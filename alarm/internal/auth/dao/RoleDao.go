package dao

import (
	"alarm/models/model"
	"alarm/util/gormutil"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RoleDAO struct {
	db *gorm.DB
}

func NewRoleDAO(ctx *gin.Context) *RoleDAO {
	return &RoleDAO{db: gormutil.GetGormDB(ctx, "")}
}

// 查询平台所有角色
func (dao *RoleDAO) GetRoles(platId int, offset, limit int) ([]model.Role, error) {
	var dataInfo []model.Role

	err := dao.db.Table("role").Offset(offset).Limit(limit).Where(`"role".plat_id = ? OR "role".plat_id = 0`, platId).Find(&dataInfo).Error
	if err != nil {
		log.Printf("Failed to get roles: %v", err)
		return nil, err
	}

	return dataInfo, nil
}

func (dao *RoleDAO) CountRoles(platId int) (int64, error) {
	var dataCount int64

	err := dao.db.Table("role").Where(`"role".plat_id = ?  OR "role".plat_id = 0 `, platId).Count(&dataCount).Error
	if err != nil {
		log.Printf("Failed to get roles: %v", err)
		return dataCount, err
	}

	return dataCount, nil
}

// 查询用户角色
func (dao *RoleDAO) GetUserRoles(userTem model.User, platId int) ([]model.Role, error) {
	var userInfo []model.Role

	err := dao.db.Table("public.users").Where(`"users".id = ?`, userTem.ID).
		Joins(`LEFT JOIN "user_role" ON "user_role".user_id = "users".id LEFT JOIN "role" ON "user_role".role_id = "role".id`).
		Select(`"role".*`).
		Scan(&userInfo).Error
	if err != nil {
		log.Printf("Failed to get role with user info: %v", err)
		return nil, err
	}

	return userInfo, nil
}

func (dao *RoleDAO) ExistRoleName(r model.Role) (model.Role, bool) {
	rl := model.Role{}

	err := dao.db.Table("role").Where("role_name = ? and plat_id = ?", r.RoleName, r.PlatId).Scan(&rl).Error
	if err != nil || rl.ID == 0 {
		log.Printf("Failed to get role with user info: %v", err)
		return r, false
	}

	return rl, true
}

func (dao *RoleDAO) ExistRoleId(r model.Role) (model.Role, bool) {
	rl := model.Role{}

	// 自动封装查询id
	err := dao.db.Table("role").Find(&r).Scan(&rl).Error
	if err != nil || rl.ID == 0 {
		log.Printf("Failed to get role with user info: %v", err)
		return r, false
	}

	return rl, true
}

// 添加角色
func (dao *RoleDAO) AddRole(r model.Role) (model.Role, bool) {
	err := dao.db.Table("role").Create(&r).Error
	if err != nil {
		log.Printf("Failed to get role with user info: %v", err)
		return r, false
	}

	return r, true

}

// 修改角色
func (dao *RoleDAO) UpdateRole(r model.Role) (model.Role, bool) {
	err := dao.db.Table("role").Updates(&r).Error
	if err != nil {
		log.Printf("Failed to get role with user info: %v", err)
		return r, false
	}

	return r, true

}

// 删除角色
func (dao *RoleDAO) DeleteRole(r model.Role) (model.Role, bool) {
	// 开启事务
	tx := dao.db.Begin()

	err := dao.db.Where("role_id = ? ", r.ID).Delete(&model.RolePermission{}).Error
	if err != nil {
		// 回滚
		log.Printf("Failed to get role with user info: %v", err)
		tx.Rollback()
		return r, false
	}

	err = dao.db.Table("role").Delete(r).Error
	if err != nil {
		// 回滚
		log.Printf("Failed to get role with user info: %v", err)
		tx.Rollback()
		return r, false
	}

	//提交事务
	tx.Commit()

	return r, true

}
