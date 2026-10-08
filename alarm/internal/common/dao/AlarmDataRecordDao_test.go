package dao

import (
	"alarm/internal/common/model"
	"fmt"
	"strings"
	"sync"
	"testing"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAlarmRecordTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.AlarmDataRecord{}); err != nil {
		t.Fatalf("migrate alarm record: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestCreateIfNotExistsUsesAlarmLifecycleKey(t *testing.T) {
	db := newAlarmRecordTestDB(t)
	dao := &AlarmDataRecordDao{db: db}
	base := model.AlarmDataRecord{
		EquipmentId: 1,
		AlarmCode:   "BATTERY_HIGH",
		BeginTime:   100,
		AlarmType:   "0",
		AlarmStage:  2,
		AlarmStatus: 1,
	}

	created, err := dao.CreateIfNotExists(base)
	if err != nil || !created {
		t.Fatalf("first create expected created=true, got created=%t err=%v", created, err)
	}
	created, err = dao.CreateIfNotExists(base)
	if err != nil || created {
		t.Fatalf("duplicate create expected created=false, got created=%t err=%v", created, err)
	}

	variants := []model.AlarmDataRecord{
		func() model.AlarmDataRecord { value := base; value.EquipmentId++; return value }(),
		func() model.AlarmDataRecord { value := base; value.AlarmCode += "_2"; return value }(),
		func() model.AlarmDataRecord { value := base; value.BeginTime++; return value }(),
		func() model.AlarmDataRecord { value := base; value.AlarmType = "1"; return value }(),
		func() model.AlarmDataRecord { value := base; value.AlarmStage++; return value }(),
	}
	for index, variant := range variants {
		created, err = dao.CreateIfNotExists(variant)
		if err != nil || !created {
			t.Fatalf("variant %d expected a distinct lifecycle, got created=%t err=%v", index, created, err)
		}
	}

	var count int64
	if err := db.Model(&model.AlarmDataRecord{}).Count(&count).Error; err != nil {
		t.Fatalf("count records: %v", err)
	}
	if count != 6 {
		t.Fatalf("expected 6 records, got %d", count)
	}
}

func TestCountActiveNormalAlarmByEquipment(t *testing.T) {
	db := newAlarmRecordTestDB(t)
	dao := &AlarmDataRecordDao{db: db}
	if err := db.Exec("DROP INDEX uni_alarm_record_lifecycle").Error; err != nil {
		t.Fatalf("drop lifecycle index: %v", err)
	}
	records := []model.AlarmDataRecord{
		{EquipmentId: 1, AlarmCode: "A", BeginTime: 1, AlarmType: "0", AlarmStage: 1, AlarmStatus: 1},
		{EquipmentId: 1, AlarmCode: "A", BeginTime: 1, AlarmType: "0", AlarmStage: 1, AlarmStatus: 1},
		{EquipmentId: 1, AlarmCode: "B", BeginTime: 2, AlarmType: "0", AlarmStage: 1, AlarmStatus: 0},
		{EquipmentId: 1, AlarmCode: "C", BeginTime: 3, AlarmType: "1", AlarmStage: 1, AlarmStatus: 1},
		{EquipmentId: 2, AlarmCode: "D", BeginTime: 4, AlarmType: "0", AlarmStage: 1, AlarmStatus: 1},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create records: %v", err)
	}

	count, ok := dao.CountActiveNormalAlarmByEquipment(1)
	if !ok || count != 1 {
		t.Fatalf("expected one active normal alarm, got count=%d ok=%t", count, ok)
	}
}

func TestFindDailyAlarmStatisticRecordsFiltersScopeTimeAndType(t *testing.T) {
	db := newAlarmRecordTestDB(t)
	dao := &AlarmDataRecordDao{db: db}
	records := []model.AlarmDataRecord{
		{EquipmentId: 1, CorpId: 10, AlarmCode: "放电过流保护", AlarmType: "0", BeginTime: 100},
		{EquipmentId: 2, CorpId: 11, AlarmCode: "充电高温告警", AlarmType: "0", BeginTime: 150},
		{EquipmentId: 3, CorpId: 10, AlarmCode: "其他告警", AlarmType: "0", BeginTime: 150},
		{EquipmentId: 4, CorpId: 10, AlarmCode: "放电高温告警", AlarmType: "1", BeginTime: 150},
		{EquipmentId: 5, CorpId: 12, AlarmCode: "放电高温告警", AlarmType: "0", BeginTime: 150},
		{EquipmentId: 6, CorpId: 10, AlarmCode: "放电高温告警", AlarmType: "0", BeginTime: 250},
		{EquipmentId: 7, CorpId: 10, AlarmCode: "放电过流", AlarmType: "10", BeginTime: 150},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create records: %v", err)
	}

	result, err := dao.FindDailyAlarmStatisticRecords([]int{1, 2, 3, 4}, 100, 200)
	if err != nil {
		t.Fatalf("find daily alarm records: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected three matching records, got %d: %#v", len(result), result)
	}
}

func TestFindRecordsFiltersHistoryStatusAndBeginTime(t *testing.T) {
	db := newAlarmRecordTestDB(t)
	dao := &AlarmDataRecordDao{db: db}
	records := []model.AlarmDataRecord{
		{EquipmentId: 1, AlarmCode: "范围内历史告警", AlarmType: "0", AlarmStatus: 0, BeginTime: 100},
		{EquipmentId: 1, AlarmCode: "范围内实时告警", AlarmType: "0", AlarmStatus: 1, BeginTime: 150},
		{EquipmentId: 1, AlarmCode: "范围外历史告警", AlarmType: "0", AlarmStatus: 0, BeginTime: 201},
		{EquipmentId: 1, AlarmCode: "范围内历史预警", AlarmType: "1", AlarmStatus: 0, BeginTime: 150},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create records: %v", err)
	}

	result, total, ok := dao.FindRecords(model.AlarmDataRecord{
		AlarmType:   "0",
		AlarmStatus: 0,
	}, 0, 10, 100, 200)
	if !ok || total != 1 || len(result) != 1 || result[0].AlarmCode != "范围内历史告警" {
		t.Fatalf("expected one history alarm in time range, got total=%d ok=%t result=%#v", total, ok, result)
	}
}

func TestFindRecordsByEquipmentFiltersTimeAndTotal(t *testing.T) {
	db := newAlarmRecordTestDB(t)
	dao := &AlarmDataRecordDao{db: db}
	records := []model.AlarmDataRecord{
		{EquipmentId: 1, AlarmCode: "边界开始", AlarmType: "1", AlarmStatus: 0, BeginTime: 100},
		{EquipmentId: 2, AlarmCode: "边界结束", AlarmType: "1", AlarmStatus: 0, BeginTime: 200},
		{EquipmentId: 3, AlarmCode: "设备范围外", AlarmType: "1", AlarmStatus: 0, BeginTime: 150},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create records: %v", err)
	}

	result, total, ok := dao.FindRecordsByEquipment(model.AlarmDataRecord{
		AlarmType:   "1",
		AlarmStatus: 0,
	}, 0, 1, []int{1, 2}, 100, 200)
	if !ok || total != 2 || len(result) != 1 {
		t.Fatalf("expected inclusive range total=2 with one paged row, got total=%d ok=%t result=%#v", total, ok, result)
	}
}

func TestCountRecordStatusExcludesRecoveredRecords(t *testing.T) {
	db := newAlarmRecordTestDB(t)
	dao := &AlarmDataRecordDao{db: db}
	records := []model.AlarmDataRecord{
		{EquipmentId: 1, AlarmCode: "已恢复告警", AlarmType: "0", AlarmStatus: 0, BeginTime: 100},
		{EquipmentId: 1, AlarmCode: "已恢复预警", AlarmType: "1", AlarmStatus: 0, BeginTime: 110},
		{EquipmentId: 2, AlarmCode: "当前告警", AlarmType: "0", AlarmStatus: 1, BeginTime: 120},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create records: %v", err)
	}

	result, ok := dao.CountRecordStatus(model.AlarmDataRecord{
		EquipmentIds: []int{1, 2},
		AlarmStatus:  1,
	})
	if !ok || len(result) != 1 || result[0].EquipmentId != 2 || result[0].AlarmCount != 1 {
		t.Fatalf("expected only active equipment status, got ok=%t result=%#v", ok, result)
	}
}

func TestCountRecordsFiltersRequestedStatus(t *testing.T) {
	db := newAlarmRecordTestDB(t)
	dao := &AlarmDataRecordDao{db: db}
	records := []model.AlarmDataRecord{
		{EquipmentId: 1, AlarmCode: "已恢复告警", AlarmType: "0", AlarmStatus: 0, BeginTime: 100},
		{EquipmentId: 1, AlarmCode: "当前告警", AlarmType: "0", AlarmStatus: 1, BeginTime: 110},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create records: %v", err)
	}

	active, ok := dao.CountRecords(model.AlarmDataRecord{
		EquipmentIds: []int{1},
		AlarmType:    "0",
		AlarmStatus:  1,
	})
	if !ok || len(active) != 1 || active[0].AlarmCount != 1 {
		t.Fatalf("expected one active alarm, got ok=%t result=%#v", ok, active)
	}
	history, ok := dao.CountRecords(model.AlarmDataRecord{
		EquipmentIds: []int{1},
		AlarmType:    "0",
		AlarmStatus:  0,
	})
	if !ok || len(history) != 1 || history[0].AlarmCount != 1 {
		t.Fatalf("expected one recovered alarm, got ok=%t result=%#v", ok, history)
	}
}

func TestFindDailyWarningStatisticRecordsFiltersScopeTimeAndType(t *testing.T) {
	db := newAlarmRecordTestDB(t)
	dao := &AlarmDataRecordDao{db: db}
	records := []model.AlarmDataRecord{
		{EquipmentId: 1, AlarmCode: "SOC过低", AlarmType: "1", AlarmStatus: 1, BeginTime: 100},
		{EquipmentId: 2, AlarmCode: "SOH过低", AlarmType: "1", AlarmStatus: 0, BeginTime: 150},
		{EquipmentId: 3, AlarmCode: "其他预警", AlarmType: "1", AlarmStatus: 1, BeginTime: 150},
		{EquipmentId: 4, AlarmCode: "普通告警", AlarmType: "0", AlarmStatus: 1, BeginTime: 150},
		{EquipmentId: 5, AlarmCode: "范围外预警", AlarmType: "1", AlarmStatus: 1, BeginTime: 250},
		{EquipmentId: 6, AlarmCode: "设备外预警", AlarmType: "1", AlarmStatus: 1, BeginTime: 150},
		{EquipmentId: 7, AlarmCode: "错误类型", AlarmType: "10", AlarmStatus: 1, BeginTime: 150},
		{EquipmentId: 8, AlarmCode: "组合类型", AlarmType: "0,1", AlarmStatus: 1, BeginTime: 150},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create records: %v", err)
	}

	result, err := dao.FindDailyWarningStatisticRecords([]int{1, 2, 3, 4, 7, 8}, 100, 200)
	if err != nil {
		t.Fatalf("find daily warning records: %v", err)
	}
	if len(result) != 4 {
		t.Fatalf("expected four matching records, got %d: %#v", len(result), result)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if !isUniqueViolation(gorm.ErrDuplicatedKey) {
		t.Fatal("expected GORM duplicate key error to match")
	}
	if !isUniqueViolation(&pq.Error{Code: pq.ErrorCode("23505")}) {
		t.Fatal("expected OpenGauss 23505 error to match")
	}
	if isUniqueViolation(&pq.Error{Code: pq.ErrorCode("42601")}) {
		t.Fatal("syntax errors must not be treated as duplicates")
	}
}

func TestConcurrentCreateIfNotExistsCreatesOneRecord(t *testing.T) {
	db := newAlarmRecordTestDB(t)
	dao := &AlarmDataRecordDao{db: db}
	record := model.AlarmDataRecord{
		EquipmentId: 1,
		AlarmCode:   "BATTERY_HIGH",
		BeginTime:   100,
		AlarmType:   "0",
		AlarmStage:  2,
		AlarmStatus: 1,
	}

	const workers = 10
	results := make(chan bool, workers)
	errors := make(chan error, workers)
	var waitGroup sync.WaitGroup
	for range workers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			created, err := dao.CreateIfNotExists(record)
			results <- created
			errors <- err
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errors)

	createdCount := 0
	for created := range results {
		if created {
			createdCount++
		}
	}
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent create failed: %v", err)
		}
	}
	if createdCount != 1 {
		t.Fatalf("expected one creator, got %d", createdCount)
	}

	var count int64
	if err := db.Model(&model.AlarmDataRecord{}).Count(&count).Error; err != nil {
		t.Fatalf("count records: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one record, got %d", count)
	}
}
