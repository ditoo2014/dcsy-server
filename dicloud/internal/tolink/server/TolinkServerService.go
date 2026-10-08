package grpcserver

import (
	"context"
	"dicloud/internal/common/dto"
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	"dicloud/pkg/cache"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

func (TolinkServer) AddGateway(ctx context.Context, request *tolink_grpc.Gateway) (*tolink_grpc.TolinkResponse, error) {
	log.Printf("[GRPC-TolinkServer] ADD GatewayId = %s, Name = %s\n", request.GatewayId, request.Name)

	// 响应体
	var responseData = &tolink_grpc.TolinkResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkServer] receive: name = %s, gatewayId = %s", request.Name, request.GatewayId),
	}

	// 网关数据封装处理
	res, gatewayTem, gatewaySTRTem := formatGateway(request)
	if !res {
		responseData.Code = 10501
		return responseData, nil
	}

	// 数据池数据封装处理// 网关数据封装处理
	res, streamTem := formatStream(request)
	if !res {
		responseData.Code = 10502
		return responseData, nil
	}

	/* 操作数据库 */
	{
		ginCtx := &gin.Context{}
		if len(request.DbName) > 0 {
			ginCtx.Set("dbName", request.DbName)
		}
		// TODO: acording to data mark. distingush plat data.
		commondataDao := dao.NewCommonDataDAO(ginCtx)
		// 数据库API 添加网关/网关数据; 除机型配置数据
		res, err := commondataDao.AddTolinkCommonDataAll(gatewayTem, gatewaySTRTem, streamTem)
		if !res || nil != err {
			responseData.Code = 10503
			return responseData, err
		}
	}

	return responseData, nil
}

func formatGateway(request *tolink_grpc.Gateway) (bool, model.Gateway, model.Gateway) {

	// 参数
	var config = model.Gateway{
		Name:         request.Name,
		GatewayId:    request.GatewayId,
		Model:        request.Model,
		Card:         request.Card,
		CreateUserId: int(request.CreateUserId),
		CorpId:       int(request.CorpId),
	}

	// 参数
	var configSTR = model.Gateway{
		Name:         request.Name,
		GatewayId:    request.GatewayId + "STR",
		Model:        request.Model,
		Card:         request.Card,
		CreateUserId: int(request.CreateUserId),
		CorpId:       int(request.CorpId),
	}

	{
		// 数据整理
		if len(request.GatewayDatas) > 0 {
			for _, dataDto := range request.GatewayDatas {
				var dataTem = model.GatewayData{
					GatewayId: request.GatewayId,
					DataId:    dataDto.DataId,
					DataType:  int(dataDto.DataType),
					Name:      dataDto.Name,
					Dbconfig:  -1,
				}
				// 字符串数据
				// remove STR gateway.
				// if len(dto.DP_STRDataMap[dataTem.DataId]) > 0 {
				// 	dataTem.DataType = 2
				// 	dataTem.GatewayId = dataTem.GatewayId + "STR"
				// 	configSTR.GatewayDatas = append(configSTR.GatewayDatas, dataTem)
				// 	continue
				// }
				config.GatewayDatas = append(config.GatewayDatas, dataTem)
			}
		}
	}

	return true, config, configSTR
}

func formatStream(request *tolink_grpc.Gateway) (bool, model.Stream) {

	// 参数
	var config = model.Stream{
		Admin:        "ditoo",
		Name:         request.Name,
		ApiKey:       request.ApiKey,
		CreateUserId: int(request.CreateUserId),
		CorpId:       int(request.CorpId),
		StreamCode:   request.GatewayId,
	}

	if len(request.GatewayDatas) > 0 {
		for _, dataDto := range request.GatewayDatas {
			var dataTem = model.StreamLabelUnit{
				GatewayId:   request.GatewayId,
				GatewayName: request.Name,
				DataId:      dataDto.DataId,
				DataName:    dataDto.Name,
				DataType:    fmt.Sprint(dataDto.DataType),
				LabelId:     dataDto.LabelId,
				Name:        dataDto.Name,
				Formula:     "x",
				Unformula:   "x",
			}

			// 字符串数据
			if len(dto.DP_STRDataMap[dataDto.DataId]) > 0 {
				dataTem.DataType = fmt.Sprint(2)
				dataTem.GatewayId = dataTem.GatewayId + "STR"
			}
			config.List = append(config.List, dataTem)
		}
	}

	return true, config
}

