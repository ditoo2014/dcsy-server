package controller

import (
	"strings"

	"dicloud/internal/business/model"
	"dicloud/util/gormutil"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type IEquipmentTypeCtrler interface {
	Add(ctx *gin.Context)
	Find(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

type EquipmentTypeCtrler struct{}

func (EquipmentTypeCtrler) Add(ctx *gin.Context) {
	var item model.EquipmentType
	if err := ctx.ShouldBindJSON(&item); err != nil {
		response.Fail(ctx, "请求参数格式错误", nil)
		return
	}
	item.EquipmentTypeName = strings.TrimSpace(item.EquipmentTypeName)
	item.EquipmentTypeDescribe = strings.TrimSpace(item.EquipmentTypeDescribe)
	if item.EquipmentTypeName == "" {
		response.Fail(ctx, "设备类型名称不能为空", nil)
		return
	}
	db := gormutil.GetGormDB(ctx, "")
	if err := db.Create(&item).Error; err != nil {
		response.Fail(ctx, "设备类型已存在或保存失败", nil)
		return
	}
	response.Success(ctx, "添加成功", gin.H{"data": item})
}

func (EquipmentTypeCtrler) Find(ctx *gin.Context) {
	db := gormutil.GetGormDB(ctx, "")
	name := strings.TrimSpace(ctx.Query("equipmentTypeName"))
	query := db.Model(&model.EquipmentType{})
	if name != "" {
		query = query.Where("equipment_type_name LIKE ?", "%"+name+"%")
	}
	var list []model.EquipmentType
	if err := query.Order("id ASC").Find(&list).Error; err != nil {
		response.Fail(ctx, "查询设备类型失败", nil)
		return
	}

	// 兼容迁移前只存在于设备表中的类型名称，并补入字典表。
	if name == "" {
		var legacyNames []string
		if err := db.Model(&model.Equipment{}).Distinct().
			Where("equipment_type_name IS NOT NULL AND trim(equipment_type_name) <> ''").
			Pluck("equipment_type_name", &legacyNames).Error; err == nil {
			known := make(map[string]bool, len(list))
			for _, item := range list {
				known[item.EquipmentTypeName] = true
			}
			for _, legacyName := range legacyNames {
				legacyName = strings.TrimSpace(legacyName)
				if legacyName == "" || legacyName == "-" || known[legacyName] {
					continue
				}
				item := model.EquipmentType{EquipmentTypeName: legacyName}
				if err := db.Where("equipment_type_name = ?", legacyName).FirstOrCreate(&item).Error; err == nil {
					list = append(list, item)
					known[legacyName] = true
				}
			}
		}
	}
	response.Success(ctx, "查询成功", gin.H{"list": list})
}

func (EquipmentTypeCtrler) Update(ctx *gin.Context) {
	var request model.EquipmentType
	if err := ctx.ShouldBindJSON(&request); err != nil || request.ID == 0 {
		response.Fail(ctx, "设备类型ID不能为空", nil)
		return
	}
	request.EquipmentTypeName = strings.TrimSpace(request.EquipmentTypeName)
	request.EquipmentTypeDescribe = strings.TrimSpace(request.EquipmentTypeDescribe)
	if request.EquipmentTypeName == "" {
		response.Fail(ctx, "设备类型名称不能为空", nil)
		return
	}
	db := gormutil.GetGormDB(ctx, "")
	err := db.Transaction(func(tx *gorm.DB) error {
		var current model.EquipmentType
		if err := tx.First(&current, request.ID).Error; err != nil {
			return err
		}
		if current.EquipmentTypeName != request.EquipmentTypeName {
			if err := tx.Model(&model.Equipment{}).
				Where("equipment_type_name = ?", current.EquipmentTypeName).
				Update("equipment_type_name", request.EquipmentTypeName).Error; err != nil {
				return err
			}
		}
		return tx.Model(&current).Updates(map[string]any{
			"equipment_type_name":     request.EquipmentTypeName,
			"equipment_type_describe": request.EquipmentTypeDescribe,
		}).Error
	})
	if err != nil {
		response.Fail(ctx, "修改设备类型失败", nil)
		return
	}
	response.Success(ctx, "修改成功", nil)
}

func (EquipmentTypeCtrler) Delete(ctx *gin.Context) {
	var request model.EquipmentType
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Fail(ctx, "请求参数格式错误", nil)
		return
	}
	db := gormutil.GetGormDB(ctx, "")
	var current model.EquipmentType
	query := db
	if request.ID > 0 {
		query = query.Where("id = ?", request.ID)
	} else {
		query = query.Where("equipment_type_name = ?", strings.TrimSpace(request.EquipmentTypeName))
	}
	if err := query.First(&current).Error; err != nil {
		response.Fail(ctx, "设备类型不存在", nil)
		return
	}
	var count int64
	if err := db.Model(&model.Equipment{}).Where("equipment_type_name = ?", current.EquipmentTypeName).Count(&count).Error; err != nil {
		response.Fail(ctx, "检查设备类型引用失败", nil)
		return
	}
	if count > 0 {
		response.Fail(ctx, "该类型仍被设备使用，不能删除", nil)
		return
	}
	if err := db.Delete(&current).Error; err != nil {
		response.Fail(ctx, "删除设备类型失败", nil)
		return
	}
	response.Success(ctx, "删除成功", nil)
}
