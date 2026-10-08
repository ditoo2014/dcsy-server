//go:build cgo

package dao

import (
	"dicloud/internal/business/model"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 使用内存 SQLite 的 public 附加库测试真实查询结果，不访问任何业务数据库。
func TestMaintainRecordScopedRowsCountPaginationAndEmpty(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	for _, statement := range []string{
		`ATTACH DATABASE ':memory:' AS public`,
		`CREATE TABLE public.equipment (id INTEGER PRIMARY KEY, corp_id INTEGER, equipment_code TEXT, equipment_name TEXT, equipment_type_name TEXT, position_status INTEGER, site_id INTEGER)`,
		`CREATE TABLE public.site (id INTEGER PRIMARY KEY, site_serial_number TEXT)`,
		`CREATE TABLE public.equipment_maintain_record (id INTEGER PRIMARY KEY, equipment_id INTEGER, create_time INTEGER, maintain_content TEXT)`,
		`INSERT INTO public.site VALUES (1, 'ST-1')`,
		`INSERT INTO public.equipment VALUES (1, 10, 'BAT-A', '电池A', '钠电池', 3, 1), (2, 20, 'BAT-B', '电池B', '锂电池', 4, 1)`,
		`INSERT INTO public.equipment_maintain_record VALUES (1, 1, 10, 'A1'), (2, 1, 20, 'A2'), (3, 1, 20, 'A3'), (4, 2, 20, 'B1'), (5, 2, 90, 'B2')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	dao := &EquipmentMaintainRecordDao{db: db}
	filter := MaintainRecordFilter{StartTime: 0, EndTime: 50, CorpIDs: []int{10}, MinimumCount: 3}
	rows, total, err := dao.FindScoped(filter, 0, 2)
	if err != nil || total != 3 || len(rows) != 2 || rows[0].ID != 3 || rows[1].ID != 2 {
		t.Fatalf("scoped page rows=%+v total=%d err=%v", rows, total, err)
	}
	if rows[0].EquipmentTypeName != "钠电池" || rows[0].EquipmentCode != "BAT-A" || rows[0].SiteSerialNumber != "ST-1" {
		t.Fatalf("metadata missing: %+v", rows[0])
	}
	if rows[0].EquipmentName != "电池A" || rows[0].PositionStatus == nil || *rows[0].PositionStatus != 3 {
		t.Fatalf("current equipment metadata missing: %+v", rows[0])
	}
	rows, total, err = dao.FindScoped(filter, 2, 2)
	if err != nil || total != 3 || len(rows) != 1 || rows[0].ID != 1 {
		t.Fatalf("second page rows=%+v total=%d err=%v", rows, total, err)
	}
	filter.EquipmentID, filter.MinimumCount = 2, 0
	rows, total, err = dao.FindScoped(filter, 0, 0)
	if err != nil || total != 0 || rows == nil || len(rows) != 0 {
		t.Fatalf("out-of-scope equipment rows=%+v total=%d err=%v", rows, total, err)
	}
	filter.EquipmentID, filter.MinimumCount, filter.IsAdmin = 0, 0, true
	rows, total, err = dao.FindScoped(filter, 0, 0)
	if err != nil || total != 4 || len(rows) != 4 {
		t.Fatalf("admin export rows=%+v total=%d err=%v", rows, total, err)
	}
	filter.MinimumCount = 2
	rows, total, err = dao.FindScoped(filter, 0, 0)
	if err != nil || total != 3 || len(rows) != 3 {
		t.Fatalf("time-scoped minimum count rows=%+v total=%d err=%v", rows, total, err)
	}
	filter.MinimumCount = 0
	filter.IsAdmin, filter.CorpIDs = false, nil
	rows, total, err = dao.FindScoped(filter, 0, 0)
	if err != nil || total != 0 || rows == nil || len(rows) != 0 {
		t.Fatalf("empty permission rows=%+v total=%d err=%v", rows, total, err)
	}
	// 旧设备没有有效编码时允许名称匹配；有编码时不能按名称误匹配。
	if err := db.Exec(`UPDATE public.equipment SET equipment_code = '-', corp_id = 0 WHERE id = 2`).Error; err != nil {
		t.Fatal(err)
	}
	filter = MaintainRecordFilter{StartTime: 0, EndTime: 100, CorpIDs: []int{10, 0}, EquipmentCode: "电池B", MinimumCount: 2}
	rows, total, err = dao.FindScoped(filter, 0, 0)
	if err != nil || total != 2 || len(rows) != 2 || rows[0].EquipmentName != "电池B" {
		t.Fatalf("legacy code fallback rows=%+v total=%d err=%v", rows, total, err)
	}
	filter.EquipmentCode = "电池A"
	rows, total, err = dao.FindScoped(filter, 0, 0)
	if err != nil || total != 0 || len(rows) != 0 {
		t.Fatalf("coded equipment should not match its name: rows=%+v total=%d err=%v", rows, total, err)
	}
}

// 同一条网页新增记录导出后再导入，按导出使用的故障描述判断重复，不把详情误作描述。
func TestMaintainRecordDuplicateUsesExportedDescription(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	for _, statement := range []string{
		`ATTACH DATABASE ':memory:' AS public`,
		`CREATE TABLE public.equipment_maintain_record (id INTEGER PRIMARY KEY, equipment_id INTEGER, create_time INTEGER, maintain_title TEXT, maintain_content TEXT)`,
		`INSERT INTO public.equipment_maintain_record VALUES (1, 1, 100, '故障描述', '故障详情'), (2, 1, 200, '-', '旧故障描述')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	dao := &EquipmentMaintainRecordDao{db: db}
	for _, test := range []struct {
		time  int64
		title string
		count int64
	}{
		{100, "故障描述", 1}, {100, "故障详情", 0}, {200, "旧故障描述", 1}, {201, "旧故障描述", 0},
	} {
		count, err := dao.CountExactRecord(model.EquipmentMaintainRecord{EquipmentId: 1, CreateTime: test.time, MaintainTitle: test.title})
		if err != nil || count != test.count {
			t.Fatalf("case=%+v count=%d err=%v", test, count, err)
		}
	}
}
