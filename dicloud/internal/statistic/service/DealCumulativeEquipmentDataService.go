package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	commonservice "dicloud/internal/common/service"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// 处理累计数据 - 保存, 生成增量数据
func dealCumulativeEquipmentData(ctx *gin.Context, dataTem model.StatisticData) bool {
	fmt.Println("[STATISTIC] DealCumulativeEquipmentData")

	// dateTypeList := GetDateMark(dataTem.EndStatisticTime)
	// 统计基础周期为天的数据, 不细化统计周期到小时
	dateTypeList := GetDayBaseDateMark(dataTem.EndStatisticTime)

	// 统计 设备, 另一纬度 部门

	// 确保设备操作
	dataTem.TargetType = 0
	oldCode := ""
	diffCode := false
	for _, v := range dateTypeList {

		// 三种粒度的数据
		dataTem.MarkDate = v["dateMark"]
		dataTem.StatisticDateType = v["dateType"]
		var writeOK bool
		oldCode, diffCode, writeOK = doOnlyCumulativeStatistic(ctx, dataTem)
		if !writeOK {
			return false
		}
	}

	// 只根据部门统计即可
	if !statisticCorpEquipmentData(ctx, dataTem, dateTypeList) {
		return false
	}

	if diffCode && len(oldCode) > 0 {
		// 设备健康度变化, 统计为新的健康度了
		// 将原来的SOH健康度(SOH优, SOH良, SOH中, SOH差), 重新统计
		log.Println("[Statistic] statistic diff CODE (SOH)", oldCode)
		dataTem.StatisticCode = oldCode
		dataTem.StatisticName = oldCode
		if !statisticCorpEquipmentData(ctx, dataTem, dateTypeList) {
			return false
		}
	}

	return true
}

// 处理设备累计数据
func doOnlyCumulativeStatistic(ctx *gin.Context, sdTem model.StatisticData) (oldCode string, diffCode bool, writeOK bool) {
	token, locked := commonservice.LockStatisticDailyData(ctx, &sdTem, 3*time.Second)
	if !locked {
		return "", false, false
	}
	defer commonservice.UnlockStatisticDailyData(ctx, &sdTem, token)

	// 查询上次数据, 相对这次
	statisticDao := dao.NewStatisticDataDao(ctx)

	// 查询是否存在
	diffCode = false
	dataTem, res := statisticDao.ExistDateMark(sdTem)
	if !res {
		// 保存设备累计数据
		writeOK = statisticDao.Create(sdTem)
	} else {
		// 根据id更新最新数据
		sdTem.ID = dataTem.ID
		writeOK = statisticDao.Update(sdTem)
		if sdTem.StatisticCode != dataTem.StatisticCode {
			oldCode = dataTem.StatisticCode
			diffCode = true
		}
	}

	return
}
