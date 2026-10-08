package controller

import (
	"strconv"
	"strings"

	commonmodel "dicloud/internal/common/model"
	"dicloud/util/gormutil"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

type IAlarmUsersCtrler interface {
	Find(ctx *gin.Context)
	Delete(ctx *gin.Context)
	BatchDelete(ctx *gin.Context)
}

type AlarmUsersCtrler struct{}

type alarmUsersRequest struct {
	ID            uint     `json:"id"`
	SiteID        int      `json:"siteId"`
	SiteName      string   `json:"siteName"`
	EquipmentID   int      `json:"equipmentId"`
	EquipmentName string   `json:"equipmentName"`
	Coding        string   `json:"coding"`
	Codings       []string `json:"codings"`
}

type alarmUsersView struct {
	ID                   uint   `json:"id"`
	TriggerID            uint   `json:"triggerId"`
	SiteID               int    `json:"siteId"`
	SiteName             string `json:"siteName"`
	EquipmentID          int    `json:"equipmentId"`
	EquipmentName        string `json:"equipmentName"`
	Coding               string `json:"coding"`
	AlarmCode            string `json:"alarmCode"`
	AlarmType            string `json:"alarmType"`
	AlarmStage           int    `json:"alarmStage"`
	AlarmChannel         string `json:"alarmChannel"`
	AlarmUsers           string `json:"alarmUsers"`
	ShortMsgAlarm        bool   `json:"shortMsgAlarm" gorm:"-"`
	ShortMsgAlarmUserIDs []int  `json:"shortMsgAlarmUserIds" gorm:"-"`
}

func (AlarmUsersCtrler) Find(ctx *gin.Context) {
	var request alarmUsersRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Fail(ctx, "请求参数格式错误", nil)
		return
	}
	db := gormutil.GetGormDB(ctx, "")
	query := db.Table("alarm_users AS au").
		Select(`au.id, au.trigger_id, tc.site_id, COALESCE(s.site_name, '') AS site_name,
			tc.equipment_id, COALESCE(e.equipment_name, '') AS equipment_name,
			tc.code AS coding, au.alarm_code, au.alarm_type, au.alarm_stage,
			au.alarm_channel, au.alarm_users`).
		Joins("JOIN trigger_config AS tc ON tc.id = au.trigger_id").
		Joins("LEFT JOIN equipment AS e ON e.id = tc.equipment_id").
		Joins("LEFT JOIN site AS s ON s.id = tc.site_id")
	if request.SiteID > 0 {
		query = query.Where("tc.site_id = ?", request.SiteID)
	}
	if strings.TrimSpace(request.SiteName) != "" {
		query = query.Where("s.site_name = ?", strings.TrimSpace(request.SiteName))
	}
	if request.EquipmentID > 0 {
		query = query.Where("tc.equipment_id = ?", request.EquipmentID)
	}
	if strings.TrimSpace(request.EquipmentName) != "" {
		query = query.Where("e.equipment_name = ?", strings.TrimSpace(request.EquipmentName))
	}
	if strings.TrimSpace(request.Coding) != "" {
		query = query.Where("tc.code = ?", strings.TrimSpace(request.Coding))
	}
	var list []alarmUsersView
	if err := query.Order("au.id ASC").Scan(&list).Error; err != nil {
		response.Fail(ctx, "查询告警接收人失败", nil)
		return
	}
	for index := range list {
		list[index].ShortMsgAlarm = csvContains(list[index].AlarmChannel, "0")
		list[index].ShortMsgAlarmUserIDs = parseCSVInts(list[index].AlarmUsers)
	}
	response.Success(ctx, "查询成功", gin.H{"list": list})
}

func (AlarmUsersCtrler) Delete(ctx *gin.Context) {
	var request alarmUsersRequest
	if err := ctx.ShouldBindJSON(&request); err != nil || request.ID == 0 {
		response.Fail(ctx, "告警接收人ID不能为空", nil)
		return
	}
	db := gormutil.GetGormDB(ctx, "")
	result := db.Delete(&commonmodel.AlarmUser{}, request.ID)
	if result.Error != nil {
		response.Fail(ctx, "删除告警接收人失败", nil)
		return
	}
	response.Success(ctx, "删除成功", nil)
}

func (AlarmUsersCtrler) BatchDelete(ctx *gin.Context) {
	var request alarmUsersRequest
	if err := ctx.ShouldBindJSON(&request); err != nil || request.EquipmentID <= 0 || len(request.Codings) == 0 {
		response.Fail(ctx, "设备和告警代码不能为空", nil)
		return
	}
	for index := range request.Codings {
		request.Codings[index] = strings.TrimSpace(request.Codings[index])
	}
	db := gormutil.GetGormDB(ctx, "")
	var triggerIDs []uint
	if err := db.Model(&commonmodel.TriggerConfig{}).
		Where("equipment_id = ? AND code IN ?", request.EquipmentID, request.Codings).
		Pluck("id", &triggerIDs).Error; err != nil {
		response.Fail(ctx, "查询触发器失败", nil)
		return
	}
	if len(triggerIDs) > 0 {
		if err := db.Where("trigger_id IN ?", triggerIDs).Delete(&commonmodel.AlarmUser{}).Error; err != nil {
			response.Fail(ctx, "批量删除告警接收人失败", nil)
			return
		}
	}
	response.Success(ctx, "删除成功", nil)
}

func csvContains(value, target string) bool {
	for _, item := range strings.Split(value, ",") {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func parseCSVInts(value string) []int {
	result := make([]int, 0)
	for _, item := range strings.Split(value, ",") {
		parsed, err := strconv.Atoi(strings.TrimSpace(item))
		if err == nil && parsed > 0 {
			result = append(result, parsed)
		}
	}
	return result
}
