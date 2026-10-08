package enum

// MQTT 服务使用的 RocketMQ 生产者和命令消费者组。
const (
	GROUP_MQTT_PRODUCER = "mqtt_producer"
	GROUP_MQTT_CONSUMER = "mqtt_consumer"
)

// MQTT 服务只向电池处理队列转发数据，并监听自己的下行命令队列。
const (
	XC_BATTERY_DEAL_QUEUE = "xc_battery_deal_msg"
	XC_MQTT_CMD_QUEUE     = "xc_mqtt_cmd_msg"
)

// TAG_MQTT_BATTERY_TTDC 标识发往 deal 的 TTDC 锂硫电池消息。
const TAG_MQTT_BATTERY_TTDC = "battery_ttdc"

const (
	TAG_MQTT_CMD  = "mqttcmd"
	TAG_MQTT_CMDO = "mqttcmdo"
)
