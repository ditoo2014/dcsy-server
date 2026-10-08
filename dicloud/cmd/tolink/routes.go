package main

import (
	"dicloud/internal/tolink/controller"
	"dicloud/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func ConnectRoute(router *gin.Engine, useSpringGateway bool) *gin.Engine {

	/* 跨域 */
	if !useSpringGateway {
		router.Use(middleware.Cors())
	}

	/* 业务模块 */

	tolink := router.Group("/tolink")
	{
		var gatewayController controller.IGatewayCtrler = controller.GatewayCtrler{}
		var gatewayDataController controller.IGatewayDataCtrler = controller.GatewayDataCtrler{}
		var timeSequenceDataController controller.ITimeSequenceDataCtrler = controller.TimeSequenceDataCtrler{}
		var ttdcTemplateController controller.TtdcTemplateController
		var ttdcTemplateGroupController controller.TtdcTemplateGroupController
		var ttdcAlarmRuleController controller.TtdcAlarmRuleController

		// middleware.AuthMiddleware() -- 登录验证
		// middleware.PlatMiddleware(2) -- 区分平台接口
		// middleware.AuthWithPlatMiddleware(2) -- 前两接口融合
		// middleware.VerifyMiddleware("", 2) -- 登录验证 + 平台内部权限

		// 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云

		/* 多租户 */
		tolink.Use(middleware.SwitchDBMiddleware())

		// 网关接口
		tolink.POST("/gateway/add", middleware.VerifyMiddleware("gateway:add", 2), middleware.SystemLogMiddleware("添加网关"), gatewayController.Create)                                 // 创建网关
		tolink.POST("/gateway/delete", middleware.VerifyMiddleware("gateway:delete", 2), middleware.SystemLogMiddleware("删除网关"), gatewayController.Delete)                           // 删除网关
		tolink.POST("/gateway/update", middleware.VerifyMiddleware("gateway:update", 2), middleware.SystemLogMiddleware("修改网关"), gatewayController.Update)                           // 删除网关
		tolink.POST("/gateway/findpage", middleware.VerifyMiddleware("gateway:findpage", 2), gatewayController.FindPage)                                                             // 查询网关列表
		tolink.POST("/gateway/copy", middleware.VerifyMiddleware("gateway:copy", 2), middleware.GatewayMiddleware(), middleware.SystemLogMiddleware("复制网关"), gatewayController.Copy) // 复制网关
		tolink.POST("/gateway/findone", middleware.VerifyMiddleware("gateway:findone", 2), middleware.GatewayMiddleware(), gatewayController.FindOne)                                // 查询一个网关
		tolink.POST("/gateway/findlinkedup", middleware.VerifyMiddleware("gateway:findlinkedup", 2), middleware.GatewayMiddleware(), gatewayController.FindLinkedUp)                 // 查询上线,未保存网关
		// 网关通道接口
		tolink.POST("/gatewaymodule/add", middleware.VerifyMiddleware("gatewaymodule:add", 2), middleware.GatewayMiddleware(), middleware.SystemLogMiddleware("添加通道"), gatewayController.CreateModule)       // 创建 通道、数据点（drive、data）
		tolink.POST("/gatewaymodule/delete", middleware.VerifyMiddleware("gatewaymodule:delete", 2), middleware.GatewayMiddleware(), middleware.SystemLogMiddleware("删除通道"), gatewayController.DeleteModule) // 删除 通道、数据点（drive、data）
		tolink.POST("/gatewaymodule/update", middleware.VerifyMiddleware("gatewaymodule:update", 2), middleware.GatewayMiddleware(), middleware.SystemLogMiddleware("修改通道"), gatewayController.UpdateModule) // 修改 通道、数据点（drive、data）
		tolink.POST("/gatewaymodule/find", middleware.VerifyMiddleware("gatewaymodule:find", 2), middleware.GatewayMiddleware(), gatewayController.FindModule)                                               // 查询 通道、数据点（drive、data）
		// TTDC template-group APIs always operate on the process default datasource.
		ttdcConfig := tolink.Group("")
		ttdcConfig.Use(controller.RequireDefaultDataSource)
		ttdcConfig.POST("/ttdcgroup/findpage", middleware.VerifyMiddleware("ttdcgroup:findpage", 2), ttdcTemplateGroupController.FindPage)
		ttdcConfig.POST("/ttdcgroup/detail", middleware.VerifyAnyMiddleware([]string{"ttdcgroup:detail", "ttdcgroup:findpage", "ttdcgroup:update"}, 2), ttdcTemplateGroupController.Detail)
		ttdcConfig.POST("/ttdcgroup/options", middleware.VerifyAnyMiddleware([]string{"ttdcgroup:options", "ttdctemplate:add", "ttdctemplate:update", "gatewaymodule:add", "gatewaymodule:update"}, 2), ttdcTemplateGroupController.Options)
		ttdcConfig.POST("/ttdcgroup/add", middleware.VerifyMiddleware("ttdcgroup:add", 2), ttdcTemplateGroupController.Save)
		ttdcConfig.POST("/ttdcgroup/update", middleware.VerifyMiddleware("ttdcgroup:update", 2), ttdcTemplateGroupController.Save)
		ttdcConfig.POST("/ttdcgroup/state", middleware.VerifyAnyMiddleware([]string{"ttdcgroup:state", "ttdcgroup:update"}, 2), ttdcTemplateGroupController.SetState)
		ttdcConfig.POST("/ttdcgroup/delete", middleware.VerifyMiddleware("ttdcgroup:delete", 2), ttdcTemplateGroupController.Delete)
		ttdcConfig.POST("/ttdctemplate/findpage", middleware.VerifyMiddleware("ttdctemplate:findpage", 2), ttdcTemplateController.FindPage)
		ttdcConfig.POST("/ttdctemplate/detail", middleware.VerifyAnyMiddleware([]string{"ttdctemplate:detail", "ttdctemplate:findpage", "ttdctemplate:update", "gatewaymodule:add", "gatewaymodule:update"}, 2), ttdcTemplateController.Detail)
		ttdcConfig.POST("/ttdctemplate/options", middleware.VerifyAnyMiddleware([]string{"ttdctemplate:options", "ttdctemplate:findpage", "gatewaymodule:add", "gatewaymodule:update"}, 2), ttdcTemplateController.Options)
		ttdcConfig.POST("/ttdctemplate/profiles", middleware.VerifyAnyMiddleware([]string{"ttdctemplate:profiles", "ttdctemplate:add", "ttdctemplate:update"}, 2), ttdcTemplateController.Profiles)
		ttdcConfig.POST("/ttdctemplate/algorithms", middleware.VerifyAnyMiddleware([]string{"ttdctemplate:algorithms", "ttdctemplate:add", "ttdctemplate:update"}, 2), ttdcTemplateController.Algorithms)
		ttdcConfig.POST("/ttdctemplate/add", middleware.VerifyMiddleware("ttdctemplate:add", 2), ttdcTemplateController.Save)
		ttdcConfig.POST("/ttdctemplate/update", middleware.VerifyMiddleware("ttdctemplate:update", 2), ttdcTemplateController.Save)
		ttdcConfig.POST("/ttdctemplate/delete", middleware.VerifyMiddleware("ttdctemplate:delete", 2), ttdcTemplateController.Delete)
		ttdcConfig.POST("/ttdctemplate/enable", middleware.VerifyAnyMiddleware([]string{"ttdctemplate:enable", "ttdctemplate:update"}, 2), ttdcTemplateController.SetManualEnabled)
		ttdcConfig.POST("/ttdctemplate/import", middleware.VerifyMiddleware("ttdctemplate:import", 2), ttdcTemplateController.Import)
		ttdcConfig.POST("/ttdctemplate/export", middleware.VerifyMiddleware("ttdctemplate:export", 2), ttdcTemplateController.Export)
		ttdcConfig.POST("/ttdcalarmrule/findpage", middleware.VerifyAnyMiddleware([]string{"ttdcalarmrule:findpage", "ttdcgroup:findpage"}, 2), ttdcAlarmRuleController.FindPage)
		ttdcConfig.POST("/ttdcalarmrule/detail", middleware.VerifyAnyMiddleware([]string{"ttdcalarmrule:detail", "ttdcalarmrule:findpage", "ttdcalarmrule:update"}, 2), ttdcAlarmRuleController.Detail)
		ttdcConfig.POST("/ttdcalarmrule/add", middleware.VerifyMiddleware("ttdcalarmrule:add", 2), ttdcAlarmRuleController.Save)
		ttdcConfig.POST("/ttdcalarmrule/update", middleware.VerifyMiddleware("ttdcalarmrule:update", 2), ttdcAlarmRuleController.Save)
		ttdcConfig.POST("/ttdcalarmrule/state", middleware.VerifyAnyMiddleware([]string{"ttdcalarmrule:state", "ttdcalarmrule:update"}, 2), ttdcAlarmRuleController.SetState)
		ttdcConfig.POST("/ttdcalarmrule/delete", middleware.VerifyMiddleware("ttdcalarmrule:delete", 2), ttdcAlarmRuleController.Delete)
		/* 网关数据点操作 */
		tolink.POST("/gatewaydata/o", middleware.VerifyMiddleware("gatewaydata:o", 2), middleware.GatewayMiddleware(), middleware.SystemLogMiddleware("网关数据点操作"), gatewayDataController.GatewayDataOperate) // 对GatewayData进行 增删改查
		// 网关时序数据
		tolink.POST("/timesequence/getdata", middleware.VerifyMiddleware("timesequence:getdata", 2), middleware.GatewayMiddleware(), timeSequenceDataController.GetData)                                                                       // 查询数据点时序数据
		tolink.POST("/timesequence/deletebyidtime", middleware.VerifyMiddleware("timesequence:deletebyidtime", 2), middleware.GatewayMiddleware(), middleware.SystemLogMiddleware("删除时序数据"), timeSequenceDataController.DeleteDataOneByIdTime) // 删除一个数据点一个时间点的时序数据
		tolink.POST("/timesequence/deleteall", middleware.VerifyMiddleware("timesequence:deleteall", 2), middleware.GatewayMiddleware(), middleware.SystemLogMiddleware("删除网关所有数据"), timeSequenceDataController.DeleteTableAllData)            // 删除所有时序数据

		/* 数据流模块 */
		var streamCtrler controller.IStreamCtrler = controller.StreamCtrler{}
		tolink.POST("/stream/create", middleware.VerifyMiddleware("stream:add", 2), streamCtrler.Create)                                                                                           //创建数据池
		tolink.POST("/stream/update", middleware.VerifyMiddleware("stream:update", 2), middleware.StreamMiddlewareNameUpdate(), middleware.SystemLogMiddleware("修改数据池"), streamCtrler.Update)      //修改数据池信息
		tolink.POST("/stream/delete", middleware.VerifyMiddleware("stream:delete", 2), middleware.StreamMiddleware(), middleware.SystemLogMiddleware("删除数据池"), streamCtrler.Delete)                //删除数据池
		tolink.POST("/stream/findpage", middleware.VerifyMiddleware("stream:findpage", 2), streamCtrler.FindPage)                                                                                  //数据池的分页查询加关键字搜索，  该接口进行修改过
		tolink.POST("/stream/label", middleware.VerifyMiddleware("stream:label", 2), middleware.StreamMiddleware(), middleware.SystemLogMiddleware("数据池数据点操作"), streamCtrler.Label)                //添加标签
		tolink.POST("/stream/getlabel", middleware.VerifyMiddleware("stream:getlabel", 2), middleware.StreamMiddleware(), streamCtrler.GetLabel)                                                   //获取标签
		tolink.POST("/stream/importexcel", middleware.VerifyMiddleware("stream:importexcel", 2), middleware.StreamMiddleware(), middleware.SystemLogMiddleware("数据池导入"), streamCtrler.ImportExcel) //导入excel表,  导入的数据点id需要网关数据点做好才修改
		tolink.POST("/stream/exportexcel", middleware.VerifyMiddleware("stream:exportexcel", 2), middleware.StreamMiddleware(), streamCtrler.ExportExcel)                                          //导出excel表
		tolink.POST("/stream/batchimport/template", middleware.VerifyMiddleware("stream:add", 2), streamCtrler.BatchImportTemplate)
		tolink.POST("/stream/batchimport", middleware.VerifyMiddleware("stream:add", 2), middleware.SystemLogMiddleware("批量导入数据池"), streamCtrler.BatchImport)
	}

	return router
}
