package controller

import (
	"log"
	"strconv"
	"strings"

	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	"dicloud/internal/tolink/service"
	usermodel "dicloud/models/model"
	normalutil "dicloud/util/normal"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

type IGatewayCtrler interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Update(ctx *gin.Context)
	Copy(ctx *gin.Context)
	FindOne(ctx *gin.Context)
	FindPage(ctx *gin.Context)
	FindLinkedUp(ctx *gin.Context)
	CreateModule(ctx *gin.Context)
	DeleteModule(ctx *gin.Context)
	UpdateModule(ctx *gin.Context)
	FindModule(ctx *gin.Context)
}

type GatewayCtrler struct{}

// Create 新增 TTDC 网关。服务端固定机型，避免前端参数绕过限制。
func (GatewayCtrler) Create(ctx *gin.Context) {
	var config model.Gateway
	if err := ctx.ShouldBindJSON(&config); err != nil {
		response.Fail(ctx, "请求参数格式错误", nil)
		return
	}
	config.Name = strings.TrimSpace(config.Name)
	config.GatewayId = strings.TrimSpace(config.GatewayId)
	if config.Name == "" || config.GatewayId == "" {
		response.Fail(ctx, "网关名称和序列号不能为空", nil)
		return
	}
	if config.Model != "" && config.Model != model.TTDC {
		response.Fail(ctx, "当前采集平台仅支持TTDC网关机型", nil)
		return
	}
	config.Model = model.TTDC
	if config.Card == "" {
		config.Card = "-"
	}
	if value, exists := ctx.Get("userInfo"); exists {
		if userInfo, ok := value.(usermodel.User); ok && userInfo.Isadmin != 1 {
			config.CreateUserId = int(userInfo.ID)
		}
	}

	gatewayDAO := dao.NewGatewayDAO(ctx)
	if _, exists := gatewayDAO.ExistName(config); exists {
		response.Fail(ctx, "网关名已被创建", nil)
		return
	}
	if _, exists := gatewayDAO.ExistID(config); exists {
		response.Fail(ctx, "序列号已被创建", nil)
		return
	}
	if _, err := gatewayDAO.AddGateway(config); err != nil {
		response.Fail(ctx, err.Error(), nil)
		return
	}
	response.Success(ctx, "操作成功", nil)
}

// Delete 删除 TTDC 网关及其模块、点位和模板绑定。
func (GatewayCtrler) Delete(ctx *gin.Context) {
	var request model.Gateway
	if err := ctx.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.GatewayId) == "" {
		response.Fail(ctx, "网关序列号不能为空", nil)
		return
	}
	gatewayDAO := dao.NewGatewayDAO(ctx)
	gateway, err := gatewayDAO.GetByGatewayID(strings.TrimSpace(request.GatewayId))
	if err != nil {
		response.Fail(ctx, "网关不存在", nil)
		return
	}
	if gateway.Model != model.TTDC {
		response.Fail(ctx, "当前采集平台仅支持TTDC网关机型", nil)
		return
	}
	if err := service.DeleteModuleTtdcByGatewayId(ctx, gateway.GatewayId); err != nil {
		response.Fail(ctx, "删除TTDC模块失败: "+err.Error(), nil)
		return
	}
	if err := gatewayDAO.DelGateway(gateway.GatewayId); err != nil {
		response.Fail(ctx, "删除失败: "+err.Error(), nil)
		return
	}
	response.Success(ctx, "删除成功", nil)
}

func (GatewayCtrler) Update(ctx *gin.Context) {
	var request model.Gateway
	if err := ctx.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.GatewayId) == "" {
		response.Fail(ctx, "网关序列号不能为空", nil)
		return
	}
	request.GatewayId = strings.TrimSpace(request.GatewayId)
	if request.Model != "" && request.Model != model.TTDC {
		response.Fail(ctx, "当前采集平台仅支持TTDC网关机型", nil)
		return
	}
	request.Model = model.TTDC
	gatewayDAO := dao.NewGatewayDAO(ctx)
	existing, err := gatewayDAO.GetByGatewayID(request.GatewayId)
	if err != nil || existing.Model != model.TTDC {
		response.Fail(ctx, "TTDC网关不存在", nil)
		return
	}
	if err := gatewayDAO.UpdateGateway(request); err != nil {
		response.Fail(ctx, "修改失败", nil)
		return
	}
	response.Success(ctx, "修改成功", nil)
}

// Copy 不复制旧网关协议数据。TTDC 配置由公共模板和绑定关系管理，应显式新建网关。
func (GatewayCtrler) Copy(ctx *gin.Context) {
	response.Fail(ctx, "TTDC网关请新建后绑定模板，不支持复制旧网关配置", nil)
}

