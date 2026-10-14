package main

import (
	"dicloud/internal/business/schedule"
	"dicloud/models/enum"
	"dicloud/pkg/cache"
	"dicloud/pkg/common"
	"dicloud/pkg/passwordpolicy"
	"dicloud/util"
	"dicloud/util/gormutil"
	"dicloud/util/opengauss"
	"dicloud/util/redisutil"
	"dicloud/util/rocketutil"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func system_CheckThread() {

	// 等待其他数据初始化完成 - 防止定时任务执行报错
	time.Sleep(time.Second * 3)

	// 心跳
	for {
		// 防止连接超时关闭
		redisutil.Check()
		opengauss.Check()
		time.Sleep(time.Second * 3)
	}
}

func schedule_Manager() {

	for {
		// 竞争锁, 每次可能不同的服务执行, 定时任务
		if !schedule.BaseScheduleLock() {
			// 间隔1秒, 竞争
			time.Sleep(1 * time.Second)
			continue
		}
		// 定时任务 - 任务内部可能有锁判断, 不影响整个流程(多一步校验流程)
		schedule.StatisticPlatEquipmentSchedule()
		schedule.RefreshDatasource()
		time.Sleep(3 * time.Second)
	}
}

func main() {

	util.Chdir("/var/lib/dicloud")
	// util.Chdir("E:/Files/WorkSpace/Visual Studio Code/dicloud")
	//util.Chdir("/app")

	/* 配置管理器 */
	if err := util.InitViper("business"); err != nil {
		panic("读取业务服务配置失败: " + err.Error())
	}
	// 从本服务的 business.yml 读取季度规则，环境变量不参与季度规则。
	if err := passwordpolicy.LoadFile(viper.ConfigFileUsed()); err != nil {
		panic("加载季度密码策略失败: " + err.Error())
	}

	/* 日志 */
	common.InitLog(enum.SERVER_LOG)

	/* 接口连接 */
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	port := viper.GetString(enum.SERVER_PORT)
	router = ConnectRoute(router)

	/* 初始化数据库连接和业务表结构 */
	if err := gormutil.InitPlatGormDBWithError(); err != nil {
		log.Printf("[BUSINESS] 数据库初始化失败: %v", err)
		return
	}

	/* 初始化Redis */
	redisutil.InitialRedis()
	go system_CheckThread()
	// 在cache中引入RDB
	cache.NewRedis()

	/* 消息队列 */
	if err := rocketutil.EnsureTopics(enum.XC_BUSINESS_STATISTIC_QUEUE); err != nil {
		log.Fatalf("[BUSINESS] 准备 RocketMQ 主题失败: %v", err)
	}
	if err := rocketutil.InitRocketProducer(enum.GROUP_BUSINESS_PRODUCER); err != nil {
		log.Fatalf("[BUSINESS] 初始化 RocketMQ 生产者失败: %v", err)
	}
	defer rocketutil.ShutdownProducer()
	if err := rocketutil.StartProducer(); err != nil {
		log.Fatalf("[BUSINESS] 启动 RocketMQ 生产者失败: %v", err)
	}

	// RocketMQ 准备完成后再启动定时任务，避免任务抢先发送消息。
	go schedule_Manager()

	/* 只信任本机的代理服务器 */
	router.SetTrustedProxies([]string{enum.SERVER_TRUSTED_PROXY})
	router.Run(":" + port)

}
