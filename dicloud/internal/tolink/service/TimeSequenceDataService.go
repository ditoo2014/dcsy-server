package service

import (
	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	"log"

	"github.com/gin-gonic/gin"
)

// 通过TDengine时序数据, 查询实时在线网关
/* 获取连接但未添加的网关列表 */
func FindLinkedUpGateways(ctx *gin.Context) (ids []string, times []int64, err error) {

	// 查询所有表名
	gatewayIds, err := dao.FindExistGateways()

	if err != nil {
		log.Println("[TDengineDataService] FindLinkedUpGateways fail", err)
		return
	}

	if len(gatewayIds) > 0 {
		gatewayDao := dao.NewGatewayDAO(ctx)
		for _, v := range gatewayIds {
			_, res := gatewayDao.ExistID(model.Gateway{GatewayId: v})
			if res {
				continue
			}
			// 获取连接但未添加的网关列表
			ids = append(ids, v)
		}
	}

	// 查询所有最新数据
	if len(ids) > 0 {
		times, err = dao.FindGatewayNewTime(ids)
	}

	return
}
