package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-clients/golang"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"log"
	"mqtt/internal/batterylog"
	"mqtt/internal/mqtt/service"
	"mqtt/models/enum"
	"mqtt/pkg/cache"
	"mqtt/pkg/common"
	"mqtt/util"
	mqttutil "mqtt/util/mqtt"
	"mqtt/util/redisutil"
	"mqtt/util/rocketutil"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var ProducerOfficial golang.Producer
var serverName string
var batteryDisabledLogOnce sync.Once
var batteryGatewayIDPattern = regexp.MustCompile(`^[0-9]{15}$`)
var batteryTopicSegmentPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)

type batteryTtdcMQTTMessage struct {
	Version    int    `json:"version"`
	MessageID  string `json:"messageId"`
	GatewayID  string `json:"gatewayId"`
	Topic      string `json:"topic"`
	Payload    string `json:"payload"`
	ReceivedAt int64  `json:"receivedAt"`
}

// buildBatteryTtdcMessage 将设备原始报文包装成处理程序统一接收的消息格式。
func buildBatteryTtdcMessage(topic, payload, gatewayID string, receivedAt int64) (string, string, error) {
	id := uuid.NewString()
	body, err := json.Marshal(batteryTtdcMQTTMessage{Version: 1, MessageID: id, GatewayID: gatewayID, Topic: topic, Payload: payload, ReceivedAt: receivedAt})
	return string(body), id, err
}

func parseBatteryCandidateTopic(topic string) (string, string, bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(topic), "/"), "/")
	if len(parts) != 4 || parts[0] != "DCBMS4G" || parts[2] != "PUB" {
		return "", "", false
	}
	topicSegment := strings.TrimSpace(parts[1])
	gatewayID := strings.TrimSpace(parts[3])
	if topicSegment == "" || !batteryGatewayIDPattern.MatchString(gatewayID) || strings.ContainsAny(topicSegment+gatewayID, " \t\r\n") {
		return "", "", false
	}

	// 默认采用明确白名单，避免通配订阅把无关设备消息转发到电池处理队列。
	mode := strings.ToLower(strings.TrimSpace(viper.GetString("battery.candidateRoutingMode")))
	if mode == "" {
		mode = "allowlist"
	}
	switch mode {
	case "allowlist":
		segments := viper.GetStringSlice("battery.candidateTopicSegments")
		if len(segments) == 0 {
			segments = []string{"SodiumBattery", "LithiumSulfurBattery"}
		}
		for _, segment := range segments {
			if strings.TrimSpace(segment) == topicSegment {
				return topicSegment, gatewayID, true
			}
		}
		return "", "", false
	case "exclusive_namespace":
		if !viper.GetBool("battery.candidateNamespaceExclusiveConfirmed") {
			return "", "", false
		}
		return topicSegment, gatewayID, true
	default:
		return "", "", false
	}
}

func validateBatteryRoutingConfig() error {
	if !viper.GetBool("battery.enabled") {
		return nil
	}
	filter := strings.TrimSpace(viper.GetString("battery.subscribeFilter"))
	if filter == "" {
		return fmt.Errorf("battery.subscribeFilter 不能为空")
	}
	mode := strings.ToLower(strings.TrimSpace(viper.GetString("battery.candidateRoutingMode")))
	switch mode {
	case "allowlist":
		segments := viper.GetStringSlice("battery.candidateTopicSegments")
		if len(segments) == 0 {
			return fmt.Errorf("allowlist 模式下 battery.candidateTopicSegments 不能为空")
		}
		seen := make(map[string]struct{}, len(segments))
		for _, value := range segments {
			segment := strings.TrimSpace(value)
			if !batteryTopicSegmentPattern.MatchString(segment) {
				return fmt.Errorf("无效的电池 Topic 段: %q", value)
			}
			if _, exists := seen[segment]; exists {
				return fmt.Errorf("电池 Topic 段重复: %s", segment)
			}
			seen[segment] = struct{}{}
		}
	case "exclusive_namespace":
		if !viper.GetBool("battery.candidateNamespaceExclusiveConfirmed") {
			return fmt.Errorf("exclusive_namespace 模式必须确认 battery.candidateNamespaceExclusiveConfirmed")
		}
	default:
		return fmt.Errorf("不支持的 battery.candidateRoutingMode: %q", mode)
	}
	return nil
}

func receiveAll(topic, msg string) {
	receivedAt := time.Now().Unix()
	ctx := &gin.Context{}
	if _, gatewayID, ok := parseBatteryCandidateTopic(topic); ok {
		sendBatteryTtdc(ctx, topic, msg, gatewayID, receivedAt)
	}
}

