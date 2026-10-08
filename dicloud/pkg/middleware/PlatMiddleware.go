package middleware

import (
	"dicloud/models/model"

	"github.com/gin-gonic/gin"
)

/* 接口定义的哪个平台,一般为权限配置相关接口,各个平台接口管理各自平台的权限 */
// -- 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云
func PlatMiddleware(platId int) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		/* 插入用户信息 */
		userInfoCache, _ := ctx.Get("userInfo")
		userInfo := userInfoCache.(model.User)

		userInfo.PlatId = platId
		ctx.Set("userInfo", userInfo)

		ctx.Next()
	}
}
