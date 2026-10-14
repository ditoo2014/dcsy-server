package middleware

import "github.com/gin-gonic/gin"

// AuthMiddleware 校验登录会话以及季度改密状态。
func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, ok := authenticateUser(ctx)
		if !ok {
			return
		}
		ctx.Set("userInfo", user)
		ctx.Next()
	}
}

// AuthWithPlatMiddleware 在统一登录校验后绑定请求所属平台。
func AuthWithPlatMiddleware(platId int) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user, ok := authenticateUser(ctx)
		if !ok {
			return
		}
		user.PlatId = platId
		ctx.Set("userInfo", user)
		ctx.Next()
	}
}
