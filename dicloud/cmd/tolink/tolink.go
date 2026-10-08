package main

import (
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/internal/migration"
	grpcserver "dicloud/internal/tolink/server"
	"dicloud/models/enum"
	"dicloud/pkg/cache"
	"dicloud/pkg/common"
	"dicloud/util"
	nacosutil "dicloud/util/nacos"
	"dicloud/util/opengauss"
	"dicloud/util/redisutil"
	"dicloud/util/tdengine"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

func InitDB() error {
	if _, err := migration.Initialize(); err != nil {
		return err
	}
	tdengine.InitTDengine()
	return nil
}

func system_CheckThread(quit chan struct{}) {

	// 心跳
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			// 收到退出信号，优雅退出
			log.Println("[TOLINK] system_CheckThread 优雅退出")
			return
		case <-ticker.C:
			// 防止连接超时关闭
			redisutil.Check()
			opengauss.Check()
		}
	}
}

func main() {

	util.Chdir("/var/lib/dicloud")
	// util.Chdir("E:/Files/WorkSpace/Visual Studio Code/dicloud")
	//util.Chdir("/app")

	/* 配置管理器 */
	util.InitViper("tolink")

	/* 日志 */
	common.InitLog(enum.SERVER_LOG)

	/* 接口连接 */
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	port := viper.GetString(enum.SERVER_PORT)

	/* 初始化数据库连接 */
	if err := InitDB(); err != nil {
		log.Println("[TOLINK] 数据库初始化失败", err)
		return
	}

	/* 初始化Redis */
	redisutil.InitialRedis()
	stopChan := make(chan struct{})
	go system_CheckThread(stopChan)
	defer close(stopChan) // 确保程序退出时发送退出信号
	cache.NewRedis()      // 在cache中引入RDB

	/* 设置路由 - 信任本机代理 */
	router.SetTrustedProxies([]string{enum.SERVER_TRUSTED_PROXY})
	// REPLACE: 替换为http分流处理, 区分GRPC和普通请求(不带证书)
	// router.Run(":" + port)

	s := grpc.NewServer()
	server := grpcserver.TolinkServer{}
	// 将server结构体注册为gRPC服务。
	tolink_grpc.RegisterTolinkServiceServer(s, &server)
	// 开始处理客户端请求。

	// 监听端口并处理服务分流 - http/gRPC
	h2Handler := h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 判断协议是否为http/2 && 是grpc
		if r.ProtoMajor == 2 &&
			strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			// 按grpc方式来请求
			s.ServeHTTP(w, r)
		} else {
			// 当作普通api
			router.ServeHTTP(w, r)
		}
	}), &http2.Server{})

	// 注册中心 - 注册http+gRPC服务
	useNacos := viper.GetBool(enum.SERVER_USE_NACOS)
	// 使用spring-cloud-gateway路由; 指定服务端口模式或Nacos的lb:server模式(集群化部署更合适), 负载均衡;
	useSpringCloudGateway := viper.GetBool(enum.SERVER_USE_SPRING_CLOUD_GATEWAY)
	if useNacos {
		/* 注册Nacos服务 */
		portInt, err := strconv.Atoi(port)
		if err != nil {
			log.Println("[TOLINK] 端口转换失败", err)
			return
		}
		nacosutil.InitNacosSVC(enum.GRPC_SERVER_TOLINK, common.GetLocalAddr(), "", "", portInt, 10, map[string]string{"test": "test"})
	}
	router = ConnectRoute(router, useSpringCloudGateway)

	// 监听HTTP服务 - 通过自定义方法分流
	if err := http.ListenAndServe(":"+port, h2Handler); err != nil {
		log.Println("http server done:", err.Error())
	}

}
