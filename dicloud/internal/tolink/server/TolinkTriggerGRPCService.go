package grpcserver

import (
	"context"
	"dicloud/internal/common/dao"
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/internal/common/model"
	tolinkdao "dicloud/internal/tolink/dao"
	tolinkmodel "dicloud/internal/tolink/model"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func (TolinkServer) AddTrigger(ctx context.Context, request *tolink_grpc.TriggerConfig) (*tolink_grpc.TolinkTriggerResponse, error) {
	log.Printf("[GRPC-TolinkTriggerServer] Add Trigger ApiKey = %s, Code = %s\n", request.ApiKey, request.Code)

	// 响应体
	var responseData = &tolink_grpc.TolinkTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkTriggerServer] ADD ApiKey = %s, Code = %s\n", request.ApiKey, request.Code),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	// 数据转换
	trigger := baseTriggerDataFormat(request)

	// 验证数据池是否存在
	streamDao := tolinkdao.NewStreamDAO(ginCtx)
	dbstream, res := streamDao.ExistApiKey(tolinkmodel.Stream{ApiKey: trigger.ApiKey})
	if !res {
		responseData.Code = 10501
		responseData.Message = "流不存在"
		return responseData, nil
	}
	// 查询数据点
	dbstream, err := streamDao.StreamLabels(tolinkmodel.Stream{ApiKey: trigger.ApiKey})
	if len(dbstream.List) == 0 || err != nil {
		responseData.Code = 10502
		responseData.Message = "数据不存在"
		return responseData, nil
	}
	// 对比数据点
	isExist := -1
	for idx, labelTem := range dbstream.List {
		if labelTem.LabelId == trigger.LabelId {
			isExist = idx
			break
		}
	}
	if isExist == -1 {
		responseData.Code = 10503
		responseData.Message = "数据点不存在"
		return responseData, nil
	}
	// 触发器标记
	trigger.Marktrigger = dbstream.List[isExist].GatewayId + " " + dbstream.List[isExist].DataId

	// 如果数据为空, 填充默认数据
	if trigger.AlarmUser.AlarmCode == "" {
		trigger.AlarmUser.AlarmCode = trigger.Code
	}
	if trigger.StatisticConfig.StatisticCode == "" {
		trigger.StatisticConfig.StatisticCode = trigger.Code
	}
	if trigger.StatisticConfig.StatisticName == "" {
		trigger.StatisticConfig.StatisticName = trigger.StatisticConfig.StatisticCode
	}
	if len(trigger.AlarmUser.AlarmType) < 1 {
		trigger.AlarmUser.AlarmType = ","
	}
	if len(trigger.AlarmUser.AlarmUsers) < 1 {
		trigger.AlarmUser.AlarmUsers = ","
	}
	if len(trigger.Formula) < 1 {
		trigger.Formula = "x"
	}

	/* 数据库连接 */
	triggerDao := dao.NewTriggerDao(ginCtx)
	_, res = triggerDao.ExistEquipmentAlarmCode(trigger)
	if res {
		responseData.Code = 10504
		responseData.Message = "创建失败, 已存在"
		return responseData, nil
	}

	res = triggerDao.Create(trigger)
	if !res {
		responseData.Code = 10505
		responseData.Message = "创建失败"
		return responseData, nil
	}

	return responseData, nil
}

func (TolinkServer) DeleteTrigger(ctx context.Context, request *tolink_grpc.TriggerConfig) (*tolink_grpc.TolinkTriggerResponse, error) {
	log.Printf("[GRPC-TolinkTriggerServer] Delete Trigger Id = %d\n", request.Id)

	// 响应体
	var responseData = &tolink_grpc.TolinkTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkTriggerServer] Delete ApiKey = %s, Code = %s\n", request.ApiKey, request.Code),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseTriggerDataFormat(request)

	/* 数据库连接 */
	triggerDao := dao.NewTriggerDao(ginCtx)
	_, res := triggerDao.ExistId(trigger)
	if !res {
		responseData.Code = 10501
		responseData.Message = "不存在的数据"
		return responseData, nil
	}

	// REMOVE:
	res = triggerDao.Delete(trigger)
	if !res {
		responseData.Code = 10502
		responseData.Message = "删除失败"
		return responseData, nil
	}

	return responseData, nil
}

func (TolinkServer) UpdateTrigger(ctx context.Context, request *tolink_grpc.TriggerConfig) (*tolink_grpc.TolinkTriggerResponse, error) {
	log.Printf("[GRPC-TolinkTriggerServer] Update Trigger ApiKey = %s, Code = %s\n", request.ApiKey, request.Code)

	// 响应体
	var responseData = &tolink_grpc.TolinkTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkTriggerServer] Update ApiKey = %s, Code = %s\n", request.ApiKey, request.Code),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseTriggerDataFormat(request)

	/* 数据库连接 */
	triggerDao := dao.NewTriggerDao(ginCtx)
	_, res := triggerDao.ExistId(trigger)
	if !res {
		responseData.Code = 10501
		responseData.Message = "数据不存在"
		return responseData, nil
	}
	_, res = triggerDao.UpdateExistEquipmentAlarmCode(trigger)
	if res {
		responseData.Code = 10502
		responseData.Message = "同名配置已存在"
		return responseData, nil
	}

	// 验证数据池是否存在
	streamDao := tolinkdao.NewStreamDAO(ginCtx)
	dbstream, res := streamDao.ExistApiKey(tolinkmodel.Stream{ApiKey: trigger.ApiKey})
	if !res {
		responseData.Code = 10503
		responseData.Message = "流不存在"
		return responseData, nil
	}
	// 查询数据点
	dbstream, err := streamDao.StreamLabels(tolinkmodel.Stream{ApiKey: trigger.ApiKey})
	if len(dbstream.List) == 0 || err != nil {
		responseData.Code = 10504
		responseData.Message = "数据不存在"
		return responseData, nil
	}
	// 对比数据点
	isExist := -1
	for idx, labelTem := range dbstream.List {
		if labelTem.LabelId == trigger.LabelId {
			isExist = idx
			break
		}
	}
	if isExist == -1 {
		responseData.Code = 10505
		responseData.Message = "数据点不存在"
		return responseData, nil
	}
	// 触发器标记
	trigger.Marktrigger = dbstream.List[isExist].GatewayId + " " + dbstream.List[isExist].DataId

	// 更新触发器
	res = triggerDao.Update(trigger)
	if !res {
		responseData.Code = 10506
		responseData.Message = "更新失败"
		return responseData, nil
	}

	return responseData, nil
}