func (GatewayCtrler) FindOne(ctx *gin.Context) {
	gateway, ok := ttdcGatewayFromContext(ctx)
	if !ok {
		return
	}
	response.Success(ctx, "操作成功", gin.H{"data": gateway})
}

func (GatewayCtrler) FindPage(ctx *gin.Context) {
	var query model.Gateway
	_ = ctx.ShouldBindJSON(&query)
	query.Model = model.TTDC
	pageNum, _ := strconv.Atoi(ctx.Query("pageNum"))
	pageSize, _ := strconv.Atoi(ctx.Query("pageSize"))
	offset, limit := normalutil.GetPageOffsetLimit(pageNum, pageSize)
	list, total, ok := dao.NewGatewayDAO(ctx).FindGatewayList(query, offset, limit)
	if !ok {
		response.Fail(ctx, "查询TTDC网关失败", nil)
		return
	}
	response.Success(ctx, "操作成功", gin.H{"data": list, "total": total})
}

func (GatewayCtrler) FindLinkedUp(ctx *gin.Context) {
	id, timestamp, err := service.FindLinkedUpGateways(ctx)
	if err != nil {
		response.Fail(ctx, err.Error(), nil)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"id": id, "time": timestamp})
}

func (GatewayCtrler) CreateModule(ctx *gin.Context) {
	gateway, ok := ttdcGatewayFromContext(ctx)
	if !ok {
		return
	}
	var module model.GatewayModuleTtdc
	if err := ctx.ShouldBindJSON(&module); err != nil {
		response.Fail(ctx, "请求参数格式错误", nil)
		return
	}
	module.GatewayId = gateway.GatewayId
	module.Name = strings.TrimSpace(module.Name)
	if module.Name == "" {
		response.Fail(ctx, "通道名称不能为空", nil)
		return
	}
	if module.Title == "" {
		module.Title = module.Name
	}
	result, err := service.CreateModuleTtdc(ctx, module)
	if err != nil {
		response.Fail(ctx, "创建TTDC通道失败: "+err.Error(), nil)
		return
	}
	response.Success(ctx, "创建成功", result)
}

func (GatewayCtrler) FindModule(ctx *gin.Context) {
	gateway, ok := ttdcGatewayFromContext(ctx)
	if !ok {
		return
	}
	result, err := service.FindModuleTtdc(ctx, gateway)
	if err != nil {
		response.Fail(ctx, "查询TTDC通道失败: "+err.Error(), nil)
		return
	}
	response.Success(ctx, "查找成功", result)
}

func (GatewayCtrler) DeleteModule(ctx *gin.Context) {
	gateway, ok := ttdcGatewayFromContext(ctx)
	if !ok {
		return
	}
	var module model.GatewayModuleTtdc
	if err := ctx.ShouldBindJSON(&module); err != nil {
		response.Fail(ctx, "请求参数格式错误", nil)
		return
	}
	module.GatewayId = gateway.GatewayId
	if _, err := service.DeleteModuleTtdc(ctx, module); err != nil {
		response.Fail(ctx, "删除TTDC通道失败: "+err.Error(), nil)
		return
	}
	response.Success(ctx, "删除成功", nil)
}

func (GatewayCtrler) UpdateModule(ctx *gin.Context) {
	gateway, ok := ttdcGatewayFromContext(ctx)
	if !ok {
		return
	}
	var module model.GatewayModuleTtdc
	if err := ctx.ShouldBindJSON(&module); err != nil {
		response.Fail(ctx, "请求参数格式错误", nil)
		return
	}
	module.GatewayId = gateway.GatewayId
	result, err := service.UpdateModuleTtdc(ctx, ctx.Query("Function"), &gateway, module)
	if err != nil {
		response.Fail(ctx, "修改TTDC通道失败: "+err.Error(), nil)
		return
	}
	response.Success(ctx, "操作成功", result)
}

func ttdcGatewayFromContext(ctx *gin.Context) (model.Gateway, bool) {
	value, exists := ctx.Get("gatewayInfo")
	if !exists {
		response.Fail(ctx, "网关信息不存在", nil)
		return model.Gateway{}, false
	}
	gateway, ok := value.(model.Gateway)
	if !ok {
		log.Println("[GatewayCtrler] gatewayInfo context type invalid")
		response.Fail(ctx, "网关信息无效", nil)
		return model.Gateway{}, false
	}
	if gateway.Model != model.TTDC {
		response.Fail(ctx, "当前采集平台仅支持TTDC网关机型", nil)
		return model.Gateway{}, false
	}
	return gateway, true
}
