package dao

import (
	"dicloud/internal/tolink/model"
	"dicloud/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CommonDataDAO struct {
	db *gorm.DB
}

// 创建新的 GatewayDAO 实例
func NewCommonDataDAO(ctx *gin.Context) *CommonDataDAO {
	return &CommonDataDAO{db: gormutil.GetGormDB(ctx, "")}
}

// 添加通用网关基本数据 - 不匹配机型, 不下发, 只查看
func (dao *CommonDataDAO) AddTolinkCommonDataAll(g model.Gateway, gSTR model.Gateway, s model.Stream) (bool, error) {
	tx := dao.db.Begin()

	// 查重, 是否存在, 存在则更新
	/**
	=========================
	== 网关
	=========================
	*/
	var gTem = model.Gateway{}
	tx.Model(&g).Where("gateway_id", g.GatewayId).Find(&gTem)
	if gTem.ID > 0 {
		// 更新
		g.ID = gTem.ID
		err := tx.Model(&g).Updates(&g).Error
		if err != nil {
			log.Printf("[TolinkCommonDataDao] Failed to Update gateway: %v", err)
			tx.Rollback()
			return false, err
		}
	} else {

		// 添加网关
		err := tx.Model(&g).Create(&g).Error
		if err != nil {
			log.Printf("[TolinkCommonDataDao] Failed to add gateway: %v", err)
			tx.Rollback()
			return false, err
		}
	}

	var gSTRTem = model.Gateway{}
	tx.Model(&gSTR).Where("gateway_id", gSTR.GatewayId).Find(&gSTRTem)
	if gSTRTem.ID > 0 {
		// 更新
		gSTR.ID = gSTRTem.ID
		err := tx.Model(&gSTR).Updates(&gSTR).Error
		if err != nil {
			log.Printf("[TolinkCommonDataDao] Failed to Update gatewaySTR: %v", err)
			tx.Rollback()
			return false, err
		}
	} else {

		// 添加网关
		err := tx.Model(&gSTR).Create(&gSTR).Error
		if err != nil {
			log.Printf("[TolinkCommonDataDao] Failed to add gatewaySTR: %v", err)
			tx.Rollback()
			return false, err
		}
	}

	/**
	=========================
	== 网关数据点
	=========================
	*/
	if len(g.GatewayDatas) > 0 {
		for i, dataTem := range g.GatewayDatas {

			gdTem := model.GatewayData{}
			// 查重
			tx.Model(&gdTem).Where("gateway_id = ? AND data_id = ?", dataTem.GatewayId, dataTem.DataId).Find(&gdTem)
			if gdTem.ID > 0 {
				// log.Println("Gateway already exist data", dataTem.Name)
				continue
			}

			// 创建 GatewayData 数据点
			if err := tx.Model(&model.GatewayData{}).Create(&dataTem).Error; err != nil {
				fmt.Printf("[TolinkCommonDataDao] 创建 GatewayData[%d] 失败: %v\nSQL: %s\n",
					i, err,
					tx.Statement.SQL.String())
				tx.Rollback()
				return false, err
			}
		}
	}

	if len(gSTR.GatewayDatas) > 0 {
		for i, dataTem := range gSTR.GatewayDatas {

			gdTem := model.GatewayData{}
			// 查重
			tx.Model(&gdTem).Where("gateway_id = ? AND data_id = ?", dataTem.GatewayId, dataTem.DataId).Find(&gdTem)
			if gdTem.ID > 0 {
				// log.Println("Gateway already exist data", dataTem.Name)
				continue
			}

			// 创建 GatewayData 数据点
			if err := tx.Model(&model.GatewayData{}).Create(&dataTem).Error; err != nil {
				fmt.Printf("[TolinkCommonDataDao] 创建 STR GatewayData[%d] 失败: %v\nSQL: %s\n",
					i, err,
					tx.Statement.SQL.String())
				tx.Rollback()
				return false, err
			}
		}
	}

	/**
	====================
	== 数据池
	====================
	*/
	var sTem = model.Stream{}
	tx.Model(&sTem).Where("stream_code = ? ", s.StreamCode).Find(&sTem)
	if sTem.ID > 0 {
		s.ID = sTem.ID
		err := tx.Model(&sTem).Updates(&s).Error
		if err != nil {
			log.Printf("[TolinkCommonDataDao] Failed to update stream: %v", err)
			tx.Rollback()
			return false, err
		}
	} else {

		// 添加数据池
		err := tx.Create(&s).Error
		if err != nil {
			log.Printf("[TolinkCommonDataDao] Failed to add stream: %v", err)
			tx.Rollback()
			return false, err
		}
	}

	/**
	====================
	== 数据池数据点
	====================
	*/
	if len(s.List) > 0 {
		for i, unitTem := range s.List {
			unitTem.StreamId = s.ID

			suTem := model.StreamLabelUnit{}
			// 查重 - 通过stream_id和data_id判断重复数据
			tx.Model(&suTem).Where("stream_id = ? AND gateway_id = ? AND data_id = ?", unitTem.StreamId, unitTem.GatewayId, unitTem.DataId).Find(&suTem)
			if len(suTem.DataName) > 0 {
				// log.Println("Stream already exist data", unitTem.Name)
				continue
			}

			// 创建 StreamUnitLabel 数据点
			if err := tx.Model(&model.StreamLabelUnit{}).Create(&unitTem).Error; err != nil {
				fmt.Printf("[TolinkCommonDataDao] 创建 StreamLabelUnit[%d] 失败: %v\nSQL: %s\n",
					i, err,
					tx.Statement.SQL.String())
				tx.Rollback()
				return false, err
			}
		}
	}

	// 提交所有数据
	tx.Commit()
	return true, nil
}

