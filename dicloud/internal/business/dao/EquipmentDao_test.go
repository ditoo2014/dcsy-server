//go:build cgo

package dao

import (
	"testing"

	"dicloud/internal/business/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestImportEquipmentsRollsBackWholeBatch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Equipment{}); err != nil {
		t.Fatal(err)
	}
	existing := model.Equipment{EquipmentCode: "BAT-OLD", EquipmentName: "BAT-OLD", EquipmentTypeName: "原类型"}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER reject_equipment_update BEFORE UPDATE ON equipment
		BEGIN SELECT RAISE(ABORT, 'reject update'); END`).Error; err != nil {
		t.Fatal(err)
	}

	created := []model.Equipment{{EquipmentCode: "BAT-NEW", EquipmentName: "BAT-NEW", EquipmentTypeName: "新类型"}}
	updated := []model.Equipment{{EquipmentCode: "BAT-OLD", EquipmentName: "BAT-OLD", EquipmentTypeName: "修改类型"}}
	if _, ok := (&EquipmentDAO{db: db}).ImportEquipments(created, updated); ok {
		t.Fatal("import succeeded despite update failure")
	}

	var createdCount int64
	if err := db.Model(&model.Equipment{}).Where("equipment_code = ?", "BAT-NEW").Count(&createdCount).Error; err != nil {
		t.Fatal(err)
	}
	if createdCount != 0 {
		t.Fatalf("new row was not rolled back, count = %d", createdCount)
	}
	var saved model.Equipment
	if err := db.Where("equipment_code = ?", "BAT-OLD").First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.EquipmentTypeName != "原类型" {
		t.Fatalf("existing row changed to %q", saved.EquipmentTypeName)
	}
}
