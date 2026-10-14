package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dicloud/models/model"
	"dicloud/pkg/passwordpolicy"
	"dicloud/util/encryption"
	"dicloud/util/opengauss"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 只用临时密钥与内存库执行真实控制器，不读取部署密钥、不连接短信和 Redis。
func passwordControllerFixture(t *testing.T) (*gorm.DB, model.User, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
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
	// 拒绝分支只需读取原凭证；改密成功的原子写入和撤销已由 DAO 测试覆盖。
	if err := db.Exec(`CREATE TABLE public.users (
		id INTEGER PRIMARY KEY, username TEXT, password TEXT, token TEXT,
		password_changed_at TIMESTAMP, must_change_password BOOLEAN DEFAULT false
	)`).Error; err != nil {
		t.Fatal(err)
	}
	oldDB := opengauss.OpengaussGormDB
	opengauss.OpengaussGormDB = db
	t.Cleanup(func() { opengauss.OpengaussGormDB = oldDB })
	privateKey, publicKey, err := encryption.GenerateRSAPKCS8Key()
	if err != nil {
		t.Fatal(err)
	}
	previousKey := viper.Get("rsa.privateKey")
	viper.Set("rsa.privateKey", privateKey)
	t.Cleanup(func() { viper.Set("rsa.privateKey", previousKey) })
	hash, err := bcrypt.GenerateFromPassword([]byte("Qa1!bXeK"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := model.User{ID: 1, Username: "example-account", Password: string(hash), Token: "current-test-token"}
	if err := db.Exec("INSERT INTO public.users (id,username,password,token) VALUES (?,?,?,?)", user.ID, user.Username, user.Password, user.Token).Error; err != nil {
		t.Fatal(err)
	}
	return db, user, publicKey
}

func passwordControllerRequest(t *testing.T, user model.User, token string, body any, handler func(*gin.Context)) *httptest.ResponseRecorder {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/auth/changepassword", bytes.NewReader(data))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Request.Header.Set("Authorization", "Bearer "+token)
	ctx.Set("userInfo", user)
	handler(ctx)
	return recorder
}

func TestChangePasswordRejectsSamePassword(t *testing.T) {
	db, user, publicKey := passwordControllerFixture(t)
	encrypted := encryption.PKCS8Encrypt(publicKey, "Qa1!bXeK")
	recorder := passwordControllerRequest(t, user, user.Token, gin.H{"password": encrypted, "newPassword": encrypted}, UserCtrler{}.ChangePassword)
	var body struct {
		Code int
		Msg  string
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 400 || body.Msg != "新密码不能与当前密码相同" {
		t.Fatalf("相同密码应明确拒绝: %s", recorder.Body.String())
	}
	var token string
	if err := db.Raw("SELECT token FROM public.users WHERE id = 1").Scan(&token).Error; err != nil || token != user.Token {
		t.Fatalf("失败的改密不能修改令牌: token=%s error=%v", token, err)
	}
}

func TestChangePasswordRejectsRevokedToken(t *testing.T) {
	_, user, publicKey := passwordControllerFixture(t)
	recorder := passwordControllerRequest(t, user, "revoked-test-token", gin.H{
		"password":    encryption.PKCS8Encrypt(publicKey, "Qa1!bXeK"),
		"newPassword": encryption.PKCS8Encrypt(publicKey, "Qe7!mXpK"),
	}, UserCtrler{}.ChangePassword)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("已撤销请求不能继续改密: %s", recorder.Body.String())
	}
}

func TestUserInfoReturnsPasswordStateWithoutHash(t *testing.T) {
	config := passwordpolicy.Current()
	t.Cleanup(func() { _ = passwordpolicy.Configure(config) })
	if err := passwordpolicy.Configure(passwordpolicy.Config{Enabled: true, Mode: passwordpolicy.CalendarQuarter, Timezone: "Asia/Shanghai"}); err != nil {
		t.Fatal(err)
	}
	for _, forced := range []bool{false, true} {
		user := model.User{ID: 1, Username: "example", Password: "never-expose-hash", MustChangePassword: forced}
		if forced {
			now := time.Now().UTC()
			user.PasswordChangedAt = &now
		}
		recorder := passwordControllerRequest(t, user, "", gin.H{}, UserCtrler{}.Info)
		var body struct {
			Return struct {
				Required bool           `json:"passwordChangeRequired"`
				Reason   string         `json:"passwordChangeReason"`
				User     map[string]any `json:"user"`
			} `json:"return"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if !body.Return.Required || body.Return.Reason == "" {
			t.Fatalf("受限本人信息须包含改密状态: %s", recorder.Body.String())
		}
		if _, exposed := body.Return.User["password"]; exposed {
			t.Fatal("本人信息不能暴露密码密文")
		}
	}
}
