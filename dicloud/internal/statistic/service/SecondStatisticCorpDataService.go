package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	"log"

	"github.com/gin-gonic/gin"
)

// 部门二次数据生成 - 通过设备数据, 生成部门数据
func statisticCorpData(ctx *gin.Context, dataTem model.StatisticData, dateTypeList []map[string]string) {

	// 值类型
	dataTem.StatisticValueType = 0
	// 累加起来
	for _, v := range dateTypeList {
		// 粒度
		dataTem.MarkDate = v["dateMark"]
		dataTem.StatisticDateType = v["dateType"]
		doStatisticCorp(ctx, dataTem)
	}

	// 值类型
	dataTem.StatisticValueType = 1
	// 累加起来
	for _, v := range dateTypeList {
		// 粒度
		dataTem.MarkDate = v["dateMark"]
		dataTem.StatisticDateType = v["dateType"]
		doStatisticCorp(ctx, dataTem)
	}
}

// 累计所有设备 累计数据, 存为当前日期部门累计
// 累计所有设备 增量数据, 存为当前日期部门增量
func doStatisticCorp(ctx *gin.Context, sdTem model.StatisticData) bool {

	statisticDao := dao.NewStatisticDataDao(ctx)

	// 查询设备数据 - 只匹配存在的设备记录, 如果被删除, 则不统计数据(统计现存设备数量)
	sdTem.TargetType = 0
	// 优化查询速度
	dataVal, res := statisticDao.FindByCorpIdExistEquipment(sdTem)
	if !res {
		// 查询部门数据失败, 有设备的数据进来, 反而没有部门的数据, 记录一下
		log.Println("[STATISTIC] statistic corp data fail, fail to find data ", sdTem)
		return false
	}

	// 部门操作
	sdTem.TargetType = 2
	// 累计数值
	sdTem.StatisticValue = dataVal
	sdTem.StartValue = 0
	sdTem.EndValue = 0

	// 存库, 或者更新
	dataTem, res := statisticDao.ExistCorpDateMark(sdTem)
	if !res {
		return statisticDao.Create(sdTem)
	} else {
		sdTem.ID = dataTem.ID
		return statisticDao.Update(sdTem)
	}
}
