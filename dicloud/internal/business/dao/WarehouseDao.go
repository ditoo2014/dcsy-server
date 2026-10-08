package dao

import (
	"dicloud/internal/business/model"
	"dicloud/util/gormutil"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type WarehouseDao struct {
	db *gorm.DB
}

func NewWarehouseDao(ctx *gin.Context) *WarehouseDao {
	return &WarehouseDao{db: gormutil.GetGormDB(ctx, "")}
}

// 创建
func (whd WarehouseDao) Create(wh model.Warehouse) (model.Warehouse, bool) {

	err := whd.db.Create(&wh).Error
	if err != nil {
		log.Println("[Warehouse] Dao create fail")
		return wh, false
	}

	return wh, true
}

// 创建库房列表
func (dao *WarehouseDao) CreateWarehouses(whs []model.Warehouse) ([]model.Warehouse, bool) {

	err := dao.db.Create(&whs).Error
	if err != nil {
		log.Println("Failed to create WAREHOUSE list: ", err)
		return whs, false
	}

	return whs, true
}

func (whd WarehouseDao) ExistName(wh model.Warehouse) (model.Warehouse, bool) {
	existing := model.Warehouse{}
	name := strings.TrimSpace(wh.Name)
	if name == "" {
		return existing, false
	}

	query := whd.db.Table("warehouse").Where("TRIM(name) = ?", name)
	if wh.ID > 0 {
		query = query.Where("id <> ?", wh.ID)
	}
	if err := query.Scan(&existing).Error; err != nil || existing.ID == 0 {
		if err != nil {
			log.Println("[Warehouse] Dao find warehouse by name fail: ", err)
		}
		return existing, false
	}

	return existing, true
}

// 根据id删除
func (whd WarehouseDao) Delete(wh model.Warehouse) bool {

	err := whd.db.Delete(&wh).Error
	if err != nil {
		log.Println("[Warehouse] Dao delete fail")
		return false
	}

	return true
}

// 根据id, 修改
func (whd WarehouseDao) Update(wh model.Warehouse) (model.Warehouse, bool) {

	res := whd.db.Updates(&wh)
	if res.RowsAffected == 0 {
		log.Println("[Warehouse] Dao update fail rows affected = 0")
		return wh, false
	}

	return wh, true
}

// 查询库房信息列表
func (whd WarehouseDao) List(wh model.Warehouse, offset, limit int) ([]model.Warehouse, int64, bool) {
	dataList := []model.Warehouse{}

	sqlStr := baseWarehouseSql(wh)

	err := whd.db.Table("warehouse").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr).Scan(&dataList).Error
	if err != nil {
		log.Println("[Warehouse] Dao List fail")
		return dataList, 0, false
	}
	var total int64
	whd.db.Table("warehouse").Where(sqlStr).Count(&total)

	return dataList, total, true
}

// 通过部门筛选
func (whd WarehouseDao) ListByCorp(wh model.Warehouse, offset, limit int, corpIds []int) ([]model.Warehouse, int64, bool) {
	dataList := []model.Warehouse{}

	sqlStr := baseWarehouseSql(wh)

	err := whd.db.Table("warehouse").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr+" AND corp_id in ?", corpIds).Scan(&dataList).Error
	if err != nil {
		log.Println("[Warehouse] Dao List fail")
		return dataList, 0, false
	}
	var total int64
	whd.db.Table("warehouse").Where(sqlStr+" AND corp_id in ?", corpIds).Count(&total)

	return dataList, total, true
}

// 基本查询sql封装
func baseWarehouseSql(wh model.Warehouse) string {

	sqlStr := "1=1 "
	if wh.ID > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND warehouse.id = %d ", wh.ID)
	}
	if len(wh.Name) > 0 {
		sqlStr = sqlStr + " AND name LIKE '%" + wh.Name + "%' "
	}
	if len(wh.WarehouseCode) > 0 {
		sqlStr = sqlStr + " AND warehouse_code LIKE '%" + wh.WarehouseCode + "%' "
	}

	return sqlStr
}

func (dao *WarehouseDao) ExistWarehouseCode(c model.Warehouse) (model.Warehouse, bool) {
	cp := model.Warehouse{}
	code := strings.TrimSpace(c.WarehouseCode)
	if code == "" {
		return c, false
	}

	err := dao.db.Model(&cp).Where("TRIM(warehouse_code) = ? ", code).Scan(&cp).Error
	if err != nil || cp.ID == 0 {
		log.Printf("Failed WarehouseCode not exist: %v", err)
		return c, false
	}

	return cp, true
}

// 导入站点
func (dao *WarehouseDao) ImportWarehouses(eqs, ueqs []model.Warehouse) ([]model.Warehouse, bool) {
	err := dao.db.Transaction(func(tx *gorm.DB) error {
		if len(eqs) > 0 {
			if err := tx.Create(&eqs).Error; err != nil {
				return err
			}
		}
		for _, v := range ueqs {
			if err := tx.Model(&model.Warehouse{}).Where("warehouse_code = ?", v.WarehouseCode).Updates(&v).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Println("Failed to import Warehouse list: ", err)
		return eqs, false
	}

	return eqs, true
}
