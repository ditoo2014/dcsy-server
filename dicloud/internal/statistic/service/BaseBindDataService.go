package service

import (
	businessdao "dicloud/internal/business/dao"
	businessmodel "dicloud/internal/business/model"
	"dicloud/internal/common/model"
	"encoding/json"
	"log"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
)

// 基础数据校验 - 填充
func BaseDataBind(ctx *gin.Context, msgOrg *primitive.MessageExt) (statisticMsg model.StatisticMQModel, res bool) {

	// 获取数据, 转换数据类型
	msg := string(msgOrg.Body)

	/* 数据解析 */

	res = true
	err := json.Unmarshal([]byte(msg), &statisticMsg)
	if err != nil {
		res = false
		log.Println("[STATISTIC] BaseDataBind Service Unmarshal failed")
		return
	}

	// 获取设备数据 - 绑定
	equipmentDao := businessdao.NewEquipmentDAO(ctx)

	// 验证设备是否存在
	equipmentTem, res := equipmentDao.FindEquipmentById(businessmodel.Equipment{ID: uint(statisticMsg.EquipmentId)})
	if !res {
		res = false
		log.Println("[STATISTIC] find mapper equipment Fail", statisticMsg)
		return
	}

	// 站点数据
	if equipmentTem.SiteId > 0 {
		statisticMsg.SiteId = equipmentTem.SiteId
	}
	// 部门数据
	if equipmentTem.CorpId > 0 {
		statisticMsg.CorpId = equipmentTem.CorpId
	}

	return
}

// 根据时间戳(单位: 秒), 获取时间标记
func GetDateMark(timestamp int64) (dateTypeList []map[string]string) {

	// 将时间戳转换为time.Time对象
	t := time.Unix(timestamp, 0) // 第二个参数是纳秒，这里为0因为只提供了秒

	var (
		hourPattern  = "2006-01-02-15"
		dayPattern   = "2006-01-02"
		monthPattern = "2006-01"
	)

	mapHour := make(map[string]string)
	mapHour["dateMark"] = t.Format(hourPattern)
	mapHour["dateType"] = "hourly"

	mapDaily := make(map[string]string)
	mapDaily["dateMark"] = t.Format(dayPattern)
	mapDaily["dateType"] = "daily"

	mapMonthly := make(map[string]string)
	mapMonthly["dateMark"] = t.Format(monthPattern)
	mapMonthly["dateType"] = "monthly"

	dateTypeList = append(dateTypeList, mapHour)
	dateTypeList = append(dateTypeList, mapDaily)
	dateTypeList = append(dateTypeList, mapMonthly)

	return
}

// 根据时间戳(单位: 秒), 获取时间标记 - 获取最小统计周期为天的数据
func GetDayBaseDateMark(timestamp int64) (dateTypeList []map[string]string) {

	// 将时间戳转换为time.Time对象
	t := time.Unix(timestamp, 0) // 第二个参数是纳秒，这里为0因为只提供了秒

	var (
		dayPattern   = "2006-01-02"
		monthPattern = "2006-01"
	)

	mapDaily := make(map[string]string)
	mapDaily["dateMark"] = t.Format(dayPattern)
	mapDaily["dateType"] = "daily"

	mapMonthly := make(map[string]string)
	mapMonthly["dateMark"] = t.Format(monthPattern)
	mapMonthly["dateType"] = "monthly"

	dateTypeList = append(dateTypeList, mapDaily)
	dateTypeList = append(dateTypeList, mapMonthly)

	return
}
