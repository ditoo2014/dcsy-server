package gormutil

import (
	"fmt"
	"log"

	"dicloud/internal/migration"
	"dicloud/util/opengauss"
	"dicloud/util/tdengine"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

/* 初始化Gorm数据库连接 */
func InitPlatGormDB() {
	if err := InitPlatGormDBWithError(); err != nil {
		log.Printf("[GORM] platform database initialization failed: %v", err)
	}
}

// InitPlatGormDBWithError initializes the platform GORM database before the
// caller starts serving requests. TTDC alarm-rule tables are auto-migrated
// together with other platform schemas during startup.
func InitPlatGormDBWithError() error {
	dbType := viper.GetString("server.gormdb.type")
	if len(dbType) == 0 {
		dbType = "OpenGauss"
	}
	if dbType != "OpenGauss" {
		return fmt.Errorf("unsupported GORM database type %q", dbType)
	}
	return initialOpenGauss()
}

/* 根据名称, 初始化数据库连接 */
func InitPlatDBByName(dbName string) {

	if len(dbName) == 0 {
		dbName = "OpenGauss"
	}

	switch dbName {
	case "OpenGauss":
		if err := initialOpenGauss(); err != nil {
			log.Printf("[GORM] 初始化OpenGauss失败: %v", err)
		}
	case "Tdengine":
		tdengine.InitTDengine()
	default:
		log.Printf("[GORM] 不支持的数据源: %s", dbName)
	}

}

/* 获取Gorm数据库连接 */
func GetGormDB(ctx *gin.Context, _ string) *gorm.DB {
	return opengauss.GetGormDB(ctx)
}

/* 初始化OpenGauss */
func initialOpenGauss() error {
	_, err := migration.Initialize()
	return err
}
