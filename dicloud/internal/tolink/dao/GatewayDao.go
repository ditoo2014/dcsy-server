package dao

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"dicloud/internal/tolink/model"
	"dicloud/util/gormutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GatewayDAO struct {
	db *gorm.DB
}

func NewGatewayDAO(ctx *gin.Context) *GatewayDAO {
	return &GatewayDAO{db: gormutil.GetGormDB(ctx, "")}
}

func (dao *GatewayDAO) AddGateway(gateway model.Gateway) (model.Gateway, error) {
	tx := dao.db.Begin()
	if tx.Error != nil {
		return gateway, tx.Error
	}
	if err := tx.Create(&gateway).Error; err != nil {
		tx.Rollback()
		return gateway, err
	}
	if tx.Migrator().HasTable(&model.TtdcConfigRevision{}) {
		revision := model.TtdcConfigRevision{ConfigType: "gateway", TargetKey: gateway.GatewayId, Operation: "create"}
		if err := tx.Create(&revision).Error; err != nil {
			tx.Rollback()
			return gateway, err
		}
	}
	return gateway, tx.Commit().Error
}

func (dao *GatewayDAO) DelGateway(gatewayID string) error {
	tx := dao.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	var gateway model.Gateway
	if err := tx.Where("gateway_id = ? AND model = ?", gatewayID, model.TTDC).First(&gateway).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("gateway_id = ?", gatewayID).Delete(&model.GatewayData{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	result := tx.Where("gateway_id = ? AND model = ?", gatewayID, model.TTDC).Delete(&model.Gateway{})
	if result.Error != nil || result.RowsAffected == 0 {
		tx.Rollback()
		if result.Error != nil {
			return result.Error
		}
		return gorm.ErrRecordNotFound
	}
	if tx.Migrator().HasTable(&model.TtdcConfigRevision{}) {
		revision := model.TtdcConfigRevision{ConfigType: "gateway", TargetKey: gatewayID, Operation: "delete"}
		if err := tx.Create(&revision).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (dao *GatewayDAO) UpdateGateway(gateway model.Gateway) error {
	tx := dao.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	var existing model.Gateway
	if err := tx.Where("gateway_id = ? AND model = ?", gateway.GatewayId, model.TTDC).First(&existing).Error; err != nil {
		tx.Rollback()
		return err
	}
	gateway.Model = model.TTDC
	result := tx.Model(&existing).Omit("ID", "GatewayId", "CreateUserId").Updates(&gateway)
	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if tx.Migrator().HasTable(&model.TtdcConfigRevision{}) {
		revision := model.TtdcConfigRevision{ConfigType: "gateway", TargetKey: gateway.GatewayId, Operation: "update"}
		if err := tx.Create(&revision).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (dao *GatewayDAO) GetByGatewayID(gatewayID string) (model.Gateway, error) {
	var gateway model.Gateway
	result := dao.db.Where("gateway_id = ?", strings.TrimSpace(gatewayID)).First(&gateway)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return model.Gateway{}, fmt.Errorf("网关不存在: gatewayId=%s", gatewayID)
	}
	if result.Error != nil {
		return model.Gateway{}, fmt.Errorf("查询网关失败: %w", result.Error)
	}
	return gateway, nil
}

func (dao *GatewayDAO) ExistName(gateway model.Gateway) (model.Gateway, bool) {
	var existing model.Gateway
	err := dao.db.Where("name = ? AND model = ?", strings.TrimSpace(gateway.Name), model.TTDC).First(&existing).Error
	return existing, err == nil
}

func (dao *GatewayDAO) ExistID(gateway model.Gateway) (model.Gateway, bool) {
	var existing model.Gateway
	err := dao.db.Where("gateway_id = ?", strings.TrimSpace(gateway.GatewayId)).First(&existing).Error
	return existing, err == nil
}

func (dao *GatewayDAO) FindGatewayList(filter model.Gateway, offset, limit int) (list []model.Gateway, total int64, ok bool) {
	query := dao.db.Model(&model.Gateway{}).Where("model = ?", model.TTDC)
	if filter.ID > 0 {
		query = query.Where("id = ?", filter.ID)
	}
	if value := strings.TrimSpace(filter.GatewayId); value != "" {
		query = query.Where("gateway_id LIKE ?", "%"+value+"%")
	}
	if value := strings.TrimSpace(filter.Card); value != "" {
		query = query.Where("card LIKE ?", "%"+value+"%")
	}
	if value := strings.TrimSpace(filter.Name); value != "" {
		query = query.Where("name LIKE ?", "%"+value+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		log.Printf("[GatewayDAO] 统计TTDC网关失败: %v", err)
		return nil, 0, false
	}
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		log.Printf("[GatewayDAO] 查询TTDC网关失败: %v", err)
		return nil, 0, false
	}
	return list, total, true
}

func (dao *GatewayDAO) FindGatewayDatas(gateway model.Gateway) (model.Gateway, bool) {
	var result model.Gateway
	err := dao.db.Preload("GatewayDatas").
		Where("gateway_id = ? AND model = ?", gateway.GatewayId, model.TTDC).
		First(&result).Error
	if err != nil {
		log.Printf("[GatewayDAO] 查询TTDC网关点位失败: %v", err)
		return result, false
	}
	return result, true
}
