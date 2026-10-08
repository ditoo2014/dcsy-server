package controller

import (
	"strconv"
	"strings"
	"time"

	commonmodel "dicloud/internal/common/model"
	"dicloud/util/gormutil"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

type ILiveAlarmCtrler interface {
	Find(ctx *gin.Context)
	Remove(ctx *gin.Context)
}

// Remove 将当前告警标记为已处理，不直接删除历史记录。
func (LiveAlarmCtrler) Remove(ctx *gin.Context) {
	var request liveAlarmRequest
	if err := ctx.ShouldBindJSON(&request); err != nil || request.ID == 0 {
		response.Fail(ctx, "实时告警ID不能为空", nil)
		return
	}
	result := gormutil.GetGormDB(ctx, "").Model(&commonmodel.AlarmDataRecord{}).
		Where("id = ? AND alarm_status = ?", request.ID, 1).
		Updates(map[string]any{"alarm_status": 0, "end_time": time.Now().Unix()})
	if result.Error != nil {
		response.Fail(ctx, "处理实时告警失败", nil)
		return
	}
	response.Success(ctx, "处理成功", nil)
}

type LiveAlarmCtrler struct{}

type liveAlarmRequest struct {
	ID            uint   `json:"id"`
	EquipmentID   int    `json:"equipmentId"`
	EquipmentName string `json:"equipmentName"`
	SiteID        int    `json:"siteId"`
	SiteName      string `json:"siteName"`
	Coding        string `json:"coding"`
	Begin         int64  `json:"begin"`
	End           int64  `json:"end"`
}

type liveAlarmView struct {
	ID            uint   `json:"id"`
	EquipmentID   int    `json:"equipmentId"`
	EquipmentName string `json:"equipmentName"`
	SiteID        int    `json:"siteId"`
	SiteName      string `json:"siteName"`
	Coding        string `json:"coding"`
	Begin         int64  `json:"begin"`
	End           int64  `json:"end"`
	Value         string `json:"value"`
	AlarmType     string `json:"alarmType"`
	AlarmStage    int    `json:"alarmStage"`
}

func (LiveAlarmCtrler) Find(ctx *gin.Context) {
	var request liveAlarmRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Fail(ctx, "请求参数格式错误", nil)
		return
	}
	page := positiveQueryInt(ctx, "pageNumber", 1)
	pageSize := positiveQueryInt(ctx, "pageSize", 20)
	if pageSize > 10000 {
		pageSize = 10000
	}
	db := gormutil.GetGormDB(ctx, "")
	query := db.Model(&commonmodel.AlarmDataRecord{}).Where("alarm_status = ?", 1)
	if request.ID > 0 {
		query = query.Where("id = ?", request.ID)
	}
	if request.EquipmentID > 0 {
		query = query.Where("equipment_id = ?", request.EquipmentID)
	}
	if strings.TrimSpace(request.EquipmentName) != "" {
		query = query.Where("equipment_name LIKE ?", "%"+strings.TrimSpace(request.EquipmentName)+"%")
	}
	if request.SiteID > 0 {
		query = query.Where("site_id = ?", request.SiteID)
	}
	if strings.TrimSpace(request.SiteName) != "" {
		query = query.Where("site_name LIKE ?", "%"+strings.TrimSpace(request.SiteName)+"%")
	}
	if strings.TrimSpace(request.Coding) != "" {
		query = query.Where("alarm_code = ?", strings.TrimSpace(request.Coding))
	}
	if request.Begin > 0 {
		query = query.Where("begin_time >= ?", request.Begin)
	}
	if request.End > 0 {
		query = query.Where("begin_time <= ?", request.End)
	}
	if start := queryInt64(ctx, "startTime"); start > 0 {
		query = query.Where("begin_time >= ?", start)
	}
	if end := queryInt64(ctx, "endTime"); end > 0 {
		query = query.Where("begin_time <= ?", end)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(ctx, "统计实时告警失败", nil)
		return
	}
	var records []commonmodel.AlarmDataRecord
	if err := query.Order("begin_time DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		response.Fail(ctx, "查询实时告警失败", nil)
		return
	}
	list := make([]liveAlarmView, 0, len(records))
	for _, record := range records {
		list = append(list, liveAlarmView{
			ID: record.ID, EquipmentID: record.EquipmentId, EquipmentName: record.EquipmentName,
			SiteID: record.SiteId, SiteName: record.SiteName, Coding: record.AlarmCode,
			Begin: record.BeginTime, End: record.EndTime, Value: record.AlarmValue,
			AlarmType: record.AlarmType, AlarmStage: record.AlarmStage,
		})
	}
	response.Success(ctx, "查询成功", gin.H{"list": list, "total": total})
}

func positiveQueryInt(ctx *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(ctx.Query(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func queryInt64(ctx *gin.Context, key string) int64 {
	value, _ := strconv.ParseInt(ctx.Query(key), 10, 64)
	return value
}
