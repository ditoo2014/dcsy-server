package main

import (
	"context"
	commonschedule "dicloud/internal/common/schedule"
	"dicloud/internal/migration"
	"dicloud/internal/trigger/schedule"
	"dicloud/internal/trigger/service"
	"dicloud/models/enum"
	"dicloud/pkg/cache"
	"dicloud/pkg/common"
	"dicloud/util"
	bloomutil "dicloud/util/bloom"
	"dicloud/util/opengauss"
	"dicloud/util/redisutil"
	"dicloud/util/rocketutil"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var serverName string

func InitDB() {
	if _, err := migration.Initialize(); err != nil {
		log.Fatalf("[TRIGGER] 数据库连接或结构校验失败: %v", err)
	}
}

func system_CheckThread() {

	// 心跳
	for {
		// 防止连接超时关闭
		// redisutil.Check() // 与RocketMQ去重部分重复, 移除
		opengauss.Check()
		schedule.RefreshDatasource()
		time.Sleep(time.Second * 5)
	}
}

func redis_StoreManager() {

	// 自动竞争锁, 管理
	for {
		if res := commonschedule.DedumplicationRocketMQCleanScheduleLock(serverName); !res {
			// log.Println("[" + strings.ToUpper(serverName) + "] Redis Dedumplication RocketMQ, RedisStore not manage by me")
			// 30秒后进行下一次尝试 - 不要老尝试, 多半有一个在管理了, 那就一直归另一个管理; 最多延时30秒
			time.Sleep(30 * time.Second)
			continue
		}
		commonschedule.DedumplicationRocketMQCleanSchedule(serverName)
		// 在获取到之后, 锁会保留一段时间, 稍微留点程序执行时间; 不要一直获取锁
		// 等待时间 > 锁保留时间; 等待锁清空, 再次公平竞争;
		// 30秒和10秒都等锁过期, > 锁时间
		time.Sleep(10 * time.Second)
	}
}

func onMessage(ctx context.Context, msg ...*primitive.MessageExt) (res consumer.ConsumeResult, err error) {
	if len(msg) == 0 || msg[0] == nil {
		log.Println("[TRIGGER] onMessage received empty message batch")
		return consumer.ConsumeSuccess, nil
	}

	defer func() {
		rec := recover()
		if rec != nil {
			log.Printf("[TRIGGER] onMessage panic, msgId=%s, reconsumeTimes=%d, err=%v\n%s", msg[0].MsgId, msg[0].ReconsumeTimes, rec, debug.Stack())
			res = consumer.ConsumeRetryLater
			err = nil
		}
	}()

	val := ctx.Value("dbName")
	ginCtx := &gin.Context{}
	if val != nil {
		ginCtx.Set("dbName", fmt.Sprint(val))
	}
	// 添加消息的Key到去重表
	handleMsg, retryLater := commonschedule.ShouldHandleDedumplicationRocketMSG(ginCtx, serverName, msg[0].MsgId, msg[0].ReconsumeTimes)
	if retryLater {
		return consumer.ConsumeRetryLater, nil
	}
	if !handleMsg {
		return consumer.ConsumeSuccess, nil
	}

	// 开始处理消息
	// msg虽然是数组，但长度都是1
	// fmt.Printf("[TRIGGER] 订阅消息：%v \n", msg[0].Topic)

	// 根据标签区分处理方式
	switch msg[0].GetTags() {
	case enum.TAG_TRIGGER_BATTERY:
		service.HandleBatteryTrigger(msg[0])

	case enum.TAG_TRIGGER_DOR:
		// fmt.Println("[TRIGGER] tag", msg[0].GetTags())
		service.HandleDorTrigger(msg[0])
		service.HandleTypeStrSwitchTrigger(msg[0])

	case enum.TAG_TRIGGER_DR:
		service.HandleDrTrigger(msg[0])

	case enum.TAG_TRIGGER_XR:
		service.HandleXrTrigger(msg[0])

	default:
		/* 打印其他数据 */
		log.Println("[TRIGGER SERVER] useless msg:", msg)
	}

	return consumer.ConsumeSuccess, nil
}

func main() {

	util.Chdir("/var/lib/dicloud")
	// util.Chdir("E:/Files/WorkSpace/Visual Studio Code/dicloud")

	/* 配置管理器 */
	util.InitViper("trigger")

	/* 日志 */
	common.InitLog(enum.SERVER_LOG)

	/* 初始化数据库连接 */
	InitDB()

	/* Bloom过滤器 */
	/* 先刷新一下数据, 防止rocket先获取了数据, bloom判断不存在 */
	bloomutil.InitialBloomFilter()

	/* 初始化Redis */
	if err := redisutil.InitialRedis(); err != nil {
		log.Fatalf("[TRIGGER] 初始化 Redis 失败: %v", err)
	}
	/* 在cache中引入RDB */
	cache.NewRedis()
	schedule.RefreshDatasource()
	go system_CheckThread()

	// RocketMQ消息去重, 及Redis内存管理
	serverName = viper.GetString(enum.SERVER_NAME)
	go redis_StoreManager()

	/* 消息队列 */
	if err := rocketutil.EnsureTopics(enum.XC_DEAL_TRIGGER_QUEUE, enum.XC_TRIGGER_ALARM_QUEUE, enum.XC_TRIGGER_STATISTIC_QUEUE); err != nil {
		log.Fatalf("[TRIGGER] 准备 RocketMQ 主题失败: %v", err)
	}
	if err := rocketutil.InitRocketConsumer(enum.GROUP_TRIGGER_CONSUMER); err != nil {
		log.Fatalf("[TRIGGER] 初始化 RocketMQ 消费者失败: %v", err)
	}
	if err := rocketutil.InitRocketProducer(enum.GROUP_TRIGGER_PRODUCER); err != nil {
		log.Fatalf("[TRIGGER] 初始化 RocketMQ 生产者失败: %v", err)
	}
	defer rocketutil.ShutdownConsumer()
	defer rocketutil.ShutdownProducer()
	if err := rocketutil.StartProducer(); err != nil {
		log.Fatalf("[TRIGGER] 启动 RocketMQ 生产者失败: %v", err)
	}

	/* 订阅主题, 先创建topic才能订阅成功(可以等生产者先启动, 发送消息, 自动生成topic之后启动; 或手动创建topic) */
	err := rocketutil.GetROCKET().Consumer.Subscribe(enum.XC_DEAL_TRIGGER_QUEUE, consumer.MessageSelector{}, onMessage)
	if err != nil {
		log.Fatalf("[TRIGGER] 订阅主题失败: %v", err)
	}
	//rocketutil.PubAsync(enum.XC_DEAL_TRIGGER_QUEUE, "test12345", "test")
	if err := rocketutil.StartConsumer(); err != nil {
		log.Fatalf("[TRIGGER] 启动 RocketMQ 消费者失败: %v", err)
	}

	/* 循环等待 */
	for {
		time.Sleep(time.Hour)
	}
}
