package client

import (
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/internal/common/model"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

func AddGeneralTrigger(ctx *gin.Context, tc *model.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {

	e := baseGeneralTriggerDataFormat(tc)

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		e.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().AddGeneralTrigger(ctx, e)
}

func DeleteGeneralTrigger(ctx *gin.Context, tc *model.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {

	e := baseGeneralTriggerDataFormat(tc)

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		e.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().DeleteGeneralTrigger(ctx, e)
}

func UpdateGeneralTrigger(ctx *gin.Context, tc *model.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {

	e := baseGeneralTriggerDataFormat(tc)

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		e.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().UpdateGeneralTrigger(ctx, e)
}

func FindGeneralTrigger(ctx *gin.Context, tc *model.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {

	e := baseGeneralTriggerDataFormat(tc)

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		e.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().FindGeneralTrigger(ctx, e)
}

// trigger数据转换 - 本地数据 -> grpc数据
func baseGeneralTriggerDataFormat(tc *model.GeneralTriggerConfig) *tolink_grpc.GeneralTriggerConfig {

	// 转换数据
	var srlRes = false
	strBytes, err := json.Marshal(tc)
	var e = tolink_grpc.GeneralTriggerConfig{}
	if err == nil {
		err := json.Unmarshal(strBytes, &e)
		if err == nil {
			srlRes = true
		}
	}

	if !srlRes {
		// 通过json序列化, 反序列化失败
		e = tolink_grpc.GeneralTriggerConfig{
			Id:           int64(tc.ID),
			EquipmentIds: tc.EquipmentIds,
			TriggerName:  tc.TriggerName,
			LabelName:    tc.LabelName,
			TriggerType:  int64(tc.TriggerType),
			TriggerModel: int64(tc.TriggerModel),
			Code:         tc.Code,
			Cycle:        tc.Cycle,
			SustainTime:  tc.SustainTime,
			Formula:      tc.Formula,
			AlarmUser: &tolink_grpc.GeneralAlarmUser{
				Id:           int64(tc.AlarmUser.ID),
				TriggerId:    int64(tc.AlarmUser.TriggerId),
				AlarmCode:    tc.AlarmUser.AlarmCode,
				AlarmType:    tc.AlarmUser.AlarmType,
				AlarmStage:   int64(tc.AlarmUser.AlarmStage),
				AlarmChannel: tc.AlarmUser.AlarmChannel,
				AlarmUsers:   tc.AlarmUser.AlarmUsers,
			},
			StatisticConfig: &tolink_grpc.GeneralStatisticConfig{
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
			e.Logics = append(e.Logics, &tolink_grpc.GeneralRealtimeLogic{
				Id:           int64(tc.Logics[idx].ID),
				TriggerId:    int64(tc.Logics[idx].TriggerId),
				Logic:        tc.Logics[idx].Logic,
				TriggerValue: tc.Logics[idx].TriggerValue,
			})
		}
		for idx := range tc.StrMatchs {
			e.StrMatchs = append(e.StrMatchs, &tolink_grpc.GeneralRealtimeStrMatch{
				Id:              int64(tc.StrMatchs[idx].ID),
				TriggerId:       int64(tc.StrMatchs[idx].TriggerId),
				MatchStr:        tc.StrMatchs[idx].MatchStr,
				ReplaceCodeFlag: tc.StrMatchs[idx].ReplaceCodeFlag,
				ReplaceCode:     tc.StrMatchs[idx].ReplaceCode,
			})
		}
		for idx := range tc.Marktriggers {
			e.Marktriggers = append(e.Marktriggers, &tolink_grpc.GeneralMarktrigger{
				Id:            int64(tc.Marktriggers[idx].ID),
				TriggerId:     int64(tc.Marktriggers[idx].TriggerId),
				SiteId:        int64(tc.Marktriggers[idx].SiteId),
				EquipmentId:   int64(tc.Marktriggers[idx].EquipmentId),
				Marktrigger:   tc.Marktriggers[idx].Marktrigger,
				Timer:         tc.Marktriggers[idx].Timer,
				Record:        tc.Marktriggers[idx].Record,
				LastStrMatch:  tc.Marktriggers[idx].LastStrMatch,
				SustainRecord: tc.Marktriggers[idx].SustainRecord,
			})
		}
	}

	return &e
}

func FindAllGeneralTriggerMarkBinds(ctx *gin.Context, tc *model.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {

	e := baseGeneralTriggerDataFormat(tc)

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		e.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().FindAllGeneralTriggerMarkBinds(ctx, e)
}

func DeleteMarkByGeneralTriggerId(ctx *gin.Context, tc *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		tc.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().DeleteMarkByGeneralTriggerId(ctx, tc)
}

func UpdateMarkTrigger(ctx *gin.Context, tc *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		tc.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().UpdateMarkTrigger(ctx, tc)
}

func DeleteMarkTrigger(ctx *gin.Context, tc *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		tc.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().DeleteMarkTrigger(ctx, tc)
}

func CreateMarkTrigger(ctx *gin.Context, tc *tolink_grpc.GeneralTriggerConfig) (*tolink_grpc.TolinkGeneralTriggerResponse, error) {

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		tc.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().CreateMarkTrigger(ctx, tc)
}
