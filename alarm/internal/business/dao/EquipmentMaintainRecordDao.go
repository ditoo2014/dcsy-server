package dao

import (
	"alarm/internal/business/model"
	"alarm/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EquipmentMaintainRecordDao struct {
	db *gorm.DB
}

func NewEquipmentMaintainRecordDao(ctx *gin.Context) *EquipmentMaintainRecordDao {
	return &EquipmentMaintainRecordDao{db: gormutil.GetGormDB(ctx, "")}
}

// 设备维保记录
func (dao *EquipmentMaintainRecordDao) Create(emr model.EquipmentMaintainRecord) (model.EquipmentMaintainRecord, bool) {

	// 创建记录
	err := dao.db.Create(&emr).Error
	if err != nil {
		log.Println("[EquipmentMaintainRecordDao] create record failed!")
		return emr, false
	}

	return emr, true
}

func (dao *EquipmentMaintainRecordDao) Find(emr model.EquipmentMaintainRecord, offset, limit, startTime, endTime int) ([]model.EquipmentMaintainRecord, bool, int64) {
	emrList := []model.EquipmentMaintainRecord{}

	sqlStr := "1=1 "
	if emr.EquipmentId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND equipment_id = %d ", emr.EquipmentId)
	}
	if len(emr.EquipmentTypeName) > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND equipment_id IN (SELECT id FROM equipment WHERE equipment_type_name LIKE '%%%s%%')", emr.EquipmentTypeName)
	}

	err := dao.db.Model(&emr).Offset(offset).Limit(limit).Where(sqlStr+" AND create_time >= ? AND create_time <= ?", startTime, endTime).Scan(&emrList).Error
	if err != nil {
		log.Println("[EquipmentMaintainRecordDao] Find records fail")
		return emrList, false, 0
	}

	// 批量填充电池类型名称
	fillMaintainRecordEquipmentTypeNames(dao.db, emrList)

	var total int64
	err = dao.db.Model(&emr).Where(sqlStr+" AND create_time >= ? AND create_time <= ?", startTime, endTime).Count(&total).Error
	if err != nil {
		log.Println("[EquipmentMaintainRecordDao] Count records fail")
		return emrList, false, 0
	}

	return emrList, true, total
}

// fillMaintainRecordEquipmentTypeNames 批量填充维保记录的电池类型名称
func fillMaintainRecordEquipmentTypeNames(db *gorm.DB, data []model.EquipmentMaintainRecord) {
	if len(data) == 0 {
		return
	}
	ids := make([]int, 0, len(data))
	seen := make(map[int]struct{}, len(data))
	for _, item := range data {
		if item.EquipmentId > 0 {
			if _, ok := seen[item.EquipmentId]; !ok {
				ids = append(ids, item.EquipmentId)
				seen[item.EquipmentId] = struct{}{}
			}
		}
	}
	if len(ids) == 0 {
		return
	}

	type equipTypeRow struct {
		ID   int    `gorm:"column:id"`
		Name string `gorm:"column:equipment_type_name"`
	}
	var rows []equipTypeRow
	db.Table("equipment").Select("id, equipment_type_name").Where("id IN ?", ids).Scan(&rows)

	typeMap := make(map[int]string, len(rows))
	for _, row := range rows {
		typeMap[row.ID] = row.Name
	}
	for i := range data {
		data[i].EquipmentTypeName = typeMap[data[i].EquipmentId]
	}
}

// 创建设备
func (dao *EquipmentMaintainRecordDao) ImportEquipmentMaintainRecords(eqs []model.EquipmentMaintainRecord) ([]model.EquipmentMaintainRecord, bool) {

	err := dao.db.Create(&eqs).Error
	if err != nil {
		log.Println("Failed to create EquipmentMaintainRecord list: ", err)
		return eqs, false
	}

	return eqs, true
}
