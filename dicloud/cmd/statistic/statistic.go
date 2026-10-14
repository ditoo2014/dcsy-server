package main

import (
	"context"
	commonschedule "dicloud/internal/common/schedule"
	"dicloud/internal/migration"
	"dicloud/internal/statistic/service"
	"dicloud/models/enum"
	"dicloud/pkg/cache"
	"dicloud/pkg/common"
	"dicloud/util"
	"dicloud/util/opengauss"
	"dicloud/util/redisutil"
	"dicloud/util/rocketutil"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"time"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

var serverName string

func InitDB() {
	if _, err := migration.Initialize(); err != nil {
		log.Fatalf("[STATISTIC] 数据库连接或结构校验失败: %v", err)
	}
}

func system_CheckThread() {

	// 心跳
	for {
		// 防止连接超时关闭
		// redisutil.Check() // 与RocketMQ去重部分重复, 移除
		opengauss.Check()
		time.Sleep(time.Second * 3)
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
		log.Println("[STATISTIC] onMessage received empty message batch")
		return consumer.ConsumeSuccess, nil
	}

	defer func() {
		rec := recover()
		if rec != nil {
			log.Printf("[STATISTIC] onMessage panic, msgId=%s, reconsumeTimes=%d, err=%v\n%s", msg[0].MsgId, msg[0].ReconsumeTimes, rec, debug.Stack())
			res = consumer.ConsumeRetryLater
			err = nil
		}
	}()

	val := ctx.Value("dbName")
	ginCtx := &gin.Context{}
	if val != nil {
		ginCtx.Set("dbName", fmt.Sprint(val))
	}
	// log.Println("[STATISTIC] dbName param check: ", val)
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
	// REMOVE:
	// fmt.Printf("[STATISTIC] 订阅消息：%v \n", msg[0])

	/* 进行数据检查保存 */

	// TODO: acording to data mark. distingush plat data.
	data, b := service.BaseDataBind(ginCtx, msg[0])
	if !b {
		/* 非法数据 */
		log.Println("[STATISTIC SERVER] illegal msg:", msg)
		// 取消去重校验
		commonschedule.CancelDedumplicationRocketMSG(ginCtx, serverName, msg[0].MsgId)
		return consumer.ConsumeRetryLater, nil
	}

	// 根据统计数据类型区分处理方式
	strs := strings.Split(data.StatisticType, ",")
	// fmt.Println("[STATISTIC] strs: ", strs)
	for _, v := range strs {
		switch v {
		case "":
			// do nothing.
			continue

		case enum.TAG_STATISTIC_NORMAL:
			// 通用处理方式, 触发方式, 简单统计
			// fmt.Println("[STATISTIC] tag", v)
			service.StatisticNormalData(ginCtx, data)

		case enum.TAG_STATISTIC_ALARM:
			// 报警处理方式, 触发方式, 统计条数
			// fmt.Println("[STATISTIC] tag", v)
			service.StatisticAlarmData(ginCtx, data)

		default:
			/* 打印其他数据 */
			log.Println("[STATISTIC SERVER] useless msg:", msg)
		}
	}

	return consumer.ConsumeSuccess, nil
}

func onBusinessMessage(ctx context.Context, msg ...*primitive.MessageExt) (res consumer.ConsumeResult, err error) {
	if len(msg) == 0 || msg[0] == nil {
		log.Println("[STATISTIC] onBusinessMessage received empty message batch")
		return consumer.ConsumeSuccess, nil
	}

	defer func() {
		rec := recover()
		if rec != nil {
			log.Printf("[STATISTIC] onBusinessMessage panic, msgId=%s, reconsumeTimes=%d, err=%v\n%s", msg[0].MsgId, msg[0].ReconsumeTimes, rec, debug.Stack())
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
	// REMOVE:
	// fmt.Printf("[STATISTIC] 订阅消息：%v \n", msg[0])

	/* 进行数据检查保存 */
	data, b := service.BaseDataBind(ginCtx, msg[0])
	if !b {
		/* 非法数据 */
		log.Println("[STATISTIC SERVER] onBusinessMessage illegal msg:", msg)
		return consumer.ConsumeRetryLater, nil
	}

	// dataStr, _ := json.Marshal(data)
	// fmt.Println("line 131 check: ", string(dataStr))
	if !service.StatisticEquipmentData(ginCtx, data) {
		commonschedule.CancelDedumplicationRocketMSG(ginCtx, serverName, msg[0].MsgId)
		return consumer.ConsumeRetryLater, nil
	}

	return consumer.ConsumeSuccess, nil
}

func main() {

	util.Chdir("/var/lib/dicloud")
	// util.Chdir("E:/Files/WorkSpace/Visual Studio Code/dicloud")
	//util.Chdir("/app")

	/* 配置管理器 */
	util.InitViper("statistic")

	/* 日志 */
	common.InitLog(enum.SERVER_LOG)

	/* 初始化数据库连接 */
	InitDB()

	/* 初始化Redis */
	if err := redisutil.InitialRedis(); err != nil {
		log.Fatalf("[STATISTIC] 初始化 Redis 失败: %v", err)
	}
	go system_CheckThread()
	/* 在cache中引入RDB */
	cache.NewRedis()

	// RocketMQ消息去重, 及Redis内存管理
	serverName = viper.GetString(enum.SERVER_NAME)
	go redis_StoreManager()

	/* 消息队列 */
	if err := rocketutil.EnsureTopics(enum.XC_TRIGGER_STATISTIC_QUEUE, enum.XC_BUSINESS_STATISTIC_QUEUE); err != nil {
		log.Fatalf("[STATISTIC] 准备 RocketMQ 主题失败: %v", err)
	}
	if err := rocketutil.InitRocketConsumer(enum.GROUP_STATISTIC_CONSUMER); err != nil {
		log.Fatalf("[STATISTIC] 初始化 RocketMQ 消费者失败: %v", err)
	}
	defer rocketutil.ShutdownConsumer()

	/* 订阅主题, 先创建topic才能订阅成功(可以等生产者先启动, 发送消息, 自动生成topic之后启动; 或手动创建topic) */
	// 为保持订阅一致性, 所有此服务(启动多任务时), 需保持以下代码一致相同
	err := rocketutil.GetROCKET().Consumer.Subscribe(enum.XC_TRIGGER_STATISTIC_QUEUE, consumer.MessageSelector{}, onMessage)
	if err != nil {
		log.Fatalf("[STATISTIC] 订阅 %s 主题失败: %v", enum.XC_TRIGGER_STATISTIC_QUEUE, err)
	}
	err = rocketutil.GetROCKET().Consumer.Subscribe(enum.XC_BUSINESS_STATISTIC_QUEUE, consumer.MessageSelector{}, onBusinessMessage)
	if err != nil {
		log.Fatalf("[STATISTIC] 订阅 %s 主题失败: %v", enum.XC_BUSINESS_STATISTIC_QUEUE, err)
	}
	if err := rocketutil.StartConsumer(); err != nil {
		log.Fatalf("[STATISTIC] 启动 RocketMQ 消费者失败: %v", err)
	}

	/* 循环等待 */
	for {
		time.Sleep(time.Hour)
	}
}
