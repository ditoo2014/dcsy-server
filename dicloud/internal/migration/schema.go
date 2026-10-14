// Package migration 统一管理 DCSY 使用的 OpenGauss 表结构。
//
// 认证服务负责创建缺失表和执行明确的增量迁移；其他服务只读校验，
// 从而既让新环境不依赖人工建表，也避免多个进程并发修改生产库。
package migration

import (
	"fmt"
	"regexp"
	"strings"
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
}

// defaultOnlyModels 是认证默认库独有的全局数据表，不应复制到租户库。
var defaultOnlyModels = []any{&authmodel.WeakPassword{}}

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

// Initialize 连接并只读校验默认库和全部租户库。
// 除认证服务外的程序使用本函数，避免多个进程并发修改同一套生产表结构。
func Initialize() (*gorm.DB, error) {
	return initializeDatabases(false)
}

// InitializeAndMigrate 连接数据库并创建缺失表、执行经过审核的增量迁移。
// 该入口只允许认证服务调用，其他服务必须使用 Initialize。
func InitializeAndMigrate() (*gorm.DB, error) {
	return initializeDatabases(true)
}

func initializeDatabases(allowMigration bool) (*gorm.DB, error) {
	if err := opengauss.InitOpenGaussDB(); err != nil {
		return nil, fmt.Errorf("连接默认OpenGauss数据库失败: %w", err)
	}
	defaultDB := opengauss.InitGormDB()
	if defaultDB == nil {
		return nil, fmt.Errorf("初始化默认GORM连接失败")
	}
	if allowMigration {
		if err := MigrateDefault(defaultDB); err != nil {
			return nil, fmt.Errorf("迁移默认数据库失败: %w", err)
		}
	} else if err := ValidateDefault(defaultDB); err != nil {
		return nil, fmt.Errorf("校验默认数据库失败: %w", err)
	}

	for _, dbName := range opengauss.ConfiguredRenterDBNames() {
		tenantDB := opengauss.GetRenterDB(dbName)
		if tenantDB == nil {
			return nil, fmt.Errorf("初始化租户数据库失败: %s", dbName)
		}
		if allowMigration {
			if err := MigrateTenant(tenantDB); err != nil {
				return nil, fmt.Errorf("迁移租户数据库%s失败: %w", dbName, err)
			}
		} else if err := ValidateTenant(tenantDB); err != nil {
			return nil, fmt.Errorf("校验租户数据库%s失败: %w", dbName, err)
		}
	}
	return defaultDB, nil
}

// ValidateDefault 只读校验默认库运行所需的表和字段。
func ValidateDefault(db *gorm.DB) error {
	models := append([]any{}, authModels...)
	models = append(models, defaultOnlyModels...)
	models = append(models, tenantModels...)
	models = append(models, globalTtdcModels...)
	if err := validateRequiredTables(db, models...); err != nil {
		return err
	}
	if err := validateCriticalColumnTypes(db); err != nil {
		return err
	}
	return validateStatisticTables(db)
}

// ValidateTenant 只读校验租户库；TTDC公共模板只位于默认库。
func ValidateTenant(db *gorm.DB) error {
	models := append([]any{}, authModels...)
	models = append(models, tenantModels...)
	if err := validateRequiredTables(db, models...); err != nil {
		return err
	}
	if err := validateCriticalColumnTypes(db); err != nil {
		return err
	}
	return validateStatisticTables(db)
}

// MigrateDefault 迁移默认库；默认库同时承担无租户场景的业务数据存储。
func MigrateDefault(db *gorm.DB) error {
	return withSchemaMigrationLock(db, migrateDefault)
}

