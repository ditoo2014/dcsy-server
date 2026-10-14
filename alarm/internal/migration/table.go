// Package migration 提供告警程序启动时使用的安全建表能力。
package migration

import (
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// CreateMissingTablesAndValidateExisting 只创建不存在的表。
// 对生产库已有表仅检查必需字段，避免 GORM AutoMigrate 修改字段或删除旧约束。
func CreateMissingTablesAndValidateExisting(db *gorm.DB, models ...any) error {
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
	if db.Dialector.Name() == "sqlite" {
		createDB = db.Session(&gorm.Session{})
		createDB.DisableForeignKeyConstraintWhenMigrating = true
	}
	if err := createDB.Migrator().CreateTable(missing...); err != nil {
		return fmt.Errorf("创建缺失数据表失败: %w", err)
	}
	return nil
}

// ValidateRequiredTables 只读校验其他服务负责创建的共享表。
func ValidateRequiredTables(db *gorm.DB, models ...any) error {
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
			return fmt.Errorf("缺少告警程序依赖的数据表 %s；请先启动认证服务完成共享数据库初始化", identity.displayName())
		}
		if err := validateExistingTableColumns(db, value); err != nil {
			return err
		}
	}
	return nil
}

// validateExistingTableColumns 只读取表结构，不执行任何结构变更。
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
	if db.Dialector.Name() == "postgres" && identity.schema == "public" && identity.name == "users" {
		var dataType string
		if err := db.Raw(`SELECT data_type FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'create_time'`).Scan(&dataType).Error; err != nil {
			return fmt.Errorf("读取 public.users.create_time 字段类型失败: %w", err)
		}
		dataType = strings.ToLower(dataType)
		if dataType != "timestamp" && dataType != "timestamp without time zone" && dataType != "timestamp with time zone" {
			return fmt.Errorf("已有字段 public.users.create_time 类型为 %q，与告警程序不兼容；请先完成经过审核的数据库增量迁移", dataType)
		}
	}
	return nil
}

type tableIdentity struct {
	schema string
	name   string
}

var safeDatabaseIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const schemaMigrationLockName = "dcsy_schema_migration_v1"

// WithSchemaLock 与主后端使用同一个 OpenGauss 锁表，避免并发创建共享表。
func WithSchemaLock(db *gorm.DB, work func(*gorm.DB) error) error {
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

func (table tableIdentity) displayName() string {
	if table.schema == "" {
		return table.name
	}
	return table.schema + "." + table.name
}

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

// modelTableExists 保留元数据查询错误，防止连接异常被误判成缺表。
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
