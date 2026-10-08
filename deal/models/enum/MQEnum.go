package enum

// deal 服务使用的 RocketMQ 生产者和消费者组。
const (
	GROUP_DEAL_PRODUCER = "deal_producer"
	GROUP_DEAL_CONSUMER = "deal_consumer"
)

// deal 只消费 TTDC 电池消息并向触发器发送解析结果。
const (
	XC_BATTERY_DEAL_QUEUE = "xc_battery_deal_msg"
	XC_DEAL_TRIGGER_QUEUE = "xc_deal_trigger_msg"
)

// 消息标签分别标识 TTDC 原始消息和解析后的电池点位。
const (
	TAG_MQTT_BATTERY_TTDC = "battery_ttdc"
	TAG_TRIGGER_BATTERY   = "trigger_battery" // 电池点位，由 trigger 根据设备绑定选择数据源
)
