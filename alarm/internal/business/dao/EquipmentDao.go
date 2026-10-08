package dao

import (
	"alarm/internal/business/model"
	"alarm/util/gormutil"
	normalutil "alarm/util/normal"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EquipmentDAO struct {
	db *gorm.DB
}

// 创建新的 EquipmentDAO 实例
func NewEquipmentDAO(ctx *gin.Context) *EquipmentDAO {
	return &EquipmentDAO{db: gormutil.GetGormDB(ctx, "")}
}

// 创建设备
func (dao *EquipmentDAO) CreateEquipment(eq model.Equipment) (model.Equipment, bool) {

	err := dao.db.Create(&eq).Error
	if err != nil || eq.ID == 0 {
		log.Println("[EquipmentDao] Failed to create EQUIPMENT: ", err)
		return eq, false
	}

	return eq, true
}

// 创建设备
func (dao *EquipmentDAO) ImportEquipments(eqs, ueqs []model.Equipment) ([]model.Equipment, bool) {
	err := dao.db.Transaction(func(tx *gorm.DB) error {
		if len(eqs) > 0 {
			if err := tx.Create(&eqs).Error; err != nil {
				return err
			}
		}

		for _, v := range ueqs {
			if err := tx.Model(&model.Equipment{}).Where("equipment_code = ?", v.EquipmentCode).Updates(&v).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Println("Failed to import EQUIPMENT list: ", err)
		return eqs, false
	}

	return eqs, true
}

// 查询列表
func (dao *EquipmentDAO) FindEquipmentAll(eq model.Equipment, offset int, limit int) ([]model.Equipment, int64, bool) {
	equipments := []model.Equipment{}

	sqlStr := baseEquipmentSql(eq)

	err := dao.db.Table("equipment").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr).Scan(&equipments).Error
	if err != nil || len(equipments) == 0 {
		log.Println("[EquipmentDao] Failed to find Equipment: ", err)
		return equipments, 0, false
	}
	var total int64
	dao.db.Table("equipment").Where(sqlStr).Count(&total)

	return equipments, total, true
}

// 计算部门中设备数量
func (dao *EquipmentDAO) CountEquipmentByCorp(eq model.Equipment, corpIds []int) (int64, bool) {
	var count int64

	sqlStr := baseEquipmentSql(eq)
	err := dao.db.Table("equipment").Select(" count(1) ").Where(sqlStr+"AND corp_id in ?", corpIds).Scan(&count).Error
	if err != nil {
		log.Println("[EquipmentDao] Failed to count corp Equipment: ", err)
		return 0, false
	}

	return count, true
}

// 计算部门中设备数量
func (dao *EquipmentDAO) CountEquipmentGroupCorp(eq model.Equipment, corpIds []int) ([]model.GroupCorpCount, bool) {
	var dataList []model.GroupCorpCount

	sqlStr := baseEquipmentSql(eq)
	err := dao.db.Table("equipment").Select(" count(1) count,corp_id ").Group("corp_id").Where(sqlStr+"AND corp_id in ?", corpIds).Scan(&dataList).Error
	if err != nil {
		log.Println("[EquipmentDao] Failed to count corp Equipment: ", err)
		return dataList, false
	}

	return dataList, true
}

// 根据用户关注查询列表 - 鉴权
func (dao *EquipmentDAO) FindEquipmentByUser(eq model.Equipment, userId, offset, limit int) ([]model.Equipment, int64, bool) {
	equipments := []model.Equipment{}

	sqlStr := baseEquipmentSql(eq)

	// 带条件查询
	err := dao.db.Table("equipment").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr+" AND (create_user_id = ? or ? = any(concern_user_ids))", userId, userId).Scan(&equipments).Error
	if err != nil || len(equipments) == 0 {
		log.Println("[EquipmentDao] Failed to find user concern Equipment: ", err)
		return equipments, 0, false
	}
	var total int64
	dao.db.Table("equipment").Where(sqlStr+" AND (create_user_id = ? or ? = any(concern_user_ids))", userId, userId).Count(&total)

	return equipments, total, true
}

// 根据部门查询设备 - 鉴权
func (dao *EquipmentDAO) FindEquipmentByCorp(eq model.Equipment, corpIds []int, offset, limit int) ([]model.Equipment, int64, bool) {
	equipments := []model.Equipment{}

	sqlStr := baseEquipmentSql(eq)
	// 未分配部门的也查出
	corpIds = append(corpIds, 0)
	corpIds = normalutil.Unique(corpIds)

	// 带条件查询
	err := dao.db.Table("equipment").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr+" AND corp_id in ? ", corpIds).Scan(&equipments).Error
	if err != nil || len(equipments) == 0 {
		log.Println("[EquipmentDao] Failed to find corp bind Equipment: ", err)
		return equipments, 0, false
	}
	var total int64
	dao.db.Table("equipment").Where(sqlStr+" AND corp_id in ? ", corpIds).Count(&total)

	return equipments, total, true
}

