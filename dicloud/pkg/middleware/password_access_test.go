package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"dicloud/models/model"
	"dicloud/pkg/common"
	"dicloud/pkg/passwordpolicy"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func usePasswordPolicy(t *testing.T) {
	t.Helper()
	config := passwordpolicy.Current()
	t.Cleanup(func() { _ = passwordpolicy.Configure(config) })
	if err := passwordpolicy.Configure(passwordpolicy.Config{Enabled: true, Mode: passwordpolicy.CalendarQuarter, Timezone: "Asia/Shanghai"}); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
}

// 改密受限时只放行精确白名单，超级管理员、业务和采集页面都不得绕过。
func TestRequiredPasswordChangeAccess(t *testing.T) {
	usePasswordPolicy(t)
	now := time.Date(2026, time.October, 9, 12, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*60*60))
	for _, item := range []struct {
		method, path string
		allowed      bool
	}{
		{http.MethodPost, "/auth/info", true},
		{http.MethodPost, "/auth/changepassword", true},
		{http.MethodGet, "/auth/info", false},
		{http.MethodPost, "/auth/info/other", false},
		{http.MethodPost, "/auth/changepassword/other", false},
		{http.MethodPost, "/auth/resetuserpassword", false},
		{http.MethodPost, "/auth/business/user/permissions", false},
		{http.MethodPost, "/business/equipment/find", false},
		{http.MethodPost, "/tolink/gateway/findpage", false},
	} {
		t.Run(item.method+item.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(item.method, item.path, nil)
			if allowed := ensurePasswordAccess(ctx, model.User{Isadmin: 1}, now); allowed != item.allowed {
				t.Fatalf("访问判断错误: allowed=%t", allowed)
			}
			if !item.allowed {
				var body struct {
					Return struct {
						ErrorCode string `json:"errorCode"`
						Required  bool   `json:"passwordChangeRequired"`
						Reason    string `json:"passwordChangeReason"`
					} `json:"return"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if recorder.Code != http.StatusForbidden || !ctx.IsAborted() || body.Return.ErrorCode != passwordpolicy.ChangeRequiredCode || !body.Return.Required || body.Return.Reason != passwordpolicy.ReasonPasswordUnset {
					t.Fatalf("受限请求应使用独立403状态: %s", recorder.Body.String())
				}
			}
		})
	}
}

// 即使 Redis 中仍有旧会话，数据库已撤销的 token 也不能通过认证。
func TestAuthoritativeTokenAndSessionValidation(t *testing.T) {
	usePasswordPolicy(t)
	token, _, err := common.ReleaseToken("test-user")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		name          string
		lookupError   error
		currentToken  string
		sessionActive bool
		wantCode      int
		wantOK        bool
	}{
		{"旧令牌已撤销", gorm.ErrRecordNotFound, "", true, http.StatusUnauthorized, false},
		{"数据库故障", errors.New("database unavailable"), "", true, http.StatusServiceUnavailable, false},
		{"Redis会话已失效", nil, token, false, http.StatusUnauthorized, false},
		{"令牌与会话均有效", nil, token, true, http.StatusOK, true},
	} {
		t.Run(item.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/auth/info", nil)
			ctx.Request.Header.Set("Authorization", "Bearer "+token)
			sessionTouched := false
			user, ok := authenticateUserWith(ctx, func(_ *gin.Context, username, currentToken string) (model.User, error) {
				return model.User{Username: username, Token: item.currentToken, Password: "never expose bcrypt"}, item.lookupError
			}, func(ctx *gin.Context, _, _ string) bool {
				sessionTouched = true
				if !item.sessionActive {
					response.ResponseCode(ctx, http.StatusUnauthorized, nil, "登录超时，请重新登录")
					ctx.Abort()
				}
				return item.sessionActive
			})
			if ok != item.wantOK || recorder.Code != item.wantCode {
				t.Fatalf("认证结果错误: ok=%t, response=%s", ok, recorder.Body.String())
			}
			if item.lookupError != nil && sessionTouched {
				t.Fatal("数据库认证失败不能续期旧会话")
			}
			if ok && user.Password != "" {
				t.Fatal("认证上下文不能暴露密码哈希")
			}
		})
	}
}
