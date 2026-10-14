package migration

import (
	"strings"
	"testing"
	"time"

	authmodel "dicloud/models/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestPasswordSecurityMigrationPreservesLegacyAccounts 验证显式增量迁移不改动旧密码和数据。
func TestPasswordSecurityMigrationPreservesLegacyAccounts(t *testing.T) {
	db := openMigrationSQLite(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("ATTACH DATABASE ':memory:' AS public").Error; err != nil {
		t.Fatal(err)
	}
	// SQLite 测试驱动只扫描其原生时间列，生产模型仍使用 OpenGauss timestamptz。
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(&authmodel.User{}); err != nil {
		t.Fatal(err)
	}
	statement.Schema.FieldsByDBName["password_changed_at"].DataType = schema.Time
	statement.Schema.FieldsByDBName["password_changed_at"].TagSettings["TYPE"] = "datetime"
	if err := db.Exec(`CREATE TABLE public.users (
		id INTEGER PRIMARY KEY,
		username TEXT NOT NULL,
		password TEXT NOT NULL,
		token TEXT,
		legacy_only TEXT,
		CONSTRAINT users_username_key UNIQUE (username)
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO public.users (id, username, password, token, legacy_only)
		VALUES (1, 'legacy-user', 'original-hash', 'old-token', '旧业务数据')`).Error; err != nil {
		t.Fatal(err)
	}
	for iteration := 0; iteration < 2; iteration++ {
		if err := ensureUserPasswordSecurityColumns(db); err != nil {
			t.Fatalf("第 %d 次增量迁移失败: %v", iteration+1, err)
		}
	}
	var row struct {
		Password           string
		Token              string
		LegacyOnly         string
		PasswordChangedAt  *time.Time
		MustChangePassword bool
	}
	if err := db.Table("public.users").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Password != "original-hash" || row.Token != "old-token" || row.LegacyOnly != "旧业务数据" || row.PasswordChangedAt != nil || row.MustChangePassword {
		t.Fatalf("增量迁移不得改写老账号或伪造改密时间: %+v", row)
	}
	var definition string
	if err := db.Raw(`SELECT sql FROM public.sqlite_master WHERE type = 'table' AND name = 'users'`).Scan(&definition).Error; err != nil {
		t.Fatal(err)
	}
	if !containsLegacyConstraint(definition) {
		t.Fatalf("旧唯一约束必须保留: %s", definition)
	}
	changedAt := time.Date(2026, 10, 9, 12, 30, 0, 0, time.UTC)
	if err := db.Table("public.users").Where("id = ?", 1).Updates(map[string]any{
		"password_changed_at": changedAt, "must_change_password": true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureUserPasswordSecurityColumns(db); err != nil {
		t.Fatal(err)
	}
	var user authmodel.User
	if err := db.First(&user, 1).Error; err != nil {
		t.Fatal(err)
	}
	if user.PasswordChangedAt == nil || !user.PasswordChangedAt.Equal(changedAt) || !user.MustChangePassword {
		t.Fatalf("重复迁移不得清除已存在的密码安全状态: %+v", user)
	}
}

func containsLegacyConstraint(definition string) bool {
	return strings.Contains(definition, "users_username_key") && strings.Contains(definition, "legacy_only")
}

func TestPasswordSecurityMigrationSkipsMissingUsers(t *testing.T) {
	db := openMigrationSQLite(t)
	if err := db.Exec("ATTACH DATABASE ':memory:' AS public").Error; err != nil {
		t.Fatal(err)
	}
	if err := ensureUserPasswordSecurityColumns(db); err != nil {
		t.Fatalf("缺失用户表应留给统一建表流程: %v", err)
	}
	if hasMigratedTable(db, &authmodel.User{}) {
		t.Fatal("增量字段迁移不应单独创建整张用户表")
	}
}

type postgresMetadataTestDialector struct {
	gorm.Dialector
}

func (postgresMetadataTestDialector) Name() string { return "postgres" }

// TestUserTokenCapacityMigrationOnlyExpandsLegacyVarchar 使用内存元数据验证生产迁移条件和幂等性。
// 仅拦截 PostgreSQL 的扩容语句，不连接真实数据库，不模拟或绕过密码迁移结果。
func TestUserTokenCapacityMigrationOnlyExpandsLegacyVarchar(t *testing.T) {
	tests := []struct {
		name      string
		dataType  string
		length    any
		wantAlter bool
	}{
		{name: "旧varchar需要扩容", dataType: "character varying", length: 255, wantAlter: true},
		{name: "varchar别名需要扩容", dataType: "varchar", length: 255, wantAlter: true},
		{name: "已达512不再改动", dataType: "character varying", length: 512},
		{name: "更大容量不能缩小", dataType: "character varying", length: 1024},
		{name: "text不转换", dataType: "text", length: nil},
		{name: "未知类型留给校验", dataType: "integer", length: nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, err := gorm.Open(postgresMetadataTestDialector{Dialector: sqlite.Open(":memory:")}, &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			sqlDB.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = sqlDB.Close() })
			if err := db.Exec("ATTACH DATABASE ':memory:' AS information_schema").Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Exec(`CREATE TABLE information_schema.columns (
				table_schema TEXT, table_name TEXT, column_name TEXT,
				data_type TEXT, character_maximum_length INTEGER
			)`).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Exec(`INSERT INTO information_schema.columns VALUES ('public', 'users', 'token', ?, ?)`, test.dataType, test.length).Error; err != nil {
				t.Fatal(err)
			}
			alterCount := 0
			originalRaw := db.Callback().Raw().Get("gorm:raw")
			if err := db.Callback().Raw().Replace("gorm:raw", func(rawDB *gorm.DB) {
				query := rawDB.Statement.SQL.String()
				if strings.HasPrefix(query, "ALTER TABLE ") {
					if query != `ALTER TABLE "public"."users" ALTER COLUMN "token" TYPE varchar(512)` {
						t.Errorf("迁移必须只扩容指定用户令牌列: %s", query)
					}
					alterCount++
					// 模拟扩容后的元数据，下一次启动应读取512而不再执行ALTER。
					_, err := sqlDB.Exec(`UPDATE information_schema.columns SET character_maximum_length = 512 WHERE column_name = 'token'`)
					rawDB.AddError(err)
					return
				}
				originalRaw(rawDB)
			}); err != nil {
				t.Fatal(err)
			}
			for iteration := 0; iteration < 2; iteration++ {
				if err := ensureUserTokenCapacity(db, tableIdentity{schema: "public", name: "users"}); err != nil {
					t.Fatal(err)
				}
			}
			wantCount := 0
			if test.wantAlter {
				wantCount = 1
			}
			if alterCount != wantCount {
				t.Fatalf("扩容次数应为 %d，实际 %d", wantCount, alterCount)
			}
		})
	}
}
