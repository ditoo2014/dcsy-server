package main

import (
	"dicloud/internal/migration"
	"dicloud/models/enum"
	"dicloud/pkg/cache"
	"dicloud/pkg/common"
	"dicloud/pkg/passwordpolicy"
	"dicloud/util"
	nacosutil "dicloud/util/nacos"
	"dicloud/util/opengauss"
	"dicloud/util/redisutil"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func InitDB() error {
	// 认证服务是唯一负责建表和增量迁移的后端进程。
	_, err := migration.InitializeAndMigrate()
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

	if err := util.Chdir("/var/lib/dicloud"); err != nil {
		panic("切换认证服务工作目录失败: " + err.Error())
	}
	// util.Chdir("E:/Files/WorkSpace/Visual Studio Code/dicloud")
	//util.Chdir("/app")

	/* 认证服务只读取 auth.yml，避免运行环境覆盖服务器配置。 */
	if err := util.InitViperFileOnly("auth"); err != nil {
		panic("读取认证服务配置失败: " + err.Error())
	}
	// 从本服务的 auth.yml 读取季度规则，不使用环境变量覆盖安全配置。
	if err := passwordpolicy.LoadFile(viper.ConfigFileUsed()); err != nil {
		panic("加载季度密码策略失败: " + err.Error())
	}

	/* 日志 */
	common.InitLog(enum.SERVER_LOG)
	log.Printf("[AUTH] 已加载配置文件: %s", viper.ConfigFileUsed())
	log.Printf(
		"[AUTH] 短信配置: simulate=%t, url=%s, timeoutSeconds=%d",
		viper.GetBool("sms.login.simulate"),
		viper.GetString("sms.login.url"),
		viper.GetInt("sms.login.timeoutSeconds"),
	)

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
