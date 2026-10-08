package main

import (
	"context"
	"deal/internal/batterylog"
	"deal/internal/deal/service"
	"deal/models/enum"
	"deal/pkg/cache"
	"deal/pkg/common"
	"deal/util"
	"deal/util/redisutil"
	"deal/util/rocketutil"
	"deal/util/tdengine"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"log"
	"runtime/debug"
	"time"
)

// 仅消费 mqtt 转发的 TTDC 锂硫电池消息。
func onBatteryMessage(ctx context.Context, messages ...*primitive.MessageExt) (result consumer.ConsumeResult, err error) {
	if len(messages) == 0 || messages[0] == nil {
		return consumer.ConsumeSuccess, nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("[DEAL-BATTERY] panic: %v\n%s", recovered, debug.Stack())
			result = consumer.ConsumeRetryLater
		}
	}()
	ginCtx := &gin.Context{}
	if dbName := ctx.Value("dbName"); dbName != nil {
		ginCtx.Set("dbName", fmt.Sprint(dbName))
	}
	for _, message := range messages {
		tag := viper.GetString("battery.tag")
		if tag == "" {
			tag = enum.TAG_MQTT_BATTERY_TTDC
		}
		if message == nil || message.GetTags() != tag {
			continue
		}
		if err := service.DealBatteryTtdc(ginCtx, message); err != nil {
			batterylog.Printf("[DEAL-BATTERY] message failed: %v\n", err)
			if service.IsBatteryErrorRetryable(err) {
				return consumer.ConsumeRetryLater, nil
			}
		}
	}
	return consumer.ConsumeSuccess, nil
}

func main() {
	_ = util.InitViper("deal")
	common.InitLog(enum.SERVER_LOG)
	tdengine.InitTDengine()
	if err := redisutil.InitialRedis(); err != nil {
		log.Fatal(err)
	}
	cache.NewRedis()
	service.InitBatteryTtdc()
	dealQueue := viper.GetString("battery.dealQueue")
	if dealQueue == "" {
		dealQueue = enum.XC_BATTERY_DEAL_QUEUE
	}
	if err := rocketutil.EnsureTopics(dealQueue, enum.XC_DEAL_TRIGGER_QUEUE); err != nil {
		log.Fatalf("[DEAL-BATTERY] 准备 RocketMQ 主题失败: %v", err)
	}
	if err := rocketutil.InitRocketProducer(enum.GROUP_DEAL_PRODUCER); err != nil {
		log.Fatalf("[DEAL-BATTERY] 初始化 RocketMQ 生产者失败: %v", err)
	}
	defer rocketutil.ShutdownProducer()
	if err := rocketutil.StartProducer(); err != nil {
		log.Fatalf("[DEAL-BATTERY] 启动 RocketMQ 生产者失败: %v", err)
	}
	if err := rocketutil.InitRocketConsumer(enum.GROUP_DEAL_CONSUMER); err != nil {
		log.Fatal(err)
	}
	defer rocketutil.ShutdownConsumer()
	if err := rocketutil.GetROCKET().Consumer.Subscribe(dealQueue, consumer.MessageSelector{}, onBatteryMessage); err != nil {
		log.Fatal(err)
	}
	if err := rocketutil.StartConsumer(); err != nil {
		log.Fatal(err)
	}
	for {
		time.Sleep(time.Hour)
	}
}
