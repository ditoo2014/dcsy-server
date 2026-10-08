package dao

import (
	"dicloud/internal/business/model"
	"dicloud/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EquipmentGeographyPositionDao struct {
	db *gorm.DB
}

func NewEquipmentGeograpyhPositionDao(ctx *gin.Context) *EquipmentGeographyPositionDao {
	return &EquipmentGeographyPositionDao{db: gormutil.GetGormDB(ctx, "")}
}

func (dao *EquipmentGeographyPositionDao) Create(egp model.EquipmentGeographyPosition) (model.EquipmentGeographyPosition, bool) {

	err := dao.db.Create(&egp).Error
	if err != nil {
		log.Println("[EquipmentGeographyPositionDao] Create fail")
		return egp, false
	}

	return egp, true
}

func (dao *EquipmentGeographyPositionDao) Find(egp model.EquipmentGeographyPosition) ([]model.EquipmentGeographyPosition, bool) {
	egpList := []model.EquipmentGeographyPosition{}

	sqlStr := "1=1 "
	if egp.EquipmentId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND equipment_id = %d ", egp.EquipmentId)
	}

	err := dao.db.Model(&egp).Where(sqlStr).Find(&egpList).Error
	if err != nil {
		log.Println("[EquipmentGeographyPositionDao] Find fail")
		return egpList, false
	}

	return egpList, true
}