// 根据部门 - 查询所有设备id
func (dao *EquipmentDAO) FindAllEquipmentIds(corpIds []int) ([]int, bool) {

	equipments := []model.Equipment{}
	equipmentIds := []int{}

	// 未分配部门的也查出
	corpIds = append(corpIds, 0)
	corpIds = normalutil.Unique(corpIds)

	// 带条件查询
	err := dao.db.Table("equipment").Where("  corp_id in ? ", corpIds).Scan(&equipments).Error
	if err != nil || len(equipments) == 0 {
		// log.Println("[EquipmentDao] Failed to find corp bind Equipment Ids: ", err)
		return equipmentIds, false
	}

	for _, vTem := range equipments {
		equipmentIds = append(equipmentIds, int(vTem.ID))
	}

	return equipmentIds, true
}

// 封装基本查询语句
func baseEquipmentSql(eq model.Equipment) (sqlStr string) {

	sqlStr = "1=1 "
	if eq.ID > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND id = %d ", eq.ID)
	}
	if len(eq.EquipmentName) > 0 {
		sqlStr = sqlStr + " AND equipment_name like '%" + eq.EquipmentName + "%' "
	}
	if eq.SiteId > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND site_id = %d ", eq.SiteId)
	}
	if len(eq.SiteName) > 0 {
		sqlStr = sqlStr + " AND site_name like '%" + eq.SiteName + "%' "
	}
	if eq.WarehouseId > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND warehouse_id = %d ", eq.WarehouseId)
	}
	if eq.PositionStatus > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND position_status = %d ", eq.PositionStatus)
	}
	if len(eq.EquipmentCode) > 0 {
		sqlStr = sqlStr + " AND equipment_code like '%" + eq.EquipmentCode + "%' "
	}
	if len(eq.Manufacturer) > 0 {
		sqlStr = sqlStr + " AND manufacturer like '%" + eq.Manufacturer + "%' "
	}
	if len(eq.BmsManufacturer) > 0 {
		sqlStr = sqlStr + " AND bms_manufacturer like '%" + eq.BmsManufacturer + "%' "
	}
	if len(eq.BmsVersion) > 0 {
		sqlStr = sqlStr + " AND bms_version like '%" + eq.BmsVersion + "%' "
	}
	if len(eq.EquipmentTypeName) > 0 {
		sqlStr = sqlStr + " AND equipment_type_name like '%" + eq.EquipmentTypeName + "%' "
	}

	return
}

// 查询设备名是否重名
func (dao *EquipmentDAO) EquipmentNameExist(et model.Equipment) (model.Equipment, bool) {
	// 新的容器
	eqpt := model.Equipment{}

	sqlStr := "1=1"
	if et.ID != 0 {
		// 除自身以外
		sqlStr = sqlStr + fmt.Sprintf(" AND id <> %d", et.ID)
	}
	if et.SiteId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND site_id = %d ", et.SiteId)
	}

	// 新结构体 判断查询值
	err := dao.db.Table("equipment").Where(sqlStr+" AND equipment_name = ?", et.EquipmentName).Scan(&eqpt).Error
	if err != nil || eqpt.ID == 0 {
		log.Println("[EquipmentDao] Failed to find one EQUIPMTN: ", err)
		return et, false
	}

	return eqpt, true
}

func (dao *EquipmentDAO) ExistEquipmentCode(c model.Equipment) (model.Equipment, bool) {
	cp := model.Equipment{}
	code := strings.TrimSpace(c.EquipmentCode)
	if code == "" {
		return c, false
	}

	err := dao.db.Model(&cp).Where("TRIM(equipment_code) = ? ", code).Scan(&cp).Error
	if err != nil || cp.ID == 0 {
		log.Printf("Failed EquipmentCode not exist: %v", err)
		return c, false
	}

	return cp, true
}

// 修改设备
func (dao *EquipmentDAO) UpdateEquipment(e model.Equipment) (model.Equipment, bool) {

	// 值更新非空部分
	err := dao.db.Table("equipment").Updates(&e).Error
	if err != nil {
		log.Println("[EquipmentDao] Failed to update EQUIPMENT: ", err)
		return e, false
	}

	return e, true
}

func (dao *EquipmentDAO) UpdateEquipmentAlarmTriggerNum(e model.Equipment) (model.Equipment, bool) {

	// 值更新非空部分
	err := dao.db.Table("equipment").Select("alarm_trigger_num").Updates(&e).Error
	if err != nil {
		log.Println("[EquipmentDao] Failed to update EQUIPMENT: ", err)
		return e, false
	}

	return e, true
}

func (dao *EquipmentDAO) DeleteEquipment(e model.Equipment) bool {

	err := dao.db.Table("equipment").Delete(&e).Error
	if err != nil {
		log.Println("[EquipmentDao] Failed to delete EQUIPMENT: ", err)
		return false
	}

	return true
}

