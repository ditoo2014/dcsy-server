package service

import (
	"dicloud/internal/common/model"
	"dicloud/models/enum"
	"dicloud/util/rocketutil"
	snowflakeutil "dicloud/util/snowflake"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

func SendGeneralTriggerMsg(ctx *gin.Context, config *model.GeneralTriggerConfig, marktriggerConfig *model.GeneralMarktrigger, value any, begin, end int64) {

	switch config.TriggerType {
	case 0:
		sendGeneralTriggerAlarm(ctx, config, marktriggerConfig, value, begin, end)
	case 1:
		sendGeneralTriggerStatistic(ctx, config, marktriggerConfig, value, begin, end)
	case 2:
		sendGeneralTriggerAlarm(ctx, config, marktriggerConfig, value, begin, end)
		sendGeneralTriggerStatistic(ctx, config, marktriggerConfig, value, begin, end)

	case 3:
		sendGeneralPrometheus(ctx, config, marktriggerConfig, value, begin, end)

	default:
		fmt.Println("无效触发器", config.ID, config.Code)
		return
	}

}

func sendGeneralTriggerAlarm(ctx *gin.Context, config *model.GeneralTriggerConfig, marktriggerConfig *model.GeneralMarktrigger, value any, begin, end int64) {

	if config.AlarmUser.AlarmCode == "" {
		config.AlarmUser.AlarmCode = config.Code
	}

	dbName, res := ctx.Get("dbName")
	if !res {
		dbName = ""
	}
	pack := model.AlarmMQModel{
		SiteId:       marktriggerConfig.SiteId,
		EquipmentId:  marktriggerConfig.EquipmentId,
		Begin:        begin,
		End:          end,
		AlarmType:    config.AlarmUser.AlarmType,
		AlarmStage:   config.AlarmUser.AlarmStage,
		AlarmCode:    config.AlarmUser.AlarmCode,
		AlarmChannel: config.AlarmUser.AlarmChannel,
		AlarmUsers:   config.AlarmUser.AlarmUsers,
		AlarmValue:   value,
		DbName:       fmt.Sprint(dbName),
	}

	jsonStr, err := pack.ToJsonStr()
	if nil != err {
		fmt.Println("[Trigger] Send Rocket Alarm 格式转换错误!")
	}

	// 发送rocketmq

	// rocketutil.PubWithTag(enum.XC_TRIGGER_ALARM_QUEUE, jsonStr, tags)
	// 使用设备ID作为顺序消息, 保证同一个shardingKey的消息进入同一个队列
	res = rocketutil.PubWithShardingKey(ctx, enum.XC_TRIGGER_ALARM_QUEUE, jsonStr, "", fmt.Sprint(pack.EquipmentId))
	retryTime := 1
	// 如果没有发送成功
	for !res {
		// 重试3次
		if retryTime > 3 {
			break
		}
		time.Sleep(1 * time.Second)
		res = rocketutil.PubWithShardingKey(ctx, enum.XC_TRIGGER_ALARM_QUEUE, jsonStr, "", fmt.Sprint(pack.EquipmentId))
		retryTime++
	}
}

func sendGeneralTriggerStatistic(ctx *gin.Context, config *model.GeneralTriggerConfig, marktriggerConfig *model.GeneralMarktrigger, value any, begin, end int64) {

	if config.StatisticConfig.StatisticCode == "" {
		config.StatisticConfig.StatisticCode = config.Code
	}

	dbName, res := ctx.Get("dbName")
	if !res {
		dbName = ""
	}
	pack := model.StatisticMQModel{
		SiteId:             marktriggerConfig.SiteId,
		EquipmentId:        marktriggerConfig.EquipmentId,
		Begin:              begin,
		End:                end,
		DataSourceType:     config.StatisticConfig.DataSourceType,
		StatisticType:      config.StatisticConfig.StatisticType,
		StatisticCode:      config.StatisticConfig.StatisticCode,
		StatisticName:      config.StatisticConfig.StatisticName,
		StatisticValue:     config.StatisticConfig.StatisticValue,
		StatisticValueType: config.StatisticConfig.StatisticValueType,
		TriggerValue:       value,
		DbName:             fmt.Sprint(dbName),
	}
	// 是报警的触发条数统计
	if strings.Contains(config.StatisticConfig.StatisticType, "1") {
		pack.StatisticAlarmType = config.AlarmUser.AlarmType
		if pack.StatisticValue == 0 {
			// 默认统计一条报警
			pack.StatisticValue = 1
		}
	}
	// 默认触发值统计, 可选覆盖原始值统计
	if config.StatisticConfig.UseStatisticValue {
		tem, err := strconv.ParseFloat(fmt.Sprint(config.StatisticConfig.StatisticValue), 64)
		if err != nil {
			log.Println("[SendRocketMsg] sendStatistic err: ", err)
		}
		pack.StatisticValue = tem
	}

	// 发送rocketmq
	jsonStr, err := pack.ToJsonStr()
	if nil != err {
		fmt.Println("[Trigger] Send Rocket Statistic 格式转换错误!")
	}

	// rocketutil.PubWithTag(enum.XC_TRIGGER_STATISTIC_QUEUE, jsonStr, tags)
	// 使用设备ID作为顺序消息, 保证同一个shardingKey的消息进入同一个队列
	res = rocketutil.PubWithShardingKey(ctx, enum.XC_TRIGGER_STATISTIC_QUEUE, jsonStr, "", fmt.Sprint(pack.EquipmentId))
	retryTime := 1
	// 如果没有发送成功
	for !res {
		// 重试3次
		if retryTime > 3 {
			break
		}
		time.Sleep(1 * time.Second)
		res = rocketutil.PubWithShardingKey(ctx, enum.XC_TRIGGER_STATISTIC_QUEUE, jsonStr, "", fmt.Sprint(pack.EquipmentId))
		retryTime++
	}
}

// 参考:
// https://help.aliyun.com/zh/prometheus/use-cases/use-pushgateway-to-push-metric-data#df5f0bb3ba5se
// https://cloud.baidu.com/doc/CProm/s/ymad9owev
// https://blog.csdn.net/weixin_31706903/article/details/141538033
// https://github.com/prometheus/client_golang/blob/main/examples/exemplars/main.go
// TODO: 可前置到解析消息之前, 触发逻辑不必通过代码, 直接交给alertmanager管理
func sendGeneralPrometheus(ctx *gin.Context, config *model.GeneralTriggerConfig, marktriggerConfig *model.GeneralMarktrigger, value any, begin, end int64) {

	if config.AlarmUser.AlarmCode == "" {
		config.AlarmUser.AlarmCode = config.Code
	}

	/**
	=========================
		需部署PushGateway
		发送到PushGateway
	=========================
	*/

	// 带标签的gauge
	metricGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trigger_to_pushgateway_alertmanager_alarm",
		Help: "信创报警消息",
	}, []string{"siteId", "equipmentId", "coding", "begin", "end", "alarmId"})

	// 转为Float64, alertmanager编写规则判断
	valTem, err := strconv.ParseFloat(fmt.Sprint(value), 64)
	if nil != err {
		fmt.Println("[Trigger] Send Prometheus Alarm 格式转换错误!")
	}

	// 写入标签值, alermanager可以接受数据, 进行匹配
	// 用于匹配报警规则
	dbName, res := ctx.Get("dbName")
	if !res {
		dbName = ""
	}
	metricGauge.With(prometheus.Labels{
		"siteId":      fmt.Sprint(marktriggerConfig.SiteId),
		"equipmentId": fmt.Sprint(marktriggerConfig.EquipmentId),
		"coding":      config.AlarmUser.AlarmCode,
		"begin":       fmt.Sprint(begin),
		"end":         fmt.Sprint(end),
		"alarmId":     fmt.Sprint(snowflakeutil.NextID()),
		"dbName":      fmt.Sprint(dbName)}).Set(valTem)

	url := "localhost:9091/"
	pusher := push.New(url, "alarm_gateway").
		Collector(metricGauge).Client(http.DefaultClient).
		// 区分报警触发器记录, 在pushgateway中追溯
		Grouping("site", fmt.Sprint(marktriggerConfig.SiteId)).Grouping("equipment", fmt.Sprint(marktriggerConfig.EquipmentId)).Grouping("code", config.AlarmUser.AlarmCode).
		Format(expfmt.NewFormat(expfmt.TypeProtoDelim))
		// 规则参考: 氧气浓度规则
		//  min({job="alarm_gateway", code="氧气浓度"}) by(siteId,equipmentId,coding) < 90
	if err := pusher.Push(); err != nil {
		fmt.Println("[TRIGGER] Could not push metric Gauge to PushGateway: ", err)
	} else {
		fmt.Println("[TRIGGER] Push PushGateway success.")
	}

}
