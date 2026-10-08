package middleware

import (
	"net/http"

	"dicloud/pkg/authsession"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

func ensureActiveSession(ctx *gin.Context, username string, token string) bool {
	// 每次接口请求都会触发 Redis 会话校验；校验通过时刷新 30 分钟空闲过期时间。
	if authsession.Touch(ctx, username, token) {
		return true
	}
	response.ResponseCode(ctx, http.StatusUnauthorized, nil, "登录超时，请重新登录")
	ctx.Abort()
	return false
}
