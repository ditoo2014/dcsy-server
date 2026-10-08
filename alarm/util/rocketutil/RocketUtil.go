package rocketutil

import (
	"alarm/util/encryption"
	snowflakeutil "alarm/util/snowflake"
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/admin"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

var rocket ROCKET

type ROCKET struct {
	Producer rocketmq.Producer
	Consumer rocketmq.PushConsumer
}

var (
	// maximum waiting time for receive func
	awaitDuration = time.Second * 5
	// maximum number of messages received at one time
	maxMessageNum int32 = 16
	// invisibleDuration should > 20s
	invisibleDuration = time.Second * 20
	// receive messages in a loop
)

// rocketCredentials 读取并解密 RocketMQ 访问凭据。
func rocketCredentials() (primitive.Credentials, error) {
	secretKey := viper.GetString("rocket.secretkey")
	if secretKey != "" {
		decoded, err := encryption.DefaultPKCS8Decrypt(secretKey)
		if err != nil {
			return primitive.Credentials{}, fmt.Errorf("解密 RocketMQ secretkey 失败: %w", err)
		}
		secretKey = decoded
	}
	return primitive.Credentials{AccessKey: viper.GetString("rocket.accesskey"), SecretKey: secretKey}, nil
}

// EnsureTopics 在服务启动前检查并按配置自动创建业务主题。
func EnsureTopics(topicNames ...string) error {
	endpoint := viper.GetString("rocket.endpoint")
	if endpoint == "" {
		return fmt.Errorf("未配置 rocket.endpoint")
	}
	credentials, err := rocketCredentials()
	if err != nil {
		return err
	}
	rocketAdmin, err := admin.NewAdmin(admin.WithResolver(primitive.NewPassthroughResolver([]string{endpoint})), admin.WithCredentials(credentials))
	if err != nil {
		return fmt.Errorf("连接 RocketMQ 管理端失败: %w", err)
	}
	defer rocketAdmin.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	topicList, err := rocketAdmin.FetchAllTopicList(ctx)
	if err != nil {
		return fmt.Errorf("查询 RocketMQ 主题列表失败: %w", err)
	}
	existing := make(map[string]struct{}, len(topicList.TopicList))
	for _, topic := range topicList.TopicList {
		existing[topic] = struct{}{}
	}
	autoCreate := true
	if viper.IsSet("rocket.autoCreateTopic") {
		autoCreate = viper.GetBool("rocket.autoCreateTopic")
	}
	readQueueNums, writeQueueNums := viper.GetInt("rocket.readQueueNums"), viper.GetInt("rocket.writeQueueNums")
	if readQueueNums <= 0 {
		readQueueNums = 4
	}
	if writeQueueNums <= 0 {
		writeQueueNums = 4
	}
	brokerAddresses := viper.GetStringSlice("rocket.brokerAddresses")
	if len(brokerAddresses) == 0 && viper.GetString("rocket.brokerAddress") != "" {
		brokerAddresses = []string{viper.GetString("rocket.brokerAddress")}
	}
	for _, topic := range topicNames {
		if topic == "" {
			continue
		}
		if _, ok := existing[topic]; ok {
			continue
		}
		if !autoCreate {
			return fmt.Errorf("RocketMQ 主题 %q 不存在，且已关闭自动创建", topic)
		}
		if len(brokerAddresses) == 0 {
			return fmt.Errorf("RocketMQ 主题 %q 不存在，未配置 rocket.brokerAddresses", topic)
		}
		var createErr error
		for _, brokerAddress := range brokerAddresses {
			if err := rocketAdmin.CreateTopic(ctx, admin.WithTopicCreate(topic), admin.WithBrokerAddrCreate(brokerAddress), admin.WithReadQueueNums(readQueueNums), admin.WithWriteQueueNums(writeQueueNums)); err != nil {
				createErr = fmt.Errorf("Broker %s: %w", brokerAddress, err)
				break
			}
		}
		if createErr != nil {
			if queues, queryErr := rocketAdmin.FetchPublishMessageQueues(ctx, topic); queryErr == nil && len(queues) > 0 {
				existing[topic] = struct{}{}
				continue
			}
			return fmt.Errorf("创建 RocketMQ 主题 %q 失败: %w", topic, createErr)
		}
		available := false
		for i := 0; i < 10; i++ {
			if queues, queryErr := rocketAdmin.FetchPublishMessageQueues(ctx, topic); queryErr == nil && len(queues) > 0 {
				available = true
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if !available {
			return fmt.Errorf("RocketMQ 主题 %q 已创建但路由尚未可用", topic)
		}
		existing[topic] = struct{}{}
	}
	return nil
}

// 初始化
func InitROCKET(GroupName string) ROCKET {
	Endpoint := viper.GetString("rocket.endpoint")
	AccessKey := viper.GetString("rocket.accesskey")
	SecretKey := viper.GetString("rocket.secretkey")

	// 解密加密数据
	SecretKey, err := encryption.DefaultPKCS8Decrypt(SecretKey)
	if err != nil {
		log.Println("[RocketUtil] 解析错误")
	}

	// GroupName := "testgroup"
	// Endpoint := "127.0.0.1:9876"
	// AccessKey := "admin"
	// SecretKey := "password"

	// 创建生产者
	pro, err := rocketmq.NewProducer(
		producer.WithGroupName(GroupName),
		// RocketMQ的端口，如127.0.0.1:9876
		producer.WithNameServer([]string{Endpoint}),
		// 配置访问密钥
		producer.WithCredentials(primitive.Credentials{
			AccessKey: AccessKey,
			SecretKey: SecretKey,
		}),
		producer.WithRetry(2),
		//指定队列选择器
		//hashQueueSelector如果消息具有分片密钥ShardingKey，则按hash选择队列，否则按随机选择队列
		producer.WithQueueSelector(producer.NewHashQueueSelector()),
	)
	if err != nil {
		log.Println("[ROCKET] 创建生产者失败：", err)
	}

	// 创建消费者
	con, err := rocketmq.NewPushConsumer(
		// 消费模式
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithGroupName(GroupName),
		// RocketMQ的端口，如127.0.0.1:9876
		consumer.WithNameServer([]string{Endpoint}),
		// 配置访问密钥
		consumer.WithCredentials(primitive.Credentials{
			AccessKey: AccessKey,
			SecretKey: SecretKey,
		}),
		// 从Broker到Consumer的重复发送次数
		consumer.WithRetry(2),
		// Consumer对消息的重新消费次数
		consumer.WithMaxReconsumeTimes(2),
		// 如要在一个消费者组中创建多个消费者，需要指定实例名
		// consumer.WithInstance(instanceName),
		consumer.WithInstance(GroupName+"_"+fmt.Sprint(snowflakeutil.NextID())[6:]+"_"+strings.ReplaceAll(uuid.NewString(), "-", "")[20:]),
	)
	if err != nil {
		log.Println("[ROCKET] 创建消费者失败：", err)
	}

	// 注入
	rocket.Producer = pro
	rocket.Consumer = con
	return rocket
}

// 初始化 -- 参数: GroupName
func InitRocketProducer(groupname string) error {
	Endpoint := viper.GetString("rocket.endpoint")
	credentials, err := rocketCredentials()
	if err != nil {
		return err
	}

	// 创建生产者
	pro, err := rocketmq.NewProducer(
		producer.WithGroupName(groupname),
		// RocketMQ的端口，如127.0.0.1:9876
		producer.WithNameServer([]string{Endpoint}),
		// 配置访问密钥
		producer.WithCredentials(credentials),
		producer.WithRetry(2),
		// 指定队列选择器
		// hashQueueSelector如果消息具有分片密钥ShardingKey，则按hash选择队列，否则按随机选择队列
		// 顺序消息选择hash, 参考: https://blog.csdn.net/csdn_tom_168/article/details/149746622
		// https://blog.csdn.net/Christhe1st_/article/details/142285636
		producer.WithQueueSelector(producer.NewHashQueueSelector()),
	)
	if err != nil {
		return fmt.Errorf("创建生产者失败: %w", err)
	}

	// 注入
	rocket.Producer = pro
	return nil
}

// 初始化 -- 参数: GroupName
func InitRocketConsumer(groupname string) error {
	Endpoint := viper.GetString("rocket.endpoint")
	credentials, err := rocketCredentials()
	if err != nil {
		return err
	}

	// 创建消费者
	con, err := rocketmq.NewPushConsumer(
		// 消费模式
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithGroupName(groupname),
		// RocketMQ的端口，如127.0.0.1:9876
		consumer.WithNameServer([]string{Endpoint}),
		// 配置访问密钥
		consumer.WithCredentials(credentials),
		// 从Broker到Consumer的重复发送次数
		consumer.WithRetry(2),
		// Consumer对消息的重新消费次数
		consumer.WithMaxReconsumeTimes(2),
		// 如要在一个消费者组中创建多个消费者，需要指定实例名
		// consumer.WithInstance(instanceName),
		consumer.WithInstance(groupname+"_"+fmt.Sprint(snowflakeutil.NextID())[6:]+"_"+strings.ReplaceAll(uuid.NewString(), "-", "")[20:]),
	)
	if err != nil {
		return fmt.Errorf("创建消费者失败: %w", err)
	}

	// 注入
	rocket.Consumer = con
	return nil
}

// 关闭Producer
func ShutdownProducer() {
	if rocket.Producer == nil {
		return
	}
	err := rocket.Producer.Shutdown()
	if err != nil {
		log.Println("[ROCKET] 关闭生产者失败：", err)
		return
	}
}

// 开启Producer
func StartProducer() error {
	if rocket.Producer == nil {
		return fmt.Errorf("生产者未初始化")
	}
	err := rocket.Producer.Start()
	if err != nil {
		return fmt.Errorf("启动生产者失败: %w", err)
	}
	return nil
}

// 关闭Consumer
func ShutdownConsumer() {
	if rocket.Consumer == nil {
		return
	}
	err := rocket.Consumer.Shutdown()
	if err != nil {
		log.Println("[ROCKET] 关闭消费者失败：", err)
		return
	}
}

// 启动Consumer
func StartConsumer() error {
	if rocket.Consumer == nil {
		return fmt.Errorf("消费者未初始化")
	}
	err := rocket.Consumer.Start()
	if err != nil {
		return fmt.Errorf("启动消费者失败: %w", err)
	}
	return nil
}

// 发送优化配置: https://developer.aliyun.com/article/1140299
// # 优化发送消息失败 - 限制发送线程池数量/设置重入锁 - broker.conf
// #sendMessageThreadPoolNums=1
// useReentrantLockWhenPutMessage=true

// 发送消息
func Pub(ctx *gin.Context, topic string, message string) bool {

	dbName, res := ctx.Get("dbName")
	if !res {
		dbName = ""
	}

	msg := primitive.NewMessage(topic, []byte(message))

	// 1.同步消息
	_, err := rocket.Producer.SendSync(context.WithValue(context.Background(), "dbName", dbName), msg)
	if err != nil {
		log.Println("[ROCKET] 发送消息失败：", err, " msg: ", msg)
		return false
	}
	// log.Println(log.Sprintf("[ROCKET] send message success: %v", res.Status))

	return true
}

// 发送消息
func PubWithTag(ctx *gin.Context, topic string, message string, tags string) bool {

	dbName, res := ctx.Get("dbName")
	if !res {
		dbName = ""
	}

	msg := primitive.NewMessage(topic, []byte(message))
	msg.WithTag(tags)

	// 1.同步消息
	_, err := rocket.Producer.SendSync(context.WithValue(context.Background(), "dbName", dbName), msg)
	if err != nil {
		log.Println("[ROCKET] 发送消息失败：", err, " msg: ", msg)
		return false
	}

	// log.Println(log.Sprintf("[ROCKET] send message: %v", message))
	// log.Println(log.Sprintf("[ROCKET] send tags: %v", tags))
	// log.Println(log.Sprintf("[ROCKET] send message success: %v", res.Status))

	return true
}

// 发送异步消息
func PubAsync(ctx *gin.Context, topic string, message string, tags string) bool {

	dbName, res := ctx.Get("dbName")
	if !res {
		dbName = ""
	}

	msg := primitive.NewMessage(topic, []byte(message))
	msg.WithTag(tags)

	// 1.异步消息
	err := rocket.Producer.SendAsync(context.WithValue(context.Background(), "dbName", dbName),
		func(ctx context.Context, result *primitive.SendResult, e error) {
			if e != nil {
				log.Printf("[ROCKET] receive message error: %v%s%s\n", e, " msg: ", msg)
			} else {
				log.Printf("[ROCKET] send message success: result=%s\n", result.String())
			}
		},
		msg)
	if err != nil {
		log.Println("[ROCKET] 发送消息失败：", err, " msg: ", msg)
		return false
	}

	return true
}

// 获取ROCKET
func GetROCKET() *ROCKET {
	return &rocket
}

// 发送消息 - 顺序消息, 单一生产者, 单线程消费者
func PubWithShardingKey(ctx *gin.Context, topic string, message string, tags, shardingKey string) bool {

	dbName, res := ctx.Get("dbName")
	if !res {
		dbName = ""
	}

	msg := primitive.NewMessage(topic, []byte(message))
	if len(shardingKey) > 0 {
		// 为消息设置分片秘钥，保证同一个ShardingKey的消息进入同一个queue
		msg.WithShardingKey(shardingKey)
	}
	if len(tags) > 0 {
		msg.WithTag(tags)
	}

	// 1.同步消息
	_, err := rocket.Producer.SendSync(context.WithValue(context.Background(), "dbName", dbName), msg)
	if err != nil {
		log.Println("[ROCKET] 发送消息失败：", err, " msg: ", msg)
		return false
	}

	return true
}

// 批量发送消息 - 批量发送, 消息大小不能超过1M, Topic必须相同
func BatchPubSync(ctx *gin.Context, data []*primitive.Message) bool {
	// https://rocketmq.apache.org/zh/docs/4.x/producer/05message4

	dbName, res := ctx.Get("dbName")
	if !res {
		dbName = ""
	}

	for _, vTem := range reduceBatchMsgSize(data) {
		res = batchSend(dbName, vTem)
		retryTime := 1
		// 重试失败, 进入
		for !res {
			// 重试机制 - 3次
			if retryTime > 3 {
				break
			}
			time.Sleep(1 * time.Second)
			res = batchSend(dbName, vTem)
			retryTime++
		}
		if !res {
			return false
		}
	}
	return true
}

func batchSend(dbName any, data []*primitive.Message) bool {

	// 1.同步消息
	resResult, err := rocket.Producer.SendSync(context.WithValue(context.Background(), "dbName", dbName), data...)
	if err != nil {
		log.Println("[ROCKET] batch 发送消息失败：", err, " msg: ", data)
		return false
	}
	if resResult.Status != 0 {
		log.Println("[ROCKET] batch send result: ", resResult, " msg: ", data)
		return false
	}

	return true
}

// 拆分一下批量消息, 防止消息过大
func reduceBatchMsgSize(data []*primitive.Message) (result [][]*primitive.Message) {

	// 不超过50条, 直接发送
	if len(data) <= 50 {
		result = append(result, data)
		return
	}

	// 50条消息打包为一批
	dataLen := len(data)
	i := 0
	for {
		end := math.Min(float64(i+50), float64(dataLen))
		result = append(result, data[i:int64(end)])
		if end >= float64(dataLen) {
			break
		}
		i = i + 50
	}

	return
}

// 基础数据初始化
func FormatMsg(topic string, message string, tags, shardingKey string, delayLevel int) *primitive.Message {
	msg := primitive.NewMessage(topic, []byte(message))
	if len(shardingKey) > 0 {
		// 为消息设置分片秘钥，保证同一个ShardingKey的消息进入同一个queue
		msg.WithShardingKey(shardingKey)
	}
	if len(tags) > 0 {
		msg.WithTag(tags)
	}
	if delayLevel > 0 && delayLevel < 19 {
		// 1 - 18
		msg.WithDelayTimeLevel(delayLevel)
	}

	return msg
}