func (TolinkServer) DeleteGateway(ctx context.Context, request *tolink_grpc.Gateway) (*tolink_grpc.TolinkResponse, error) {
	log.Printf("[GRPC-TolinkServer] DELETE GatewayId = %s, Name = %s\n", request.GatewayId, request.Name)

	// 响应体
	var responseData = &tolink_grpc.TolinkResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkServer] receive: name = %s, gatewayId = %s", request.Name, request.GatewayId),
	}

	var gatewayTem = model.Gateway{
		GatewayId: request.GatewayId,
	}
	// 网关数据处理
	// TODO: acording to data mark. distingush plat data.
	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}
	gatewayDao := dao.NewGatewayDAO(ginCtx)
	gatewayTem, res := gatewayDao.ExistID(gatewayTem)
	if !res {
		log.Println("[GRPC-TolinkServer] deleteGateway gatewayId not exist")
		responseData.Code = 10504
		return responseData, errors.New("网关序列号不存在")
	}

	/* 操作数据库 */
	{
		// TODO: acording to data mark. distingush plat data.
		commondataDao := dao.NewCommonDataDAO(&gin.Context{})
		// 数据库API 添加网关/网关数据; 除机型配置数据
		res, err := commondataDao.DeleteTolinkCommonDataAll(gatewayTem)
		if !res || nil != err {
			responseData.Code = 10505
			return responseData, err
		}
	}

	return responseData, nil
}

func (TolinkServer) GetGatewayIdSOH(ctx context.Context, request *tolink_grpc.Gateway) (*tolink_grpc.TolinkSequenceData, error) {
	log.Printf("[GRPC-TolinkServer] GetGatewayIdSOH GatewayId = %s ", request.GatewayId)

	// 响应体
	var responseData = &tolink_grpc.TolinkSequenceData{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkServer] receive: gatewayId = %s", request.GatewayId),
	}

	var gatewayTem = model.Gateway{
		GatewayId: request.GatewayId,
	}
	// 网关数据处理
	// TODO: acording to data mark. distingush plat data.
	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}
	gatewayDao := dao.NewGatewayDAO(ginCtx)
	gatewayTem, res := gatewayDao.ExistID(gatewayTem)
	if !res {
		log.Println("[GRPC-TolinkServer] GetGatewayIdSOH gatewayId not exist")
		responseData.Code = 10505
		return responseData, errors.New("网关序列号不存在")
	}

	seqData := tolink_grpc.SequenceData{}

	// 从Redis获取
	var (
		_sta  int64
		_val  float64
		_time int64
	)
	result := cache.HMGet(ginCtx, "gateway:"+gatewayTem.GatewayId, dto.DP_DEFAULT_SOH_DATAID)
	if len(result) > 0 {
		if _, ok := result[0].(string); ok {
			if n, err := fmt.Sscanf(result[0].(string), "%d %f %d", &_sta, &_val, &_time); n == 3 && err == nil {
				seqData.Val = _val
				seqData.CreateTime = _time
				seqData.Status = _sta
				responseData.SequenceData = &seqData
				return responseData, nil
			}
		}
	}

	// 从时序数据库获取 - 最新30条数据
	gatewayDataDao := dao.NewGatewayDataDao(ginCtx)
	dataTem, res := gatewayDataDao.FindOne(model.GatewayData{GatewayId: request.GatewayId, DataId: dto.DP_DEFAULT_SOH_DATAID})

	if !res {
		log.Println("[GRPC-TolinkServer] GetGatewayIdSOH dataId not exist")
		return nil, errors.New("数据点ID不存在")
	}

	formula := "x"
	times, values, status, err := dao.GetDataNew(dataTem, formula, false)
	if nil != err {
		log.Println("[GRPC-TolinkServer] GetGatewayIdSOH: ", err)
		return nil, err
	}

	if len(values) > 0 {
		varTem, err := strconv.ParseFloat(fmt.Sprint(values[0]), 64)
		if err != nil {
			varTem = 0
		}
		seqData.Val = varTem
	}
	seqData.CreateTime = times[0]
	seqData.Status = status[0]
	responseData.SequenceData = &seqData
	return responseData, nil
}

