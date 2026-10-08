package mqttutil

import (
	"log"
	"mqtt/util/encryption"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
)

var mqtt MQTTClient

// add 2024.12.05.
// 带密码的连接,不影响以前的服务;因为以前的服务允许匿名连接,没有禁用
var MqttAuth MQTTClient

/* 客户端结构体 */
type MQTTClient struct {
	Client   MQTT.Client
	Options  *MQTT.ClientOptions // 连接参数
	SubInfos []SubInfo           // 订阅参数
}

type SubInfo struct {
	Topic       string                  // 订阅Topic
	HandlerFunc func(topic, msg string) // 处理方法
}

/* 发布 */
func (m *MQTTClient) Pub(topic, msg string) {

	m.Client.Publish(topic, 2, false, msg)
}

func (m *MQTTClient) PubWithQos(topic, msg string, qos int) {

	m.Client.Publish(topic, byte(qos), false, msg)
}

/* MQTT订阅 */
func (m *MQTTClient) Sub(topic string, handler func(topic, msg string)) {

	// 重连时, 重新初始化之后的topic/handler
	var info = SubInfo{Topic: topic, HandlerFunc: handler}
	m.SubInfos = append(m.SubInfos, info)
	// m.Topic = topic
	// m.HandlerFunc = handler

	var callback MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		handler(string(msg.Topic()), string(msg.Payload()))
	}

	if token := m.Client.Subscribe(topic, 0, callback); token.WaitTimeout(5*time.Second) && token.Error() != nil {
		log.Println("[MQTTUTIL] subscribe fail: ", token.Error())
	}
}

//###########################################
//## 认证客户端
//###########################################

/* MQTT初始化 */
func InitAuthMqtt() *MQTTClient {

	url := viper.GetString("mqtt.url")
	username := viper.GetString("mqtt.username")
	password := viper.GetString("mqtt.password")

	if len(password) == 0 {
		log.Println("[MQTTUTIL] InitAuthMqtt 请配置密码")
		return nil
	}

	// 解密加密数据
	password, err := encryption.DefaultPKCS8Decrypt(password)
	if err != nil {
		log.Println("[MQTTUTIL] 解析错误")
		return nil
	}

	opts := MQTT.NewClientOptions()

	opts.AddBroker(url)
	opts.SetUsername(username)
	opts.SetPassword(password)

	// 连接丢失重连
	var connectLostHandler MQTT.ConnectionLostHandler = func(client MQTT.Client, err error) {
		log.Printf("Connect lost: %v\n", err)
		//这里可以做自定义重连机制
		for {
			log.Println("[MQTTUTIL] AuthMQTT Try Reconnect...")
			err := ReconnectClient(&MqttAuth)
			if err == nil {
				break
			}
			time.Sleep(30 * time.Second)
		}
	}
	opts.OnConnectionLost = connectLostHandler
	MqttAuth.Options = opts

	MqttAuth.Client = MQTT.NewClient(opts)

	if token := MqttAuth.Client.Connect(); token.Wait() && token.Error() != nil {
		// panic(token.Error())
		return &MqttAuth
	}

	return &MqttAuth
}

/* 客户端断连后自动重连, 并重新订阅主题 */
func ReconnectClient(mqttTem *MQTTClient) error {

	// 老数据, mqttTem
	if mqttTem.Client == nil {
		// 使用就参数, 重建客户端
		c := MQTT.NewClient(mqttTem.Options)
		if token := c.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}

		// 重新订阅主题
		mqttTem.Client = c
		// mqttTem.Sub(mqttTem.Topic, mqttTem.HandlerFunc)
		if len(mqttTem.SubInfos) > 0 {
			for idx := range mqttTem.SubInfos {
				mqttTem.Sub(mqttTem.SubInfos[idx].Topic, mqttTem.SubInfos[idx].HandlerFunc)
			}
		}
	} else {
		// 断开连接之后重连
		mqttTem.Client.Disconnect(250)
		if token := mqttTem.Client.Connect(); token.Wait() && token.Error() != nil {
			return token.Error()
		}
		// 重新订阅主题
		// mqttTem.Sub(mqttTem.Topic, mqttTem.HandlerFunc)
		if len(mqttTem.SubInfos) > 0 {
			for idx := range mqttTem.SubInfos {
				mqttTem.Sub(mqttTem.SubInfos[idx].Topic, mqttTem.SubInfos[idx].HandlerFunc)
			}
		}
	}

	log.Println("[MQTTUTIL] MQTT Reconnected! ")
	return nil
}

/* MQTT关闭 */
func CloseAuthMqtt() {
	if MqttAuth.Client != nil {
		MqttAuth.Client.Disconnect(250)
	}
}

/* 获取操作体 */
func GetAuthMqtt() MQTTClient {
	return MqttAuth
}

/* 多租户MQTT初始化 */
func InitMultiUserMqtt() *MQTTClient {

	url := viper.GetString("mqtt.url")
	username := viper.GetString("mqtt.username")
	password := viper.GetString("mqtt.password")

	if len(password) == 0 {
		log.Println("[MQTTUTIL] InitAuthMqtt 请配置密码")
		return nil
	}

	// 解密加密数据
	password, err := encryption.DefaultPKCS8Decrypt(password)
	if err != nil {
		log.Println("[MQTTUTIL] 解析错误")
		return nil
	}

	opts := MQTT.NewClientOptions()

	opts.AddBroker(url)
	opts.SetUsername(username)
	opts.SetPassword(password)

	// 连接丢失重连
	var connectLostHandler MQTT.ConnectionLostHandler = func(client MQTT.Client, err error) {
		log.Printf("Connect lost: %v\n", err)
		//这里可以做自定义重连机制
		for {
			log.Println("[MQTTUTIL] AuthMQTT Try Reconnect...")
			err := ReconnectClient(&MqttAuth)
			if err == nil {
				break
			}
			time.Sleep(30 * time.Second)
		}
	}
	opts.OnConnectionLost = connectLostHandler
	MqttAuth.Options = opts

	MqttAuth.Client = MQTT.NewClient(opts)

	if token := MqttAuth.Client.Connect(); token.Wait() && token.Error() != nil {
		// panic(token.Error())
		return &MqttAuth
	}

	return &MqttAuth
}
