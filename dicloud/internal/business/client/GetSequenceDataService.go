package client

import (
	"context"
	"dicloud/internal/common/grpc/tolink_grpc"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func GetTolinkLastSequenceDataBatch(ctx context.Context, requests []*tolink_grpc.GetDataDto) (*tolink_grpc.TolinkBatchSequenceData, error) {
	return getTolinkClient().GetLastDataByTimeBatch(ctx, &tolink_grpc.BatchGetDataDto{Requests: requests})
}

// 获取采集云数据 - 根据设备编码, 获取采集云SOH实时数据(统计健康度)
func GetTolinkSequence(ctx *gin.Context, g *tolink_grpc.Gateway) (res *tolink_grpc.SequenceData) {

	/**
	===========================
	== [GRPC] - TOLINK调用
	===========================
	*/

	// grpcConn := GetGrpcConnectionByServiceName(enum.XC_GRPC_SERVICE_TOLINK)
	// if grpcConn == nil {
	// 	return
	// }

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		g.DbName = fmt.Sprint(nameTem)
	}

	result, err := getTolinkClient().GetGatewayIdSOH(ctx, g)

	// 调用失败
	if err != nil {
		log.Println("[GRPC-TolinkClient] GRPC call Tolink GetGatewayIdSOH fail; GatewayId: ", g.GetGatewayId(), " err: ", err)
		return nil
	}

	return result.SequenceData
}

// 根据时间获取最新采集云数据 - 根据设备编码, 获取采集云最新一条数据
func GetTolinkLastSequenceData(ctx *gin.Context, g *tolink_grpc.GetDataDto) (res *tolink_grpc.SequenceData) {

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

	result, err := getTolinkClient().GetLastDataByTime(ctx, g)

	// 调用失败
	if err != nil {
		log.Println("[GRPC-TolinkClient] GRPC call Tolink GetTolinkLastSequenceData fail; GatewayId: ", g.GetGatewayId(), " dataId: ", g.DataId, " err: ", err)
		return nil
	}

	return result.SequenceData
}
