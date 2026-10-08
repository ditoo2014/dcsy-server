package service

import (
	"dicloud/internal/business/client"
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/internal/common/dto"
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

func StatisticDurationData(ctx *gin.Context, equipmentList []model.Equipment, beginTime, endTime int64) bool {
	// 传入数据 - 默认单位: 秒

	if beginTime > endTime {
		log.Println("[Business-Schedule] statistic duration data time callback")
		return false
	}

	// 获取当前时间的年月日时分秒
	t := time.Unix(beginTime, 0) // 第二个参数是纳秒，这里设置为0因为用的是秒数时间戳
	year, month, day := t.Date()
	hour, _, _ := t.Clock()

	tEnd := time.Unix(endTime, 0)
	endYear, endMonth, endDay := tEnd.Date()
	endHour, _, _ := tEnd.Clock()

	// 将分钟和秒设置为0，即整点时间
	truncatedTime := time.Date(year, month, day, hour, 0, 0, 0, t.Location())
	actualEndTime := time.Date(endYear, endMonth, endDay, endHour, 0, 0, 0, tEnd.Location())

	// 从上一个小时开始 - 减去1秒, 统计上一个小时内最新数据
	countTime := truncatedTime.Unix() - 1
	// deadLineTime := actualEndTime.Unix() + 60*60
	deadLineTime := actualEndTime.Unix() + 24*60*60

	// 多租户
	dbName, exist := ctx.Get("dbName")
	if !exist {
		dbName = ""
	}

	// 健康度标准
	healthStandardDao := dao.NewEquipmentHealthStandardDao(ctx)
	equipmentHealthStandard, res := healthStandardDao.Find()
	pointResolver := newStatisticDataPointResolver(ctx)
	allPublished := true

	// 开始计算
	for {
		// 如果计算时间有效
		if countTime > deadLineTime {
			// 统计完成, 跳出循环
			break
		}

		log.Printf("[Business-Schedule] statistic before %s last data\n", time.Unix(countTime, 0).Format("2006-01-02 15:04:05"))
		for _, equipmentTem := range equipmentList {
			// 计算
			// 轮询设备

			// 普通数据累计统计
			if !normalDataCumulative(ctx, equipmentTem, countTime, dbName, pointResolver) {
				allPublished = false
			}
			// 健康度数据统计
			if res {
				if !healthDataCumulative(ctx, equipmentTem, countTime, dbName, &equipmentHealthStandard, pointResolver) {
					allPublished = false
				}
			}
			time.Sleep(30 * time.Millisecond)
		}
		time.Sleep(100 * time.Millisecond)

		// 累加1小时 - 计算下一个周期
		// countTime = countTime + 60*60

		// 累加1天 - 计算下一个周期
		countTime = countTime + 24*60*60
	}
	return allPublished
}

// StatisticCurrentDayData 每小时刷新当天的累计数据和健康度数据。
// 只读取当前最新值，避免为当天重复回算整段历史数据。
func StatisticCurrentDayData(ctx *gin.Context, equipmentList []model.Equipment, currentTime int64) bool {
	if len(equipmentList) == 0 || currentTime <= 0 {
		return false
	}

	dbName, exist := ctx.Get("dbName")
	if !exist {
		dbName = ""
	}

	healthStandardDao := dao.NewEquipmentHealthStandardDao(ctx)
	equipmentHealthStandard, res := healthStandardDao.Find()
	pointResolver := newStatisticDataPointResolver(ctx)
	allPublished := true
	for _, equipmentTem := range equipmentList {
		if !normalDataCumulative(ctx, equipmentTem, currentTime, dbName, pointResolver) {
			allPublished = false
		}
		if res {
			if !healthDataCumulative(ctx, equipmentTem, currentTime, dbName, &equipmentHealthStandard, pointResolver) {
				allPublished = false
			}
		}
	}
	return allPublished
}

func normalDataCumulative(ctx *gin.Context, equipmentTem model.Equipment, countTime int64, dbName interface{}, pointResolver *statisticDataPointResolver) bool {

	normalDataMsgList := []*primitive.Message{}
	legacyPoints := make([]statisticDataPoint, 0, len(dto.DP_STATISTIC_DATA_LIST))
	for _, dataID := range dto.DP_STATISTIC_DATA_LIST {
		legacyPoints = append(legacyPoints, statisticDataPoint{DataID: dataID, Code: dto.DP_STATISTIC_CODE_MAP[dataID]})
	}
	dataPoints := pointResolver.resolve(equipmentTem.EquipmentCode, legacyPoints)
	// 每种统计值只写一条，多个地址候选中选择截止时间前最新的有效数据。
	for _, legacyPoint := range legacyPoints {
		res := latestStatisticSequenceData(ctx, equipmentTem.EquipmentCode, dataPoints, legacyPoint.Code, countTime, dbName)
		if res == nil {
			continue
		}

		// 实际数据时间 -> 统计时间
		statisticTime := res.CreateTime
		if countTime-res.CreateTime > 60*60 {
			// 如果时间超过一小时
			statisticTime = countTime
		}
		// 发送统计数据
		// 封装数据
		pack := commonmodel.StatisticMQModel{
			SiteId:             equipmentTem.SiteId,
			EquipmentId:        int(equipmentTem.ID),
			Begin:              statisticTime, // 统计时间
			End:                statisticTime, //
			DataSourceType:     3,             // 数据源
			StatisticType:      "0",           // 简单累加
			StatisticCode:      legacyPoint.Code,
			StatisticName:      legacyPoint.Code,
			StatisticValue:     res.Val, // 统计值
			StatisticValueType: 0,       // 累计值统计
			DbName:             fmt.Sprint(dbName),
		}

		// 发送rocketmq
		tags := ""

		jsonStr, err := pack.ToJsonStr()
		if nil != err {
			log.Println("[Business] Send Rocket Equipment last data Statistic 格式转换错误!")
			return false
		}

		// rocketutil.PubWithTag(enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags)

		// 如果需要有增量数据, 统计
		// tags = enum.TAG_STATISTIC_NORMAL
		// rocketutil.PubWithTag(enum.XC_TRIGGER_STATISTIC_QUEUE, jsonStr, tags)

		// 使用设备ID作为顺序消息, 保证同一个shardingKey的消息进入同一个队列
		normalDataMsgList = append(normalDataMsgList, rocketutil.FormatMsg(enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags, fmt.Sprint(pack.EquipmentId), 0))
	}

	if len(normalDataMsgList) > 0 {
		sendRes := rocketutil.BatchPubSync(ctx, normalDataMsgList)
		for retryTime := 1; !sendRes && retryTime <= 3; retryTime++ {
			time.Sleep(time.Second)
			sendRes = rocketutil.BatchPubSync(ctx, normalDataMsgList)
		}
		if !sendRes {
			log.Printf("[Business-Statistic] cumulative statistic publish failed equipmentId=%d gateway=%s messageCount=%d", equipmentTem.ID, equipmentTem.EquipmentCode, len(normalDataMsgList))
			return false
		}
	}
	return true
}

func latestStatisticSequenceData(ctx *gin.Context, gatewayID string, points []statisticDataPoint, code string, countTime int64, dbName interface{}) *tolink_grpc.SequenceData {
	var latest *tolink_grpc.SequenceData
	for _, point := range points {
		if point.Code != code {
			continue
		}
		res := client.GetTolinkLastSequenceData(ctx, &tolink_grpc.GetDataDto{
			GatewayId: gatewayID,
			DataId:    point.DataID,
			CurTime:   countTime,
			DbName:    fmt.Sprint(dbName),
		})
		if res == nil || (latest != nil && res.CreateTime <= latest.CreateTime) {
			continue
		}
		latest = res
	}
	return latest
}

func healthDataCumulative(ctx *gin.Context, equipmentTem model.Equipment, countTime int64, dbName interface{}, equipmentHealthStandard *model.EquipmentHealthStandard, pointResolver *statisticDataPointResolver) bool {

	// 调用gRPC, 获取统计周期时间数据
	legacyPoint := statisticDataPoint{DataID: dto.DP_DEFAULT_SOH_DATAID, Code: commonmodel.StatisticHealthCode_YOU}
	resolvedPoints := pointResolver.resolve(equipmentTem.EquipmentCode, []statisticDataPoint{legacyPoint})
	if len(resolvedPoints) == 0 {
		return true
	}
	res := latestStatisticSequenceData(ctx, equipmentTem.EquipmentCode, resolvedPoints, legacyPoint.Code, countTime, dbName)
	if res == nil {
		return true
	}

	// 实际数据时间 -> 统计时间
	statisticTime := res.CreateTime
	if countTime-res.CreateTime > 60*60 {
		// 如果时间超过一小时
		statisticTime = countTime
	}

	// 封装数据
	pack := commonmodel.StatisticMQModel{
		SiteId:             equipmentTem.SiteId,
		EquipmentId:        int(equipmentTem.ID),
		Begin:              statisticTime,
		End:                statisticTime,
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
		log.Println("[Business] Send Rocket Equipment Health duration Statistic 格式转换错误!")
		return false
	}

	// rocketutil.PubWithTag(enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags)
	// 使用设备ID作为顺序消息, 保证同一个shardingKey的消息进入同一个队列
	sendRes := rocketutil.PubWithShardingKey(ctx, enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags, fmt.Sprint(pack.EquipmentId))
	retryTime := 1
	for !sendRes {
		// 重试3次
		if retryTime > 3 {
			break
		}
		time.Sleep(1 * time.Second)
		sendRes = rocketutil.PubWithShardingKey(ctx, enum.XC_BUSINESS_STATISTIC_QUEUE, jsonStr, tags, fmt.Sprint(pack.EquipmentId))
		retryTime++
	}
	if !sendRes {
		log.Printf("[Business-Statistic] health statistic publish failed equipmentId=%d gateway=%s", equipmentTem.ID, equipmentTem.EquipmentCode)
	}
	return sendRes
}
