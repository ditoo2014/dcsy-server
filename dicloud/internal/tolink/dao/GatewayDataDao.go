package dao

import (
	"dicloud/internal/tolink/model"
	"dicloud/util/gormutil"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GatewayDataDao struct {
	db *gorm.DB
}

func NewGatewayDataDao(ctx *gin.Context) *GatewayDataDao {
	return &GatewayDataDao{db: gormutil.GetGormDB(ctx, "")}
}

// 查询一个 GatewayData 数据点
func (dao *GatewayDataDao) FindOne(gd model.GatewayData) (model.GatewayData, bool) {

	sqlStr := baseGatewayDataSql(gd)
	err := dao.db.Where(sqlStr).Find(&gd).Error
	if err != nil || gd.ID == 0 {
		log.Println("[GatewayDataDao] findone GatewayData Fail")
		return gd, false
	}

	return gd, true
}

// 基本查询sql封装
func baseGatewayDataSql(gd model.GatewayData) string {

	sqlStr := "1=1"

	if len(gd.GatewayId) > 0 {
		sqlStr = sqlStr + " AND gateway_id = '" + gd.GatewayId + "'"
	}
	if len(gd.DataId) > 0 {
		// 手动封装sql, 在字符串查询中加''单引号, 防止特殊符号引起歧义
		// 自动封装不用
		sqlStr = sqlStr + " AND data_id = '" + gd.DataId + "' "
	}
	if len(gd.Name) > 0 {
		sqlStr = sqlStr + " AND name = " + gd.Name
	}
	if gd.ID > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND id = %d", gd.ID)
	}

	return sqlStr
}

// 根据网关序列号,数据点名称 获取数据点
func (dao *GatewayDataDao) ExistDataName(gd model.GatewayData) (model.GatewayData, bool) {
	c := model.GatewayData{}

	err := dao.db.Table("gateway_data").Where("gateway_id = ? and name = ?", gd.GatewayId, gd.Name).Scan(&c).Error
	if err != nil || c.ID == 0 {
		log.Printf("Failed gateway DATA NAME not exist: %v", err)
		return c, false
	}

	return c, true
}

// 根据网关序列号,数据点id 获取数据点
func (dao *GatewayDataDao) ExistDataId(gd model.GatewayData) (model.GatewayData, bool) {
	c := model.GatewayData{}

	err := dao.db.Table("gateway_data").Where("gateway_id = ? and data_id = ?", gd.GatewayId, gd.DataId).Scan(&c).Error
	if err != nil || c.ID == 0 {
		log.Printf("Failed gateway DATA ID not exist: %v", err)
		return c, false
	}

	return c, true
}

// 根据数据库id 获取数据点
func (dao *GatewayDataDao) ExistData(gd model.GatewayData) (model.GatewayData, bool) {
	c := model.GatewayData{}

	err := dao.db.Table("gateway_data").Find(&gd).Error
	if err != nil || c.ID == 0 {
		log.Printf("Failed gateway DATA not exist: %v", err)
		return c, false
	}

	return c, true
}

// 创建数据点
func (dao *GatewayDataDao) CreateData(gd model.GatewayData) (model.GatewayData, bool) {
	c := model.GatewayData{}

	err := dao.db.Table("gateway_data").Create(&gd).Error
	if err != nil {
		log.Printf("Failed Create gateway DATA: %v", err)
		return c, false
	}

	return c, true
}

// 删除数据点
func (dao *GatewayDataDao) DeleteData(gd model.GatewayData) (model.GatewayData, bool) {
	c := model.GatewayData{}

	err := dao.db.Table("gateway_data").Where("gateway_id = ? and data_id = ?", gd.GatewayId, gd.DataId).Delete(&gd).Error
	if err != nil {
		log.Printf("Failed Delete gateway DATA: %v", err)
		return c, false
	}

	return c, true
}

// 根据数据库id 删除数据点
func (dao *GatewayDataDao) DeleteDataDirect(gd model.GatewayData) (model.GatewayData, bool) {
	c := model.GatewayData{}

	err := dao.db.Delete(&gd).Error
	if err != nil {
		log.Printf("Failed Delete gateway DATA: %v", err)
		return c, false
	}

	return c, true
}

// 根据数据库id 更新数据点
func (dao *GatewayDataDao) UpdateDataDirect(gd model.GatewayData) (model.GatewayData, bool) {
	c := model.GatewayData{}

	err := dao.db.Updates(&gd).Error
	if err != nil {
		log.Printf("Failed Update gateway DATA: %v", err)
		return c, false
	}

	return c, true
}

// 根据网关序列号,数据点id 更新数据点
func (dao *GatewayDataDao) UpdateData(gd model.GatewayData) (model.GatewayData, bool) {
	c := model.GatewayData{}

	err := dao.db.Table("gateway_data").Where("gateway_id = ? and data_id = ?", gd.GatewayId, gd.DataId).Updates(&gd).Error
	if err != nil {
		log.Printf("Failed Update gateway DATA: %v", err)
		return c, false
	}

	return c, true
}

// 自生成数据点ID
func (dao *GatewayDataDao) AutoIdGenerate(gatewayId string) (id string, err error) {

	/* 自生成数据点ID -- 100次尝试 */
	for _idx := 100; _idx > 0; _idx-- {

		id = fmt.Sprintf("%d", time.Now().UnixNano()%10000000000000000)

		// 是否存在随机生成的 data_id
		dataTem := model.GatewayData{DataId: id}
		err = dao.db.Model(&dataTem).Where("gateway_id = ? AND data_id = ?", gatewayId, id).First(&dataTem).Error
		if dataTem.ID > 0 {
			// 重复值
			id = ""
		}
		// 不存在, 跳出循环
		if id != "" {
			break
		}
	}

	return
}
