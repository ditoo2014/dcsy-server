package main

import (
	"dicloud/internal/apiserver/controller"
	"dicloud/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func ConnectRoute(router *gin.Engine) *gin.Engine {

	/* 跨域 */
	router.Use(middleware.Cors())
	api := router.Group("/api")
	{
		/* 多租户 */
		api.Use(middleware.SwitchDBMiddleware(), middleware.ApiMiddleware())

		var apiCtrler controller.IApiCtrler = controller.ApiCtrler{}

		router.GET("/api/getlive", apiCtrler.GetLive) //获取网关状态 - 不查询数据池
		api.POST("/getlabel", apiCtrler.GetLabel)     //获取标签列表
		api.POST("/getdata", apiCtrler.GetData)       //获取数据

		// 批量查询实时数据接口
		router.POST("/api/getapilistdata", apiCtrler.GetApiListDataAll)
	}
	return router
}