func (TolinkServer) GetLastDataByTime(ctx context.Context, request *tolink_grpc.GetDataDto) (*tolink_grpc.TolinkSequenceData, error) {
	log.Printf("[GRPC-TolinkServer] GetLastDataByTime GatewayId = %s ", request.GatewayId)

	// 响应体
	var responseData = &tolink_grpc.TolinkSequenceData{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkServer] receive: gatewayId = %s", request.GatewayId),
	}

	var gatewayTem = model.Gateway{
		GatewayId: request.GatewayId,
	}
	// 网关数据处理
	// TODO: acording to data mark. distingush plat data.
	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	existData := false
	result := cache.HMGet(ginCtx, "gatewayId:exist", request.GatewayId)
	if len(result) > 0 {
		if dbGatewyIdExistStr, ok := result[0].(string); ok {
			if len(dbGatewyIdExistStr) > 0 && dbGatewyIdExistStr == request.GatewayId {
				// redis校验成功
				existData = true
			}
		}
	}
	if !existData {
		gatewayDao := dao.NewGatewayDAO(ginCtx)
		_, res := gatewayDao.ExistID(gatewayTem)
		if !res {
			log.Println("[GRPC-TolinkServer] GetLastDataByTime gatewayId not exist")
			responseData.Code = 10505
			return responseData, errors.New("网关序列号不存在")
		}
		cache.HMSet(ginCtx, "gatewayId:exist", map[string]interface{}{request.GatewayId: request.GatewayId})
	}

	existDataId := false
	result = cache.HMGet(ginCtx, "dataId:exist", request.DataId)
	if len(result) > 0 {
		if dbDataIdExistStr, ok := result[0].(string); ok {
			if len(dbDataIdExistStr) > 0 && dbDataIdExistStr == request.DataId {
				// redis校验成功
				existDataId = true
			}
		}
	}
	if !existDataId {
		// 从时序数据库获取 - 最新30条数据
		gatewayDataDao := dao.NewGatewayDataDao(ginCtx)
		_, res := gatewayDataDao.FindOne(model.GatewayData{GatewayId: request.GatewayId, DataId: request.DataId})

		if !res {
			log.Println("[GRPC-TolinkServer] GetLastDataByTime dataId not exist")
			return nil, errors.New("数据点ID不存在")
		}
		cache.HMSet(ginCtx, "dataId:exist", map[string]interface{}{request.DataId: request.DataId})
	}

	seqData := tolink_grpc.SequenceData{}
	var dataTem = model.GatewayData{GatewayId: request.GatewayId, DataId: request.DataId}
	// 获取最新一条数据
	times, values, status, err := dao.GetLastDataByTime(dataTem, request.CurTime, "x", false)
	if nil != err || len(values) == 0 {
		log.Println("[GRPC-TolinkServer] GetLastDataByTime getdata err: ", err)
		return nil, errors.New("数据查询失败")
	}

	varTem, err := strconv.ParseFloat(fmt.Sprint(values[0]), 64)
	if err != nil {
		log.Println("[GRPC-TolinkServer] GetLastDataByTime format data err: ", err)
		return responseData, errors.New("数据类型转换失败")
	}
	seqData.Val = varTem
	seqData.CreateTime = times[0]
	seqData.Status = status[0]
	responseData.SequenceData = &seqData
	return responseData, nil
}

const (
	maxHistoryBatchSize    = 1000
	historyBatchWorkerSize = 16
)

func (server TolinkServer) GetLastDataByTimeBatch(ctx context.Context, request *tolink_grpc.BatchGetDataDto) (*tolink_grpc.TolinkBatchSequenceData, error) {
	responseData := &tolink_grpc.TolinkBatchSequenceData{
		Code:    200,
		Message: "batch history query completed",
	}
	if request == nil || len(request.Requests) == 0 {
		return responseData, nil
	}
	if len(request.Requests) > maxHistoryBatchSize {
		responseData.Code = 10506
		responseData.Message = fmt.Sprintf("batch size exceeds limit %d", maxHistoryBatchSize)
		return responseData, nil
	}

	responseData.Data = make([]*tolink_grpc.TolinkBatchSequenceItem, len(request.Requests))
	jobs := make(chan int)
	workerCount := historyBatchWorkerSize
	if len(request.Requests) < workerCount {
		workerCount = len(request.Requests)
	}

	var waitGroup sync.WaitGroup
	waitGroup.Add(workerCount)
	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		go func() {
			defer waitGroup.Done()
			for requestIndex := range jobs {
				itemRequest := request.Requests[requestIndex]
				item := &tolink_grpc.TolinkBatchSequenceItem{Request: itemRequest}
				if itemRequest == nil {
					item.Error = "request is nil"
					responseData.Data[requestIndex] = item
					continue
				}
				result, err := server.GetLastDataByTime(ctx, itemRequest)
				if err != nil {
					item.Error = err.Error()
				} else if result != nil {
					item.SequenceData = result.SequenceData
				}
				responseData.Data[requestIndex] = item
			}
		}()
	}

	for requestIndex := range request.Requests {
		select {
		case jobs <- requestIndex:
		case <-ctx.Done():
			close(jobs)
			waitGroup.Wait()
			return nil, ctx.Err()
		}
	}
	close(jobs)
	waitGroup.Wait()
	return responseData, nil
}
