package middleware

import (
	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

func GatewayMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		/* 获取网关信息 */

		name := ctx.Query("Name")
		gatewayID := ctx.Query("GatewayID")

		/* 验证网关 */

		var dbGateway = model.Gateway{Name: name, GatewayId: gatewayID}
		var res bool

		gatewaDao := dao.NewGatewayDAO(ctx)
		if len(gatewayID) != 0 {
			dbGateway, res = gatewaDao.ExistID(dbGateway)
			if !res {
				response.Fail(ctx, "网关不存在", nil)
				ctx.Abort()
				return
			}
		} else if len(name) != 0 {
			dbGateway, res = gatewaDao.ExistName(dbGateway)
			if !res {
				response.Fail(ctx, "网关不存在", nil)
				ctx.Abort()
				return
			}
		} else {
			response.Fail(ctx, "网关参数为空", nil)
			ctx.Abort()
			return
		}

		/* 插入信息 */

		ctx.Set("gatewayInfo", dbGateway)

		ctx.Next()
	}
}
