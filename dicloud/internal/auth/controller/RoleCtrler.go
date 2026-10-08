package controller

import (
	"dicloud/internal/auth/dao"
	"dicloud/models/model"
	"dicloud/pkg/cache"
	normalutil "dicloud/util/normal"
	"dicloud/util/response"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type IRoleCtrler interface {
	UserOwnRoles(ctx *gin.Context)
	Roles(ctx *gin.Context)

	AddRole(ctx *gin.Context)
	UpdateRole(ctx *gin.Context)
	DeleteRole(ctx *gin.Context)
	FindRole(ctx *gin.Context)
}

type RoleCtrler struct {
}

/* 采集云查询所属角色 -- 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云 */
func (r RoleCtrler) UserOwnRoles(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	/* 查询用户权限 */

	roleDao := dao.NewRoleDAO(ctx)
	dataDtos, _ := roleDao.GetUserRoles(userInfo, userInfo.PlatId)

	/* 存入redis */

	cache.HMSet(ctx, fmt.Sprintf("user:%v", userInfo.Username), map[string]interface{}{
		fmt.Sprintf("role%v", userInfo.PlatId): dataDtos,
	})

	response.Success(ctx, "success", gin.H{"data": dataDtos})
}

/* 采集云查询所有角色 -- 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云 */
func (r RoleCtrler) Roles(ctx *gin.Context) {

	var (
		pageStr     string
		pageSizeStr string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	// fmt.Println("pageNum = ", pageStr, ", pageSize = ", pageSizeStr)
	pageNum, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(pageNum, pageSize)

	roleDao := dao.NewRoleDAO(ctx)
	dataDtos, _ := roleDao.GetRoles(userInfo.PlatId, offset, limit)
	total, _ := roleDao.CountRoles(userInfo.PlatId)

	response.Success(ctx, "success", gin.H{"data": dataDtos, "total": total})
}

/* 添加角色 */
func (r RoleCtrler) AddRole(ctx *gin.Context) {

	/* 获取用户信息 */

	// userInfoCache, _ := ctx.Get("userInfo")
	// userInfo := userInfoCache.(model.User)

	roleDao := dao.NewRoleDAO(ctx)
	var role = model.Role{}
	ctx.Bind(&role)

	// 查询名称是否已存在
	role, res := roleDao.ExistRoleName(role)
	if res {
		response.Fail(ctx, "已存在同名角色", nil)

		return
	}

	// 保存
	role.CreateTime = time.Now()
	data, res := roleDao.AddRole(role)
	if !res {
		response.Fail(ctx, "添加角色失败", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": data})
}

/* 修改角色 */
func (r RoleCtrler) UpdateRole(ctx *gin.Context) {

	/* 获取用户信息 */

	// userInfoCache, _ := ctx.Get("userInfo")
	// userInfo := userInfoCache.(model.User)

	/* 获取角色数据信息 */

	roleDao := dao.NewRoleDAO(ctx)
	var role = model.Role{}
	ctx.Bind(&role)

	// 查询名称是否已存在
	roleTem, res := roleDao.ExistRoleName(role)
	if res && roleTem.ID != role.ID {
		response.Fail(ctx, "已存在同名角色", nil)

		return
	}

	roleTem, res = roleDao.ExistRoleId(role)
	if !res {
		response.Fail(ctx, "不存在的数据", nil)

		return
	}

	// 修改
	data, res := roleDao.UpdateRole(role)
	if !res {
		response.Fail(ctx, "修改角色失败", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": data})

}

/* 删除角色 */
func (r RoleCtrler) DeleteRole(ctx *gin.Context) {

	/* 获取用户信息 */

	// userInfoCache, _ := ctx.Get("userInfo")
	// userInfo := userInfoCache.(model.User)

	roleDao := dao.NewRoleDAO(ctx)
	var role = model.Role{}
	ctx.Bind(&role)

	// 查询数据是否存在
	role, res := roleDao.ExistRoleId(role)
	if !res {
		response.Fail(ctx, "不存在的数据", nil)

		return
	}

	// 查询用户是否绑定角色
	userRoleDao := dao.NewUserRoleDAO(ctx)
	dataTem, res := userRoleDao.ExistUserBindRole(&role)
	if res && dataTem.ID != 0 {
		response.Fail(ctx, "请先将用户解绑再删除", nil)

		return
	}

	// 删除角色
	// 删除角色绑定其权限
	_, res = roleDao.DeleteRole(role)

	if !res {
		response.Fail(ctx, "删除角色失败", nil)

		return
	}

	response.Success(ctx, "success", nil)

}

/* 查询角色 */
func (r RoleCtrler) FindRole(ctx *gin.Context) {

	/* 获取用户信息 */

	// userInfoCache, _ := ctx.Get("userInfo")
	// userInfo := userInfoCache.(model.User)

	roleDao := dao.NewRoleDAO(ctx)
	var role = model.Role{}
	ctx.Bind(&role)

	// 查询数据是否存在
	role, res := roleDao.ExistRoleId(role)
	if !res {
		response.Fail(ctx, "不存在数据", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": role})
}
