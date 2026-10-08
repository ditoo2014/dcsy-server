// Package migration 统一管理 DCSY 使用的 OpenGauss 表结构。
//
// 所有服务必须先完成这里的迁移，再启动 HTTP、定时任务或消息消费者，
// 从而保证新环境不再依赖人工执行建表 SQL。
package migration

import (
	"fmt"
	"time"

	businessmigration "dicloud/internal/business/migration"
	businessmodel "dicloud/internal/business/model"
	commonmodel "dicloud/internal/common/model"
	tolinkmodel "dicloud/internal/tolink/model"
	authmodel "dicloud/models/model"
	"dicloud/util/opengauss"

	"gorm.io/gorm"
)

// authModels 是登录、组织、角色和权限接口实际使用的表。
var authModels = []any{
	&authmodel.User{},
	&authmodel.Permission{},
	&authmodel.Role{},
	&authmodel.UserRole{},
	&authmodel.RolePermission{},
	&authmodel.Corp{},
	&authmodel.UserCorp{},
	&authmodel.SystemLog{},
	&authmodel.WeakPassword{},
}

// tenantModels 是每个 DCSY 租户独立保存的业务、TTDC网关和触发数据。
var tenantModels = []any{
	&businessmodel.Site{},
	&businessmodel.SiteGroup{},
	&businessmodel.Warehouse{},
	&businessmodel.Equipment{},
	&businessmodel.EquipmentType{},
	&businessmodel.DataDetails{},
	&businessmodel.EquipmentCirculationLog{},
	&businessmodel.EquipmentGeographyPosition{},
	&businessmodel.EquipmentMaintainRecord{},
	&businessmodel.EquipmentHealthStandard{},
	&businessmodel.TtdcAlarmTriggerBinding{},
	&tolinkmodel.Gateway{},
	&tolinkmodel.GatewayData{},
	&tolinkmodel.GatewayModuleTtdc{},
	&tolinkmodel.ModuleTtdcConfig{},
	&tolinkmodel.ModuleTtdcData{},
	&tolinkmodel.Stream{},
	&tolinkmodel.StreamLabelUnit{},
	&commonmodel.TriggerConfig{},
	&commonmodel.RealtimeLogic{},
	&commonmodel.RealtimeStrMatch{},
	&commonmodel.AlarmUser{},
	&commonmodel.StatisticConfig{},
	&commonmodel.GeneralTriggerConfig{},
	&commonmodel.GeneralMarktrigger{},
	&commonmodel.GeneralRealtimeLogic{},
	&commonmodel.GeneralRealtimeStrMatch{},
	&commonmodel.GeneralAlarmUser{},
	&commonmodel.GeneralStatisticConfig{},
	&commonmodel.AlarmDataRecord{},
}

// globalTtdcModels 只保存在默认库，供所有租户共享TTDC公共模板。
var globalTtdcModels = []any{
	&tolinkmodel.TtdcTemplateGroup{},
	&tolinkmodel.TtdcTemplate{},
	&tolinkmodel.TtdcTemplateTopic{},
	&tolinkmodel.TtdcTemplatePoint{},
	&tolinkmodel.GatewayTtdcAddressGroupBinding{},
	&tolinkmodel.GatewayTtdcTemplateBinding{},
	&tolinkmodel.TtdcConfigRevision{},
	&tolinkmodel.TtdcSchemaVersionRecord{},
	&tolinkmodel.TtdcAlarmRule{},
	&tolinkmodel.TtdcAlarmRuleLogic{},
}

// Initialize 连接默认库，并自动迁移默认库和配置中的全部租户库。
func Initialize() (*gorm.DB, error) {
	if err := opengauss.InitOpenGaussDB(); err != nil {
		return nil, fmt.Errorf("连接默认OpenGauss数据库失败: %w", err)
	}
	defaultDB := opengauss.InitGormDB()
	if defaultDB == nil {
		return nil, fmt.Errorf("初始化默认GORM连接失败")
	}
	if err := MigrateDefault(defaultDB); err != nil {
		return nil, fmt.Errorf("迁移默认数据库失败: %w", err)
	}

	for _, dbName := range opengauss.ConfiguredRenterDBNames() {
		tenantDB := opengauss.GetRenterDB(dbName)
		if tenantDB == nil {
			return nil, fmt.Errorf("初始化租户数据库失败: %s", dbName)
		}
		if err := MigrateTenant(tenantDB); err != nil {
			return nil, fmt.Errorf("迁移租户数据库%s失败: %w", dbName, err)
		}
	}
	return defaultDB, nil
}

// MigrateDefault 迁移默认库；默认库同时承担无租户场景的业务数据存储。
func MigrateDefault(db *gorm.DB) error {
	if err := businessmigration.EnsureBusinessRecordTimeColumns(db); err != nil {
		return err
	}
	models := append([]any{}, authModels...)
	models = append(models, tenantModels...)
	models = append(models, globalTtdcModels...)
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	if err := migrateStatisticTables(db); err != nil {
		return err
	}
	if err := businessmigration.EnsureEquipmentWarehouseCodeIndexes(db); err != nil {
		return err
	}
	if err := ensureBusinessPermissions(db); err != nil {
		return err
	}
	return ensureTtdcSchemaReady(db)
}

// MigrateTenant 迁移单个租户库。TTDC公共模板仍只使用默认库。
func MigrateTenant(db *gorm.DB) error {
	if err := businessmigration.EnsureBusinessRecordTimeColumns(db); err != nil {
		return err
	}
	models := append([]any{}, authModels...)
	models = append(models, tenantModels...)
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}
	if err := migrateStatisticTables(db); err != nil {
		return err
	}
	if err := businessmigration.EnsureEquipmentWarehouseCodeIndexes(db); err != nil {
		return err
	}
	return ensureBusinessPermissions(db)
}

// migrateStatisticTables 根据模型支持的维度生成全部合法动态表名，
// 不接收外部任意表名，避免SQL注入和散落的表名硬编码。
func migrateStatisticTables(db *gorm.DB) error {
	for targetType := 0; targetType <= 2; targetType++ {
		for _, dateType := range []string{"hourly", "daily", "monthly"} {
			for valueType := 0; valueType <= 1; valueType++ {
				model := commonmodel.StatisticData{
					TargetType:         targetType,
					StatisticDateType:  dateType,
					StatisticValueType: valueType,
				}
				if err := db.Table(model.GetTableName()).AutoMigrate(&commonmodel.StatisticData{}); err != nil {
					return fmt.Errorf("迁移统计表%s失败: %w", model.GetTableName(), err)
				}
			}
		}
	}
	return nil
}

// ensureTtdcSchemaReady 创建或更新TTDC结构版本记录。
// AutoMigrate成功后才写入ready，解析程序据此判断模板结构可安全使用。
func ensureTtdcSchemaReady(db *gorm.DB) error {
	now := time.Now()
	state := tolinkmodel.TtdcSchemaVersionRecord{Component: tolinkmodel.TtdcSchemaComponent}
	return db.Where("component = ?", tolinkmodel.TtdcSchemaComponent).
		Assign(map[string]any{
			"version":    tolinkmodel.TtdcSchemaVersion,
			"status":     tolinkmodel.TtdcSchemaReady,
			"updated_at": now,
		}).
		FirstOrCreate(&state).Error
}
