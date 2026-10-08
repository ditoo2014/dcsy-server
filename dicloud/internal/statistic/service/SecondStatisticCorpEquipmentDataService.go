package service

import (
	"dicloud/internal/common/model"

	"github.com/gin-gonic/gin"
)

// 部门二次数据生成 - 通过设备数据, 生成部门数据
// 这里只生成累计值数据, 列举一下: 设备数量, 设备健康度, 循环次数, 累计充电量, 累计放电量
func statisticCorpEquipmentData(ctx *gin.Context, dataTem model.StatisticData, dateTypeList []map[string]string) bool {

	// 值类型
	dataTem.StatisticValueType = 0
	// 累加起来
	for _, v := range dateTypeList {
		// 粒度
		dataTem.MarkDate = v["dateMark"]
		dataTem.StatisticDateType = v["dateType"]
		if !doStatisticCorp(ctx, dataTem) {
			return false
		}
	}

	// 不需要处理增量值
	return true
}