// 删除通用网关基本数据 - 网关/网关数据/数据池/数据池数据
func (dao *CommonDataDAO) DeleteTolinkCommonDataAll(g model.Gateway) (bool, error) {

	tx := dao.db.Begin()

	/**
	=========================
	== 网关
	=========================
	*/
	err := tx.Model(&g).Delete(&g).Error
	if err != nil {
		log.Printf("[TolinkCommonDataDao] Failed to Delete gateway: %v", err)
		tx.Rollback()
		return false, err
	}

	/**
	=========================
	== 网关数据点
	=========================
	*/
	gdTem := model.GatewayData{
		GatewayId: g.GatewayId,
	}
	err = tx.Model(&gdTem).Where("gateway_id", g.GatewayId).Delete(&gdTem).Error
	if err != nil {
		fmt.Printf("[TolinkCommonDataDao] 删除 GatewayData 失败: %v\nSQL: %s\n",
			err,
			tx.Statement.SQL.String())
		tx.Rollback()
		return false, err
	}

	/**
	====================
	== 数据池
	====================
	*/
	var sTem = model.Stream{StreamCode: g.GatewayId}
	err = tx.Model(&sTem).Where("stream_code = ? ", g.GatewayId).Delete(&sTem).Error
	if err != nil {
		log.Printf("[TolinkCommonDataDao] Failed to delete stream: %v", err)
		tx.Rollback()
		return false, err
	}

	/**
	====================
	== 数据池数据点
	====================
	*/
	suTem := model.StreamLabelUnit{GatewayId: g.GatewayId}
	// 查重 - 通过stream_id和data_id判断重复数据
	err = tx.Model(&suTem).Where("gateway_id = ? ", g.GatewayId).Delete(&suTem).Error
	if err != nil {
		fmt.Printf("[TolinkCommonDataDao] 删除 StreamLabelUnit 失败: %v\nSQL: %s\n",
			err,
			tx.Statement.SQL.String())
		tx.Rollback()
		return false, err
	}

	// 提交所有数据
	tx.Commit()
	return true, nil
}
