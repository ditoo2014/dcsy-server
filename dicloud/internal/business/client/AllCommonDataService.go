package client

import (
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/internal/common/dto"
	"dicloud/internal/common/grpc/tolink_grpc"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

// 添加采集云通用数据 - 网关数据/数据池数据
func AddBusinessTolinkCommonData(ctx *gin.Context, g *tolink_grpc.Gateway, e *model.Equipment, statisticDataList []dto.StatisticData) (bool, error) {

	// 开启事务, 添加设备
	businesscommondataDao := dao.NewBusinessCommonDataDao(ctx)
	res, tx := businesscommondataDao.AddBusinessCommonDataAll(e)
	if !res {
		log.Println("[GRPC-TolinkClient] business create common data fail")
		tx.Rollback()
		return false, fmt.Errorf("添加业务云数据出错")
	}

	/**
	===========================
	== [GRPC] - TOLINK调用
	===========================
	*/

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		g.DbName = fmt.Sprint(nameTem)
	}
	result, err := getTolinkClient().AddGateway(ctx, g)

	// 调用失败, 回滚事务
	if err != nil {
		log.Println("[GRPC-TolinkClient] GRPC call Tolink addGateway fail; GatewayId: ", g.GetGatewayId(), " err: ", err)
		tx.Rollback()
		return false, err
	}
	if result.Code != 200 {
		log.Printf("[GRPC-TolinkClient] GRPC call Response code = %d, message = %s\n", result.Code, result.Message)
		tx.Rollback()
		return false, err
	}

	// 添加统计池数据 - 不需要, 转为由统计任务统计
	requestStatisticStreamDataAdd(statisticDataList, int(e.ID))
	// 提交事务
	tx.Commit()
	return true, nil
}

// 添加采集云通用数据 - 网关数据/数据池数据
func DeleteBusinessTolinkCommonData(ctx *gin.Context, g *tolink_grpc.Gateway, e *model.Equipment) (bool, error) {

	// 开启事务, 添加设备
	businesscommondataDao := dao.NewBusinessCommonDataDao(ctx)
	res, tx := businesscommondataDao.DeleteBusinessCommonDataAll(e)
	if !res {
		log.Println("[GRPC-TolinkClient] delete business common data fail")
		tx.Rollback()
		return false, fmt.Errorf("删除业务云数据出错")
	}

	/**
	===========================
	== [GRPC] - TOLINK调用
	===========================
	*/

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		g.DbName = fmt.Sprint(nameTem)
	}
	result, err := getTolinkClient().DeleteGateway(ctx, g)

	if err != nil {
		log.Println("[GRPC-TolinkClient] GRPC call Tolink deleteGateway fail; GatewayId: ", g.GetGatewayId(), " err: ", err)
		tx.Rollback()
		return false, err
	}
	if result.Code != 200 {
		log.Printf("[GRPC-TolinkClient] GRPC call Response code = %d, message = %s\n", result.Code, result.Message)
		tx.Rollback()
		return false, err
	}

	// 提交事务
	tx.Commit()
	return true, nil
}