// 根据站点id 查询设备
func (dao *EquipmentDAO) FindEquipmentBySiteId(siteId, offset int, limit int) ([]model.Equipment, bool) {
	equipments := []model.Equipment{}

	err := dao.db.Table("equipment").Offset(offset).Limit(limit).Order("id desc").Where("site_id = ?", siteId).Scan(&equipments).Error
	if err != nil || len(equipments) == 0 {
		log.Println("[EquipmentDao] Failed to find Equipment: ", err)
		return equipments, false
	}

	return equipments, true
}

// 根据id 查询设备
func (dao *EquipmentDAO) FindEquipmentById(equipment model.Equipment) (model.Equipment, bool) {

	err := dao.db.Find(&equipment).Order("id desc").Error
	if err != nil || len(equipment.EquipmentName) == 0 {
		log.Println("[EquipmentDao] Failed to find Equipment: ", err)
		return equipment, false
	}

	return equipment, true
}

// 根据id,和部门 查询设备
func (dao *EquipmentDAO) FindEquipmentByIdAndCorp(equipment model.Equipment, corpIds []int) (model.Equipment, bool) {

	err := dao.db.Where(" corp_id in ? ", corpIds).Order("id desc").Find(&equipment).Error
	if err != nil || len(equipment.EquipmentName) == 0 {
		log.Println("[EquipmentDao] Failed to find Equipment: ", err)
		return equipment, false
	}

	return equipment, true
}

// 根据部门查询设备 - 鉴权
func (dao *EquipmentDAO) FindEquipmentAllDataByCorp(eq model.Equipment, corpIds []int, offset, limit int) ([]model.Equipment, int64, bool) {
	equipments := []model.Equipment{}

	sqlStr := baseEquipmentSql(eq)
	// 未分配部门的也查出
	corpIds = append(corpIds, 0)
	corpIds = normalutil.Unique(corpIds)

	// 带条件查询
	err := dao.db.Table("equipment").Preload("DataDetails").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr+" AND corp_id in ? ", corpIds).Find(&equipments).Error
	if err != nil || len(equipments) == 0 {
		log.Println("[EquipmentDao] Failed to find corp bind Equipment: ", err)
		return equipments, 0, false
	}
	var total int64
	dao.db.Table("equipment").Where(sqlStr+" AND corp_id in ? ", corpIds).Count(&total)

	return equipments, total, true
}

// 查询列表
func (dao *EquipmentDAO) FindEquipmentAllDataAll(eq model.Equipment, offset int, limit int) ([]model.Equipment, int64, bool) {
	equipments := []model.Equipment{}

	sqlStr := baseEquipmentSql(eq)

	err := dao.db.Table("equipment").Preload("DataDetails").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr).Find(&equipments).Error
	if err != nil || len(equipments) == 0 {
		log.Println("[EquipmentDao] Failed to find Equipment: ", err)
		return equipments, 0, false
	}
	var total int64
	dao.db.Table("equipment").Where(sqlStr).Count(&total)

	return equipments, total, true
}

// 根据用户关注查询列表 - 鉴权
func (dao *EquipmentDAO) FindEquipmentAllDataByUser(eq model.Equipment, userId, offset, limit int) ([]model.Equipment, int64, bool) {
	equipments := []model.Equipment{}

	sqlStr := baseEquipmentSql(eq)

	// 带条件查询
	err := dao.db.Table("equipment").Preload("DataDetails").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr+" AND (create_user_id = ? or ? = any(concern_user_ids))", userId, userId).Find(&equipments).Error
	if err != nil || len(equipments) == 0 {
		log.Println("[EquipmentDao] Failed to find user concern Equipment: ", err)
		return equipments, 0, false
	}
	var total int64
	dao.db.Table("equipment").Where(sqlStr+" AND (create_user_id = ? or ? = any(concern_user_ids))", userId, userId).Count(&total)

	return equipments, total, true
}

func (dao *EquipmentDAO) FindEquipmentByIdArr(eq model.Equipment, idArr []int) ([]model.Equipment, bool) {
	equipments := []model.Equipment{}

	sqlStr := baseEquipmentSql(eq)

	err := dao.db.Table("equipment").Order("id").Where(sqlStr+" AND id in ?", idArr).Scan(&equipments).Error
	if err != nil || len(equipments) == 0 {
		log.Println("[EquipmentDao] Failed to find Equipment by ids: ", err)
		return equipments, false
	}

	return equipments, true
}

// FindEquipmentByCorpIDsExact returns registered equipment in exactly the
// requested departments. Unlike the legacy department query, it does not add
// unassigned equipment implicitly.
func (dao *EquipmentDAO) FindEquipmentByCorpIDsExact(corpIDs []int) ([]model.Equipment, error) {
	equipments := []model.Equipment{}
	query := dao.db.Table("equipment").Order("id")
	if corpIDs != nil {
		if len(corpIDs) == 0 {
			return equipments, nil
		}
		query = query.Where("corp_id IN ?", corpIDs)
	}
	if err := query.Scan(&equipments).Error; err != nil {
		return nil, err
	}
	return equipments, nil
}
