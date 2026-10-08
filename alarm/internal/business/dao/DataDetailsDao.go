package dao

import (
	"alarm/internal/business/model"
	"alarm/util/gormutil"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DataDetailsDao struct {
	db *gorm.DB
}

// 创建新的 DataDetailsDao 实例
func NewDataDetailsDao(ctx *gin.Context) *DataDetailsDao {
	return &DataDetailsDao{db: gormutil.GetGormDB(ctx, "")}
}

func (dao *DataDetailsDao) FindDataDetails(dd model.DataDetails) ([]model.DataDetails, bool) {
	ddss := []model.DataDetails{}

	err := dao.db.Order("sort_num").Find(&ddss, dd).Error
	if err != nil {
		log.Println("[DataDetailsDao] find list failed")
		return ddss, false
	}

	return ddss, true
}

func (dao *DataDetailsDao) FindDataDetailsByLabelNameOrName(labelName string) ([]model.DataDetails, bool) {
	ddss := []model.DataDetails{}

	err := dao.db.Order("sort_num").Where("label_name = ? OR name = ?", labelName, labelName).Find(&ddss).Error
	if err != nil {
		log.Println("[DataDetailsDao] find list by label_name or name failed")
		return ddss, false
	}

	return ddss, true
}

func (dao *DataDetailsDao) AddDataDetails(dds model.DataDetails) (model.DataDetails, bool) {

	err := dao.db.Create(&dds).Error
	if err != nil {
		log.Println("[DataDetailsDao] create fail")
		return dds, false
	}

	return dds, true
}

// 批量添加
func (dao *DataDetailsDao) AddDataDetailsList(dds []model.DataDetails) ([]model.DataDetails, bool) {

	err := dao.db.Create(&dds).Error
	if err != nil {
		log.Println("[DataDetailsDao] batch create fail")
		return dds, false
	}

	return dds, true
}

func (dao *DataDetailsDao) UpdateDataDetails(dds model.DataDetails) bool {

	// 更新
	err := dao.db.Updates(&dds).Error
	if err != nil {
		log.Println("[DataDetailsDao] update fail")
		return false
	}

	return true
}

// 更新
func (dao *DataDetailsDao) UpdateDataDetailsList(dds []model.DataDetails) bool {

	// 循环遍历更新
	for _, v := range dds {
		if v.ID == 0 {
			log.Println("[DataDetailsDao] batch update fail: id is zero")
			return false
		}

		// 更新
		err := dao.db.Updates(&v).Error
		if err != nil {
			log.Println("[DataDetailsDao] batch update fail")
			return false
		}
	}

	return true
}

// 根据id删除
func (dao *DataDetailsDao) DeleteDataDetails(dds model.DataDetails) (model.DataDetails, bool) {

	// 删除
	err := dao.db.Delete(&dds).Error
	if err != nil {
		log.Println("[DataDetailsDao] delete fail")
		return dds, false
	}

	return dds, true
}
