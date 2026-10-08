package controller

import (
	"dicloud/internal/tolink/service"
	"dicloud/models/model"
	"dicloud/util/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TtdcAlarmRuleController manages platform-level alarm definitions. The
// definitions are stored in the default datasource by the service layer.
type TtdcAlarmRuleController struct{}

func (TtdcAlarmRuleController) FindPage(ctx *gin.Context) {
	var req service.TtdcAlarmRulePageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	rows, total, err := service.NewTtdcAlarmRuleService().FindPage(req)
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"data": rows, "total": total})
}

func (TtdcAlarmRuleController) Detail(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Query("id"), 10, 64)
	if err != nil || id == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	item, err := service.NewTtdcAlarmRuleService().Detail(uint(id))
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"data": item})
}

func (TtdcAlarmRuleController) Save(ctx *gin.Context) {
	var req service.TtdcAlarmRuleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	userID := 0
	if value, ok := ctx.Get("userInfo"); ok {
		if user, ok := value.(model.User); ok {
			userID = int(user.ID)
		}
	}
	item, err := service.NewTtdcAlarmRuleService().Save(req, userID)
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "保存成功", gin.H{"data": item})
}

func (TtdcAlarmRuleController) SetState(ctx *gin.Context) {
	var req struct {
		ID    uint `json:"id"`
		Value bool `json:"value"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	if err := service.NewTtdcAlarmRuleService().SetEnabled(req.ID, req.Value); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "状态更新成功", nil)
}

func (TtdcAlarmRuleController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Query("id"), 10, 64)
	if err != nil || id == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	if err = service.NewTtdcAlarmRuleService().Delete(uint(id)); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "删除成功", nil)
}