func (TolinkServer) FindTrigger(ctx context.Context, request *tolink_grpc.TriggerConfig) (*tolink_grpc.TolinkTriggerResponse, error) {
	log.Printf("[GRPC-TolinkTriggerServer] Find Trigger EquipmentId = %d\n", request.EquipmentId)

	// 响应体
	var responseData = &tolink_grpc.TolinkTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkTriggerServer] Find ApiKey = %s, Code = %s\n", request.ApiKey, request.Code),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseTriggerDataFormat(request)

	triggerDao := dao.NewTriggerDao(ginCtx)
	data, err := triggerDao.FindTriggerLogics(trigger)
	if err != nil {
		responseData.Code = 10501
		responseData.Message = "查询失败"
		return responseData, nil
	}

	// 处理敏感数据
	if len(data) > 0 {
		for idx := range data {
			data[idx].Marktrigger = ""
		}
	}

	// 转换返回数据
	dataStr, err := json.Marshal(data)
	if err != nil {
		log.Println("[GRPC-TolinkTriggerServer] A transform data fail, may have fields ignored")
		responseData.Message = "数据转换失败"
		return responseData, nil
	}
	var grpcTriggers = []*tolink_grpc.TriggerConfig{}
	err = json.Unmarshal(dataStr, &grpcTriggers)
	if err != nil {
		log.Println("[GRPC-TolinkTriggerServer] B transform data fail, may have fields ignored")
		responseData.Message = "数据转换失败"
		return responseData, nil
	}
	responseData.Triggers = grpcTriggers

	return responseData, nil
}

// trigger数据转换 - grpc数据 -> 本地数据
func baseTriggerDataFormat(request *tolink_grpc.TriggerConfig) model.TriggerConfig {

	// 通过json转换数据
	var srlRes = false
	strBytes, err := json.Marshal(request)
	var trigger = model.TriggerConfig{}
	if err == nil {
		err = json.Unmarshal(strBytes, &trigger)
		if err == nil {
			srlRes = true
		}
	}

	// 序列化转换失败, 手动操作
	if !srlRes {
		// 反序列化失败, 手动填充数据
		trigger = model.TriggerConfig{
			ID:           uint(request.Id),
			SiteId:       int(request.SiteId),
			EquipmentId:  int(request.EquipmentId),
			TriggerType:  int(request.TriggerType),
			TriggerModel: int(request.TriggerModel),
			Code:         request.Code,
			ApiKey:       request.ApiKey,
			LabelName:    request.LabelName,
			LabelId:      request.LabelId,
			Cycle:        request.Cycle,
			Timer:        request.Timer,
			Record:       request.Record,
			LastStrMatch: request.LastStrMatch,
			Marktrigger:  request.Marktrigger,
			Formula:      request.Formula,
			AlarmUser: model.AlarmUser{
				ID:           uint(request.AlarmUser.Id),
				TriggerId:    uint(request.AlarmUser.TriggerId),
				AlarmCode:    request.AlarmUser.AlarmCode,
				AlarmType:    request.AlarmUser.AlarmType,
				AlarmStage:   int(request.AlarmUser.AlarmStage),
				AlarmChannel: request.AlarmUser.AlarmChannel,
				AlarmUsers:   request.AlarmUser.AlarmUsers,
			},
			StatisticConfig: model.StatisticConfig{
				ID:                 uint(request.StatisticConfig.Id),
				TriggerId:          uint(request.StatisticConfig.TriggerId),
				StatisticCode:      request.StatisticConfig.StatisticCode,
				StatisticName:      request.StatisticConfig.StatisticName,
				DataSourceType:     int(request.StatisticConfig.DataSourceType),
				StatisticType:      request.StatisticConfig.StatisticType,
				UseStatisticValue:  request.StatisticConfig.UseStatisticValue,
				StatisticValue:     request.StatisticConfig.StatisticValue,
				StatisticValueType: int(request.StatisticConfig.StatisticValueType),
			},
		}

		for idx := range request.Logics {
			trigger.Logics = append(trigger.Logics, model.RealtimeLogic{
				ID:           uint(request.Logics[idx].Id),
				TriggerId:    uint(request.Logics[idx].TriggerId),
				Logic:        request.Logics[idx].Logic,
				TriggerValue: request.Logics[idx].TriggerValue,
			})
		}
		for idx := range request.StrMatchs {
			trigger.StrMatchs = append(trigger.StrMatchs, model.RealtimeStrMatch{
				ID:              uint(request.Id),
				TriggerId:       uint(request.StrMatchs[idx].TriggerId),
				MatchStr:        request.StrMatchs[idx].MatchStr,
				ReplaceCodeFlag: request.StrMatchs[idx].ReplaceCodeFlag,
				ReplaceCode:     request.StrMatchs[idx].ReplaceCode,
			})
		}
	}

	return trigger
}