func migrateDefault(db *gorm.DB) error {
	if err := ensureUserPasswordSecurityColumns(db); err != nil {
		return err
	}
	// 旧版曾把三张记录表的时间列建成 JSON；该显式迁移会先逐条校验，
	// 只在能够无损转换时修改列类型，不能转换则回滚并保留原数据。
	if err := businessmigration.EnsureBusinessRecordTimeColumns(db); err != nil {
		return err
	}
	if err := ensureGatewayIdentityIndex(db); err != nil {
		return err
	}
	if err := ensureAlarmLifecycleIndex(db); err != nil {
		return err
	}
	models := append([]any{}, authModels...)
	models = append(models, defaultOnlyModels...)
	models = append(models, tenantModels...)
	models = append(models, globalTtdcModels...)
	if err := createMissingTablesAndValidateExisting(db, models...); err != nil {
		return err
	}
	if err := validateCriticalColumnTypes(db); err != nil {
		return err
	}
	if err := migrateStatisticTables(db); err != nil {
		return err
	}
	// 编码索引是经过单独审核的增量迁移；创建前会检查重复值且不修改记录。
	if err := businessmigration.EnsureEquipmentWarehouseCodeIndexes(db); err != nil {
		return err
	}
	// 权限初始化只补缺失项，保留全部已有 ID、名称、层级和授权关系。
	if err := ensureBusinessPermissions(db); err != nil {
		return err
	}
	return ensureTtdcSchemaReady(db)
}

// MigrateTenant 迁移单个租户库。TTDC公共模板仍只使用默认库。
func MigrateTenant(db *gorm.DB) error {
	return withSchemaMigrationLock(db, migrateTenant)
}

func migrateTenant(db *gorm.DB) error {
	if err := ensureUserPasswordSecurityColumns(db); err != nil {
		return err
	}
	if err := businessmigration.EnsureBusinessRecordTimeColumns(db); err != nil {
		return err
	}
	if err := ensureGatewayIdentityIndex(db); err != nil {
		return err
	}
	if err := ensureAlarmLifecycleIndex(db); err != nil {
		return err
	}
	models := append([]any{}, authModels...)
	models = append(models, tenantModels...)
	if err := createMissingTablesAndValidateExisting(db, models...); err != nil {
		return err
	}
	if err := validateCriticalColumnTypes(db); err != nil {
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

const schemaMigrationLockName = "dcsy_schema_migration_v1"

// ensureUserPasswordSecurityColumns 是认证服务专用的幂等增量迁移。
// 只补两列、扩大旧令牌列容量，不回填老账号改密时间，不改写密码、数据或约束。
func ensureUserPasswordSecurityColumns(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("数据库连接为空")
	}
	user := &authmodel.User{}
	exists, err := modelTableExists(db, user)
	if err != nil || !exists {
		// 缺失表由后续统一 CreateTable 创建，模型自然包含新字段。
		return err
	}
	identity, _, err := resolveModelTable(db, user)
	if err != nil {
		return err
	}
	columns, err := readExistingColumnNames(db, user, identity)
	if err != nil {
		return fmt.Errorf("读取用户密码安全字段失败: %w", err)
	}
	for _, column := range []string{"password_changed_at", "must_change_password"} {
		// 元数据读取保留错误；GORM HasColumn/AddColumn 只用于明确允许增加的列。
		if _, ok := columns[column]; ok {
			continue
		}
		if db.Migrator().HasColumn(user, column) {
			continue
		}
		if err := db.Migrator().AddColumn(user, column); err != nil {
			return fmt.Errorf("添加用户密码安全字段 %s 失败: %w", column, err)
		}
	}
	return ensureUserTokenCapacity(db, identity)
}

// ensureUserTokenCapacity 只允许将旧 varchar 令牌列扩容到 512，不缩小、不转换其他类型。
// 直接读取元数据并执行固定目标 ALTER，避免旧 OpenGauss 的 ColumnTypes 兼容问题。
func ensureUserTokenCapacity(db *gorm.DB, identity tableIdentity) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	var column struct {
		DataType string `gorm:"column:data_type"`
		Length   *int64 `gorm:"column:character_maximum_length"`
	}
	if err := db.Raw(`SELECT data_type, character_maximum_length FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ? AND column_name = 'token'`, identity.schema, identity.name).Scan(&column).Error; err != nil {
		return fmt.Errorf("读取用户令牌字段容量失败: %w", err)
	}
	if (column.DataType != "character varying" && column.DataType != "varchar") || column.Length == nil || *column.Length >= 512 {
		return nil
	}
	// identity 已由 resolveModelTable 限制为安全标识符，并且只来源于固定 User 模型。
	table := fmt.Sprintf(`"%s"."%s"`, identity.schema, identity.name)
	if err := db.Exec(`ALTER TABLE ` + table + ` ALTER COLUMN "token" TYPE varchar(512)`).Error; err != nil {
		return fmt.Errorf("扩容用户令牌字段失败: %w", err)
	}
	return nil
}

