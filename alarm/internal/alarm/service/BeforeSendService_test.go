package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	alarmmodel "alarm/internal/alarm/model"
	businessmodel "alarm/internal/business/model"
	commonmodel "alarm/internal/common/model"
	"alarm/util/opengauss"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAlarmServiceTest(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.Exec(`CREATE TABLE equipment (
		id integer primary key, equipment_name text not null, site_id integer not null default 0,
		site_name text not null default '', corp_id integer not null default 0,
		alarm_trigger_num integer not null default 0
	)`).Error; err != nil {
		t.Fatalf("create equipment table: %v", err)
	}
	if err := db.AutoMigrate(&commonmodel.AlarmDataRecord{}, &alarmmodel.AlarmEventInbox{}, &alarmmodel.AlarmNotificationDelivery{}); err != nil {
		t.Fatalf("migrate alarm tables: %v", err)
	}
	statements := []string{
		`INSERT INTO equipment (id, equipment_name, site_id, site_name, corp_id, alarm_trigger_num)
			VALUES (1, 'Battery 1', 2, 'Site 1', 3, 0)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("prepare test database: %v", err)
		}
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	previousDB := opengauss.OpengaussGormDB
	opengauss.OpengaussGormDB = db
	t.Cleanup(func() {
		opengauss.OpengaussGormDB = previousDB
		_ = sqlDB.Close()
	})
	return db
}

func alarmMessage(t *testing.T, message commonmodel.AlarmMQModel, msgID string) *primitive.MessageExt {
	t.Helper()
	body, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal alarm message: %v", err)
	}
	return &primitive.MessageExt{Message: primitive.Message{Body: body}, MsgId: msgID}
}

func baseAlarmMessage() commonmodel.AlarmMQModel {
	return commonmodel.AlarmMQModel{
		SiteId: 2, EquipmentId: 1, AlarmCode: "BATTERY_HIGH", AlarmType: "0", AlarmStage: 2,
		AlarmChannel: "0,1,2", AlarmUsers: "2,1", Begin: 100, End: 0, AlarmValue: 4.2,
	}
}

func TestAcceptAlarmEventIsIdempotentByMessageAndLifecycle(t *testing.T) {
	db := setupAlarmServiceTest(t)
	message := baseAlarmMessage()
	accepted, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, message, "msg-1"))
	if err != nil || !accepted {
		t.Fatalf("first event expected accepted, accepted=%t err=%v", accepted, err)
	}
	accepted, err = AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, message, "msg-1"))
	if err != nil || accepted {
		t.Fatalf("same message expected duplicate, accepted=%t err=%v", accepted, err)
	}

	message.AlarmValue = 4.3
	accepted, err = AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, message, "msg-2"))
	if err != nil || !accepted {
		t.Fatalf("new event expected accepted, accepted=%t err=%v", accepted, err)
	}
	assertRecordAndDeliveryCounts(t, db, 1, 2, 8)

	var equipment businessmodel.Equipment
	if err := db.Table("equipment").Where("id = ?", 1).Take(&equipment).Error; err != nil {
		t.Fatalf("find equipment: %v", err)
	}
	if equipment.AlarmTriggerNum != 0 {
		t.Fatalf("equipment count must remain asynchronous, got %d", equipment.AlarmTriggerNum)
	}
}

func TestAcceptAlarmEventMarksPayloadConflict(t *testing.T) {
	db := setupAlarmServiceTest(t)
	message := baseAlarmMessage()
	if _, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, message, "msg-conflict")); err != nil {
		t.Fatalf("accept first event: %v", err)
	}
	message.AlarmValue = 9.9
	accepted, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, message, "msg-conflict"))
	if err != nil || accepted {
		t.Fatalf("conflicting duplicate expected acknowledged, accepted=%t err=%v", accepted, err)
	}
	var inbox alarmmodel.AlarmEventInbox
	if err := db.Where("event_id = ?", "msg-conflict").Take(&inbox).Error; err != nil {
		t.Fatalf("find inbox: %v", err)
	}
	if inbox.Status != alarmmodel.AlarmInboxConflict || inbox.ConflictHash == "" {
		t.Fatalf("expected conflict inbox, got %#v", inbox)
	}
}

func TestAcceptAlarmEventUsesNonEmptyAuditDefaults(t *testing.T) {
	db := setupAlarmServiceTest(t)
	if _, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, baseAlarmMessage(), "audit-defaults")); err != nil {
		t.Fatalf("accept alarm event: %v", err)
	}
	var inbox alarmmodel.AlarmEventInbox
	if err := db.Where("event_id = ?", "audit-defaults").Take(&inbox).Error; err != nil {
		t.Fatalf("find inbox: %v", err)
	}
	if inbox.ConflictHash != alarmmodel.AlarmAuditUnset {
		t.Fatalf("unexpected conflict hash default: %q", inbox.ConflictHash)
	}
	var delivery alarmmodel.AlarmNotificationDelivery
	if err := db.Where("event_id = ? AND status = ?", "audit-defaults", alarmmodel.AlarmDeliveryPending).
		Take(&delivery).Error; err != nil {
		t.Fatalf("find pending delivery: %v", err)
	}
	if delivery.LeaseOwner != alarmmodel.AlarmAuditUnset ||
		delivery.ProviderRequestID != alarmmodel.AlarmAuditUnset ||
		delivery.ResponseCode != alarmmodel.AlarmAuditUnset || delivery.Response != alarmmodel.AlarmAuditUnset ||
		delivery.LastError != alarmmodel.AlarmAuditUnset {
		t.Fatalf("delivery contains empty audit defaults: %#v", delivery)
	}
}

func TestAcceptAlarmEventRecoveryAndContinued(t *testing.T) {
	db := setupAlarmServiceTest(t)
	start := baseAlarmMessage()
	if _, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, start, "start")); err != nil {
		t.Fatalf("accept start: %v", err)
	}
	continued := start
	continued.End = 1
	if _, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, continued, "continued")); err != nil {
		t.Fatalf("accept continued: %v", err)
	}
	recovery := start
	recovery.End = 200
	recovery.AlarmValue = 3.9
	if _, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, recovery, "recovery")); err != nil {
		t.Fatalf("accept recovery: %v", err)
	}
	var record commonmodel.AlarmDataRecord
	if err := db.Take(&record).Error; err != nil {
		t.Fatalf("find alarm record: %v", err)
	}
	if record.AlarmStatus != 0 || record.EndTime != 200 {
		t.Fatalf("expected recovered record, got %#v", record)
	}
	assertRecordAndDeliveryCounts(t, db, 1, 3, 11)
}

func TestAcceptAlarmEventRecoverySurvivesAlarmLevelChange(t *testing.T) {
	db := setupAlarmServiceTest(t)
	start := baseAlarmMessage()
	if _, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, start, "level-start")); err != nil {
		t.Fatalf("accept start: %v", err)
	}

	recovery := start
	recovery.AlarmStage = start.AlarmStage + 1
	recovery.End = 200
	recovery.AlarmValue = 0
	if _, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, recovery, "level-recovery")); err != nil {
		t.Fatalf("accept recovery after level change: %v", err)
	}

	var record commonmodel.AlarmDataRecord
	if err := db.Take(&record).Error; err != nil {
		t.Fatalf("find alarm record: %v", err)
	}
	if record.AlarmStatus != 0 || record.EndTime != recovery.End || record.RecoverValue != "0" {
		t.Fatalf("expected recovered original record, got %#v", record)
	}
}

func TestAcceptAlarmEventRollsBackWhenOutboxFails(t *testing.T) {
	db := setupAlarmServiceTest(t)
	if err := db.Migrator().DropTable(&alarmmodel.AlarmNotificationDelivery{}); err != nil {
		t.Fatalf("drop delivery table: %v", err)
	}
	if _, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, baseAlarmMessage(), "rollback")); err == nil {
		t.Fatal("expected outbox failure")
	}
	var recordCount, inboxCount int64
	_ = db.Model(&commonmodel.AlarmDataRecord{}).Count(&recordCount).Error
	_ = db.Model(&alarmmodel.AlarmEventInbox{}).Count(&inboxCount).Error
	if recordCount != 0 || inboxCount != 0 {
		t.Fatalf("transaction must roll back, records=%d inbox=%d", recordCount, inboxCount)
	}
}

func TestConcurrentAlarmEventsCreateOneLifecycle(t *testing.T) {
	db := setupAlarmServiceTest(t)
	const workers = 8
	var waitGroup sync.WaitGroup
	errors := make(chan error, workers)
	for index := 0; index < workers; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			_, err := AcceptAlarmEvent(&gin.Context{}, alarmMessage(t, baseAlarmMessage(), fmt.Sprintf("concurrent-%d", index)))
			errors <- err
		}(index)
	}
	waitGroup.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent accept failed: %v", err)
		}
	}
	var count int64
	if err := db.Model(&commonmodel.AlarmDataRecord{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected one lifecycle, count=%d err=%v", count, err)
	}
}

func assertRecordAndDeliveryCounts(t *testing.T, db *gorm.DB, records, inboxes, deliveries int64) {
	t.Helper()
	checks := []struct {
		model interface{}
		want  int64
	}{
		{&commonmodel.AlarmDataRecord{}, records},
		{&alarmmodel.AlarmEventInbox{}, inboxes},
		{&alarmmodel.AlarmNotificationDelivery{}, deliveries},
	}
	for _, check := range checks {
		var count int64
		if err := db.Model(check.model).Count(&count).Error; err != nil || count != check.want {
			t.Fatalf("unexpected count for %T: got=%d want=%d err=%v", check.model, count, check.want, err)
		}
	}
}
