package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"dicloud/models/model"
	"dicloud/pkg/common"
	"dicloud/pkg/passwordpolicy"
	"dicloud/util/gormutil"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type lookupAuthenticatedUser func(*gin.Context, string, string) (model.User, error)
type touchAuthenticatedSession func(*gin.Context, string, string) bool

// authenticateUser 统一所有用户认证入口；每次从数据库核验最新令牌和密码状态。
// Redis 仍用于空闲超时校验，但缓存中的旧用户记录不能让失效令牌重新通过。
func authenticateUser(ctx *gin.Context) (model.User, bool) {
	return authenticateUserWith(ctx, func(ctx *gin.Context, username, token string) (model.User, error) {
		var user model.User
		db := gormutil.GetGormDB(ctx, "")
		if db == nil {
			return user, errors.New("用户数据库未连接")
		}
		err := db.WithContext(ctx.Request.Context()).Where("username = ? AND token = ?", username, token).First(&user).Error
		return user, err
	}, ensureActiveSession)
}

func authenticateUserWith(ctx *gin.Context, lookup lookupAuthenticatedUser, touch touchAuthenticatedSession) (model.User, bool) {
	var emptyUser model.User
	authorization := ctx.GetHeader("authorization")
	if !strings.HasPrefix(authorization, "Bearer ") || len(authorization) <= len("Bearer ") {
		response.Fail(ctx, "auth header格式错误", nil)
		ctx.Abort()
		return emptyUser, false
	}
	tokenString := strings.TrimPrefix(authorization, "Bearer ")
	token, claims, err := common.ParseToken(tokenString)
	if err != nil || token == nil || !token.Valid || claims.UserName == "" {
		response.ResponseCode(ctx, http.StatusUnauthorized, nil, "登录已失效，请重新登录")
		ctx.Abort()
		return emptyUser, false
	}
	user, err := lookup(ctx, claims.UserName, tokenString)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 数据库故障时拒绝请求，但保留登录态供恢复后重试，不误导用户反复登录。
		response.ResponseCode(ctx, http.StatusServiceUnavailable, nil, "暂时无法校验账号状态，请稍后重试")
		ctx.Abort()
		return emptyUser, false
	}
	if err != nil || user.Username != claims.UserName || user.Token != tokenString {
		response.ResponseCode(ctx, http.StatusUnauthorized, nil, "登录已失效，请重新登录")
		ctx.Abort()
		return emptyUser, false
	}
	if !touch(ctx, claims.UserName, tokenString) {
		return emptyUser, false
	}
	user.Password = ""
	if !ensurePasswordAccess(ctx, user, time.Now()) {
		return emptyUser, false
	}
	return user, true
}

// passwordRecoveryRequest 只允许精确的本人状态和改密请求，避免路径前缀绕过。
// 退出登录接口原本允许匿名调用，无须在此列入白名单。
func passwordRecoveryRequest(ctx *gin.Context) bool {
	return ctx.Request.Method == http.MethodPost &&
		(ctx.Request.URL.Path == "/auth/info" || ctx.Request.URL.Path == "/auth/changepassword")
}

func ensurePasswordAccess(ctx *gin.Context, user model.User, now time.Time) bool {
	status := passwordpolicy.Evaluate(user.PasswordChangedAt, user.MustChangePassword, now)
	ctx.Set(passwordpolicy.ContextStatusKey, status)
	if !status.PasswordChangeRequired || passwordRecoveryRequest(ctx) {
		return true
	}
	// 在判断超级管理员权限之前执行，所有用户均不能绕过季度改密。
	response.ResponseCode(ctx, http.StatusForbidden, gin.H{
		"errorCode":              passwordpolicy.ChangeRequiredCode,
		"passwordChangeRequired": status.PasswordChangeRequired,
		"passwordChangeReason":   status.PasswordChangeReason,
		"passwordExpiresAt":      status.PasswordExpiresAt,
	}, passwordpolicy.ReasonMessage(status.PasswordChangeReason))
	ctx.Abort()
	return false
}
