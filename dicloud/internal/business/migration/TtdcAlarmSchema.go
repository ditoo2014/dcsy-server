package migration

import (
	businessmodel "dicloud/internal/business/model"
	tolinkmodel "dicloud/internal/tolink/model"
	"errors"

	"gorm.io/gorm"
)

// EnsureTtdcAlarmRuleSchema 使用GORM自动创建TTDC公共告警规则表。
// 保留该函数是为了兼容现有调用方，但不再要求人工执行SQL文件。
func EnsureTtdcAlarmRuleSchema(db *gorm.DB) error {
	if db == nil {
		return errors.New("TTDC alarm rule database is nil")
	}
	return db.AutoMigrate(&tolinkmodel.TtdcAlarmRule{}, &tolinkmodel.TtdcAlarmRuleLogic{})
}

// EnsureTtdcAlarmTriggerBindingSchema 使用GORM创建租户侧的规则绑定表。
func EnsureTtdcAlarmTriggerBindingSchema(db *gorm.DB) error {
	if db == nil {
		return errors.New("TTDC alarm trigger binding database is nil")
	}
	return db.AutoMigrate(&businessmodel.TtdcAlarmTriggerBinding{})
}
