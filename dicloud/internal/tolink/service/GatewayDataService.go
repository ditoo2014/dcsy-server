package service

import (
	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

func FindGatewayDatas(ctx *gin.Context, gatewayId string) (resMsg gin.H, err error) {

	dataTem := model.Gateway{}

	gatewayDao := dao.NewGatewayDAO(ctx)
	dataTem, res := gatewayDao.FindGatewayDatas(model.Gateway{GatewayId: gatewayId})
	if !res {
		response.Fail(ctx, "网关数据点为空", nil)
		return
	}
	resMsg = gin.H{"acount": len(dataTem.GatewayDatas), "data": dataTem}

	return
}
