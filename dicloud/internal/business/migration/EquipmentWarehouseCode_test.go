package migration

import (
	"strings"
	"testing"

	"dicloud/internal/business/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCodeMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	if err := db.Exec("CREATE TABLE equipment (id INTEGER PRIMARY KEY, equipment_code TEXT)").Error; err != nil {
		t.Fatalf("create equipment table: %v", err)
	}
	if err := db.Exec("CREATE TABLE warehouse (id INTEGER PRIMARY KEY, warehouse_code TEXT)").Error; err != nil {
		t.Fatalf("create warehouse table: %v", err)
	}
	return db
}

func TestEnsureEquipmentWarehouseCodeIndexesCreatesAndIsIdempotent(t *testing.T) {
	db := newCodeMigrationTestDB(t)

	if err := EnsureEquipmentWarehouseCodeIndexes(db); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}
	if !db.Migrator().HasIndex(&model.Equipment{}, equipmentCodeIndexName) {
		t.Fatal("equipment code index was not created")
	}
	if !db.Migrator().HasIndex(&model.Warehouse{}, warehouseCodeIndexName) {
		t.Fatal("warehouse code index was not created")
	}
	if err := EnsureEquipmentWarehouseCodeIndexes(db); err != nil {
		t.Fatalf("ensure indexes a second time: %v", err)
	}

	if err := db.Exec("INSERT INTO equipment (equipment_code) VALUES (?)", "BAT-001").Error; err != nil {
		t.Fatalf("insert first equipment code: %v", err)
	}
	if err := db.Exec("INSERT INTO equipment (equipment_code) VALUES (?)", " BAT-001 ").Error; err == nil {
		t.Fatal("expected trimmed duplicate equipment code to be rejected")
	}
	if err := db.Exec("INSERT INTO equipment (equipment_code) VALUES (?)", "-").Error; err != nil {
		t.Fatalf("insert first placeholder code: %v", err)
	}
	if err := db.Exec("INSERT INTO equipment (equipment_code) VALUES (?)", "-").Error; err != nil {
		t.Fatalf("placeholder code should not be unique: %v", err)
	}
	if err := db.Exec("INSERT INTO equipment (equipment_code) VALUES (''), (''), (NULL), (NULL)").Error; err != nil {
		t.Fatalf("blank and null codes should not be unique: %v", err)
	}

	if err := db.Exec("INSERT INTO warehouse (warehouse_code) VALUES (?)", "WH-001").Error; err != nil {
		t.Fatalf("insert first warehouse code: %v", err)
	}
	if err := db.Exec("INSERT INTO warehouse (warehouse_code) VALUES (?)", " WH-001 ").Error; err == nil {
		t.Fatal("expected trimmed duplicate warehouse code to be rejected")
	}
}

func TestEnsureEquipmentWarehouseCodeIndexesRejectsExistingDuplicates(t *testing.T) {
	db := newCodeMigrationTestDB(t)
	if err := db.Exec("INSERT INTO equipment (equipment_code) VALUES (?)", "BAT-002").Error; err != nil {
		t.Fatalf("insert first duplicate: %v", err)
	}
	if err := db.Exec("INSERT INTO equipment (equipment_code) VALUES (?)", " BAT-002 ").Error; err != nil {
		t.Fatalf("insert second duplicate: %v", err)
	}

	err := EnsureEquipmentWarehouseCodeIndexes(db)
	if err == nil || !strings.Contains(err.Error(), "duplicate equipment codes") {
		t.Fatalf("expected duplicate equipment code error, got %v", err)
	}
	if db.Migrator().HasIndex(&model.Equipment{}, equipmentCodeIndexName) {
		t.Fatal("equipment code index should not be created when duplicates exist")
	}
}

func TestEnsureEquipmentWarehouseCodeIndexesRejectsExistingWarehouseDuplicates(t *testing.T) {
	db := newCodeMigrationTestDB(t)
	if err := db.Exec("INSERT INTO warehouse (warehouse_code) VALUES (?), (?)", "WH-002", " WH-002 ").Error; err != nil {
		t.Fatalf("insert duplicate warehouse codes: %v", err)
	}

	err := EnsureEquipmentWarehouseCodeIndexes(db)
	if err == nil || !strings.Contains(err.Error(), "duplicate warehouse codes") {
		t.Fatalf("expected duplicate warehouse code error, got %v", err)
	}
	if db.Migrator().HasIndex(&model.Equipment{}, equipmentCodeIndexName) {
		t.Fatal("no index should be created when warehouse duplicates exist")
	}
}
