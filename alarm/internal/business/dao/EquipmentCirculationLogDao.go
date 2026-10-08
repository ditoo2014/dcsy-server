package dao

import (
	"alarm/internal/business/model"
	"alarm/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EquipmentCirculationLogDao struct {
	db *gorm.DB
}

func NewEquipmentCirculationLogDao(ctx *gin.Context) *EquipmentCirculationLogDao {
	return &EquipmentCirculationLogDao{db: gormutil.GetGormDB(ctx, "")}
}

// 记录流转日志
func (dao *EquipmentCirculationLogDao) Create(ecl model.EquipmentCirculationLog) (model.EquipmentCirculationLog, bool) {
	tx := dao.db.Begin()

	// 创建记录
	err := tx.Create(&ecl).Error
	if err != nil {
		tx.Rollback()
		log.Println("[EquipmentCirculationLogDao] create log failed!")
		return ecl, false
	}

	// 更新设备流转状态
	rwef := tx.Table("equipment").Where("id = ? ", ecl.EquipmentId).Update("position_status", ecl.OperateType).RowsAffected
	if rwef == 0 {
		tx.Rollback()
		log.Println("[EquipmentCirculationLogDao] update equipment positionstatus fail!")
		return ecl, false
	}

	// 提交
	tx.Commit()
	return ecl, true
}

func (dao *EquipmentCirculationLogDao) Find(ecl model.EquipmentCirculationLog) ([]model.EquipmentCirculationLog, bool) {
	eclList := []model.EquipmentCirculationLog{}

	sqlStr := "1=1 "
	if ecl.EquipmentId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND equipment_id = %d ", ecl.EquipmentId)
	}
	if len(ecl.EquipmentTypeName) > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND equipment_id IN (SELECT id FROM equipment WHERE equipment_type_name LIKE '%%%s%%')", ecl.EquipmentTypeName)
	}

	err := dao.db.Model(&ecl).Order("id asc").Where(sqlStr).Scan(&eclList).Error
	if err != nil {
		log.Println("[EquipmentCirculationLogDao] Find logs fail")
		return eclList, false
	}

	// 批量填充电池类型名称
	fillCirculationLogEquipmentTypeNames(dao.db, eclList)

	return eclList, true
}

// FindLatestByEquipmentIDs returns at most one latest circulation log for each equipment.
func (dao *EquipmentCirculationLogDao) FindLatestByEquipmentIDs(equipmentIDs []int) ([]model.EquipmentCirculationLog, bool) {
	eclList := []model.EquipmentCirculationLog{}
	if len(equipmentIDs) == 0 {
		return eclList, true
	}

	latestIDs := dao.db.Table("equipment_circulation_log").
		Select("MAX(id)").
		Where("equipment_id IN ?", equipmentIDs).
		Group("equipment_id")
	err := dao.db.Table("equipment_circulation_log").
		Where("id IN (?)", latestIDs).
		Order("equipment_id asc").
		Scan(&eclList).Error
	if err != nil {
		log.Println("[EquipmentCirculationLogDao] find latest logs fail: ", err)
		return eclList, false
	}

	return eclList, true
}

// fillCirculationLogEquipmentTypeNames 批量填充流转记录的电池类型名称
func fillCirculationLogEquipmentTypeNames(db *gorm.DB, data []model.EquipmentCirculationLog) {
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
