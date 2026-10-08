package controller

import (
	"dicloud/internal/auth/dao"
	"dicloud/models/model"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

type IRolePermissionCtrler interface {
	List(ctx *gin.Context)
	Permissions(ctx *gin.Context)
	AddRolePermission(ctx *gin.Context)
	DeleteRolePermission(ctx *gin.Context)
	BatchAddRolePermissions(ctx *gin.Context)
	BatchDeleteRolePermissions(ctx *gin.Context)
	DeleteRolePermissionList(ctx *gin.Context)
}

// (鉴权) 查询所属权限 -- 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云
type RolePermissionCtrler struct {
}

// 查询平台所有角色权限
func (rp RolePermissionCtrler) List(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	var roleP = model.RolePermission{}
	ctx.Bind(&roleP)

	/* 查询角色权限 */

	rolePermissionDao := dao.NewRolePermissionDAO(ctx)
	dataDtos, err := rolePermissionDao.GetRolePermissions(roleP, userInfo.PlatId)
	if err != nil {
		response.Fail(ctx, "查询数据失败", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": dataDtos})
}

// 查询角色权限项详情
func (rp RolePermissionCtrler) Permissions(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	var roleP = model.RolePermission{}
	ctx.Bind(&roleP)

	/* 查询用户权限 */

	rolePermissionDao := dao.NewRolePermissionDAO(ctx)
	dataDtos, err := rolePermissionDao.Permissions(roleP, userInfo.PlatId)
	if err != nil {
		response.Fail(ctx, "查询数据失败", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": dataDtos})
}

// (鉴权) 查询所属权限 -- 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云
func (rp RolePermissionCtrler) AddRolePermission(ctx *gin.Context) {

	/* 获取用户信息 */

	// userInfoCache, _ := ctx.Get("userInfo")
	// userInfo := userInfoCache.(model.User)

	var roleP = model.RolePermission{}
	ctx.Bind(&roleP)

	rolePermissionDao := dao.NewRolePermissionDAO(ctx)

	/* 添加rolepermission */
	dataDtos, err := rolePermissionDao.AddRolePermission(roleP)
	if err != nil {
		response.Fail(ctx, "添加数据失败", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": dataDtos})
}

func (rolePerCon RolePermissionCtrler) DeleteRolePermission(ctx *gin.Context) {

	/* 获取要删除的id */

	var rp = model.RolePermission{}
	ctx.Bind(&rp)

	rolePermissionDao := dao.NewRolePermissionDAO(ctx)

	/* 删除rolepermission */
	err := rolePermissionDao.DeleteRolePermission(rp)
	if err != nil {
		response.Fail(ctx, "删除失败", nil)
		return
	}

	response.Success(ctx, "success", nil)
}

// (鉴权) 查询所属权限 -- 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云
func (rp RolePermissionCtrler) BatchAddRolePermissions(ctx *gin.Context) {

	/* 获取用户信息 */

	// userInfoCache, _ := ctx.Get("userInfo")
	// userInfo := userInfoCache.(model.User)

	var rolePs = []model.RolePermission{}
	ctx.Bind(&rolePs)

	rolePermissionDao := dao.NewRolePermissionDAO(ctx)

	/* 删除rolepermission */
	// err := rolePermissionDao.DeleteRolePermissionsByRole(int(rolePs[0].RoleId), userInfo.PlatId)
	// if err != nil {
	// 	response.Fail(ctx, "添加失败", nil)
	//
	// 	return
	// }

	/* 添加rolepermission */
	dataDtos, err := rolePermissionDao.AddRolePermissions(rolePs)
	if err != nil {
		response.Fail(ctx, "添加数据失败", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": dataDtos})
}

func (rp RolePermissionCtrler) BatchDeleteRolePermissions(ctx *gin.Context) {

	/* 获取要删除的id */

	var ids = []int{}
	ctx.Bind(&ids)

	if len(ids) == 0 {
		response.Fail(ctx, "数据格式错误", nil)
		return
	}

	rolePermissionDao := dao.NewRolePermissionDAO(ctx)

	/* 删除rolepermission */
	err := rolePermissionDao.DeleteRolePermissionsByIds(ids)
	if err != nil {
		response.Fail(ctx, "删除失败", nil)
		return
	}

	response.Success(ctx, "success", nil)
}

func (rp RolePermissionCtrler) DeleteRolePermissionList(ctx *gin.Context) {

	/* 获取要删除数据 */

	var rolePs = []model.RolePermission{}
	ctx.Bind(&rolePs)

	if len(rolePs) == 0 {
		response.Fail(ctx, "数据格式错误", nil)
		return
	}

	rolePermissionDao := dao.NewRolePermissionDAO(ctx)

	/* 删除rolepermission */
	err := rolePermissionDao.BatchDeleteRolePermissions(rolePs)
	if err != nil {
		response.Fail(ctx, "删除失败", nil)
		return
	}

	response.Success(ctx, "success", nil)
}
