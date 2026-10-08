package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	"log"

	"github.com/gin-gonic/gin"
)

// 站点二次数据生成 - 通过设备数据, 生成站点数据
func statisticSiteData(ctx *gin.Context, dataTem model.StatisticData) {

	dateTypeList := GetDateMark(dataTem.EndStatisticTime)

	// 值类型
	dataTem.StatisticValueType = 0
	// 累加起来
	for _, v := range dateTypeList {
		// 粒度
		dataTem.MarkDate = v["dateMark"]
		dataTem.StatisticDateType = v["dateType"]
		doStatisticSite(ctx, dataTem)
	}

	// 值类型
	dataTem.StatisticValueType = 1
	// 累加起来
	for _, v := range dateTypeList {
		// 粒度
		dataTem.MarkDate = v["dateMark"]
		dataTem.StatisticDateType = v["dateType"]
		doStatisticSite(ctx, dataTem)
	}
}

// 累计所有设备 累计数据, 存为当前日期站点累计
// 累计所有设备 增量数据, 存为当前日期站点增量
func doStatisticSite(ctx *gin.Context, sdTem model.StatisticData) {

	statisticDao := dao.NewStatisticDataDao(ctx)

	// 查询设备数据
	sdTem.TargetType = 0
	dataList, res := statisticDao.FindBySiteId(sdTem)
	if !res {
		// 查询站点数据失败, 有设备的数据进来, 反而没有站点的数据, 记录一下
		log.Println("[STATISTIC] statistic site data fail, fail to find data ", sdTem)
		log.Println("[STATISTIC] target type: ", sdTem.TargetType)
		log.Println("[STATISTIC] date type: ", sdTem.StatisticDateType)
		log.Println("[STATISTIC] value type: ", sdTem.StatisticValueType)
		return
	}

	// 站点操作
	sdTem.TargetType = 1
	// 累计数值
	var countAll float64
	for _, v := range dataList {
		countAll = countAll + v.StatisticValue
	}
	sdTem.StatisticValue = countAll
	sdTem.StartValue = 0
	sdTem.EndValue = 0

	// 存库, 或者更新
	dataTem, res := statisticDao.ExistSiteDateMark(sdTem)
	if !res {
		statisticDao.Create(sdTem)
	} else {
		sdTem.ID = dataTem.ID
		statisticDao.Update(sdTem)
	}
}
