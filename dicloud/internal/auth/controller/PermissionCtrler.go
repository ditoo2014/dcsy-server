package controller

import (
	"dicloud/internal/auth/dao"
	"dicloud/models/model"
	"dicloud/pkg/cache"
	"dicloud/util/response"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

/* 权限管理 */
type IPermissionCtrler interface {
	UserOwnPermissions(ctx *gin.Context)
	Permissions(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

type PermissionCtrler struct {
}

// (鉴权) 查询所属权限 -- 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云
func (p PermissionCtrler) UserOwnPermissions(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	/* 查询用户权限 */

	permissionDao := dao.NewPermissionDAO(ctx)
	dataDtos, _ := permissionDao.GetUserPermissions(userInfo, userInfo.PlatId)

	/* 存入redis */

	var permissions = model.Permissions{}
	permissions.Data = dataDtos
	cache.HMSet(ctx, fmt.Sprintf("user:%v", userInfo.Username), map[string]interface{}{
		fmt.Sprintf("permission%v", userInfo.PlatId): permissions,
	})

	response.Success(ctx, "success", gin.H{"data": dataDtos})
}

// (鉴权) 全部权限 -- 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云
func (p PermissionCtrler) Permissions(ctx *gin.Context) {
	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	/* 查询平台权限 */
	permissionDao := dao.NewPermissionDAO(ctx)
	dataDtos, _ := permissionDao.GetPermissions(userInfo.PlatId)

	response.Success(ctx, "success", gin.H{"data": dataDtos})
}

func (p PermissionCtrler) Create(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	/* 绑定数据 */
	var iPms = model.Permission{}
	ctx.Bind(&iPms)

	if len(iPms.PermissionName) == 0 {
		log.Println("[PermissionCtrler] name is null")
		response.Fail(ctx, "名称为空", nil)
		return
	}

	// 填充数据
	iPms.PlatId = userInfo.PlatId

	// 创建
	pmsDao := dao.NewPermissionDAO(ctx)
	dataTem, res := pmsDao.Create(iPms)
	if !res {
		log.Println("[PermissionCtrler] Create fail")
		response.Fail(ctx, "权限创建失败", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": dataTem})
}

func (p PermissionCtrler) Update(ctx *gin.Context) {

	/* 绑定数据 */
	var iPms = model.Permission{}
	ctx.Bind(&iPms)

	if iPms.ID == 0 {
		log.Println("[PermissionCtrler] id is zero")
		response.Fail(ctx, "非法数据", nil)
		return
	}

	// 防止丢失, 自己绑定自己
	if iPms.ID == uint(iPms.ParentId) {
		iPms.ParentId = 0
	}

	// 修改
	pmsDao := dao.NewPermissionDAO(ctx)
	dataTem, res := pmsDao.Update(iPms)
	if !res {
		log.Println("[PermissionCtrler] update fail")
		response.Fail(ctx, "修改权限成功", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": dataTem})
}

func (p PermissionCtrler) Delete(ctx *gin.Context) {

	/* 绑定数据 */
	var iPms = model.Permission{}
	ctx.Bind(&iPms)

	if iPms.ID == 0 {
		log.Println("[PermissionCtrler] id is zero")
		response.Fail(ctx, "非法数据", nil)
		return
	}

	// 删除
	pmsDao := dao.NewPermissionDAO(ctx)
	res := pmsDao.Delete(iPms)
	if !res {
		log.Println("[PermissionCtrler] delete fail")
		response.Fail(ctx, "删除失败，可能需要从角色解绑此权限后操作", nil)
		return
	}

	response.Success(ctx, "success", nil)
}
