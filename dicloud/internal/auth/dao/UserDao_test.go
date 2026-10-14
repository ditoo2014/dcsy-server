package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	entity "dicloud/models/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func newPasswordTestDAO(t *testing.T) (*UserDAO, entity.User) {
	t.Helper()
	dsn := fmt.Sprintf("file:dao_%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("ATTACH DATABASE ':memory:' AS public").Error; err != nil {
		t.Fatal(err)
	}
	// SQLite 不识别 OpenGauss timestamptz；测试映射为其原生时间列，仍验证保存的绝对时刻。
	statement := &gorm.Statement{DB: db}
	if err := statement.Parse(&entity.User{}); err != nil {
		t.Fatal(err)
	}
	statement.Schema.FieldsByDBName["password_changed_at"].DataType = schema.Time
	statement.Schema.FieldsByDBName["password_changed_at"].TagSettings["TYPE"] = "datetime"
	if err := db.Migrator().CreateTable(&entity.User{}); err != nil {
		t.Fatal(err)
	}
	user := entity.User{Username: "test-user", Password: "old-hash", Token: "old-token", TrueName: "测试账号", Phone: "13183639612", LoginIp: "192.168.0.1", TotalLogin: 5}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return &UserDAO{db: db}, user
}

func loadPasswordTestUser(t *testing.T, dao *UserDAO, id uint) entity.User {
	t.Helper()
	var user entity.User
	if err := dao.db.First(&user, id).Error; err != nil {
		t.Fatal(err)
	}
	return user
}

// TestChangePasswordInvalidatesToken 验证密码、改密状态和令牌同时更新，且 false 和空串不会被忽略。
func TestChangePasswordInvalidatesToken(t *testing.T) {
	dao, user := newPasswordTestDAO(t)
	if err := dao.db.Model(&user).Update("must_change_password", true).Error; err != nil {
		t.Fatal(err)
	}
	changedAt := time.Date(2026, 10, 9, 12, 30, 0, 0, time.UTC)
	user.Password = "new-hash"
	if err := dao.ChangePassword(user, "old-hash", changedAt, false); err != nil {
		t.Fatal(err)
	}
	stored := loadPasswordTestUser(t, dao, user.ID)
	if stored.Password != "new-hash" || stored.Token != "" || stored.MustChangePassword || stored.PasswordChangedAt == nil || !stored.PasswordChangedAt.Equal(changedAt) {
		t.Fatalf("改密必须同步更新安全状态并失效令牌: %+v", stored)
	}
	if stored.TrueName != "测试账号" || stored.Phone != "13183639612" || stored.TotalLogin != 5 {
		t.Fatal("改密不得改写无关用户资料")
	}
}

func TestChangePasswordRejectsChangedCredentials(t *testing.T) {
	for _, changedColumn := range []string{"password", "token"} {
		t.Run(changedColumn, func(t *testing.T) {
			dao, user := newPasswordTestDAO(t)
			if err := dao.db.Model(&entity.User{}).Where("id = ?", user.ID).Update(changedColumn, "concurrent-change").Error; err != nil {
				t.Fatal(err)
			}
			user.Password = "attempted-hash"
			err := dao.ChangePassword(user, "old-hash", time.Now(), false)
			if !errors.Is(err, ErrPasswordChanged) {
				t.Fatalf("并发凭证变化必须拒绝覆盖: %v", err)
			}
			stored := loadPasswordTestUser(t, dao, user.ID)
			if stored.Password == "attempted-hash" || stored.PasswordChangedAt != nil {
				t.Fatal("失败改密不能写入部分安全状态")
			}
		})
	}
}

func TestAdministratorResetKeepsForceFlag(t *testing.T) {
	dao, user := newPasswordTestDAO(t)
	user.Password = "administrator-reset-hash"
	if err := dao.ChangePassword(user, "old-hash", time.Now(), true); err != nil {
		t.Fatal(err)
	}
	stored := loadPasswordTestUser(t, dao, user.ID)
	if !stored.MustChangePassword || stored.Token != "" || stored.PasswordChangedAt == nil {
		t.Fatal("管理员重置后必须保留本人改密标记并失效旧令牌")
	}
}

