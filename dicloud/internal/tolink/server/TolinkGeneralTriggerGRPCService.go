package grpcserver

import (
	"context"
	"dicloud/internal/common/dao"
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/internal/common/model"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func (TolinkServer) AddGeneralTrigger(ctx context.Context, request *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {
	log.Printf("[GRPC-TolinkGeneralTriggerServer] Add Trigger triggerName = %s\n", request.TriggerName)

	// 响应体
	var responseData = &tolink_grpc.TolinkGeneralTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkGeneralTriggerServer] Delete LabelName = %s, Code = %s\n", request.LabelName, request.Code),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseGeneralTriggerDataFormat(request)

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
	generaltriggerDao := dao.NewGeneralTriggerDao(ginCtx)
	_, res := generaltriggerDao.ExistTriggerName(trigger)
	if res {
		responseData.Code = 10501
		responseData.Message = "创建失败, 已存在"
		return responseData, nil
	}

	res = generaltriggerDao.Create(trigger)
	if !res {
		responseData.Code = 10502
		responseData.Message = "创建失败"
		return responseData, nil
	}

	return responseData, nil
}

func (TolinkServer) DeleteGeneralTrigger(ctx context.Context, request *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {
	log.Printf("[GRPC-TolinkGeneralTriggerServer] Delete Trigger Id = %d\n", request.Id)

	// 响应体
	var responseData = &tolink_grpc.TolinkGeneralTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkGeneralTriggerServer] Delete LabelName = %s, Code = %s\n", request.LabelName, request.Code),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseGeneralTriggerDataFormat(request)

	/* 数据库连接 */
	generaltriggerDao := dao.NewGeneralTriggerDao(ginCtx)
	_, res := generaltriggerDao.ExistId(trigger)
	if !res {
		responseData.Code = 10501
		responseData.Message = "不存在的数据"
		return responseData, nil
	}

	// REMOVE:
	res = generaltriggerDao.Delete(trigger)
	if !res {
		responseData.Code = 10502
		responseData.Message = "删除失败"
		return responseData, nil
	}

	return responseData, nil
}

func (TolinkServer) UpdateGeneralTrigger(ctx context.Context, request *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {
	log.Printf("[GRPC-TolinkGeneralTriggerServer] Update Trigger Id = %d\n", request.Id)

	// 响应体
	var responseData = &tolink_grpc.TolinkGeneralTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkGeneralTriggerServer] Update LabelName = %s, Code = %s\n", request.LabelName, request.Code),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseGeneralTriggerDataFormat(request)

	/* 数据库连接 */
	generaltriggerDao := dao.NewGeneralTriggerDao(ginCtx)

	_, res := generaltriggerDao.ExistId(trigger)
	if !res {
		responseData.Code = 10501
		responseData.Message = "不存在的数据"
		return responseData, nil
	}
	_, res = generaltriggerDao.UpdateExistEquipmentAlarmCode(trigger)
	if res {
		responseData.Code = 10502
		responseData.Message = "同名配置已存在"
		return responseData, nil
	}

	// 更新触发器
	res = generaltriggerDao.Update(trigger)
	if !res {
		responseData.Code = 10503
		responseData.Message = "更新失败"
		return responseData, nil
	}

	return responseData, nil
}

func (TolinkServer) FindGeneralTrigger(ctx context.Context, request *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {
	log.Printf("[GRPC-TolinkGeneralTriggerServer] Find Trigger Id = %d\n", request.Id)

	// 响应体
	var responseData = &tolink_grpc.TolinkGeneralTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkGeneralTriggerServer] Find LabelName = %s, Code = %s\n", request.LabelName, request.Code),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseGeneralTriggerDataFormat(request)

	/* 数据库连接 */
	generaltriggerDao := dao.NewGeneralTriggerDao(ginCtx)
	data, err := generaltriggerDao.FindTriggerLogics(trigger)
	if err != nil {
		responseData.Code = 10501
		responseData.Message = "查询失败"
		return responseData, nil
	}

	// 转换返回数据
	dataStr, err := json.Marshal(data)
	if err != nil {
		log.Println("[GRPC-TolinkGeneralTriggerServer] A transform data fail, may have fields ignored")
		responseData.Message = "数据转换失败"
		return responseData, nil
	}
	var grpcTriggers = []*tolink_grpc.GeneralTriggerConfig{}
	err = json.Unmarshal(dataStr, &grpcTriggers)
	if err != nil {
		log.Println("[GRPC-TolinkGeneralTriggerServer] B transform data fail, may have fields ignored")
		responseData.Message = "数据转换失败"
		return responseData, nil
	}
	responseData.Triggers = grpcTriggers

	return responseData, nil
}

