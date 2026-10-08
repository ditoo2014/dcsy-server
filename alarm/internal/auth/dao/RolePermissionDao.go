package dao

import (
	"alarm/models/model"
	"alarm/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RolePermissionDAO struct {
	db *gorm.DB
}

func NewRolePermissionDAO(ctx *gin.Context) *RolePermissionDAO {
	return &RolePermissionDAO{db: gormutil.GetGormDB(ctx, "")}
}

// 查询所属
func (dao *RolePermissionDAO) GetRolePermissions(roleP model.RolePermission, platId int) ([]model.RolePermission, error) {
	var rps []model.RolePermission

	sqlStr := "1=1 "
	if roleP.RoleId > 0 {
		sqlStr = sqlStr + fmt.Sprintf(`and role_id = %d `, roleP.RoleId)
	}

	err := dao.db.Table("role_permission").Where(sqlStr+` and ("role".plat_id = ? OR "role".plat_id = 0)`, platId).
		Joins(`left JOIN "role" ON "role_permission".role_id = "role".id `).
		Select(`"role_permission".*`).Find(&rps).Error
	if err != nil {
		log.Printf("Failed to get rolepermisison info: %v", err)
		return nil, err
	}

	return rps, nil
}

// 查询角色权限项
func (dao *RolePermissionDAO) Permissions(roleP model.RolePermission, platId int) ([]model.Permission, error) {
	var dataInfo []model.Permission

	err := dao.db.Table("role").Where(`"role".id = ? and "permission".plat_id = ?`, roleP.RoleId, platId).
		Joins(`left join "role_permission" on "role".id = "role_permission".role_id left join "permission" on "role_permission".permission_id = "permission".id`).
		Select(`"permission".*`).
		Scan(&dataInfo).Error
	if err != nil {
		log.Printf("Failed to get permissions with role info: %v", err)
		return nil, err
	}

	return dataInfo, nil
}

// 通过roleId 删除rolepermission
func (dao *RolePermissionDAO) DeleteRolePermissionsByRole(roleId int, platId int) error {

	err := dao.db.Model(&model.RolePermission{}).Where("role_id = ? ", roleId).Delete(&model.RolePermission{}).Error
	if err != nil {
		log.Printf("Failed to delete rolepermisisons: %v", err)
		return err
	}

	return nil
}

func (dao *RolePermissionDAO) BatchDeleteRolePermissions(rolepermissions []model.RolePermission) error {

	for _, v := range rolepermissions {
		err := dao.db.Model(&model.RolePermission{}).Where("role_id = ? AND permission_id = ?", v.RoleId, v.PermissionId).Delete(&model.RolePermission{}).Error
		if err != nil {
			log.Printf("Failed to delete rolepermisisons: %v", err)
			return err
		}
	}

	return nil
}

// 删除
func (dao *RolePermissionDAO) DeleteRolePermission(rp model.RolePermission) error {

	err := dao.db.Delete(&rp).Error
	if err != nil {
		log.Printf("Failed to delete rolepermisisons: %v", err)
		return err
	}

	return nil
}

// 批量通过ids删除
func (dao *RolePermissionDAO) DeleteRolePermissionsByIds(ids []int) error {

	err := dao.db.Delete(&model.RolePermission{}, ids).Error
	if err != nil {
		log.Printf("Failed to delete rolepermisisons: %v", err)
		return err
	}

	return nil
}

// 添加rolepermission
func (dao *RolePermissionDAO) AddRolePermission(rolepermission model.RolePermission) (model.RolePermission, error) {

	err := dao.db.Create(&rolepermission).Error
	if err != nil {
		log.Printf("Failed to create rolepermisison: %v", err)
		return rolepermission, err
	}

	return rolepermission, nil
}

// 批量添加rolepermissions
func (dao *RolePermissionDAO) AddRolePermissions(rolepermissions []model.RolePermission) ([]model.RolePermission, error) {

	for _, v := range rolepermissions {
		dao.db.Model(&v).Where("role_id = ? AND permission_id = ? ", v.RoleId, v.PermissionId).First(&v)
		if v.ID > 0 {
			// log.Printf("Batch create rolepermisisons already exist: %v \n", v)
			continue
		}
		err := dao.db.Create(&v).Error
		if err != nil {
			log.Printf("Failed to batch create rolepermisisons: %v", err)
			return nil, err
		}
	}

	return rolepermissions, nil
}
