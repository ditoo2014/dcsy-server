package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

// 处理累计数据 - 保存, 生成增量数据
func dealCumulativeData(ctx *gin.Context, dataTem model.StatisticData) {
	fmt.Println("[STATISTIC] DealCumulativeData")

	dateTypeList := GetDateMark(dataTem.EndStatisticTime)

	// 统计站点, 设备, 另一纬度 部门
	// 统计时候, 通过累加设备数据, 获取站点或部门的二次数据, 进行计算站点和部门数据
	// 以防设备减少, 变化导致的数据变化

	// 确保设备操作
	dataTem.TargetType = 0
	for _, v := range dateTypeList {
		// 三种粒度的数据
		dataTem.MarkDate = v["dateMark"]
		dataTem.StatisticDateType = v["dateType"]
		doCumulativeStatistic(ctx, dataTem)
	}

	statisticSiteData(ctx, dataTem)
	statisticCorpData(ctx, dataTem, dateTypeList)
}

// 处理设备累计数据
func doCumulativeStatistic(ctx *gin.Context, sdTem model.StatisticData) (oldCode string, diffCode bool) {

	// 查询上次数据, 相对这次
	lastTem := model.StatisticData{}
	statisticDao := dao.NewStatisticDataDao(ctx)

	// 查询是否存在
	diffCode = false
	dataTem, res := statisticDao.ExistDateMark(sdTem)
	if !res {
		// 保存设备累计数据
		statisticDao.Create(sdTem)
	} else {
		// 根据id更新最新数据
		sdTem.ID = dataTem.ID
		statisticDao.Update(sdTem)
		if sdTem.StatisticCode != dataTem.StatisticCode {
			oldCode = dataTem.StatisticCode
			diffCode = true
		}
	}

	// 查缓存 - 如果不按时间顺序, 则出现问题
	// 查询数据库
	lastTem, res = statisticDao.ExistLastCumData(sdTem)
	if !res {
		// 不存在, 不计算增量数据
		log.Println("[STATISTIC] statistic cumulative data, not exist last cum, ignore increment")
		return
	}

	if lastTem.ID == 0 {
		// 查询失败
		return
	}

	// 两条(及以上), 计算增量数据
	countInc := sdTem.StatisticValue - lastTem.StatisticValue
	if countInc < 0 {
		// 非法数据, 不存储
		log.Println("[STATISTIC] statistic cumulative data, countInc fail: ", countInc)
		return
	}

	// 存储数据 - 增量
	sdTem.StatisticValueType = 1
	// 开始时间 - 结束时间
	sdTem.StartStatisticTime = lastTem.EndStatisticTime
	// 开始值 - 结束值
	sdTem.StartValue = lastTem.StatisticValue
	sdTem.EndValue = sdTem.StatisticValue
	sdTem.StatisticValue = countInc

	// 是否存在当前增量数据
	curTem, res := statisticDao.ExistDateMark(sdTem)
	if res {
		// 存在, 更新
		sdTem.ID = curTem.ID
		statisticDao.Update(sdTem)
		return
	}

	// 保存增量数据
	statisticDao.Create(sdTem)

	return
}
