package dao

import (
	"alarm/models/model"
	"alarm/util/gormutil"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SystemLogDao struct {
	db *gorm.DB
}

func NewSystemLogDao(ctx *gin.Context) *SystemLogDao {
	return &SystemLogDao{db: gormutil.GetGormDB(ctx, "")}
}

func (dao *SystemLogDao) Create(systemLog model.SystemLog) bool {

	// 使用 GORM 保存
	if err := dao.db.Create(&systemLog).Error; err != nil {
		log.Printf("Failed to save system log: %v", err)
		return false
	}

	return true
}

func (dao *SystemLogDao) Find(systemLog model.SystemLog, offset, limit, startTime, endTime int) ([]model.SystemLog, int64, bool) {

	var dataList []model.SystemLog
	var total int64

	query := dao.baseSystemLogQuery(systemLog, startTime, endTime)
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		log.Printf("Failed to count system log: %v", err)
		return dataList, 0, false
	}

	if err := query.Session(&gorm.Session{}).Order("log_time DESC").Offset(offset).Limit(limit).Find(&dataList).Error; err != nil {
		log.Printf("Failed to find system log: %v", err)
		return dataList, 0, false
	}

	return dataList, total, true
}

func (dao *SystemLogDao) baseSystemLogQuery(systemLog model.SystemLog, startTime, endTime int) *gorm.DB {
	query := dao.db.Model(&model.SystemLog{}).Where("log_time >= ? AND log_time <= ?", startTime, endTime)

	if systemLog.PlatId > 0 {
		if systemLog.PlatId == 1 {
			query = query.Where("(plat_id = ? OR plat_id = 0)", systemLog.PlatId)
		} else {
			query = query.Where("plat_id = ?", systemLog.PlatId)
		}
	}
	if systemLog.UserId > 0 {
		query = query.Where("user_id = ?", systemLog.UserId)
	}
	if value := likeValue(systemLog.Username); value != "" {
		query = query.Where("username LIKE ?", value)
	}
	if value := likeValue(systemLog.TrueName); value != "" {
		query = query.Where("true_name LIKE ?", value)
	}

	return query
}

func likeValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "%" + value + "%"
}
