package client

import (
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/internal/common/model"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

func AddTrigger(ctx *gin.Context, tc *model.TriggerConfig) (*tolink_grpc.TolinkTriggerResponse, error) {

	e := baseTriggerDataFormat(tc)

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		e.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().AddTrigger(ctx, e)
}

func DeleteTrigger(ctx *gin.Context, tc *model.TriggerConfig) (*tolink_grpc.TolinkTriggerResponse, error) {

	e := baseTriggerDataFormat(tc)

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		e.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().DeleteTrigger(ctx, e)
}

func UpdateTrigger(ctx *gin.Context, tc *model.TriggerConfig) (*tolink_grpc.TolinkTriggerResponse, error) {

	e := baseTriggerDataFormat(tc)

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		e.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().UpdateTrigger(ctx, e)
}

func FindTrigger(ctx *gin.Context, tc *model.TriggerConfig) (*tolink_grpc.TolinkTriggerResponse, error) {

	e := baseTriggerDataFormat(tc)

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		e.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().FindTrigger(ctx, e)
}

// trigger数据转换 - 本地数据 -> grpc数据
func baseTriggerDataFormat(tc *model.TriggerConfig) *tolink_grpc.TriggerConfig {

	// 转换数据
	var srlRes = false
	strBytes, err := json.Marshal(tc)
	var e = tolink_grpc.TriggerConfig{}
	if err == nil {
		err := json.Unmarshal(strBytes, &e)
		if err == nil {
			srlRes = true
		}
	}

	if !srlRes {
		// 通过json序列化, 反序列化失败
		e = tolink_grpc.TriggerConfig{
			Id:            int64(tc.ID),
			SiteId:        int64(tc.SiteId),
			EquipmentId:   int64(tc.EquipmentId),
			TriggerType:   int64(tc.TriggerType),
			TriggerModel:  int64(tc.TriggerModel),
			Code:          tc.Code,
			ApiKey:        tc.ApiKey,
			LabelName:     tc.LabelName,
			LabelId:       tc.LabelId,
			Cycle:         tc.Cycle,
			Timer:         tc.Timer,
			Record:        tc.Record,
			LastStrMatch:  tc.LastStrMatch,
			Marktrigger:   tc.Marktrigger,
			Formula:       tc.Formula,
			SustainTime:   tc.SustainTime,
			SustainRecord: tc.SustainRecord,
			AlarmUser: &tolink_grpc.AlarmUser{
				Id:           int64(tc.AlarmUser.ID),
				TriggerId:    int64(tc.AlarmUser.TriggerId),
				AlarmCode:    tc.AlarmUser.AlarmCode,
				AlarmType:    tc.AlarmUser.AlarmType,
				AlarmStage:   int64(tc.AlarmUser.AlarmStage),
				AlarmChannel: tc.AlarmUser.AlarmChannel,
				AlarmUsers:   tc.AlarmUser.AlarmUsers,
			},
			StatisticConfig: &tolink_grpc.StatisticConfig{
				Id:                 int64(tc.StatisticConfig.ID),
				TriggerId:          int64(tc.StatisticConfig.TriggerId),
				StatisticCode:      tc.StatisticConfig.StatisticCode,
				StatisticName:      tc.StatisticConfig.StatisticName,
				DataSourceType:     int64(tc.StatisticConfig.DataSourceType),
				StatisticType:      tc.StatisticConfig.StatisticType,
				UseStatisticValue:  tc.StatisticConfig.UseStatisticValue,
				StatisticValue:     tc.StatisticConfig.StatisticValue,
				StatisticValueType: int64(tc.StatisticConfig.StatisticValueType),
			},
		}

		// 切片数据封装
		for idx := range tc.Logics {
			e.Logics = append(e.Logics, &tolink_grpc.RealtimeLogic{
				Id:           int64(tc.Logics[idx].ID),
				TriggerId:    int64(tc.Logics[idx].TriggerId),
				Logic:        tc.Logics[idx].Logic,
				TriggerValue: tc.Logics[idx].TriggerValue,
			})
		}
		for idx := range tc.StrMatchs {
			e.StrMatchs = append(e.StrMatchs, &tolink_grpc.RealtimeStrMatch{
				Id:              int64(tc.StrMatchs[idx].ID),
				TriggerId:       int64(tc.StrMatchs[idx].TriggerId),
				MatchStr:        tc.StrMatchs[idx].MatchStr,
				ReplaceCodeFlag: tc.StrMatchs[idx].ReplaceCodeFlag,
				ReplaceCode:     tc.StrMatchs[idx].ReplaceCode,
			})
		}
	}

	return &e
}
