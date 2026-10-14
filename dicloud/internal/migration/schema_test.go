package migration

import (
	"fmt"
	"strings"
	"testing"

	commonmodel "dicloud/internal/common/model"
	tolinkmodel "dicloud/internal/tolink/model"
	authmodel "dicloud/models/model"

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

	// 重复执行不得修改已有表、删除旧约束或重复写入初始化数据。
	var permissionCount int64
	if err := db.Model(&authmodel.Permission{}).Count(&permissionCount).Error; err != nil {
		t.Fatalf("统计首次权限数据失败: %v", err)
	}
	if err := MigrateDefault(db); err != nil {
		t.Fatalf("重复迁移应保持幂等: %v", err)
	}
	var secondPermissionCount int64
	if err := db.Model(&authmodel.Permission{}).Count(&secondPermissionCount).Error; err != nil {
		t.Fatalf("统计重复迁移后的权限数据失败: %v", err)
	}
	if secondPermissionCount != permissionCount {
		t.Fatalf("重复迁移不应改写权限字典: before=%d after=%d", permissionCount, secondPermissionCount)
	}
	if err := ValidateDefault(db); err != nil {
		t.Fatalf("迁移完成后的只读结构校验失败: %v", err)
	}
}

type legacyConstraintModel struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"column:username"`
	Required string `gorm:"column:required_value"`
}

func (legacyConstraintModel) TableName() string { return "legacy_constraint_test" }

func openMigrationSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开迁移测试数据库失败: %v", err)
	}
	return db
}

// TestExistingTableKeepsLegacyConstraintName 模拟生产库由数据库自动命名的唯一约束。
// 模型没有唯一标签时也不得执行 ALTER/DROP，更不能推导错误的约束名。
func TestExistingTableKeepsLegacyConstraintName(t *testing.T) {
	db := openMigrationSQLite(t)
	if err := db.Exec(`CREATE TABLE legacy_constraint_test (
		id INTEGER PRIMARY KEY,
		username TEXT NOT NULL,
		required_value TEXT NOT NULL,
		legacy_only TEXT,
		CONSTRAINT users_username_key UNIQUE (username)
	)`).Error; err != nil {
		t.Fatalf("创建旧表失败: %v", err)
	}

	before := sqliteTableDefinition(t, db, "legacy_constraint_test")
	if err := createMissingTablesAndValidateExisting(db, &legacyConstraintModel{}); err != nil {
		t.Fatalf("已有完整表应只读通过: %v", err)
	}
	after := sqliteTableDefinition(t, db, "legacy_constraint_test")
	if after != before || !strings.Contains(after, "users_username_key") || !strings.Contains(after, "legacy_only") {
		t.Fatalf("已有表或旧约束被修改:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestExistingUsersTableKeepsDatabaseGeneratedConstraint(t *testing.T) {
	db := openMigrationSQLite(t)
	if err := db.Exec("ATTACH DATABASE ':memory:' AS public").Error; err != nil {
		t.Fatalf("创建 public 测试 schema 失败: %v", err)
	}
	if err := db.Exec(`CREATE TABLE public.users (
		id INTEGER PRIMARY KEY,
		username TEXT NOT NULL,
		password TEXT NOT NULL,
		password_changed_at TIMESTAMP WITH TIME ZONE,
		must_change_password BOOLEAN NOT NULL DEFAULT false,
		true_name TEXT,
		phone TEXT,
		email TEXT,
		create_time TIMESTAMP,
		total_login INTEGER,
		token TEXT,
		login_ip TEXT,
		last_ip TEXT,
		isadmin INTEGER,
		client_id TEXT,
		parent_id INTEGER,
		plat_name TEXT,
		plat_type TEXT,
		plat_id INTEGER,
		baned INTEGER,
		corp_ids TEXT,
		renter TEXT,
		legacy_only TEXT,
		CONSTRAINT users_username_key UNIQUE (username)
	)`).Error; err != nil {
		t.Fatalf("创建旧 users 表失败: %v", err)
	}

	var before string
	if err := db.Raw(`SELECT sql FROM public.sqlite_master WHERE type = 'table' AND name = 'users'`).Scan(&before).Error; err != nil {
		t.Fatalf("读取旧 users 表定义失败: %v", err)
	}
	if err := createMissingTablesAndValidateExisting(db, &authmodel.User{}); err != nil {
		t.Fatalf("旧 users 表应只读通过: %v", err)
	}
	var after string
	if err := db.Raw(`SELECT sql FROM public.sqlite_master WHERE type = 'table' AND name = 'users'`).Scan(&after).Error; err != nil {
		t.Fatalf("读取校验后的 users 表定义失败: %v", err)
	}
	if after != before || !strings.Contains(after, "users_username_key") || !strings.Contains(after, "legacy_only") {
		t.Fatalf("users 表或真实约束名被修改:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestExistingTableMissingColumnFailsWithoutModification(t *testing.T) {
	db := openMigrationSQLite(t)
	if err := db.Exec(`CREATE TABLE legacy_constraint_test (
		id INTEGER PRIMARY KEY,
		username TEXT NOT NULL,
		CONSTRAINT users_username_key UNIQUE (username)
	)`).Error; err != nil {
		t.Fatalf("创建缺字段旧表失败: %v", err)
	}
	before := sqliteTableDefinition(t, db, "legacy_constraint_test")
	err := createMissingTablesAndValidateExisting(db, &legacyConstraintModel{})
	if err == nil || !strings.Contains(err.Error(), "required_value") {
		t.Fatalf("应明确报告缺少 required_value，实际错误: %v", err)
	}
	after := sqliteTableDefinition(t, db, "legacy_constraint_test")
	if after != before || db.Migrator().HasColumn(&legacyConstraintModel{}, "required_value") {
		t.Fatal("缺字段校验不应自动增加字段或改写旧表")
	}
}

func sqliteTableDefinition(t *testing.T, db *gorm.DB, table string) string {
	t.Helper()
	var definition string
	if err := db.Raw(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&definition).Error; err != nil {
		t.Fatalf("读取表定义失败: %v", err)
	}
	return definition
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
