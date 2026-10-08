package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	"fmt"

	"github.com/gin-gonic/gin"
)

// 处理增量数据 - 保存, 生成累计数据
func dealIncrementData(ctx *gin.Context, dataTem model.StatisticData) {
	fmt.Println("[STATISTIC] DealIncremtData")

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
		doIncrementStatistic(ctx, dataTem)
	}

	statisticSiteData(ctx, dataTem)
	statisticCorpData(ctx, dataTem, dateTypeList)
}

// 处理设备增量数据
func doIncrementStatistic(ctx *gin.Context, sdTem model.StatisticData) {

	// 查询已存在增量, 累加 -> 生成新增量
	statisticDao := dao.NewStatisticDataDao(ctx)
	dataTem, res := statisticDao.ExistDateMark(sdTem)
	if res {
		// 相加 -> 更新数据
		sdTem.StatisticValue = dataTem.StatisticValue + sdTem.StatisticValue
		sdTem.ID = dataTem.ID
		statisticDao.Update(sdTem)
	} else {
		// 直接存库
		statisticDao.Create(sdTem)
	}

	// 累计表
	sdTem.StatisticValueType = 0
	// 计算累计 - 将增量加到之前累计, 生成现在累计 -> 已存在更新

	dataTem, ans := statisticDao.ExistLastCumData(sdTem)
	if ans {
		// 更新累计值 - 累计量 + 增量
		sdTem.StatisticValue = dataTem.StatisticValue + sdTem.StatisticValue
	}

	// 是否存在当前累计数据
	curTem, res := statisticDao.ExistDateMark(sdTem)
	if res {
		// 不创建新数据
		// 增量传输一次, 累加一次
		sdTem.ID = curTem.ID
		statisticDao.Update(sdTem)
	} else {
		// 存为最新累计值
		statisticDao.Create(sdTem)
	}
}
