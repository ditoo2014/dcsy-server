package dao

import (
	"alarm/models/model"
	"alarm/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserRoleDAO struct {
	db *gorm.DB
}

func NewUserRoleDAO(ctx *gin.Context) *UserRoleDAO {
	return &UserRoleDAO{db: gormutil.GetGormDB(ctx, "")}
}

func (dao *UserRoleDAO) List(platId int, userRole model.UserRole) ([]*model.UserRole, error) {
	urs := []*model.UserRole{}

	sqlStr := "1=1"
	if userRole.UserId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND user_role.user_id = %d ", userRole.UserId)
	}
	if userRole.RoleId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND user_role.role_id = %d ", userRole.RoleId)
	}

	err := dao.db.Table("role").Where(sqlStr+` AND ("role".plat_id = ? OR "role".plat_id = 0)`, platId).
		Joins(`left JOIN "user_role" ON "user_role".role_id = "role".id `).
		Select(`"user_role".*`).Find(&urs).Error
	if err != nil {
		log.Printf("Failed to find user roles: %v", err)
		return nil, err
	}

	return urs, nil
}

func (dao *UserRoleDAO) UserRoles(u *model.User) ([]*model.UserRole, error) {
	urs := []*model.UserRole{}

	err := dao.db.Table("user_role").Where("user_id = ? ", u.ID).Scan(&urs).Error
	if err != nil {
		log.Printf("Failed to find user_roles: %v", err)
		return nil, err
	}

	return urs, nil
}

func (dao *UserRoleDAO) ExistUserBindRole(r *model.Role) (model.UserRole, bool) {
	urs := model.UserRole{}

	err := dao.db.Table("user_role").Where("role_id = ? ", r.ID).Scan(&urs).Error
	if err != nil {
		log.Printf("Failed to find user_roles: %v", err)
		return urs, false
	}

	return urs, true
}

func (dao *UserRoleDAO) ExistRoleBindUser(u *model.User) (model.UserRole, bool) {
	urs := model.UserRole{}

	err := dao.db.Table("user_role").Where("user_id = ? ", u.ID).Scan(&urs).Error
	if err != nil {
		log.Printf("Failed to find user_roles: %v", err)
		return urs, false
	}

	return urs, true
}

func (dao *UserRoleDAO) Add(ur *model.UserRole) (*model.UserRole, error) {

	// 保存
	err := dao.db.Create(&ur).Error
	if err != nil {
		log.Printf("Failed to save user_role: %v", err)
		return ur, err
	}
	return ur, nil
}

func (dao *UserRoleDAO) Delete(ur *model.UserRole) (*model.UserRole, error) {

	// 删除
	err := dao.db.Delete(&ur).Error
	if err != nil {
		log.Printf("Failed to delete user_role: %v", err)
		return ur, err
	}
	return ur, nil
}
