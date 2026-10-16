package middleware

import (
	"log"
	"net/http"

	"dicloud/pkg/authsession"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

func ensureActiveSession(ctx *gin.Context, username string, token string) bool {
	// 每次接口请求都会触发 Redis 会话校验；校验通过时刷新 30 分钟空闲过期时间。
	active, err := authsession.Touch(ctx, username, token)
	if err != nil {
		// 缓存故障时拒绝访问并记录原因，保留浏览器令牌，恢复后允许重新校验。
		log.Printf("[AUTH] 校验登录会话失败 user=%q: %v", username, err)
		response.ResponseCode(ctx, http.StatusServiceUnavailable, nil, "暂时无法校验登录状态，请稍后重试")
		ctx.Abort()
		return false
	}
	if active {
		return true
	}
	response.ResponseCode(ctx, http.StatusUnauthorized, nil, "登录超时，请重新登录")
	ctx.Abort()
	return false
}
