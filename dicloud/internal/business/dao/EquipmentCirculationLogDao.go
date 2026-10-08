package dao

import (
	"dicloud/internal/business/model"
	"dicloud/util/gormutil"
	"log"
	"strings"

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
	query, valid := dao.findQuery(ecl)
	if !valid {
		// 流转记录必须绑定具体设备，防止编号0退化为全表查询。
		log.Println("[EquipmentCirculationLogDao] invalid equipment id")
		return eclList, false
	}

	err := query.Scan(&eclList).Error
	if err != nil {
		log.Println("[EquipmentCirculationLogDao] Find logs fail")
		return eclList, false
	}

	// 批量填充电池类型名称
	fillCirculationLogEquipmentTypeNames(dao.db, eclList)

	return eclList, true
}

// findQuery 构造流转记录查询，单独保留便于离线 DryRun 测试校验参数和排序。
func (dao *EquipmentCirculationLogDao) findQuery(ecl model.EquipmentCirculationLog) (*gorm.DB, bool) {
	if ecl.EquipmentId <= 0 {
		return nil, false
	}

	// 设备编号由控制器校验为正数；这里仍使用参数绑定，避免拼接查询条件带来的注入风险。
	query := dao.db.Model(&model.EquipmentCirculationLog{}).Where("equipment_id = ?", ecl.EquipmentId)
	if equipmentTypeName := strings.TrimSpace(ecl.EquipmentTypeName); equipmentTypeName != "" {
		// 类型名称来自查询参数，使用子查询和占位符保持查询安全。
		query = query.Where(
			"equipment_id IN (?)",
			dao.db.Table("equipment").Select("id").Where("equipment_type_name LIKE ?", "%"+equipmentTypeName+"%"),
		)
	}

	// 保持历史调用方需要的正序：先按发生时间，再按主键稳定排序。
	return query.Order("create_time ASC").Order("id ASC"), true
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
