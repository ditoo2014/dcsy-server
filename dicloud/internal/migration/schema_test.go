package migration

import (
	"fmt"
	"strings"
	"testing"

	commonmodel "dicloud/internal/common/model"
	tolinkmodel "dicloud/internal/tolink/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestMigrateDefaultCreatesAllRuntimeTables 验证空数据库能够一次性创建
// DCSY 后端实际使用的认证、业务、TTDC 配置及动态统计表。
func TestMigrateDefaultCreatesAllRuntimeTables(t *testing.T) {
	dsn := fmt.Sprintf("file:migration_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 SQLite 测试库失败: %v", err)
	}
	// 生产库使用 OpenGauss 的 public schema；SQLite 通过附加内存库模拟该 schema。
	if err := db.Exec("ATTACH DATABASE ':memory:' AS public").Error; err != nil {
		t.Fatalf("创建 public 测试 schema 失败: %v", err)
	}

	// 从空库执行完整迁移，验证首次部署不依赖手工建表。
	if err := MigrateDefault(db); err != nil {
		t.Fatalf("首次自动迁移失败: %v", err)
	}

	models := append([]any{}, authModels...)
	models = append(models, tenantModels...)
	models = append(models, globalTtdcModels...)
	for _, value := range models {
		if !hasMigratedTable(db, value) {
			t.Fatalf("自动迁移缺少模型对应的数据表: %T", value)
		}
	}

	for targetType := 0; targetType <= 2; targetType++ {
		for _, dateType := range []string{"hourly", "daily", "monthly"} {
			for valueType := 0; valueType <= 1; valueType++ {
				statistic := commonmodel.StatisticData{
					TargetType:         targetType,
					StatisticDateType:  dateType,
					StatisticValueType: valueType,
				}
				if !db.Migrator().HasTable(statistic.GetTableName()) {
					t.Fatalf("自动迁移缺少动态统计表: %s", statistic.GetTableName())
				}
			}
		}
	}

	var schemaState tolinkmodel.TtdcSchemaVersionRecord
	if err := db.Where("component = ?", tolinkmodel.TtdcSchemaComponent).First(&schemaState).Error; err != nil {
		t.Fatalf("读取 TTDC 结构版本失败: %v", err)
	}
	if schemaState.Version != tolinkmodel.TtdcSchemaVersion || schemaState.Status != tolinkmodel.TtdcSchemaReady {
		t.Fatalf("TTDC 结构版本状态不正确: version=%d status=%s", schemaState.Version, schemaState.Status)
	}
}

// hasMigratedTable 兼容 SQLite 附加库对 public.table 的检测限制。
// 生产环境仍由 GORM 的 OpenGauss Migrator 负责结构判断和幂等迁移。
func hasMigratedTable(db *gorm.DB, value any) bool {
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(value); err != nil || statement.Schema == nil {
		return false
	}
	parts := strings.SplitN(statement.Schema.Table, ".", 2)
	if len(parts) != 2 {
		return db.Migrator().HasTable(value)
	}
	var count int64
	err := db.Raw(
		fmt.Sprintf("SELECT COUNT(*) FROM %s.sqlite_master WHERE type = 'table' AND name = ?", parts[0]),
		parts[1],
	).Scan(&count).Error
	return err == nil && count == 1
}