// withSchemaMigrationLock 使用 OpenGauss 锁对象串行化多副本冷启动。
// 默认库与各租户库分别加锁，事务无论提交或回滚都会执行解锁。
func withSchemaMigrationLock(db *gorm.DB, work func(*gorm.DB) error) error {
	if db == nil {
		return fmt.Errorf("数据库连接为空")
	}
	if db.Dialector.Name() != "postgres" {
		return work(db)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS " + schemaMigrationLockName + " (id integer PRIMARY KEY)").Error; err != nil {
		return fmt.Errorf("创建数据库迁移锁表失败: %w", err)
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE " + schemaMigrationLockName + " IN ACCESS EXCLUSIVE MODE").Error; err != nil {
			return fmt.Errorf("获取数据库迁移锁失败: %w", err)
		}
		return work(tx)
	})
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
				// 动态统计表同样只在缺失时创建，避免启动时修改生产库已有表结构。
				tableDB := db.Table(model.GetTableName())
				if err := createMissingTablesAndValidateExisting(tableDB, &commonmodel.StatisticData{}); err != nil {
					return fmt.Errorf("迁移统计表%s失败: %w", model.GetTableName(), err)
				}
			}
		}
	}
	return nil
}

func validateStatisticTables(db *gorm.DB) error {
	for targetType := 0; targetType <= 2; targetType++ {
		for _, dateType := range []string{"hourly", "daily", "monthly"} {
			for valueType := 0; valueType <= 1; valueType++ {
				model := commonmodel.StatisticData{
					TargetType:         targetType,
					StatisticDateType:  dateType,
					StatisticValueType: valueType,
				}
				if err := validateRequiredTables(db.Table(model.GetTableName()), &commonmodel.StatisticData{}); err != nil {
					return fmt.Errorf("校验统计表%s失败: %w", model.GetTableName(), err)
				}
			}
		}
	}
	return nil
}

// createMissingTablesAndValidateExisting 兼顾新环境自动建表与旧库安全启动。
//
// GORM AutoMigrate 会比较并修改已有表。旧版 OpenGauss 数据库中的约束名称
// 由数据库自动生成，可能与 GORM 推导的名称不同；直接执行 AutoMigrate 会因此
// 尝试删除不存在的约束，甚至改变已有字段类型、默认值或非空属性。
// 这里对缺失表使用 GORM CreateTable，对已有表只核对程序必需的字段是否存在。
func createMissingTablesAndValidateExisting(db *gorm.DB, models ...any) error {
	if db == nil {
		return fmt.Errorf("数据库连接为空")
	}

	missing := make([]any, 0, len(models))
	for _, value := range models {
		exists, err := modelTableExists(db, value)
		if err != nil {
			return err
		}
		if !exists {
			missing = append(missing, value)
			continue
		}
		if err := validateExistingTableColumns(db, value); err != nil {
			return err
		}
	}

	if len(missing) == 0 {
		return nil
	}
	createDB := db
	// SQLite 只用于单元测试，且不支持跨附加库外键；生产 OpenGauss 仍创建模型外键。
	if db.Dialector.Name() == "sqlite" {
		createDB = db.Session(&gorm.Session{})
		createDB.DisableForeignKeyConstraintWhenMigrating = true
	}
	if err := createDB.Migrator().CreateTable(missing...); err != nil {
		return fmt.Errorf("创建缺失数据表失败: %w", err)
	}
	return nil
}

func validateRequiredTables(db *gorm.DB, models ...any) error {
	if db == nil {
		return fmt.Errorf("数据库连接为空")
	}
	for _, value := range models {
		exists, err := modelTableExists(db, value)
		if err != nil {
			return err
		}
		if !exists {
			identity, _, parseErr := resolveModelTable(db, value)
			if parseErr != nil {
				return parseErr
			}
			return fmt.Errorf("缺少程序必需的数据表 %s；请先启动认证服务完成数据库初始化", identity.displayName())
		}
		if err := validateExistingTableColumns(db, value); err != nil {
			return err
		}
	}
	return nil
}

