package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	"log"

	"github.com/gin-gonic/gin"
)

// 部门二次数据生成 - 通过设备数据, 生成部门数据
func statisticCorpAlarmData(ctx *gin.Context, dataTem model.StatisticData, dateTypeList []map[string]string) {

	// 值类型
	dataTem.StatisticValueType = 0
	// 累加起来
	for _, v := range dateTypeList {
		// 粒度
		dataTem.MarkDate = v["dateMark"]
		dataTem.StatisticDateType = v["dateType"]
		doStatisticCorpAlarm(ctx, dataTem)
	}

	// 值类型
	dataTem.StatisticValueType = 1
	// 累加起来
	for _, v := range dateTypeList {
		// 粒度
		dataTem.MarkDate = v["dateMark"]
		dataTem.StatisticDateType = v["dateType"]
		doStatisticCorpAlarm(ctx, dataTem)
	}
}

// 累计所有设备 累计数据, 存为当前日期部门累计
// 累计所有设备 增量数据, 存为当前日期部门增量
func doStatisticCorpAlarm(ctx *gin.Context, sdTem model.StatisticData) {

	statisticDao := dao.NewStatisticDataDao(ctx)

	// 查询设备数据
	sdTem.TargetType = 0
	dataList, res := statisticDao.FindAlarmByCorpId(sdTem)
	if !res {
		// 查询部门数据失败, 有设备的数据进来, 反而没有部门的数据, 记录一下
		log.Println("[STATISTIC] statistic corp data fail, fail to find data ", sdTem)
		log.Println("[STATISTIC] target type: ", sdTem.TargetType)
		log.Println("[STATISTIC] date type: ", sdTem.StatisticDateType)
		log.Println("[STATISTIC] value type: ", sdTem.StatisticValueType)
		return
	}

	// 部门操作
	sdTem.TargetType = 2
	// 累计数值
	var countAll float64
	for _, v := range dataList {
		countAll = countAll + v.StatisticValue
	}
	sdTem.StatisticValue = countAll
	sdTem.StartValue = 0
	sdTem.EndValue = 0

	// 存库, 或者更新
	dataTem, res := statisticDao.ExistCorpAlarmDateMark(sdTem)
	if !res {
		statisticDao.Create(sdTem)
	} else {
		sdTem.ID = dataTem.ID
		statisticDao.Update(sdTem)
	}
}
