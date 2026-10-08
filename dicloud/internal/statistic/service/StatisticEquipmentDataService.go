package service

import (
	"dicloud/internal/common/model"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	muxEquipment sync.Mutex
)

func StatisticEquipmentData(ctx *gin.Context, ststMsg model.StatisticMQModel) bool {

	// 加锁, 直到一条数据完全处理完成, 才释放
	// 如果启用集群模式, 需要MVCC, TODO: 通过Redis实现简单加锁, 解锁 - BaseLockMsg
	muxEquipment.Lock()
	defer muxEquipment.Unlock()

	// 转为存储格式
	dataTem := model.StatisticData{
		SiteId:             ststMsg.SiteId,
		EquipmentId:        ststMsg.EquipmentId,
		CorpId:             ststMsg.CorpId,
		DataSourceType:     ststMsg.DataSourceType,
		StatisticCode:      ststMsg.StatisticCode,
		StatisticName:      ststMsg.StatisticName,
		StatisticAlarmType: ststMsg.StatisticAlarmType,
		StartStatisticTime: ststMsg.Begin,
		EndStatisticTime:   ststMsg.End,
		EndValue:           ststMsg.StatisticValue,
		StatisticValue:     ststMsg.StatisticValue,
		StatisticValueType: ststMsg.StatisticValueType,
		CreateTime:         time.Now().Unix(),
	}

	if dataTem.EndStatisticTime == 0 {
		dataTem.EndStatisticTime = dataTem.StartStatisticTime
	}
	if len(dataTem.StatisticName) == 0 {
		dataTem.StatisticName = dataTem.StatisticCode
	}

	switch dataTem.StatisticValueType {
	case 0:
		// 累计设备上线数, 数据处理 - 只需要处理累计值, 不生成增量值, 不关注且不需要
		// 这里只生成累计值数据, 列举一下: 设备数量, 设备健康度, 循环次数, 累计充电量, 累计放电量
		return dealCumulativeEquipmentData(ctx, dataTem)
	default:
		log.Println("[STATISTIC SERVER] illegal msg unknown value type:", dataTem)
		return false
	}
}