// validateExistingTableColumns 只读取结构信息，不增加、删除或修改任何字段和约束。
func validateExistingTableColumns(db *gorm.DB, value any) error {
	identity, statement, err := resolveModelTable(db, value)
	if err != nil {
		return err
	}
	columns, err := readExistingColumnNames(db, value, identity)
	if err != nil {
		return fmt.Errorf("读取已有数据表 %s 的字段失败: %w", identity.displayName(), err)
	}

	missingColumns := make([]string, 0)
	for _, column := range statement.Schema.DBNames {
		field := statement.Schema.FieldsByDBName[column]
		if field == nil || field.IgnoreMigration {
			continue
		}
		if _, exists := columns[column]; !exists {
			missingColumns = append(missingColumns, column)
		}
	}
	if len(missingColumns) > 0 {
		return fmt.Errorf(
			"已有数据表 %s 缺少程序必需字段 %s；为保护生产数据，程序未自动修改该表，请先执行经过审核的增量迁移",
			identity.displayName(),
			strings.Join(missingColumns, ", "),
		)
	}
	return nil
}

type tableIdentity struct {
	schema string
	name   string
}

var safeDatabaseIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func (table tableIdentity) displayName() string {
	if table.schema == "" {
		return table.name
	}
	return table.schema + "." + table.name
}

// resolveModelTable 同时兼容模型 TableName 和 db.Table 指定的动态统计表名。
func resolveModelTable(db *gorm.DB, value any) (tableIdentity, *gorm.Statement, error) {
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(value); err != nil {
		return tableIdentity{}, nil, fmt.Errorf("解析数据表模型失败: %w", err)
	}
	if statement.Schema == nil {
		return tableIdentity{}, nil, fmt.Errorf("解析数据表模型失败: 未取得表结构")
	}

	tableName := statement.Schema.Table
	if db.Statement != nil && db.Statement.Table != "" {
		tableName = db.Statement.Table
	}
	parts := strings.SplitN(tableName, ".", 2)
	identity := tableIdentity{name: parts[0]}
	if len(parts) == 2 {
		identity.schema, identity.name = parts[0], parts[1]
	}
	if !safeDatabaseIdentifier.MatchString(identity.name) ||
		(identity.schema != "" && !safeDatabaseIdentifier.MatchString(identity.schema)) {
		return tableIdentity{}, nil, fmt.Errorf("数据表名称不合法: %s", tableName)
	}
	return identity, statement, nil
}

// modelTableExists 直接查询元数据并保留数据库错误，避免 Migrator.HasTable
// 将连接或权限错误吞掉后误判成缺表并尝试执行 CREATE TABLE。
func modelTableExists(db *gorm.DB, value any) (bool, error) {
	identity, _, err := resolveModelTable(db, value)
	if err != nil {
		return false, err
	}

	var count int64
	switch db.Dialector.Name() {
	case "postgres":
		query := `SELECT COUNT(*) FROM information_schema.tables
			WHERE table_schema = CURRENT_SCHEMA() AND table_name = ? AND table_type = 'BASE TABLE'`
		args := []any{identity.name}
		if identity.schema != "" {
			query = `SELECT COUNT(*) FROM information_schema.tables
				WHERE table_schema = ? AND table_name = ? AND table_type = 'BASE TABLE'`
			args = []any{identity.schema, identity.name}
		}
		if err := db.Raw(query, args...).Scan(&count).Error; err != nil {
			return false, fmt.Errorf("检查数据表 %s 是否存在失败: %w", identity.displayName(), err)
		}
	case "sqlite":
		schema := identity.schema
		if schema == "" {
			schema = "main"
		}
		query := fmt.Sprintf(`SELECT COUNT(*) FROM "%s".sqlite_master WHERE type = 'table' AND name = ?`, schema)
		if err := db.Raw(query, identity.name).Scan(&count).Error; err != nil {
			return false, fmt.Errorf("检查数据表 %s 是否存在失败: %w", identity.displayName(), err)
		}
	default:
		tables, err := db.Migrator().GetTables()
		if err != nil {
			return false, fmt.Errorf("读取数据表清单失败: %w", err)
		}
		for _, table := range tables {
			if table == identity.name || table == identity.displayName() {
				return true, nil
			}
		}
	}
	return count > 0, nil
}

