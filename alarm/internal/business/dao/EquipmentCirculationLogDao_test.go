//go:build cgo

package dao

import (
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
