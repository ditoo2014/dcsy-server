package controller

import (
	"dicloud/internal/auth/dao"
	"dicloud/models/model"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

type IUserRoleCtrler interface {
	List(ctx *gin.Context)
	UserRoles(ctx *gin.Context)
	AddUserRole(ctx *gin.Context)
	DeleteUserRole(ctx *gin.Context)
}

type UserRoleCtrler struct {
}

// 查询全部
func (ur UserRoleCtrler) List(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	/* 绑定用户信息 */
	var userRole = model.UserRole{}
	ctx.Bind(&userRole)

	/* 查询平台所有 */

	userRoleDao := dao.NewUserRoleDAO(ctx)
	data, err := userRoleDao.List(userInfo.PlatId, userRole)
	if err != nil {
		response.Fail(ctx, "查询失败", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": data})
}

// 查询
func (ur UserRoleCtrler) UserRoles(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	/* 查询 */

	userRoleDao := dao.NewUserRoleDAO(ctx)
	data, err := userRoleDao.UserRoles(&userInfo)
	if err != nil {
		response.Fail(ctx, "查询失败", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": data})
}

// 添加用户角色
func (ur UserRoleCtrler) AddUserRole(ctx *gin.Context) {

	var userRole = model.UserRole{}
	ctx.Bind(&userRole)

	// 保存
	userRoleDao := dao.NewUserRoleDAO(ctx)
	data, err := userRoleDao.Add(&userRole)
	if err != nil {
		response.Fail(ctx, "保存失败", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": data})
}

// 删除用户角色
func (ur UserRoleCtrler) DeleteUserRole(ctx *gin.Context) {
	var userRole = model.UserRole{}
	ctx.Bind(&userRole)

	// 删除
	userRoleDao := dao.NewUserRoleDAO(ctx)
	data, err := userRoleDao.Delete(&userRole)
	if err != nil {
		response.Fail(ctx, "保存失败", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": data})
}
