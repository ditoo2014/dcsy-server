package controller

import (
	"errors"
	"sort"
	"strings"

	businessmodel "dicloud/internal/business/model"
	authmodel "dicloud/models/model"
	"dicloud/util/gormutil"
	"dicloud/util/response"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// IUserEquipmentPermissionCtrler 管理用户对设备的关注和控制授权。
type IUserEquipmentPermissionCtrler interface {
	AddConcern(ctx *gin.Context)
	DeleteConcern(ctx *gin.Context)
	FindConcern(ctx *gin.Context)
	AddControl(ctx *gin.Context)
	DeleteControl(ctx *gin.Context)
	FindControl(ctx *gin.Context)
}

type UserEquipmentPermissionCtrler struct{}

type userEquipmentPermissionRequest struct {
	GrantedUser  string `json:"grantedUser"`
	EquipmentIDs []int  `json:"equipmentIds"`
}

func (UserEquipmentPermissionCtrler) AddConcern(ctx *gin.Context) {
	updateUserEquipmentPermission(ctx, "concern_user_ids", true)
}

func (UserEquipmentPermissionCtrler) DeleteConcern(ctx *gin.Context) {
	updateUserEquipmentPermission(ctx, "concern_user_ids", false)
}

func (UserEquipmentPermissionCtrler) FindConcern(ctx *gin.Context) {
	findUserEquipmentPermission(ctx, true)
}

func (UserEquipmentPermissionCtrler) AddControl(ctx *gin.Context) {
	updateUserEquipmentPermission(ctx, "control_user_ids", true)
}

func (UserEquipmentPermissionCtrler) DeleteControl(ctx *gin.Context) {
	updateUserEquipmentPermission(ctx, "control_user_ids", false)
}

func (UserEquipmentPermissionCtrler) FindControl(ctx *gin.Context) {
	findUserEquipmentPermission(ctx, false)
}

func updateUserEquipmentPermission(ctx *gin.Context, column string, add bool) {
	request, userID, db, ok := prepareUserEquipmentPermission(ctx, true)
	if !ok {
		return
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		var equipments []businessmodel.Equipment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ?", request.EquipmentIDs).Find(&equipments).Error; err != nil {
			return err
		}
		if len(equipments) != len(uniquePositiveIDs(request.EquipmentIDs)) {
			return errors.New("部分设备不存在")
		}
		for _, equipment := range equipments {
			values := equipment.ConcernUserIds
			if column == "control_user_ids" {
				values = equipment.ControlUserIds
			}
			values = changeInt64Set(values, userID, add)
			if err := tx.Model(&businessmodel.Equipment{}).Where("id = ?", equipment.ID).
				Update(column, values).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		response.Fail(ctx, err.Error(), nil)
		return
	}
	response.Success(ctx, "设备授权已更新", nil)
}

func findUserEquipmentPermission(ctx *gin.Context, concern bool) {
	_, userID, db, ok := prepareUserEquipmentPermission(ctx, false)
	if !ok {
		return
	}
	var equipments []businessmodel.Equipment
	if err := db.Select("id", "concern_user_ids", "control_user_ids").Find(&equipments).Error; err != nil {
		response.Fail(ctx, "查询设备授权失败", nil)
		return
	}
	ids := make([]int, 0)
	for _, equipment := range equipments {
		values := equipment.ControlUserIds
		if concern {
			values = equipment.ConcernUserIds
		}
		if containsInt64(values, userID) {
			ids = append(ids, int(equipment.ID))
		}
	}
	sort.Ints(ids)
	response.Success(ctx, "查询成功", gin.H{"data": ids})
}

func prepareUserEquipmentPermission(ctx *gin.Context, requireEquipmentIDs bool) (userEquipmentPermissionRequest, int64, *gorm.DB, bool) {
	var request userEquipmentPermissionRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Fail(ctx, "请求参数格式错误", nil)
		return request, 0, nil, false
	}
	request.GrantedUser = strings.TrimSpace(request.GrantedUser)
	if request.GrantedUser == "" {
		response.Fail(ctx, "授权用户不能为空", nil)
		return request, 0, nil, false
	}
	request.EquipmentIDs = uniquePositiveIDs(request.EquipmentIDs)
	if requireEquipmentIDs && len(request.EquipmentIDs) == 0 {
		response.Fail(ctx, "设备列表不能为空", nil)
		return request, 0, nil, false
	}
	db := gormutil.GetGormDB(ctx, "")
	if db == nil {
		response.Fail(ctx, "数据库未初始化", nil)
		return request, 0, nil, false
	}
	var user authmodel.User
	if err := db.Where("username = ?", request.GrantedUser).First(&user).Error; err != nil {
		response.Fail(ctx, "授权用户不存在", nil)
		return request, 0, nil, false
	}
	return request, int64(user.ID), db, true
}

func uniquePositiveIDs(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func changeInt64Set(values pq.Int64Array, value int64, add bool) pq.Int64Array {
	result := make(pq.Int64Array, 0, len(values)+1)
	found := false
	for _, current := range values {
		if current == value {
			found = true
			if !add {
				continue
			}
		}
		result = append(result, current)
	}
	if add && !found {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func containsInt64(values pq.Int64Array, target int64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