func sendBatteryTtdc(ctx *gin.Context, topic, payload, gatewayID string, receivedAt int64) {
	if !viper.GetBool("battery.enabled") {
		batteryDisabledLogOnce.Do(func() { batterylog.Printf("[MQTT-BATTERY] forwarding disabled, topic=%s\n", topic) })
		return
	}
	body, _, err := buildBatteryTtdcMessage(topic, payload, gatewayID, receivedAt)
	if err != nil {
		batterylog.Printf("[MQTT-BATTERY] marshal failed: %v\n", err)
		return
	}
	dealQueue := viper.GetString("battery.dealQueue")
	if dealQueue == "" {
		dealQueue = enum.XC_BATTERY_DEAL_QUEUE
	}
	tag := viper.GetString("battery.tag")
	if tag == "" {
		tag = enum.TAG_MQTT_BATTERY_TTDC
	}
	if !rocketutil.PubWithShardingKey(ctx, dealQueue, string(body), tag, gatewayID) {
		batterylog.Printf("[MQTT-BATTERY] publish failed, gatewayID=%s\n", gatewayID)
	}
}

func onMessage(ctx context.Context, msg ...*primitive.MessageExt) (res consumer.ConsumeResult, err error) {
	if len(msg) == 0 || msg[0] == nil {
		return consumer.ConsumeSuccess, nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("[MQTT] panic: %v\n%s", recovered, debug.Stack())
			res = consumer.ConsumeRetryLater
		}
	}()
	ginCtx := &gin.Context{}
	if val := ctx.Value("dbName"); val != nil {
		ginCtx.Set("dbName", fmt.Sprint(val))
	}
	var topic, data string
	switch msg[0].GetTags() {
	case enum.TAG_MQTT_CMD:
		topic, data = service.DealCmd(string(msg[0].Body))
	case enum.TAG_MQTT_CMDO:
		topic, data = service.DealCmdo(string(msg[0].Body))
	default:
		return consumer.ConsumeSuccess, nil
	}
	if topic != "" {
		mqttutil.MqttAuth.Pub(topic, data)
	}
	return consumer.ConsumeSuccess, nil
}

func main() {
	_ = util.Chdir(".")
	if err := util.InitViper("mqtt"); err != nil {
		log.Fatal(err)
	}
	if err := validateBatteryRoutingConfig(); err != nil {
		log.Fatal(err)
	}
	common.InitLog(enum.SERVER_LOG)
	if err := redisutil.InitialRedis(); err != nil {
		log.Fatal(err)
	}
	cache.NewRedis()
	serverName = viper.GetString(enum.SERVER_NAME)
	mqtt := mqttutil.InitAuthMqtt()
	if mqtt == nil || mqtt.Client == nil {
		log.Fatal("[MQTT] 初始化 MQTT 客户端失败，请检查地址、账号、密码和 RSA 私钥配置")
	}
	defer mqttutil.CloseAuthMqtt()
	dealQueue := viper.GetString("battery.dealQueue")
	if dealQueue == "" {
		dealQueue = enum.XC_BATTERY_DEAL_QUEUE
	}
	// 先准备 RocketMQ 主题和客户端，避免设备消息先到导致生产者为空。
	if err := rocketutil.EnsureTopics(dealQueue, enum.XC_MQTT_CMD_QUEUE); err != nil {
		log.Fatalf("[MQTT] 准备 RocketMQ 主题失败: %v", err)
	}
	if err := rocketutil.InitRocketProducer(enum.GROUP_MQTT_PRODUCER); err != nil {
		log.Fatalf("[MQTT] 初始化 RocketMQ 生产者失败: %v", err)
	}
	defer rocketutil.ShutdownProducer()
	if err := rocketutil.StartProducer(); err != nil {
		log.Fatalf("[MQTT] 启动 RocketMQ 生产者失败: %v", err)
	}
	if err := rocketutil.InitRocketConsumer(enum.GROUP_MQTT_CONSUMER); err != nil {
		log.Fatal(err)
	}
	defer rocketutil.ShutdownConsumer()
	if err := rocketutil.GetROCKET().Consumer.Subscribe(enum.XC_MQTT_CMD_QUEUE, consumer.MessageSelector{}, onMessage); err != nil {
		log.Fatal(err)
	}
	if err := rocketutil.StartConsumer(); err != nil {
		log.Fatal(err)
	}
	subscribeFilter := viper.GetString("battery.subscribeFilter")
	if subscribeFilter == "" {
		subscribeFilter = "/DCBMS4G/+/PUB/+"
	}
	mqtt.Sub(subscribeFilter, receiveAll)
	batterylog.Printf("[MQTT-BATTERY] 订阅已启用 filter=%s routingMode=%s allowlist=%v queue=%s tag=%s\n",
		subscribeFilter, viper.GetString("battery.candidateRoutingMode"),
		viper.GetStringSlice("battery.candidateTopicSegments"),
		viper.GetString("battery.dealQueue"), viper.GetString("battery.tag"))
	for {
		time.Sleep(time.Hour)
	}
}
