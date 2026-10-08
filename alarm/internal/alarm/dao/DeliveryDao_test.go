package dao

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	alarmmodel "alarm/internal/alarm/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDeliveryTestDAO(t *testing.T) (*DeliveryDAO, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&alarmmodel.AlarmEventInbox{}, &alarmmodel.AlarmNotificationDelivery{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	return NewDeliveryDAOWithDB(db), db
}

func TestDeliveryClaimAndLifecycleOrder(t *testing.T) {
	dao, db := newDeliveryTestDAO(t)
	now := time.Now()
	items := []alarmmodel.AlarmNotificationDelivery{
		{EventID: "start", LifecycleKey: "life", Channel: "0", RecipientKey: "one", Status: alarmmodel.AlarmDeliveryPending, MaxAttempts: 5, NextRetryAt: now.Unix(), CreateTime: now, UpdateTime: now},
		{EventID: "recover", LifecycleKey: "life", Channel: "0", RecipientKey: "one", Status: alarmmodel.AlarmDeliveryPending, MaxAttempts: 5, NextRetryAt: now.Unix(), CreateTime: now, UpdateTime: now},
	}
	if err := db.Create(&items).Error; err != nil {
		t.Fatalf("create deliveries: %v", err)
	}
	due, err := dao.FindDue(now, 20)
	if err != nil || len(due) != 1 || due[0].EventID != "start" {
		t.Fatalf("expected only start due, due=%#v err=%v", due, err)
	}
	claimed, err := dao.TryClaim(due[0].ID, "owner", now)
	if err != nil || !claimed {
		t.Fatalf("claim start: claimed=%t err=%v", claimed, err)
	}
	claimed, err = dao.TryClaim(due[0].ID, "other", now)
	if err != nil || claimed {
		t.Fatalf("second claim must fail, claimed=%t err=%v", claimed, err)
	}
	if err := dao.UpdateAttempt(due[0].ID, "owner", map[string]interface{}{
		"status": alarmmodel.AlarmDeliverySuccess, "next_retry_at": 0,
	}); err != nil {
		t.Fatalf("mark success: %v", err)
	}
	due, err = dao.FindDue(now.Add(time.Second), 20)
	if err != nil || len(due) != 1 || due[0].EventID != "recover" {
		t.Fatalf("expected recovery after start terminal, due=%#v err=%v", due, err)
	}
}

func TestDeliveryReclaimsExpiredLeaseAndStopsAtMaxAttempts(t *testing.T) {
	dao, db := newDeliveryTestDAO(t)
	now := time.Now()
	item := alarmmodel.AlarmNotificationDelivery{
		EventID: "event", LifecycleKey: "life", Channel: "0", RecipientKey: "one",
		Status: alarmmodel.AlarmDeliverySending, AttemptCount: 4, MaxAttempts: 5,
		LeaseOwner: "dead-owner", LeaseUntil: now.Add(-time.Second).Unix(), CreateTime: now, UpdateTime: now,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	due, err := dao.FindDue(now, 20)
	if err != nil || len(due) != 1 {
		t.Fatalf("expected expired lease due, due=%#v err=%v", due, err)
	}
	claimed, err := dao.TryClaim(item.ID, "new-owner", now)
	if err != nil || !claimed {
		t.Fatalf("reclaim delivery: claimed=%t err=%v", claimed, err)
	}
	if err := db.Model(&alarmmodel.AlarmNotificationDelivery{}).Where("id = ?", item.ID).
		Updates(map[string]interface{}{"lease_until": now.Add(-time.Second).Unix()}).Error; err != nil {
		t.Fatalf("expire final lease: %v", err)
	}
	if _, err := dao.FindDue(now, 20); err != nil {
		t.Fatalf("finalize exhausted lease: %v", err)
	}
	if err := db.First(&item, item.ID).Error; err != nil || item.Status != alarmmodel.AlarmDeliveryDead {
		t.Fatalf("expected dead delivery, status=%s err=%v", item.Status, err)
	}
}

func TestAutoMigrateReliabilitySchemaCreatesNamedIndexes(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := AutoMigrateReliabilitySchema(db); err != nil {
		t.Fatalf("auto migrate reliability schema: %v", err)
	}
	if err := ValidateReliabilitySchema(db); err != nil {
		t.Fatalf("validate complete reliability schema: %v", err)
	}
}

func TestDeleteInboxWaitsUntilDeliveriesAreRemoved(t *testing.T) {
	dao, db := newDeliveryTestDAO(t)
	now := time.Now()
	old := now.AddDate(0, 0, -100)
	inbox := alarmmodel.AlarmEventInbox{
		EventID: "retained", PayloadHash: "hash", Payload: "{}", Status: alarmmodel.AlarmInboxAccepted,
		CreateTime: old, UpdateTime: old,
	}
	if err := db.Create(&inbox).Error; err != nil {
		t.Fatalf("create inbox: %v", err)
	}
	delivery := alarmmodel.AlarmNotificationDelivery{
		EventID: inbox.EventID, LifecycleKey: "life", Channel: "0", RecipientKey: "one",
		Status: alarmmodel.AlarmDeliverySuccess, CreateTime: old, UpdateTime: now,
	}
	if err := db.Create(&delivery).Error; err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	cutoff := now.AddDate(0, 0, -90)
	if err := dao.DeleteInboxBefore(cutoff); err != nil {
		t.Fatalf("cleanup inbox with retained delivery: %v", err)
	}
	if err := db.First(&alarmmodel.AlarmEventInbox{}, inbox.ID).Error; err != nil {
		t.Fatalf("inbox must remain while a delivery exists: %v", err)
	}
	if err := db.Model(&delivery).Update("update_time", old).Error; err != nil {
		t.Fatalf("age terminal delivery: %v", err)
	}
	if err := dao.DeleteFinishedBefore(cutoff); err != nil {
		t.Fatalf("cleanup terminal delivery: %v", err)
	}
	if err := dao.DeleteInboxBefore(cutoff); err != nil {
		t.Fatalf("cleanup inbox after delivery removal: %v", err)
	}
	if err := db.First(&alarmmodel.AlarmEventInbox{}, inbox.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected inbox cleanup, err=%v", err)
	}
}
