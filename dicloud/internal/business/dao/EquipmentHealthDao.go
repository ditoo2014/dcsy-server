package dao

import (
	"dicloud/internal/business/model"
	"dicloud/util/gormutil"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EquipmentHealthStandardDao struct {
	db *gorm.DB
}

func NewEquipmentHealthStandardDao(ctx *gin.Context) *EquipmentHealthStandardDao {
	return &EquipmentHealthStandardDao{db: gormutil.GetGormDB(ctx, "")}
}

// 创建
func (whd EquipmentHealthStandardDao) Create(wh model.EquipmentHealthStandard) (model.EquipmentHealthStandard, bool) {

	err := whd.db.Create(&wh).Error
	if err != nil {
		log.Println("[EquipmentHealthStandard] Dao create fail")
		return wh, false
	}

	return wh, true
}

// 根据id删除
func (whd EquipmentHealthStandardDao) Delete(wh model.EquipmentHealthStandard) bool {

	err := whd.db.Delete(&wh).Error
	if err != nil {
		log.Println("[EquipmentHealthStandard] Dao delete fail")
		return false
	}

	return true
}

// 根据id, 修改
func (whd EquipmentHealthStandardDao) Update(wh model.EquipmentHealthStandard) (model.EquipmentHealthStandard, bool) {

	res := whd.db.Updates(&wh)
	if res.RowsAffected == 0 {
		log.Println("[EquipmentHealthStandard] Dao update fail rows affected = 0")
		return wh, false
	}

	return wh, true
}

// 查询
func (whd EquipmentHealthStandardDao) Find() (model.EquipmentHealthStandard, bool) {
	dataList := model.EquipmentHealthStandard{}

	err := whd.db.First(&dataList).Error
	if err != nil {
		log.Println("[EquipmentHealthStandard] Dao Find fail")
		return dataList, false
	}

	return dataList, true
}
