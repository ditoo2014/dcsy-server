package service

import (
	"dicloud/internal/common/model"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	muxNormal sync.Mutex
)

func StatisticNormalData(ctx *gin.Context, ststMsg model.StatisticMQModel) {

	// 加锁, 直到一条数据完全处理完成, 才释放
	// 如果启用集群模式, 需要MVCC, TODO: 通过Redis实现简单加锁, 解锁 - BaseLockMsg
	muxNormal.Lock()
	defer muxNormal.Unlock()

	// 转为存储格式
	dataTem := model.StatisticData{
		SiteId:             ststMsg.SiteId,
		EquipmentId:        ststMsg.EquipmentId,
		CorpId:             ststMsg.CorpId,
		DataSourceType:     ststMsg.DataSourceType,
		StatisticCode:      ststMsg.StatisticCode,
		StatisticName:      ststMsg.StatisticName,
		StartStatisticTime: ststMsg.Begin,
		EndStatisticTime:   ststMsg.End,
		EndValue:           ststMsg.StatisticValue,
		StatisticValue:     ststMsg.StatisticValue,
		StatisticValueType: ststMsg.StatisticValueType,
		CreateTime:         time.Now().Unix(),
	}
	// 如果统计值 - 手动矫正值为0, 则计算触发值
	if dataTem.StatisticValue == 0 {
		tem, err := strconv.ParseFloat(fmt.Sprint(ststMsg.TriggerValue), 64)
		if err != nil {
			log.Println("[StatisticAlarmData] statistic triggerValue err: ", err)
		}
		dataTem.StatisticValue = tem
	}
	if dataTem.EndStatisticTime == 0 {
		dataTem.EndStatisticTime = dataTem.StartStatisticTime
	}
	if len(dataTem.StatisticName) == 0 {
		dataTem.StatisticName = dataTem.StatisticCode
	}

	switch dataTem.StatisticValueType {
	case 0:
		// 累计数据处理
		dealCumulativeData(ctx, dataTem)
	case 1:
		// 增量数据处理
		dealIncrementData(ctx, dataTem)
	default:
		log.Println("[STATISTIC SERVER] illegal msg unknown value type:", dataTem)
	}

}
