package service

import (
	"dicloud/internal/business/client"
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/internal/common/grpc/tolink_grpc"
	commonmodel "dicloud/internal/common/model"
	"dicloud/models/enum"
	"dicloud/util/rocketutil"
	"fmt"
	"log"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
)

// 统计设备健康度 - 统计平台设备健康度(数量)占比
func SendEquipmentHealthRocketMsgToStatistic(ctx *gin.Context, equipmentList []model.Equipment) {

	// 多租户
	dbName, exist := ctx.Get("dbName")
	if !exist {
		dbName = ""
	}
	/* 时间戳标准 - 暂定为系统时间, 统计当前时间的设备数据 - 最新设备状态 */
	currentTime := time.Now().Unix()

	healthStandardDao := dao.NewEquipmentHealthStandardDao(ctx)
	equipmentHealthStandard, res := healthStandardDao.Find()
	if !res {
		log.Println("[Business] send EquipmentHealth RocketMsg To Statistic - standard not exist")
		return
	}

	var data = []*primitive.Message{}
	for _, equipmentTem := range equipmentList {

		// 调用gRPC, 获取实时SOH数据
		g := tolink_grpc.Gateway{
			GatewayId: equipmentTem.EquipmentCode,
		}
		res := client.GetTolinkSequence(ctx, &g)
		if res == nil {
			continue
		}

		// 封装数据
		pack := commonmodel.StatisticMQModel{
			SiteId:             equipmentTem.SiteId,
			EquipmentId:        int(equipmentTem.ID),
			Begin:              currentTime,
			End:                currentTime,
			DataSourceType:     3,
			StatisticType:      "0",
			StatisticCode:      commonmodel.StatisticHealthCode_YOU,
			StatisticName:      commonmodel.StatisticHealthCode_YOU,
			StatisticValue:     1,
			StatisticValueType: 0,
			DbName:             fmt.Sprint(dbName),
		}

		// 健康度判断判断
		for idx := range equipmentHealthStandard.LevelStandards {
			// 从高到低, 一一匹配, 匹配成功即退出
			if res.Val >= equipmentHealthStandard.LevelStandards[idx] {
				pack.StatisticCode = commonmodel.StatisticHealthCodeArr[idx]
				pack.StatisticName = commonmodel.StatisticHealthCodeArr[idx]
				break
			}
		}

		// 发送rocketmq
		tags := ""

		jsonStr, err := pack.ToJsonStr()
		if nil != err {
			log.Println("[Business] Send Rocket Equipment Health Statistic 格式转换错误!")
		}

		// rocketutil.PubWithTag(enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags)
		// 使用设备ID作为顺序消息, 保证同一个shardingKey的消息进入同一个队列
		// rocketutil.PubWithShardingKey(enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags, fmt.Sprint(pack.EquipmentId))

		data = append(data, rocketutil.FormatMsg(enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags, fmt.Sprint(pack.EquipmentId), 0))
	}

	rocketutil.BatchPubSync(ctx, data)
}
