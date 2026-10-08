package enum

// mqtt 监听 topic
const (
	MQTT_TOPIC_NORMAL                 = "/dtdp/data/"
	MQTT_TOPIC_NORMAL_LIVE            = "/dtdp/live/"
	MQTT_TOPIC_NORMALSTR              = "/dtdp/dstr/"
	MQTT_TOPIC_DIGATE                 = "digate/up/data/"
	MQTT_TOPIC_FBOX                   = "fbox/data/"
	MQTT_TOPIC_FBOX_LIVE              = "fbox/live/"
	MQTT_TOPIC_X                      = "/ge/data/"
	MQTT_TOPIC_BATTERY_LITHIUM_SULFUR = "/DCBMS4G/LithiumSulfurBattery/PUB/"
)

// groupname
const (
	GROUP_TRIGGER_PRODUCER   = "trigger_producer"
	GROUP_TRIGGER_CONSUMER   = "trigger_consumer"
	GROUP_ALARM_CONSUMER     = "alarm_consumer"
	GROUP_STATISTIC_CONSUMER = "statistic_consumer"
	GROUP_BUSINESS_PRODUCER  = "business_producer"
)

// rocketmq topic: 信创_producer_consumer_msg_queue
const (
	XC_BATTERY_DEAL_QUEUE       = "xc_battery_deal_msg"
	XC_DEAL_TRIGGER_QUEUE       = "xc_deal_trigger_msg"
	XC_TRIGGER_ALARM_QUEUE      = "xc_trigger_alarm_msg"
	XC_TRIGGER_STATISTIC_QUEUE  = "xc_trigger_statistic_msg"
	XC_MQTT_CMD_QUEUE           = "xc_mqtt_cmd_msg"
	XC_BUSINESS_STATISTIC_QUEUE = "xc_business_statistic_msg"
)

// mqtt to deal tag
const (
	TAG_MQTT_NORMAL       = "normal"
	TAG_MQTT_NORMAL_LIVE  = "normal_live"
	TAG_MQTT_NORMALSTR    = "normalstr"
	TAG_MQTT_DIGATE       = "digate"
	TAG_MQTT_FBOX         = "fbox"
	TAG_MQTT_FBOX_LIVE    = "fbox_live"
	TAG_MQTT_X            = "x"
	TAG_MQTT_ZYDP         = "zydp" // udp ?
	TAG_MQTT_BATTERY_TTDC = "battery_ttdc"
)

const (
	TAG_MQTT_CMD  = "mqttcmd"
	TAG_MQTT_CMDO = "mqttcmdo"
)

// deal to trigger tag
const (
	// 示例
	TAG_TRIGGER_DOR     = "trigger_dor"     // gateway dataId value 格式数据
	TAG_TRIGGER_DR      = "trigger_dr"      // gateway{datastr} 格式数据
	TAG_TRIGGER_XR      = "trigger_xr"      // x 格式数据
	TAG_TRIGGER_BATTERY = "trigger_battery" // 电池点位，由 trigger 根据设备绑定选择数据源
)

// trigger to alarm tag
const (
	TAG_ALARM_CHANNEL_SHORTMSG = "0" // 短信报警
)

// trigger to alarm tag
const (
	TAG_STATISTIC_NORMAL = "0"
	TAG_STATISTIC_ALARM  = "1"
)
