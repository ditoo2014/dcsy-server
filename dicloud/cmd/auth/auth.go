package main

import (
	"dicloud/internal/migration"
	"dicloud/models/enum"
	"dicloud/pkg/cache"
	"dicloud/pkg/common"
	"dicloud/util"
	nacosutil "dicloud/util/nacos"
	"dicloud/util/opengauss"
	"dicloud/util/redisutil"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func InitDB() error {
	_, err := migration.Initialize()
	return err
}

func system_CheckThread() {

	// 心跳
	for {
		// 防止连接超时关闭
		redisutil.Check()
		opengauss.Check()
		time.Sleep(time.Second * 3)
	}
}

func main() {

	util.Chdir("/var/lib/dicloud")
	// util.Chdir("E:/Files/WorkSpace/Visual Studio Code/dicloud")
	//util.Chdir("/app")

	/* 配置管理器 */
	util.InitViper("auth")

	/* 日志 */
	common.InitLog(enum.SERVER_LOG)

	/* 初始化数据库连接 */
	if err := InitDB(); err != nil {
		panic("自动迁移数据库失败: " + err.Error())
	}

	/* 初始化Redis */
	redisutil.InitialRedis()
	go system_CheckThread()
	// 在cache中引入RDB
	cache.NewRedis()

	/* 接口连接 */
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	port := viper.GetString(enum.SERVER_PORT)
	useNacos := viper.GetBool(enum.SERVER_USE_NACOS)
	// 使用spring-cloud-gateway路由; 指定服务端口模式或Nacos的lb:server模式(集群化部署更合适), 负载均衡;
	useSpringCloudGateway := viper.GetBool(enum.SERVER_USE_SPRING_CLOUD_GATEWAY)

	if useNacos {
		/* 注册Nacos服务 */
		portInt, _ := strconv.Atoi(port)
		nacosutil.InitNacosSVC(enum.GRPC_SERVER_AUTH, "127.0.0.1", "", "", portInt, 10, map[string]string{"test": "test"})
	}
	router = ConnectRoute(router, useSpringCloudGateway)

	/* 只信任本机的代理服务器 */
	router.SetTrustedProxies([]string{enum.SERVER_TRUSTED_PROXY})
	router.Run(":" + port)
}
