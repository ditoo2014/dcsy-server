package middleware

import (
	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

func StreamMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		/* 获取流信息 */

		name := ctx.Query("Name")
		apiKey := ctx.Query("ApiKey")
		/* 验证信息格式 */
		if 1 > len(name) && 1 > len(apiKey) {
			response.Fail(ctx, "流名称为空", nil)
			ctx.Abort()
			return
		}

		/* 验证流 */

		var dbStream = model.Stream{Name: name, ApiKey: apiKey}
		var res bool
		streamDao := dao.NewStreamDAO(ctx)
		dbStream, res = streamDao.ExistName(dbStream)
		if false == res {
			response.Fail(ctx, "流不存在", nil)
			ctx.Abort()
			return
		}

		/* 插入信息 */

		ctx.Set("streamInfo", dbStream)

		ctx.Next()
	}
}

func StreamMiddlewareNameUpdate() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		/* 获取流信息 */

		name := ctx.Query("ApiKey")

		/* 验证信息格式 */

		if 1 > len(name) {
			response.Fail(ctx, "流ApiKey为空", nil)
			ctx.Abort()
			return
		}

		/* 验证流 */

		var dbStream = model.Stream{ApiKey: name}
		var res bool
		streamDao := dao.NewStreamDAO(ctx)
		dbStream, res = streamDao.ExistApiKey(dbStream)
		if false == res {
			response.Fail(ctx, "流不存在", nil)
			ctx.Abort()
			return
		}

		/* 插入信息 */

		ctx.Set("streamInfo", dbStream)

		ctx.Next()
	}
}
