package middleware

import (
	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

func ApiMiddleware() gin.HandlerFunc {

	return func(ctx *gin.Context) {

		//获取密钥
		apikey := ctx.Query("Apikey")

		//验证信息格式
		if len(apikey) == 0 {
			response.Fail(ctx, "密钥为空", nil)
			ctx.Abort()
			return
		}
		streamDao := dao.NewStreamDAO(ctx)

		//验证apikey是否存在
		dbStream, res := streamDao.ExistApiKey(model.Stream{ApiKey: apikey})
		if !res {
			response.Fail(ctx, "密钥不存在", nil)
			ctx.Abort()
			return
		}
		//插入信息
		ctx.Set("streamInfo", dbStream)
		ctx.Next()
	}

}
