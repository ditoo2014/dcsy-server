package main

import (
	"alarm/internal/alarm/controller"
	"alarm/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func ConnectRoute(router *gin.Engine) *gin.Engine {

	/* 跨域 */
	router.Use(middleware.Cors())
	api := router.Group("/alarm")
	{
		var receiveAlertmanagerCtrler controller.IReceiveAlertmanagerCtrler = controller.ReceiveAlertmanagerCtrler{}
		// 接受alertmanager报警消息, 进行报警
		api.POST("/receive/alertmanager", receiveAlertmanagerCtrler.ReceiveAlertmanager)
	}
	return router
}