// trigger数据转换 - grpc数据 -> 本地数据
func baseGeneralTriggerDataFormat(request *tolink_grpc.GeneralTriggerConfig) model.GeneralTriggerConfig {

	// 通过json转换数据
	var srlRes = false
	strBytes, err := json.Marshal(request)
	var trigger = model.GeneralTriggerConfig{}
	if err == nil {
		err = json.Unmarshal(strBytes, &trigger)
		if err == nil {
			srlRes = true
		}
	}

	// 序列化转换失败, 手动操作
	if !srlRes {
		// 反序列化失败, 手动填充数据
		trigger = model.GeneralTriggerConfig{
			ID:           uint(request.Id),
			EquipmentIds: request.EquipmentIds,
			TriggerName:  request.TriggerName,
			LabelName:    request.LabelName,
			TriggerType:  int(request.TriggerType),
			TriggerModel: int(request.TriggerModel),
			Code:         request.Code,
			Cycle:        request.Cycle,
			Formula:      request.Formula,
			AlarmUser: model.GeneralAlarmUser{
				ID:           uint(request.AlarmUser.Id),
				TriggerId:    uint(request.AlarmUser.TriggerId),
				AlarmCode:    request.AlarmUser.AlarmCode,
				AlarmType:    request.AlarmUser.AlarmType,
				AlarmStage:   int(request.AlarmUser.AlarmStage),
				AlarmChannel: request.AlarmUser.AlarmChannel,
				AlarmUsers:   request.AlarmUser.AlarmUsers,
			},
			StatisticConfig: model.GeneralStatisticConfig{
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
			trigger.Logics = append(trigger.Logics, model.GeneralRealtimeLogic{
				ID:           uint(request.Logics[idx].Id),
				TriggerId:    uint(request.Logics[idx].TriggerId),
				Logic:        request.Logics[idx].Logic,
				TriggerValue: request.Logics[idx].TriggerValue,
			})
		}
		for idx := range request.StrMatchs {
			trigger.StrMatchs = append(trigger.StrMatchs, model.GeneralRealtimeStrMatch{
				ID:              uint(request.Id),
				TriggerId:       uint(request.StrMatchs[idx].TriggerId),
				MatchStr:        request.StrMatchs[idx].MatchStr,
				ReplaceCodeFlag: request.StrMatchs[idx].ReplaceCodeFlag,
				ReplaceCode:     request.StrMatchs[idx].ReplaceCode,
			})
		}
		for idx := range request.Marktriggers {
			trigger.Marktriggers = append(trigger.Marktriggers, model.GeneralMarktrigger{
				ID:            uint(request.Marktriggers[idx].Id),
				TriggerId:     uint(request.Marktriggers[idx].TriggerId),
				SiteId:        int(request.Marktriggers[idx].SiteId),
				EquipmentId:   int(request.Marktriggers[idx].EquipmentId),
				Marktrigger:   request.Marktriggers[idx].Marktrigger,
				Timer:         request.Marktriggers[idx].Timer,
				Record:        request.Marktriggers[idx].Record,
				LastStrMatch:  request.Marktriggers[idx].LastStrMatch,
				SustainRecord: request.Marktriggers[idx].SustainRecord,
			})
		}
	}

	return trigger
}

// 查询全部通用触发器, +关联标记数据
func (TolinkServer) FindAllGeneralTriggerMarkBinds(ctx context.Context, request *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {
	// log.Printf("[GRPC-TolinkGeneralTriggerServer] FindAllGeneralTriggerMarkBinds Trigger \n")

	// 响应体
	var responseData = &tolink_grpc.TolinkGeneralTriggerResponse{
		Code:    200,
		Message: "[GRPC-TolinkGeneralTriggerServer] FindAllGeneralTriggerMarkBinds",
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	/* 数据库连接 */
	// 数据库数据
	generaltriggerDao := dao.NewGeneralTriggerDao(ginCtx)
	datas, res := generaltriggerDao.Find(model.GeneralTriggerConfig{})
	if !res {
		responseData.Code = 10501
		responseData.Message = "查询失败"
		return responseData, nil
	}

	// 转换返回数据
	dataStr, err := json.Marshal(datas)
	if err != nil {
		log.Println("[GRPC-TolinkGeneralTriggerServer] A transform data fail, may have fields ignored")
		responseData.Message = "数据转换失败"
		return responseData, nil
	}
	var grpcTriggers = []*tolink_grpc.GeneralTriggerConfig{}
	err = json.Unmarshal(dataStr, &grpcTriggers)
	if err != nil {
		log.Println("[GRPC-TolinkGeneralTriggerServer] B transform data fail, may have fields ignored")
		responseData.Message = "数据转换失败"
		return responseData, nil
	}
	responseData.Triggers = grpcTriggers

	return responseData, nil
}

func (TolinkServer) DeleteMarkByGeneralTriggerId(ctx context.Context, request *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {
	log.Printf("[GRPC-TolinkGeneralTriggerServer] DeleteMarkByGeneralTriggerId Trigger Id = %d\n", request.Id)

	// 响应体
	var responseData = &tolink_grpc.TolinkGeneralTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkGeneralTriggerServer] DeleteMarkByGeneralTriggerId TriggerName = %s, Code = %s\n", request.TriggerName, request.Code),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseGeneralTriggerDataFormat(request)

	/* 数据库连接 */
	generaltriggerDao := dao.NewGeneralTriggerDao(ginCtx)
	res := generaltriggerDao.DeleteMarktriggers(trigger)
	if !res {
		responseData.Code = 10501
		responseData.Message = "删除失败"
		return responseData, nil
	}

	return responseData, nil
}

func validateGeneralMarktriggerRequest(action string, request *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, bool) {
	if request != nil && len(request.Marktriggers) > 0 {
		for _, mark := range request.Marktriggers {
			if mark == nil {
				log.Printf("[GRPC-TolinkGeneralTriggerServer] %s Marktriggers item is nil\n", action)
				return &tolink_grpc.TolinkGeneralTriggerResponse{
					Code:    10501,
					Message: "marktriggers item is nil",
				}, false
			}
		}
		return nil, true
	}

	log.Printf("[GRPC-TolinkGeneralTriggerServer] %s Marktriggers is empty\n", action)
	return &tolink_grpc.TolinkGeneralTriggerResponse{
		Code:    10501,
		Message: "marktriggers is empty",
	}, false
}

func (TolinkServer) UpdateMarkTrigger(ctx context.Context, request *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {
	if response, ok := validateGeneralMarktriggerRequest("UpdateMarkTrigger", request); !ok {
		return response, nil
	}

	log.Printf("[GRPC-TolinkGeneralTriggerServer] UpdateMarkTrigger Id = %d\n", request.Marktriggers[0].Id)

	// 响应体
	var responseData = &tolink_grpc.TolinkGeneralTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkGeneralTriggerServer] UpdateMarkTrigger EquipmentId = %d, Marktrigger = %s\n", request.Marktriggers[0].EquipmentId, request.Marktriggers[0].Marktrigger),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseGeneralTriggerDataFormat(request)

	/* 数据库连接 */
	generaltriggerDao := dao.NewGeneralTriggerDao(ginCtx)
	res := generaltriggerDao.UpdateMarkTrigger(trigger.Marktriggers[0])
	if !res {
		responseData.Code = 10501
		responseData.Message = "修改失败"
		return responseData, nil
	}

	return responseData, nil
}

func (TolinkServer) DeleteMarkTrigger(ctx context.Context, request *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {
	if response, ok := validateGeneralMarktriggerRequest("DeleteMarkTrigger", request); !ok {
		return response, nil
	}

	log.Printf("[GRPC-TolinkGeneralTriggerServer] DeleteMarkTrigger Id = %d\n", request.Marktriggers[0].Id)

	// 响应体
	var responseData = &tolink_grpc.TolinkGeneralTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkGeneralTriggerServer] DeleteMarkTrigger EquipmentId = %d, Marktrigger = %s\n", request.Marktriggers[0].EquipmentId, request.Marktriggers[0].Marktrigger),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseGeneralTriggerDataFormat(request)

	/* 数据库连接 */
	generaltriggerDao := dao.NewGeneralTriggerDao(ginCtx)
	res := generaltriggerDao.DeleteMarkTrigger(trigger.Marktriggers[0])
	if !res {
		responseData.Code = 10501
		responseData.Message = "删除失败"
		return responseData, nil
	}

	return responseData, nil
}

func (TolinkServer) CreateMarkTrigger(ctx context.Context, request *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {
	if response, ok := validateGeneralMarktriggerRequest("CreateMarkTrigger", request); !ok {
		return response, nil
	}

	log.Printf("[GRPC-TolinkGeneralTriggerServer] CreateMarkTrigger Id = %d\n", request.Marktriggers[0].Id)

	// 响应体
	var responseData = &tolink_grpc.TolinkGeneralTriggerResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkGeneralTriggerServer] CreateMarkTrigger EquipmentId = %d, Marktrigger = %s\n", request.Marktriggers[0].EquipmentId, request.Marktriggers[0].Marktrigger),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	trigger := baseGeneralTriggerDataFormat(request)

	/* 数据库连接 */
	generaltriggerDao := dao.NewGeneralTriggerDao(ginCtx)
	res := generaltriggerDao.CreateMarkTrigger(trigger.Marktriggers...)
	if !res {
		responseData.Code = 10501
		responseData.Message = "删除失败"
		return responseData, nil
	}

	return responseData, nil
}
