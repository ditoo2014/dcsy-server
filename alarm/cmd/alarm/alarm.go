package main

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"alarm/internal/alarm/service"
	"alarm/models/enum"
	"alarm/pkg/common"
	"alarm/util"
	"alarm/util/opengauss"
	"alarm/util/rocketutil"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func InitDB() error {
	if err := opengauss.InitOpenGaussDB(); err != nil {
		return fmt.Errorf("initialize OpenGauss: %w", err)
	}
	if db := opengauss.InitGormDB(); db == nil {
		return fmt.Errorf("initialize OpenGauss GORM failed")
	}
	if err := service.AutoMigrateAlarmDatabases(); err != nil {
		return fmt.Errorf("auto migrate alarm schema: %w", err)
	}
	// 自动迁移后立即校验关键唯一索引，避免缺少索引时并发产生重复告警或重复通知。
	if err := service.ValidateAlarmReliabilityDatabases(); err != nil {
		return fmt.Errorf("validate alarm schema: %w", err)
	}

	// Notification configuration must not block alarm persistence.
	if err := service.InitAllSmsClient(&gin.Context{}); err != nil {
		log.Printf("[ALARM] initialize SMS client failed: %v", err)
	}
	return nil
}

func systemCheckThread() {
	for {
		_ = opengauss.Check()
		time.Sleep(30 * time.Second)
	}
}

func onMessage(ctx context.Context, messages ...*primitive.MessageExt) (result consumer.ConsumeResult, err error) {
	if len(messages) == 0 || messages[0] == nil {
		log.Println("[ALARM] onMessage received empty message batch")
		return consumer.ConsumeSuccess, nil
	}
	message := messages[0]
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("[ALARM] onMessage panic, msgId=%s, reconsumeTimes=%d, err=%v\n%s",
				message.MsgId, message.ReconsumeTimes, recovered, debug.Stack())
			result = consumer.ConsumeRetryLater
			err = nil
		}
	}()

	ginCtx := &gin.Context{}
	if dbName := ctx.Value("dbName"); dbName != nil {
		ginCtx.Set("dbName", fmt.Sprint(dbName))
	}
	if _, persistErr := service.AcceptAlarmEvent(ginCtx, message); persistErr != nil {
		log.Printf("[ALARM] persist message failed, msgId=%s, reconsumeTimes=%d, err=%v",
			message.MsgId, message.ReconsumeTimes, persistErr)
		return consumer.ConsumeRetryLater, nil
	}
	return consumer.ConsumeSuccess, nil
}

func main() {
	util.Chdir("/app")
	if err := util.InitViper("alarm"); err != nil {
		log.Fatalf("[ALARM] initialize config failed: %v", err)
	}
	common.InitLog(enum.SERVER_LOG)

	if err := InitDB(); err != nil {
		log.Fatalf("[ALARM] initialize database failed: %v", err)
	}
	go systemCheckThread()
	go service.RunAlarmDeliveryWorker()
	go service.RunAlarmReliabilityCleanupWorker()

	inputQueue := viper.GetString("alarm.inputQueue")
	if inputQueue == "" {
		inputQueue = enum.XC_TRIGGER_ALARM_QUEUE
	}
	if err := rocketutil.EnsureTopics(inputQueue); err != nil {
		log.Fatalf("[ALARM] 准备 RocketMQ 主题失败: %v", err)
	}
	if err := rocketutil.InitRocketConsumer(enum.GROUP_ALARM_CONSUMER); err != nil {
		log.Fatalf("[ALARM] initialize RocketMQ consumer failed: %v", err)
	}
	defer rocketutil.ShutdownConsumer()
	if err := rocketutil.GetROCKET().Consumer.Subscribe(
		inputQueue, consumer.MessageSelector{}, onMessage,
	); err != nil {
		log.Fatalf("[ALARM] subscribe RocketMQ topic failed: %v", err)
	}
	if err := rocketutil.StartConsumer(); err != nil {
		log.Fatalf("[ALARM] start RocketMQ consumer failed: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := ConnectRoute(gin.Default())
	_ = router.SetTrustedProxies([]string{enum.SERVER_TRUSTED_PROXY})
	if err := router.Run(":" + viper.GetString(enum.SERVER_PORT)); err != nil {
		log.Fatalf("[ALARM] start HTTP server failed: %v", err)
	}
}
