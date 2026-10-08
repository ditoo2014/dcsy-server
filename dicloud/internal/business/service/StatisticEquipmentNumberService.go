package service

import (
	"dicloud/internal/business/model"
	commonmode "dicloud/internal/common/model"
	"dicloud/models/enum"
	"dicloud/util/rocketutil"
	"fmt"
	"log"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
)

var (
	StatisticNumberCode = "电池数量"
)

func SendEquipmentNumberRocketMsgToStatistic(ctx *gin.Context, equipmentList []model.Equipment) {

	dbName, _ := ctx.Get("dbName")
	currentTime := time.Now().Unix()

	var data = []*primitive.Message{}

	for _, equipmentTem := range equipmentList {

		pack := commonmode.StatisticMQModel{
			SiteId:             equipmentTem.SiteId,
			EquipmentId:        int(equipmentTem.ID),
			Begin:              currentTime,
			End:                currentTime,
			DataSourceType:     3,
			StatisticType:      "0",
			StatisticCode:      StatisticNumberCode,
			StatisticName:      StatisticNumberCode,
			StatisticValue:     1,
			StatisticValueType: 0,
			DbName:             fmt.Sprint(dbName),
		}

		// 发送rocketmq
		tags := ""
		// types := strings.Split(config.StatisticConfig.StatisticType, ",")
		// for _, x := range types {
		// 	if x == "" {
		// 		continue
		// 	}
		// 	tags += enum.TiggerStatisticMap[x] + ","
		// }

		jsonStr, err := pack.ToJsonStr()
		if nil != err {
			log.Println("[Business] Send Rocket Equipment Number Statistic 格式转换错误!")
		}

		// rocketutil.PubWithTag(enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags)
		// 使用设备ID作为顺序消息, 保证同一个shardingKey的消息进入同一个队列
		// rocketutil.PubWithShardingKey(enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags, fmt.Sprint(pack.EquipmentId))

		data = append(data, rocketutil.FormatMsg(enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags, fmt.Sprint(pack.EquipmentId), 0))
	}

	rocketutil.BatchPubSync(ctx, data)
}
