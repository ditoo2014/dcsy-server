package main

import (
	"dicloud/internal/business/controller"
	"dicloud/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func ConnectRoute(router *gin.Engine) *gin.Engine {

	/* 跨域 */
	router.Use(middleware.Cors())

	/* 业务模块 */
	business := router.Group("/business")
	{
		var triggerController controller.ITriggerCtrler = controller.TriggerCtrler{}
		var siteGroupController controller.ISiteGroupCtrler = controller.SiteGroupCtrler{}
		var siteController controller.ISiteCtrler = controller.SiteCtrler{}
		var equipmentController controller.IEquipmentCtrler = controller.EquipmentCtrler{}
		var equipmentTypeController controller.IEquipmentTypeCtrler = controller.EquipmentTypeCtrler{}
		var dataDetailController controller.IDataDetailsCtrler = controller.DataDetailsCtrler{}
		var warehouseController controller.IWarehouseCtrler = controller.WarehouseCtrler{}
		var equipmentCirculationLogController controller.IEquipmentCirculationLogCtrler = controller.EquipmentCirculationLogCtrler{}
		var equipmentGeographyPositionController controller.IEquipmentGeographyPositionCtrler = controller.EquipmentGeographyPositionCtrler{}
		var equipmentMaintainRecordController controller.IEquipmentMaintainRecordCtrler = controller.EquipmentMaintainRecordCtrler{}
		var alarmDataRecordCtrler controller.IAlarmDataRecordCtrler = controller.AlarmDataRecordCtrler{}
		var statisticDataCtrler controller.IStatisticDataCtrler = controller.StatisticDataCtrler{}
		var commonDataCtrler controller.IAutoSyncAllCommonDataCtrler = controller.AutoSyncAllCommonDataCtrler{}
		var healthStandardCtrler controller.IEquipmentHealthCtrler = controller.EquipmentHealthCtrler{}
		var statisticDurationDataCtrler controller.StatisticDurationDataCtrler = controller.StatisticDurationDataCtrler{}
		// 通用触发器
		var generaltriggerCtrler controller.IGeneralTriggerCtrler = controller.GeneralTriggerCtrler{}
		var alarmUsersCtrler controller.IAlarmUsersCtrler = controller.AlarmUsersCtrler{}
		var liveAlarmCtrler controller.ILiveAlarmCtrler = controller.LiveAlarmCtrler{}

		// middleware.AuthMiddleware() -- 登录验证
		// middleware.PlatMiddleware(2) -- 区分平台接口
		// middleware.AuthWithPlatMiddleware(2) -- 前两接口融合
		// middleware.VerifyMiddleware("", 2) -- 登录验证 + 平台内部权限
		// 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云

		/* 多租户 */
		business.Use(middleware.SwitchDBMiddleware())

		/* 触发器 */
		business.POST("/trigger/add", middleware.VerifyMiddleware("trigger:add", 1), middleware.SystemLogMiddleware("添加触发器"), triggerController.AddTrigger)
		business.POST("/trigger/delete", middleware.VerifyMiddleware("trigger:delete", 1), middleware.SystemLogMiddleware("删除触发器"), triggerController.DeleteTrigger)
		business.POST("/trigger/update", middleware.VerifyMiddleware("trigger:update", 1), middleware.SystemLogMiddleware("修改触发器"), triggerController.UpdateTrigger)
		business.POST("/trigger/find", middleware.VerifyMiddleware("trigger:find", 1), triggerController.FindTrigger)
		// 通用触发器
		business.POST("/generaltrigger/add", middleware.VerifyMiddleware("generaltrigger:add", 1), middleware.SystemLogMiddleware("添加通用触发器"), generaltriggerCtrler.AddGeneralTrigger)
		business.POST("/generaltrigger/delete", middleware.VerifyMiddleware("generaltrigger:delete", 1), middleware.SystemLogMiddleware("删除通用触发器"), generaltriggerCtrler.DeleteGeneralTrigger)
		business.POST("/generaltrigger/update", middleware.VerifyMiddleware("generaltrigger:update", 1), middleware.SystemLogMiddleware("修改通用触发器"), generaltriggerCtrler.UpdateGeneralTrigger)
		business.POST("/generaltrigger/find", middleware.VerifyMiddleware("generaltrigger:find", 1), generaltriggerCtrler.FindGeneralTrigger)

		/* 统计数据查询 */
		business.POST("/statisticdata/find", middleware.VerifyMiddleware("statisticdata:find", 1), statisticDataCtrler.FindStatisticData)
		business.POST("/statisticdata/count", middleware.VerifyMiddleware("statisticdata:count", 1), statisticDataCtrler.CountStatisticData)
		business.POST("/statisticdata/levelcorp", middleware.VerifyMiddleware("statisticdata:levelcorp", 1), statisticDataCtrler.FindStatisticDataByLevelCorp)
		business.POST("/statisticdata/markdatelist", middleware.VerifyMiddleware("statisticdata:markdatelist", 1), statisticDataCtrler.FindStatisticDataMakrListByCorpCode)
		business.POST("/statisticdata/charge-discharge-daily", middleware.VerifyMiddleware("statisticdata:markdatelist", 1), statisticDataCtrler.FindChargeDischargeDailyStatistics)
		business.POST("/statisticdata/alarmdaily", middleware.VerifyMiddleware("statisticdata:markdatelist", 1), statisticDataCtrler.FindDailyAlarmStatistics)
		business.POST("/statisticdata/warningdaily", middleware.VerifyMiddleware("statisticdata:markdatelist", 1), statisticDataCtrler.FindDailyWarningStatistics)

		/* 报警记录 */
		business.POST("/alarmrecord/find", middleware.VerifyMiddleware("alarmrecord:find", 1), alarmDataRecordCtrler.FindAlarm)
		business.POST("/alarmrecord/delete", middleware.VerifyMiddleware("alarmrecord:delete", 1), middleware.SystemLogMiddleware("删除报警记录"), alarmDataRecordCtrler.DeleteAlarm)
		business.POST("/alarmrecord/countbyequipment", middleware.VerifyMiddleware("alarmrecord:countbyequipment", 1), alarmDataRecordCtrler.BatchCountAlarm)
		business.POST("/alarmrecord/statusbyequipment", middleware.VerifyMiddleware("alarmrecord:countbyequipment", 1), alarmDataRecordCtrler.BatchAlarmStatus)
		business.POST("/alarmUsers/findAlarmUsers", middleware.VerifyMiddleware("alarmrecord:find", 1), alarmUsersCtrler.Find)
		business.POST("/alarmUsers/deleteAlarmUsers", middleware.VerifyMiddleware("trigger:delete", 1), alarmUsersCtrler.Delete)
		business.POST("/alarmUsers/batchDeleteAlarmUsers", middleware.VerifyMiddleware("trigger:delete", 1), alarmUsersCtrler.BatchDelete)
		business.POST("/collect/findCollectLiveTriggerTempData", middleware.VerifyMiddleware("alarmrecord:find", 1), liveAlarmCtrler.Find)
		business.POST("/collect/removeCollectLiveTriggerTempData", middleware.VerifyMiddleware("alarmrecord:delete", 1), liveAlarmCtrler.Remove)

		/* 站点分组 */
		business.POST("/sitegrouparr/add", middleware.VerifyMiddleware("sitegrouparr:add", 1), siteGroupController.AddSiteGroupArr)
		business.POST("/sitegroup/delete", middleware.VerifyMiddleware("sitegroup:delete", 1), middleware.SystemLogMiddleware("删除站点分组"), siteGroupController.DeleteSiteGroup)
		business.POST("/sitegrouparr/update", middleware.VerifyMiddleware("sitegrouparr:update", 1), middleware.SystemLogMiddleware("修改站点分组"), siteGroupController.UpdateSiteGroupArr)
		business.POST("/sitegrouparr/find", middleware.VerifyMiddleware("sitegrouparr:find", 1), siteGroupController.FindSiteGroupArr)
		business.GET("/sitegrouparr/find", middleware.VerifyMiddleware("sitegrouparr:find", 1), siteGroupController.FindSiteGroupArr)

		/* 站点 */
		business.POST("/site/add", middleware.VerifyMiddleware("site:add", 1), middleware.SystemLogMiddleware("添加站点"), siteController.AddSite)
		business.POST("/site/delete", middleware.VerifyMiddleware("site:delete", 1), middleware.SystemLogMiddleware("删除站点"), siteController.DeleteSite)
		business.POST("/site/update", middleware.VerifyMiddleware("site:update", 1), middleware.SystemLogMiddleware("修改站点"), siteController.UpdateSite)
		business.POST("/site/find", middleware.VerifyMiddleware("site:find", 1), siteController.FindSite)
		business.POST("/site/details/find", middleware.VerifyMiddleware("site:details:find", 1), siteController.FindSiteDetail)
		business.POST("/site/export", middleware.VerifyMiddleware("site:export", 1), siteController.ExportExcel)
		business.POST("/site/import", middleware.VerifyMiddleware("site:import", 1), middleware.SystemLogMiddleware("导入站点数据"), siteController.ImportExcel)
		business.POST("/site/import/template", middleware.VerifyMiddleware("site:import", 1), siteController.ImportTemplate)

		/* 设备 */
		business.POST("/equipment/add", middleware.VerifyMiddleware("equipment:add", 1), equipmentController.AddEquipment)
		business.POST("/equipment/find", middleware.VerifyMiddleware("equipment:find", 1), equipmentController.FindEquipment)
		business.POST("/equipment/health/statistics", middleware.VerifyMiddleware("equipment:find", 1), healthStandardCtrler.FindEquipmentHealthStatistics)
		business.POST("/equipment/countbycorp", middleware.VerifyMiddleware("equipment:countbycorp", 1), equipmentController.CountEquipmentByCorp)
		business.POST("/equipment/update", middleware.VerifyMiddleware("equipment:update", 1), middleware.SystemLogMiddleware("修改设备"), equipmentController.UpdateEquipment)
		business.POST("/equipment/delete", middleware.VerifyMiddleware("equipment:delete", 1), middleware.SystemLogMiddleware("删除设备"), equipmentController.DeleteEquipment)
		business.POST("/equipment/export", middleware.VerifyMiddleware("equipment:export", 1), equipmentController.ExportExcel)
		business.POST("/equipment/import", middleware.VerifyMiddleware("equipment:import", 1), middleware.SystemLogMiddleware("导入设备数据"), equipmentController.ImportExcel)
		business.POST("/equipment/import/template", middleware.VerifyMiddleware("equipment:import", 1), equipmentController.ImportTemplate)
		business.POST("/equipment/alldata", middleware.VerifyMiddleware("equipment:alldata", 1), equipmentController.FindEquipmentAllData)
		business.GET("/equipment/findEquipmentTypeList", middleware.VerifyMiddleware("equipment:find", 1), equipmentTypeController.Find)
		business.POST("/equipment/saveEquipmentType", middleware.VerifyMiddleware("equipment:add", 1), equipmentTypeController.Add)
		business.POST("/equipment/modifyEquipmentType", middleware.VerifyMiddleware("equipment:update", 1), equipmentTypeController.Update)
		business.POST("/equipment/deleteEquipmentType", middleware.VerifyMiddleware("equipment:delete", 1), equipmentTypeController.Delete)

		/* 数据详情 */
		business.POST("/datadetails/add", middleware.VerifyMiddleware("datadetails:add", 1), middleware.SystemLogMiddleware("添加数据详情"), dataDetailController.AddDataDetails)
		business.POST("/datadetails/batchadd", middleware.VerifyMiddleware("datadetails:batchadd", 1), middleware.SystemLogMiddleware("批量添加数据详情"), dataDetailController.AddDataDetailsList)
		business.POST("/datadetails/update", middleware.VerifyMiddleware("datadetails:update", 1), middleware.SystemLogMiddleware("修改数据详情"), dataDetailController.UpdateDataDetails)
		business.POST("/datadetails/batchupdate", middleware.VerifyMiddleware("datadetails:batchupdate", 1), middleware.SystemLogMiddleware("批量修改数据详情"), dataDetailController.UpdateDataDetailsList)
		business.POST("/datadetails/find", middleware.VerifyMiddleware("datadetails:find", 1), dataDetailController.FindDataDetails)
		business.POST("/datadetails/delete", middleware.VerifyMiddleware("datadetails:delete", 1), middleware.SystemLogMiddleware("删除数据详情"), dataDetailController.DeleteDataDetails)

		/* 仓库信息 */
		business.POST("/warehouse/add", middleware.VerifyMiddleware("warehouse:add", 1), middleware.SystemLogMiddleware("添加库房"), warehouseController.AddWarehouse)
		business.POST("/warehouse/update", middleware.VerifyMiddleware("warehouse:update", 1), middleware.SystemLogMiddleware("修改库房"), warehouseController.UpdateWarehouse)
		business.POST("/warehouse/delete", middleware.VerifyMiddleware("warehouse:delete", 1), middleware.SystemLogMiddleware("删除库房"), warehouseController.DeleteWarehouse)
		business.POST("/warehouse/find", middleware.VerifyMiddleware("warehouse:find", 1), warehouseController.FindWarehouse)
		business.POST("/warehouse/export", middleware.VerifyMiddleware("warehouse:export", 1), warehouseController.ExportExcel)
		business.POST("/warehouse/import", middleware.VerifyMiddleware("warehouse:import", 1), middleware.SystemLogMiddleware("导入库房数据"), warehouseController.ImportExcel)
		business.POST("/warehouse/import/template", middleware.VerifyMiddleware("warehouse:import", 1), warehouseController.ImportTemplate)

		/* 设备流转记录 */
		business.POST("/equipmentcirlog/add", middleware.VerifyMiddleware("equipmentcirlog:add", 1), middleware.SystemLogMiddleware("添加设备流转记录"), equipmentCirculationLogController.Create)
		business.POST("/equipmentcirlog/find", middleware.VerifyMiddleware("equipmentcirlog:find", 1), equipmentCirculationLogController.Find)

		/* 设备地理位置信息上报 */
		business.POST("/equipmentgoppst/add", middleware.VerifyMiddleware("equipmentgoppst:add", 1), middleware.SystemLogMiddleware("添加设备位置信息"), equipmentGeographyPositionController.Create)
		business.POST("/equipmentgoppst/find", middleware.VerifyMiddleware("equipmentgoppst:find", 1), equipmentGeographyPositionController.Find)

		/* 设备维保记录 */
		business.POST("/equipmentmitrcd/add", middleware.VerifyMiddleware("equipmentmitrcd:add", 1), middleware.SystemLogMiddleware("添加设备维保记录"), equipmentMaintainRecordController.Create)
		business.POST("/equipmentmitrcd/find", middleware.VerifyMiddleware("equipmentmitrcd:find", 1), equipmentMaintainRecordController.Find)
		business.POST("/equipmentmitrcd/export", middleware.VerifyMiddleware("equipmentmitrcd:export", 1), middleware.SystemLogMiddleware("导出设备维保记录"), equipmentMaintainRecordController.ExportExcel)
		business.POST("/equipmentmitrcd/import", middleware.VerifyMiddleware("equipmentmitrcd:import", 1), middleware.SystemLogMiddleware("导入设备维保记录"), equipmentMaintainRecordController.ImportExcel)

		/* 通用数据导入接口 */
		business.POST("/commondata/import",
			// middleware.VerifyMiddleware("commondata:import", 1), middleware.SystemLogMiddleware("导入平台通用配置"),
			commonDataCtrler.ImportCommonData)
		business.POST("/commondata/delete",
			// middleware.VerifyMiddleware("commondata:import", 1), middleware.SystemLogMiddleware("删除平台通用配置"),
			commonDataCtrler.DeleteCommonDataByEquipmentId)

		/* 设备健康度判断标准接口 */
		business.POST("/equipmenthltstd/update", middleware.VerifyMiddleware("equipmenthltstd:update", 1), middleware.SystemLogMiddleware("修改健康度标准"), healthStandardCtrler.UpdateEquipmentHealth)
		business.POST("/equipmenthltstd/find", middleware.VerifyMiddleware("equipmenthltstd:find", 1), healthStandardCtrler.FindEquipmentHealth)

		/* 执行统计计划 - 根据时间统计固定数据点数据 */
		business.POST("/statistictask/start", middleware.VerifyMiddleware("statistictask:start", 1), middleware.SystemLogMiddleware("执行统计任务"), statisticDurationDataCtrler.StartStatisticTask)

	}

	image := router.Group("/go/images")
	{
		// 上传沿用前端已携带的登录令牌，防止匿名请求写入服务器文件。
		image.POST("/upload", middleware.AuthWithPlatMiddleware(1), controller.Upload)
		// 与上传地址同源，文件名和真实路径由控制器校验。
		image.GET("/file/:filename", controller.ReadUploadedImage)
	}

	return router
}