// readExistingColumnNames 只读取字段名，不调用会比较类型和约束的 AutoMigrate。
func readExistingColumnNames(db *gorm.DB, value any, identity tableIdentity) (map[string]struct{}, error) {
	type columnRow struct {
		Name string `gorm:"column:column_name"`
	}
	rows := make([]columnRow, 0)

	switch db.Dialector.Name() {
	case "postgres":
		query := `SELECT column_name FROM information_schema.columns
			WHERE table_schema = CURRENT_SCHEMA() AND table_name = ?`
		args := []any{identity.name}
		if identity.schema != "" {
			query = `SELECT column_name FROM information_schema.columns
				WHERE table_schema = ? AND table_name = ?`
			args = []any{identity.schema, identity.name}
		}
		if err := db.Raw(query, args...).Scan(&rows).Error; err != nil {
			return nil, err
		}
	case "sqlite":
		schema := identity.schema
		if schema == "" {
			schema = "main"
		}
		var sqliteRows []struct {
			Name string `gorm:"column:name"`
		}
		query := fmt.Sprintf(`PRAGMA "%s".table_info("%s")`, schema, identity.name)
		if err := db.Raw(query).Scan(&sqliteRows).Error; err != nil {
			return nil, err
		}
		for _, row := range sqliteRows {
			rows = append(rows, columnRow{Name: row.Name})
		}
	default:
		columnTypes, err := db.Table(identity.displayName()).Migrator().ColumnTypes(value)
		if err != nil {
			return nil, err
		}
		for _, column := range columnTypes {
			rows = append(rows, columnRow{Name: column.Name()})
		}
	}

	columns := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		columns[row.Name] = struct{}{}
	}
	return columns, nil
}

// validateCriticalColumnTypes 对运行时无法兼容的关键类型做只读校验。
// 通用校验不比较所有类型，避免旧库等价类型名称差异造成误报。
func validateCriticalColumnTypes(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	targets := []struct {
		schema  string
		table   string
		column  string
		allowed map[string]bool
	}{
		{"public", "users", "create_time", map[string]bool{
			"timestamp without time zone": true,
			"timestamp with time zone":    true,
			"timestamp":                   true,
		}},
		{"public", "equipment_maintain_record", "create_time", map[string]bool{"bigint": true}},
		{"public", "equipment_circulation_log", "create_time", map[string]bool{"bigint": true}},
		{"public", "equipment_geography_position", "create_time", map[string]bool{"bigint": true}},
	}
	for _, target := range targets {
		var dataType string
		err := db.Raw(`SELECT data_type FROM information_schema.columns
			WHERE table_schema = ? AND table_name = ? AND column_name = ?`,
			target.schema, target.table, target.column).Scan(&dataType).Error
		if err != nil {
			return fmt.Errorf("读取 %s.%s.%s 字段类型失败: %w", target.schema, target.table, target.column, err)
		}
		if !target.allowed[strings.ToLower(dataType)] {
			return fmt.Errorf(
				"已有字段 %s.%s.%s 类型为 %q，与当前程序不兼容；程序未自动转换，请先备份并执行经过审核的增量迁移",
				target.schema, target.table, target.column, dataType,
			)
		}
	}
	return nil
}

