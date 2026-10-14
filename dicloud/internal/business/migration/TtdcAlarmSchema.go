package migration

import (
	businessmodel "dicloud/internal/business/model"
	tolinkmodel "dicloud/internal/tolink/model"
	"errors"

	"gorm.io/gorm"
)

// EnsureTtdcAlarmRuleSchema 仅创建缺失的TTDC公共告警规则表。
// 已有表由统一迁移入口只读校验，禁止在这里执行通用 AutoMigrate。
func EnsureTtdcAlarmRuleSchema(db *gorm.DB) error {
	if db == nil {
		return errors.New("TTDC alarm rule database is nil")
	}
	return createMissingTables(db, &tolinkmodel.TtdcAlarmRule{}, &tolinkmodel.TtdcAlarmRuleLogic{})
}

// EnsureTtdcAlarmTriggerBindingSchema 使用GORM创建租户侧的规则绑定表。
func EnsureTtdcAlarmTriggerBindingSchema(db *gorm.DB) error {
	if db == nil {
		return errors.New("TTDC alarm trigger binding database is nil")
	}
	return createMissingTables(db, &businessmodel.TtdcAlarmTriggerBinding{})
}

func createMissingTables(db *gorm.DB, models ...any) error {
	for _, value := range models {
		if db.Migrator().HasTable(value) {
			continue
		}
		if err := db.Migrator().CreateTable(value); err != nil {
			return err
		}
	}
	return nil
}
