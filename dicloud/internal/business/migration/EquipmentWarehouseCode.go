package migration

import (
	"errors"
	"fmt"
	"strings"

	"dicloud/internal/business/model"

	"gorm.io/gorm"
)

const (
	equipmentCodeIndexName = "uk_equipment_equipment_code"
	warehouseCodeIndexName = "uk_warehouse_warehouse_code"
)

type duplicateCode struct {
	Code  string `gorm:"column:code"`
	Count int64  `gorm:"column:count"`
}

// EnsureEquipmentWarehouseCodeIndexes 创建业务模块依赖的编码唯一索引。
// 若现有数据中存在重复编码，则先返回明确错误，不自动删除或修改业务数据。
func EnsureEquipmentWarehouseCodeIndexes(db *gorm.DB) error {
	if db == nil {
		return errors.New("business database is nil")
	}

	if duplicates, err := findDuplicateCodes(db, "equipment", "equipment_code"); err != nil {
		return fmt.Errorf("check duplicate equipment codes: %w", err)
	} else if len(duplicates) > 0 {
		return fmt.Errorf("cannot create equipment code unique index: duplicate equipment codes: %s", formatDuplicateCodes(duplicates))
	}

	if duplicates, err := findDuplicateCodes(db, "warehouse", "warehouse_code"); err != nil {
		return fmt.Errorf("check duplicate warehouse codes: %w", err)
	} else if len(duplicates) > 0 {
		return fmt.Errorf("cannot create warehouse code unique index: duplicate warehouse codes: %s", formatDuplicateCodes(duplicates))
	}

	if err := ensureIndex(db, &model.Equipment{}, equipmentCodeIndexName); err != nil {
		return fmt.Errorf("create equipment code unique index: %w", err)
	}
	if err := ensureIndex(db, &model.Warehouse{}, warehouseCodeIndexName); err != nil {
		return fmt.Errorf("create warehouse code unique index: %w", err)
	}

	return nil
}

func findDuplicateCodes(db *gorm.DB, tableName, columnName string) ([]duplicateCode, error) {
	var duplicates []duplicateCode
	where := fmt.Sprintf("%s IS NOT NULL AND TRIM(%s) <> '' AND TRIM(%s) <> ?", columnName, columnName, columnName)
	query := db.Table(tableName).
		Select(fmt.Sprintf("TRIM(%s) AS code, COUNT(*) AS count", columnName)).
		Where(where, "-").
		Group(fmt.Sprintf("TRIM(%s)", columnName)).
		Having("COUNT(*) > ?", 1).
		Scan(&duplicates)
	return duplicates, query.Error
}

func ensureIndex(db *gorm.DB, value interface{}, name string) error {
	if db.Migrator().HasIndex(value, name) {
		return nil
	}
	return db.Migrator().CreateIndex(value, name)
}

func formatDuplicateCodes(duplicates []duplicateCode) string {
	values := make([]string, 0, len(duplicates))
	for _, duplicate := range duplicates {
		values = append(values, fmt.Sprintf("%q (%d)", duplicate.Code, duplicate.Count))
	}
	return strings.Join(values, ", ")
}