// ensureGatewayIdentityIndex 是针对旧库的显式、幂等增量迁移。
// gateway_data 的外键依赖该唯一键；创建前先拒绝重复数据，绝不删除或改写网关。
func ensureGatewayIdentityIndex(db *gorm.DB) error {
	exists, err := modelTableExists(db, &tolinkmodel.Gateway{})
	if err != nil || !exists {
		return err
	}

	unique := false
	if db.Dialector.Name() == "postgres" {
		var count int64
		err = db.Raw(`SELECT COUNT(*) FROM pg_indexes
			WHERE schemaname = CURRENT_SCHEMA()
			  AND tablename = 'gateway'
			  AND UPPER(indexdef) LIKE 'CREATE UNIQUE INDEX%'
			  AND REPLACE(indexdef, ' ', '') LIKE '%(gateway_id)%'`).Scan(&count).Error
		if err != nil {
			return fmt.Errorf("检查 gateway.gateway_id 唯一约束失败: %w", err)
		}
		unique = count > 0
	} else {
		// SQLite 仅用于自动化测试；字段名会由 GORM 解析为默认索引名。
		unique = db.Migrator().HasIndex(&tolinkmodel.Gateway{}, "GatewayId")
	}
	if !unique {
		var duplicateCount int64
		if err := db.Table("gateway").
			Select("gateway_id").
			Where("gateway_id IS NOT NULL").
			Group("gateway_id").
			Having("COUNT(*) > 1").
			Limit(1).
			Count(&duplicateCount).Error; err != nil {
			return fmt.Errorf("检查重复网关编号失败: %w", err)
		}
		if duplicateCount > 0 {
			return fmt.Errorf("无法为 gateway.gateway_id 创建唯一索引：现有数据包含重复网关编号，请先人工核对数据")
		}
		if err := db.Migrator().CreateIndex(&tolinkmodel.Gateway{}, "GatewayId"); err != nil {
			return fmt.Errorf("创建 gateway.gateway_id 唯一索引失败: %w", err)
		}
	}
	return nil
}

// ensureAlarmLifecycleIndex 保证告警接收幂等所需的五列唯一索引存在。
// 这是明确限定字段的增量迁移，创建前只读检查重复生命周期，不修改记录内容。
func ensureAlarmLifecycleIndex(db *gorm.DB) error {
	exists, err := modelTableExists(db, &commonmodel.AlarmDataRecord{})
	if err != nil || !exists {
		return err
	}
	if db.Migrator().HasIndex(&commonmodel.AlarmDataRecord{}, "uni_alarm_record_lifecycle") {
		return nil
	}

	var duplicateCount int64
	if err := db.Table("alarm_data_record").
		Select("equipment_id, alarm_code, begin_time, alarm_type, alarm_stage").
		Group("equipment_id, alarm_code, begin_time, alarm_type, alarm_stage").
		Having("COUNT(*) > 1").
		Limit(1).
		Count(&duplicateCount).Error; err != nil {
		return fmt.Errorf("检查重复告警生命周期失败: %w", err)
	}
	if duplicateCount > 0 {
		return fmt.Errorf("无法创建告警生命周期唯一索引：alarm_data_record 存在重复记录，请先人工核对数据")
	}
	if err := db.Migrator().CreateIndex(&commonmodel.AlarmDataRecord{}, "uni_alarm_record_lifecycle"); err != nil {
		return fmt.Errorf("创建告警生命周期唯一索引失败: %w", err)
	}
	return nil
}

// ensureTtdcSchemaReady 创建或按需更新TTDC结构版本记录。
// 版本和状态已经正确时不执行 UPDATE，避免每次启动无意义地改写生产数据。
func ensureTtdcSchemaReady(db *gorm.DB) error {
	var state tolinkmodel.TtdcSchemaVersionRecord
	err := db.Where("component = ?", tolinkmodel.TtdcSchemaComponent).Take(&state).Error
	if err == gorm.ErrRecordNotFound {
		state = tolinkmodel.TtdcSchemaVersionRecord{
			Component: tolinkmodel.TtdcSchemaComponent,
			Version:   tolinkmodel.TtdcSchemaVersion,
			Status:    tolinkmodel.TtdcSchemaReady,
			UpdatedAt: time.Now(),
		}
		return db.Create(&state).Error
	}
	if err != nil {
		return err
	}
	if state.Version == tolinkmodel.TtdcSchemaVersion && state.Status == tolinkmodel.TtdcSchemaReady {
		return nil
	}
	return db.Model(&state).Updates(map[string]any{
		"version":    tolinkmodel.TtdcSchemaVersion,
		"status":     tolinkmodel.TtdcSchemaReady,
		"updated_at": time.Now(),
	}).Error
}
