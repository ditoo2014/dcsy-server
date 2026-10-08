//go:build cgo

package dao

import (
	"dicloud/internal/business/model"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindLatestByEquipmentIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE equipment_circulation_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		equipment_id INTEGER NOT NULL,
		operator_user_id INTEGER NOT NULL DEFAULT 0,
		operate_type INTEGER NOT NULL DEFAULT 0,
		operate_content TEXT NOT NULL DEFAULT '',
		operate_details BLOB NOT NULL DEFAULT '{}',
		create_time INTEGER NOT NULL DEFAULT 0
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO equipment_circulation_log
		(equipment_id, operate_type, operate_details, create_time) VALUES
		(1, 1, CAST('{"commander":"first"}' AS BLOB), 100),
		(1, 2, CAST('{"commander":"latest"}' AS BLOB), 200),
		(2, 3, CAST('{"commander":"other"}' AS BLOB), 150),
		(3, 4, CAST('{"commander":"not-requested"}' AS BLOB), 300)`).Error; err != nil {
		t.Fatal(err)
	}

	logs, ok := (&EquipmentCirculationLogDao{db: db}).FindLatestByEquipmentIDs([]int{1, 2})
	if !ok {
		t.Fatal("latest circulation query failed")
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 latest logs, got %d", len(logs))
	}
	if logs[0].EquipmentId != 1 || logs[0].OperateType != 2 {
		t.Fatalf("unexpected latest log for equipment 1: %+v", logs[0])
	}
	if logs[1].EquipmentId != 2 || logs[1].OperateType != 3 {
		t.Fatalf("unexpected latest log for equipment 2: %+v", logs[1])
	}
}

func TestFindOrdersByCreateTimeAndIDAndBindsFilters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// 模型表名带有 public 架构前缀，SQLite 附加同名内存库模拟 PostgreSQL 架构。
	if err := db.Exec(`ATTACH DATABASE ':memory:' AS public`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE public.equipment_circulation_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		equipment_id INTEGER NOT NULL,
		operator_user_id INTEGER NOT NULL DEFAULT 0,
		operate_type INTEGER NOT NULL DEFAULT 0,
		operate_content TEXT NOT NULL DEFAULT '',
		operate_details BLOB NOT NULL DEFAULT '{}',
		create_time INTEGER NOT NULL DEFAULT 0
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE equipment (
		id INTEGER PRIMARY KEY,
		equipment_type_name TEXT NOT NULL DEFAULT ''
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO equipment (id, equipment_type_name) VALUES
		(1, '锂电池'), (2, '钠电池')`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO public.equipment_circulation_log
		(equipment_id, operate_type, operate_details, create_time) VALUES
		(1, 2, CAST('{"commander":"same-time-later-id"}' AS BLOB), 100),
		(1, 1, CAST('{"commander":"same-time-first-id"}' AS BLOB), 100),
		(1, 3, CAST('{"commander":"earlier"}' AS BLOB), 90),
		(2, 4, CAST('{"commander":"other-type"}' AS BLOB), 80)`).Error; err != nil {
		t.Fatal(err)
	}

	logs, ok := (&EquipmentCirculationLogDao{db: db}).Find(model.EquipmentCirculationLog{
		EquipmentId:       1,
		EquipmentTypeName: "锂电池",
	})
	if !ok {
		t.Fatal("circulation log query failed")
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}
	if logs[0].CreateTime != 90 || logs[1].ID >= logs[2].ID || logs[1].CreateTime != 100 || logs[2].CreateTime != 100 {
		t.Fatalf("unexpected stable order: %+v", logs)
	}

	// 恶意类型名称只能作为参数值参与匹配，不能改变查询语义。
	logs, ok = (&EquipmentCirculationLogDao{db: db}).Find(model.EquipmentCirculationLog{
		EquipmentId:       1,
		EquipmentTypeName: "' OR 1=1 --",
	})
	if !ok {
		t.Fatal("parameterized type query failed")
	}
	if len(logs) != 0 {
		t.Fatalf("unescaped type filter returned %d rows", len(logs))
	}
}
