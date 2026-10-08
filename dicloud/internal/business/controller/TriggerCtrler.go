package controller

import (
	"dicloud/internal/business/client"
	businessdao "dicloud/internal/business/dao"
	businessmodel "dicloud/internal/business/model"
	"dicloud/internal/common/model"
	"dicloud/util/response"
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
)

type ITriggerCtrler interface {
	AddTrigger(ctx *gin.Context)
	DeleteTrigger(ctx *gin.Context)
	UpdateTrigger(ctx *gin.Context)
	FindTrigger(ctx *gin.Context)
}

type TriggerCtrler struct {
}

func (r TriggerCtrler) AddTrigger(ctx *gin.Context) {

	// 绑定触发器数据
	var trigger = model.TriggerConfig{}
	ctx.Bind(&trigger)

	/** 进行一些前期的校验 TODO: 站点,设备,null数据校验 */
	if (len(trigger.Logics) < 1 && len(trigger.StrMatchs) < 1) || len(trigger.Code) < 1 {
		response.Fail(ctx, "数据格式错误", nil)
		return
	}
	if trigger.EquipmentId == 0 {
		response.Fail(ctx, "数据不匹配", nil)
		return
	}
	if len(trigger.Code) == 0 {
		response.Fail(ctx, "报警代码为空", nil)
		return
	}
	if len(trigger.ApiKey) == 0 || len(trigger.LabelId) == 0 {
		response.Fail(ctx, "触发数据为空", nil)
		return
	}
	// 验证设备是否存在
	equipmentDao := businessdao.NewEquipmentDAO(ctx)
	_, res := equipmentDao.FindEquipmentById(businessmodel.Equipment{ID: uint(trigger.EquipmentId)})
	if !res {
		response.Fail(ctx, "设备不存在", nil)
		return
	}

	// 这里分离: 采集云/业务云, 进行远程调用
	resp, err := client.AddTrigger(ctx, &trigger)
	if resp.Code != 200 {
		log.Println("[BUSINESS] add trigger fail: ", err)
		response.Fail(ctx, resp.Message, nil)
		return
	}

	response.Success(ctx, "创建成功", nil)
}

func (r TriggerCtrler) DeleteTrigger(ctx *gin.Context) {

	// 绑定触发器数据
	var trigger = model.TriggerConfig{}
	ctx.Bind(&trigger)

	if trigger.ID == 0 {
		response.Fail(ctx, "无效数据", nil)
		return
	}

	// 这里分离: 采集云/业务云, 进行远程调用
	resp, err := client.DeleteTrigger(ctx, &trigger)
	if resp.Code != 200 {
		log.Println("[BUSINESS] delete trigger fail: ", err)
		response.Fail(ctx, resp.Message, nil)
		return
	}

	response.Success(ctx, "删除成功", nil)
}

func (r TriggerCtrler) UpdateTrigger(ctx *gin.Context) {

	/* 获取信息 */

	var trigger model.TriggerConfig
	ctx.Bind(&trigger)

	// 绑定触发器数据
	if trigger.ID == 0 {
		response.Fail(ctx, "无效数据", nil)
		return
	}

	// 这里分离: 采集云/业务云, 进行远程调用
	resp, err := client.UpdateTrigger(ctx, &trigger)
	if resp.Code != 200 {
		log.Println("[BUSINESS] update trigger fail: ", err)
		response.Fail(ctx, resp.Message, nil)
		return
	}

	response.Success(ctx, "更新成功", nil)
}

func (r TriggerCtrler) FindTrigger(ctx *gin.Context) {

	// 绑定触发器数据
	var trigger = model.TriggerConfig{}
	ctx.Bind(&trigger)

	if trigger.EquipmentId == 0 {
		response.Fail(ctx, "无效数据", nil)
		return
	}

	// 这里分离: 采集云/业务云, 进行远程调用
	resp, err := client.FindTrigger(ctx, &trigger)
	if resp.Code != 200 {
		log.Println("[BUSINESS] find trigger fail: ", err)
		response.Fail(ctx, resp.Message, nil)
		return
	}

	// 转换数据 - 有些非0数据grpc会忽略, 造成前端判断数据缺失
	var triggers = []model.TriggerConfig{}
	grpcStrs, err := json.Marshal(resp.Triggers)
	if err != nil {
		log.Println("[BUSINESS] A trigger local transform data fail, err: ", err)
		response.Fail(ctx, "查询失败", nil)
		return
	}
	err = json.Unmarshal(grpcStrs, &triggers)
	if err != nil {
		log.Println("[BUSINESS] B trigger local transform data fail, err: ", err)
		response.Fail(ctx, "查询失败", nil)
		return
	}

	response.Success(ctx, "查询成功", gin.H{"data": triggers})
}
