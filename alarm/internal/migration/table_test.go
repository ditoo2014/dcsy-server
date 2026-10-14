package migration

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type legacyTable struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"column:name"`
	Required string `gorm:"column:required_value"`
}

func (legacyTable) TableName() string { return "legacy_alarm_table" }

func openTableTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	return db
}

func TestCreateMissingTablesAndValidateExisting(t *testing.T) {
	db := openTableTestDB(t)
	if err := CreateMissingTablesAndValidateExisting(db, &legacyTable{}); err != nil {
		t.Fatalf("创建缺失表失败: %v", err)
	}
	if !db.Migrator().HasTable(&legacyTable{}) {
		t.Fatal("未创建缺失表")
	}

	before := sqliteDefinition(t, db, "legacy_alarm_table")
	if err := CreateMissingTablesAndValidateExisting(db, &legacyTable{}); err != nil {
		t.Fatalf("已有完整表应只读通过: %v", err)
	}
	if after := sqliteDefinition(t, db, "legacy_alarm_table"); after != before {
		t.Fatalf("重复初始化修改了已有表:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestExistingTableMissingColumnIsNotModified(t *testing.T) {
	db := openTableTestDB(t)
	if err := db.Exec(`CREATE TABLE legacy_alarm_table (
		id INTEGER PRIMARY KEY,
		name TEXT,
		legacy_only TEXT,
		CONSTRAINT legacy_alarm_name_key UNIQUE (name)
	)`).Error; err != nil {
		t.Fatalf("创建旧表失败: %v", err)
	}

	before := sqliteDefinition(t, db, "legacy_alarm_table")
	err := CreateMissingTablesAndValidateExisting(db, &legacyTable{})
	if err == nil || !strings.Contains(err.Error(), "required_value") {
		t.Fatalf("应明确报告缺少 required_value，实际错误: %v", err)
	}
	after := sqliteDefinition(t, db, "legacy_alarm_table")
	if after != before || !strings.Contains(after, "legacy_alarm_name_key") || !strings.Contains(after, "legacy_only") {
		t.Fatal("结构校验不应修改已有字段或旧约束")
	}
}

func sqliteDefinition(t *testing.T, db *gorm.DB, table string) string {
	t.Helper()
	var definition string
	if err := db.Raw(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&definition).Error; err != nil {
		t.Fatalf("读取表定义失败: %v", err)
	}
	return definition
}