// TestChangePasswordPreservesQuarterBoundaryInstant 验证服务器时区差异不改变季度边界的真实时刻。
func TestChangePasswordPreservesQuarterBoundaryInstant(t *testing.T) {
	dao, user := newPasswordTestDAO(t)
	beijing := time.FixedZone("Asia/Shanghai", 8*60*60)
	changedAt := time.Date(2026, 10, 1, 0, 0, 0, 0, beijing)
	user.Password = "new-quarter-hash"
	if err := dao.ChangePassword(user, "old-hash", changedAt, false); err != nil {
		t.Fatal(err)
	}
	stored := loadPasswordTestUser(t, dao, user.ID)
	if stored.PasswordChangedAt == nil || !stored.PasswordChangedAt.Equal(changedAt) {
		t.Fatalf("跨时区保存必须保留绝对时刻: %+v", stored.PasswordChangedAt)
	}
	if stored.PasswordChangedAt.In(beijing).Month() != time.October || stored.PasswordChangedAt.In(beijing).Day() != 1 {
		t.Fatal("改密时刻转换为北京时间后必须仍处于新季度")
	}
}

func TestUpdateLoginDoesNotReviveChangedPassword(t *testing.T) {
	dao, stale := newPasswordTestDAO(t)
	if err := dao.db.Model(&entity.User{}).Where("id = ?", stale.ID).Updates(map[string]any{"password": "new-hash", "token": ""}).Error; err != nil {
		t.Fatal(err)
	}
	stale.Token = "stale-login-token"
	if _, ok := dao.UpdateLogin(stale); ok {
		t.Fatal("旧密码校验结果不得恢复失效令牌")
	}
	stored := loadPasswordTestUser(t, dao, stale.ID)
	if stored.Token != "" || stored.Password != "new-hash" || stored.TotalLogin != 5 {
		t.Fatal("拒绝的并发登录不得改写账号")
	}
}

func TestUpdateLoginWritesOnlySessionColumns(t *testing.T) {
	dao, user := newPasswordTestDAO(t)
	changedAt := time.Date(2026, 10, 9, 12, 30, 0, 0, time.UTC)
	if err := dao.db.Model(&user).Updates(map[string]any{"password_changed_at": changedAt, "must_change_password": true}).Error; err != nil {
		t.Fatal(err)
	}
	user.Token = "new-token"
	user.LoginIp = "192.168.0.2"
	user.TrueName = "不能通过登录修改"
	user.Isadmin = 1
	user.TotalLogin = 1000
	actual, ok := dao.UpdateLogin(user)
	if !ok {
		t.Fatal("匹配当前密码的登录应成功")
	}
	if actual.ID != user.ID || actual.Username != user.Username || actual.Token != "new-token" || actual.TotalLogin != 6 || actual.LastIp != "192.168.0.1" || actual.LoginIp != "192.168.0.2" {
		t.Fatalf("应返回数据库实际登录记录: %+v", actual)
	}
	if actual.TrueName != "测试账号" || actual.Isadmin != 0 || !actual.MustChangePassword || actual.PasswordChangedAt == nil || !actual.PasswordChangedAt.Equal(changedAt) {
		t.Fatal("登录不得篡改资料或密码安全状态")
	}
}

func TestUpdateProfileRejectsCredentialFields(t *testing.T) {
	dao, user := newPasswordTestDAO(t)
	changedAt := time.Now()
	user.TrueName = "更新后的姓名"
	user.Phone = ""
	user.Username = "injected-name"
	user.Password = "injected-password"
	user.Token = "injected-token"
	user.Isadmin = 1
	user.Renter = "injected-renter"
	user.PasswordChangedAt = &changedAt
	user.MustChangePassword = true
	actual, ok := dao.Update(user)
	if !ok {
		t.Fatal("正常资料更新应成功")
	}
	if actual.ID != user.ID || actual.Username != "test-user" || actual.Password != "old-hash" || actual.Token != "old-token" || actual.Isadmin != 0 || actual.Renter != "" || actual.PasswordChangedAt != nil || actual.MustChangePassword {
		t.Fatalf("资料接口不得更新凭证和密码安全状态: %+v", actual)
	}
	if actual.TrueName != "更新后的姓名" || actual.Phone != "13183639612" {
		t.Fatal("白名单资料应更新普通值，并沿用原接口忽略零值的语义")
	}
}

func TestPasswordSecurityFieldsCannotBindJSON(t *testing.T) {
	var user entity.User
	err := json.Unmarshal([]byte(`{"password_changed_at":"2026-10-09T12:30:00Z","must_change_password":true,"passwordChangedAt":"2026-10-09T12:30:00Z","mustChangePassword":true}`), &user)
	if err != nil {
		t.Fatal(err)
	}
	if user.PasswordChangedAt != nil || user.MustChangePassword {
		t.Fatal("客户端 JSON 不得绑定密码安全元数据")
	}
}
