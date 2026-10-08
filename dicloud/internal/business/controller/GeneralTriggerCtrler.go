package controller

import (
	"dicloud/internal/business/client"
	"dicloud/internal/business/service"
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/internal/common/model"
	"dicloud/util/response"
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
)

type IGeneralTriggerCtrler interface {
	AddGeneralTrigger(ctx *gin.Context)
	DeleteGeneralTrigger(ctx *gin.Context)
	UpdateGeneralTrigger(ctx *gin.Context)
	FindGeneralTrigger(ctx *gin.Context)
}

type GeneralTriggerCtrler struct {
}

// 远程调用失败时响应可能为空；先处理连接错误，避免错误分支再次触发空指针。
func generalTriggerFailed(ctx *gin.Context, resp *tolink_grpc.TolinkGeneralTriggerResponse, err error) bool {
	if err != nil {
		log.Printf("[BUSINESS] 通用触发器采集服务调用失败: %v", err)
		response.Fail(ctx, "采集服务暂不可用，请检查业务云与采集服务连接", nil)
		return true
	}
	if resp == nil {
		response.Fail(ctx, "采集服务未返回结果，请稍后重试", nil)
		return true
	}
	if resp.Code != 200 {
		message := resp.Message
		if message == "" {
			message = "通用触发器操作失败"
		}
		response.Fail(ctx, message, nil)
		return true
	}
	return false
}

func (r GeneralTriggerCtrler) AddGeneralTrigger(ctx *gin.Context) {

	// 绑定触发器数据
	var trigger = model.GeneralTriggerConfig{}
	ctx.Bind(&trigger)

	/** 进行一些前期的校验 TODO: 站点,设备,null数据校验 */
	if (len(trigger.Logics) < 1 && len(trigger.StrMatchs) < 1) || len(trigger.Code) < 1 {
		response.Fail(ctx, "数据格式错误", nil)
		return
	}
	if len(trigger.Code) == 0 {
		response.Fail(ctx, "报警代码为空", nil)
		return
	}
	if len(trigger.LabelName) == 0 {
		response.Fail(ctx, "触发数据为空", nil)
		return
	}

	// 查询所有设备 -> 通过设备查询数据池 -> 通过数据池数据点生成匹配数据
	gmts := service.GetGeneralMarktriggerListByLabelName(ctx, trigger.EquipmentIds, trigger.LabelName)
	trigger.Marktriggers = gmts

	// 调用远程过程 - 创建
	resp, err := client.AddGeneralTrigger(ctx, &trigger)
	if generalTriggerFailed(ctx, resp, err) {
		return
	}

	response.Success(ctx, "创建成功", nil)
}

func (r GeneralTriggerCtrler) DeleteGeneralTrigger(ctx *gin.Context) {

	// 绑定触发器数据
	var trigger = model.GeneralTriggerConfig{}
	ctx.Bind(&trigger)

	if trigger.ID == 0 {
		response.Fail(ctx, "无效数据", nil)
		return
	}

	// 调用远程过程 - 创建
	resp, err := client.DeleteGeneralTrigger(ctx, &trigger)
	if generalTriggerFailed(ctx, resp, err) {
		return
	}

	response.Success(ctx, "删除成功", nil)
}

func (r GeneralTriggerCtrler) UpdateGeneralTrigger(ctx *gin.Context) {

	/* 获取信息 */

	var trigger model.GeneralTriggerConfig
	ctx.Bind(&trigger)

	// 绑定触发器数据
	if trigger.ID == 0 {
		response.Fail(ctx, "无效数据", nil)
		return
	}

	// 调用远程过程 - 创建
	resp, err := client.UpdateGeneralTrigger(ctx, &trigger)
	if generalTriggerFailed(ctx, resp, err) {
		return
	}

	response.Success(ctx, "更新成功", nil)
}

func (r GeneralTriggerCtrler) FindGeneralTrigger(ctx *gin.Context) {

	// 绑定触发器数据
	var trigger = model.GeneralTriggerConfig{}
	ctx.Bind(&trigger)

	resp, err := client.FindGeneralTrigger(ctx, &trigger)
	if generalTriggerFailed(ctx, resp, err) {
		return
	}

	// 转换数据 - 有些非0数据grpc会忽略, 造成前端判断数据缺失
	var triggers = []model.GeneralTriggerConfig{}
	grpcStrs, err := json.Marshal(resp.Triggers)
	if err != nil {
		log.Println("[BUSINESS] A generaltrigger local transform data fail, err: ", err)
		response.Fail(ctx, "查询失败", nil)
		return
	}
	err = json.Unmarshal(grpcStrs, &triggers)
	if err != nil {
		log.Println("[BUSINESS] B generaltrigger local transform data fail, err: ", err)
		response.Fail(ctx, "查询失败", nil)
		return
	}

	response.Success(ctx, "查询成功", gin.H{"data": triggers})
}
