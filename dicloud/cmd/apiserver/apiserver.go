package main

import (
	"dicloud/internal/migration"
	"dicloud/models/enum"
	"dicloud/pkg/cache"
	"dicloud/pkg/common"
	"dicloud/util"
	"dicloud/util/opengauss"
	"dicloud/util/redisutil"
	"dicloud/util/tdengine"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func InitDB() error {
	if _, err := migration.Initialize(); err != nil {
		return err
	}
	tdengine.InitTDengine()
	return nil
}

func system_CheckThread() {

	// 心跳
	for {
		// 防止连接超时关闭
		redisutil.Check()
		opengauss.Check()
		tdengine.Check()
		time.Sleep(time.Second * 3)
	}
}

func main() {

	// util.Chdir("/var/lib/dicloud")
	// util.Chdir("E:/Files/WorkSpace/Visual Studio Code/dicloud")
	util.Chdir("/app")

	/* 配置管理器 */
	util.InitViper("apiserver")

	/* 日志 */
	common.InitLog(enum.SERVER_LOG)

	/* 接口连接 */
	router := gin.Default()
	port := viper.GetString(enum.SERVER_PORT)
	router = ConnectRoute(router) // 绑定路由处理函数

	/* 初始化数据库连接 */
	if err := InitDB(); err != nil {
		panic("数据库连接或结构校验失败: " + err.Error())
	}

	/* 初始化Redis */
	redisutil.InitialRedis()
	go system_CheckThread()
	cache.NewRedis() // 在cache中引入RDB

	/* 轮询处理 */
	router.Run(":" + port)
}
