package dao

import (
	"alarm/models/model"
	"alarm/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PermissionDAO struct {
	db *gorm.DB
}

// NewUserDAO 创建新的 UserDAO 实例
func NewPermissionDAO(ctx *gin.Context) *PermissionDAO {
	return &PermissionDAO{db: gormutil.GetGormDB(ctx, "")}
}

// 查询平台所有权限项
func (dao *PermissionDAO) GetPermissions(platId int) ([]model.Permission, error) {
	var dataInfo []model.Permission

	err := dao.db.Table("permission").Where(`("permission".plat_id = 0 OR "permission".plat_id = ?)`, platId).Find(&dataInfo).Error
	if err != nil {
		log.Printf("Failed to get permissions: %v", err)
		return nil, err
	}

	return dataInfo, nil
}

// 查询平台所有权限项
func (dao *PermissionDAO) GetPermissionList(p model.Permission, ids []int64) ([]model.Permission, error) {
	var dataInfo []model.Permission

	sqlStr := "1=1"
	sqlStr = basePermissionSqlFormat(sqlStr, p)

	if len(ids) > 0 {
		err := dao.db.Table("permission").Where(sqlStr+" AND id in ?", ids).Find(&dataInfo).Error
		if err != nil {
			log.Printf("Failed to get permissions: %v", err)
			return nil, err
		}
	} else {
		err := dao.db.Table("permission").Where(sqlStr).Find(&dataInfo).Error
		if err != nil {
			log.Printf("Failed to get permissions: %v", err)
			return nil, err
		}
	}

	return dataInfo, nil
}

// 查询用户权限
func (dao *PermissionDAO) GetUserPermissions(userTem model.User, platId int) ([]model.Permission, error) {
	var dataInfo []model.Permission

	err := dao.db.Table("public.users").Where(`"users".id = ? and ("permission".plat_id = 0 OR "permission".plat_id = ?) `, userTem.ID, platId).
		Joins(`left JOIN "user_role" ON "user_role".user_id = "users".id left JOIN "role" ON "user_role".role_id = "role".id left join "role_permission" on "role".id = "role_permission".role_id left join "permission" on "role_permission".permission_id = "permission".id`).
		Select(`"permission".*`).
		Scan(&dataInfo).Error
	if err != nil {
		log.Printf("Failed to get permissions with user info: %v", err)
		return nil, err
	}

	return dataInfo, nil
}

// 创建
func (dao *PermissionDAO) Create(pms model.Permission) (model.Permission, bool) {

	err := dao.db.Create(&pms).Error
	if err != nil {
		log.Println("[PermissionDao] create fail", err)
		return pms, false
	}

	return pms, true
}

// 修改
func (dao *PermissionDAO) Update(pms model.Permission) (model.Permission, bool) {

	res := dao.db.Updates(&pms)
	if res.RowsAffected == 0 {
		log.Println("[PermissionDao] Dao update fail rows affected = 0")
		return pms, false
	}

	return pms, true
}

// 删除
func (dao *PermissionDAO) Delete(pms model.Permission) bool {

	res := dao.db.Delete(&pms)
	if res.RowsAffected == 0 {
		log.Println("[PermissionDao] Dao delete fail rows affected = 0")
		return false
	}

	return true
}

func basePermissionSqlFormat(sqlStr string, p model.Permission) string {

	if p.Level != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND level = %d", p.Level)
	}

	if p.ApiStage > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND api_stage = %d", p.ApiStage)
	}

	return sqlStr
}
