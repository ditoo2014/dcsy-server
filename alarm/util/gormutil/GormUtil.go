package gormutil

import (
	"alarm/util/opengauss"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetGormDB 返回告警程序使用的 OpenGauss 连接。
// dbTypeParam 仅为兼容现有调用签名保留，告警项目不再携带 Oracle、TDengine 实现。
func GetGormDB(ctx *gin.Context, dbTypeParam string) *gorm.DB {
	return opengauss.GetGormDB(ctx)
}
