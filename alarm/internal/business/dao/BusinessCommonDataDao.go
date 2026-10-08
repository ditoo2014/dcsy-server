package dao

import (
	"alarm/internal/business/model"
	"alarm/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BusinessCommonDataDao struct {
	db *gorm.DB
}

// 创建新的 BusinessCommonDataDao 实例
func NewBusinessCommonDataDao(ctx *gin.Context) *BusinessCommonDataDao {
	return &BusinessCommonDataDao{db: gormutil.GetGormDB(ctx, "")}
}

func (dao *BusinessCommonDataDao) AddBusinessCommonDataAll(e *model.Equipment) (bool, *gorm.DB) {
	tx := dao.db.Begin()

	eTem := model.Equipment{}
	// 查重
	tx.Model(&eTem).Where("equipment_code = ?", e.EquipmentCode).Find(&eTem)
	if eTem.ID > 0 {
		// 更新设备
		e.ID = eTem.ID
		err := tx.Model(&eTem).Updates(e).Error
		if err != nil {
			log.Println("[BusinessCommonDataDao] update Equipment fail: ", err)
			tx.Rollback()
			return false, tx
		}
	} else {

		// 创建设备
		err := tx.Model(model.Equipment{}).Create(e).Error
		if err != nil {
			log.Println("[BusinessCommonDataDao] create Equipment fail: ", err)
			tx.Rollback()
			return false, tx
		}
	}

	if len(e.DataDetails) > 0 {
		for i, v := range e.DataDetails {
			v.EquipmentId = int(e.ID)

			ddTem := model.DataDetails{}
			// 查重 - 通过名称查重, 不知道data_id数据
			tx.Model(&ddTem).Where("equipment_id = ? AND label_name = ?", v.EquipmentId, v.LabelName).Find(&ddTem)
			if ddTem.ID > 0 {
				// log.Println("DataDetails already exist data", ddTem.Name)
				continue
			}

			// 创建 DataDetails 数据点
			if err := tx.Model(&ddTem).Create(&v).Error; err != nil {
				fmt.Printf("[BusinessCommonDataDao] 创建 DataDetails[%d] 失败: %v\nSQL: %s\n",
					i, err,
					tx.Statement.SQL.String())
				tx.Rollback()
				return false, tx
			}
		}
	}

	return true, tx
}

func (dao *BusinessCommonDataDao) DeleteBusinessCommonDataAll(e *model.Equipment) (bool, *gorm.DB) {

	tx := dao.db.Begin()

	// 查重
	err := tx.Model(&e).Delete(&e).Error
	if err != nil {
		log.Println("[BusinessCommonDataDao] delete Equipment fail: ", err)
		tx.Rollback()
		return false, tx
	}

	var ddTem = model.DataDetails{}
	err = tx.Model(&ddTem).Where("equipment_id = ?", e.ID).Delete(&ddTem).Error
	if err != nil {
		log.Println("[BusinessCommonDataDao] delete DataDetails fail: ", err)
		tx.Rollback()
		return false, tx
	}

	return true, tx
}
