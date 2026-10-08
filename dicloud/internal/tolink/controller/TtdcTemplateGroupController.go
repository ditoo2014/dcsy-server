package controller

import (
	"dicloud/internal/tolink/service"
	"dicloud/models/model"
	"dicloud/util/response"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type TtdcTemplateGroupController struct{}

// RequireDefaultDataSource keeps global TTDC configuration out of tenant DBs.
func RequireDefaultDataSource(ctx *gin.Context) {
	if value, exists := ctx.Get("dbName"); exists {
		requested := strings.TrimSpace(fmt.Sprint(value))
		defaultName := strings.TrimSpace(viper.GetString("opengauss.dbname"))
		if requested != "" && !strings.EqualFold(requested, defaultName) {
			response.Fail(ctx, "TTDC电池配置只支持默认数据源", nil)
			ctx.Abort()
			return
		}
	}
	ctx.Next()
}

func (TtdcTemplateGroupController) FindPage(ctx *gin.Context) {
	var req service.TtdcTemplatePageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	rows, total, err := service.NewTtdcTemplateGroupService().FindPage(req)
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"data": rows, "total": total})
}

func (TtdcTemplateGroupController) Detail(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Query("id"), 10, 64)
	if err != nil || id == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	item, err := service.NewTtdcTemplateGroupService().Detail(uint(id))
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"data": item})
}

func (TtdcTemplateGroupController) Options(ctx *gin.Context) {
	rows, err := service.NewTtdcTemplateGroupService().Options()
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"data": rows})
}

func (TtdcTemplateGroupController) Save(ctx *gin.Context) {
	var req service.TtdcTemplateGroupRequest
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
	item, err := service.NewTtdcTemplateGroupService().Save(req, userID)
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "保存成功", gin.H{"data": item})
}

func (TtdcTemplateGroupController) SetState(ctx *gin.Context) {
	var req struct {
		ID      uint   `json:"id"`
		Field   string `json:"field"`
		Value   bool   `json:"value"`
		Confirm bool   `json:"confirm"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	if err := service.NewTtdcTemplateGroupService().SetState(req.ID, req.Field, req.Value, req.Confirm); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "状态更新成功", nil)
}

func (TtdcTemplateGroupController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Query("id"), 10, 64)
	if err != nil || id == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	if err := service.NewTtdcTemplateGroupService().Delete(uint(id)); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "删除成功", nil)
}
