package main

import (
	"dicloud/internal/auth/controller"
	"dicloud/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func ConnectRoute(router *gin.Engine, useSpringGateway bool) *gin.Engine {

	/* 跨域 */
	if !useSpringGateway {
		router.Use(middleware.Cors())
	}

	/* 用户管理 + 权限管理 */

	auth := router.Group("/auth")
	{
		var userController controller.IUserCtrler = controller.UserCtrler{}
		var userRoleController controller.IUserRoleCtrler = controller.UserRoleCtrler{}
		var permissionController controller.IPermissionCtrler = controller.PermissionCtrler{}
		var roleController controller.IRoleCtrler = controller.RoleCtrler{}
		var rolePermissionController controller.IRolePermissionCtrler = controller.RolePermissionCtrler{}
		var corpController controller.ICorpCtrler = controller.CorpCtrler{}
		var syslogController controller.ISystemLogController = controller.SystemLogController{}
		var syncPlatLoginController controller.ISyncPlatLoginInfoCtrler = controller.SyncPlatLoginInfoCtrler{}
		var userEquipmentPermissionController controller.IUserEquipmentPermissionCtrler = controller.UserEquipmentPermissionCtrler{}

		/* 多租户 */
		auth.Use(middleware.SwitchDBMiddleware())

		/* 用户登录认证 */
		// 登录接口
		auth.POST("/smscode", userController.SendSmsCode)
		auth.POST("/login", middleware.SystemLogMiddleware("登录", middleware.AllowAnonymousLog()), userController.Login)
		auth.POST("/logout", userController.Logout)

		// middleware.AuthMiddleware() -- 登录验证
		// middleware.PlatMiddleware(2) -- 区分平台接口
		// middleware.AuthWithPlatMiddleware(2) -- 前两接口融合
		// middleware.VerifyMiddleware("", 2) -- 平台内部权限

		/* 用户信息管理 -- 默认所有用户都同步在每个平台,一个地方登录全部平台通用 -- 权限则各自平台管理 */
		auth.POST("/register", middleware.VerifyMiddleware("user:register", 1), middleware.SystemLogMiddleware("创建用户"), userController.Register)
		auth.POST("/delete", middleware.VerifyMiddleware("user:delete", 1), middleware.SystemLogMiddleware("删除用户"), userController.Delete)
		auth.POST("/update", middleware.VerifyMiddleware("user:update", 1), middleware.SystemLogMiddleware("修改用户"), userController.Update)
		auth.POST("/findpage", middleware.VerifyMiddleware("user:findpage", 1), userController.FindPage)
		auth.POST("/info", middleware.AuthMiddleware(), userController.Info)
		auth.POST("/changepassword", middleware.AuthMiddleware(), userController.ChangePassword)
		auth.POST("/resetuserpassword", middleware.VerifyMiddleware("user:resetuserpassword", 1), middleware.SystemLogMiddleware("重置用户密码"), userController.ResetUserPassword)

		// 使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云
		/**************************** 采集云权限管理 ****************************/
		auth.POST("/tolink/user/roles", middleware.AuthWithPlatMiddleware(2), roleController.UserOwnRoles)
		auth.POST("/tolink/user/permissions", middleware.AuthWithPlatMiddleware(2), permissionController.UserOwnPermissions)

		auth.POST("/tolink/role/list", middleware.VerifyMiddleware("role:list", 2), roleController.Roles)
		auth.POST("/tolink/role/add", middleware.VerifyMiddleware("role:add", 2), middleware.SystemLogMiddleware("[TOLINK]添加角色"), roleController.AddRole)
		auth.POST("/tolink/role/delete", middleware.VerifyMiddleware("role:delete", 2), middleware.SystemLogMiddleware("[TOLINK]删除角色"), roleController.DeleteRole)
		auth.POST("/tolink/role/update", middleware.VerifyMiddleware("role:update", 2), middleware.SystemLogMiddleware("[TOLINK]修改角色"), roleController.UpdateRole)

		auth.POST("/tolink/permission/list", middleware.VerifyMiddleware("permission:list", 2), permissionController.Permissions)
		auth.POST("/tolink/permission/add", middleware.VerifyMiddleware("permission:add", 2), middleware.SystemLogMiddleware("[TOLINK]添加权限"), permissionController.Create)
		auth.POST("/tolink/permission/update", middleware.VerifyMiddleware("permission:update", 2), middleware.SystemLogMiddleware("[TOLINK]修改权限"), permissionController.Update)
		auth.POST("/tolink/permission/delete", middleware.VerifyMiddleware("permission:delete", 2), middleware.SystemLogMiddleware("[TOLINK]删除权限"), permissionController.Delete)

		auth.POST("/tolink/rolepermission/list", middleware.VerifyMiddleware("rolepermission:list", 2), rolePermissionController.List)
		auth.POST("/tolink/rolepermission/permissions", middleware.VerifyMiddleware("rolepermission:permissions", 2), rolePermissionController.Permissions)
		auth.POST("/tolink/rolepermission/add", middleware.VerifyMiddleware("rolepermission:add", 2), middleware.SystemLogMiddleware("[TOLINK]绑定角色权限"), rolePermissionController.AddRolePermission)
		auth.POST("/tolink/rolepermission/delete", middleware.VerifyMiddleware("rolepermission:delete", 2), middleware.SystemLogMiddleware("[TOLINK]解绑角色权限"), rolePermissionController.DeleteRolePermission)
		auth.POST("/tolink/rolepermission/batchadd", middleware.VerifyMiddleware("rolepermission:batchadd", 2), middleware.SystemLogMiddleware("[TOLINK]批量绑定角色权限"), rolePermissionController.BatchAddRolePermissions)
		auth.POST("/tolink/rolepermission/batchdelete", middleware.VerifyMiddleware("rolepermission:batchdelete", 2), middleware.SystemLogMiddleware("[TOLINK]批量解绑角色权限"), rolePermissionController.BatchDeleteRolePermissions)
		auth.POST("/tolink/rolepermission/batchdeletelist", middleware.VerifyMiddleware("rolepermission:batchdeletelist", 1), middleware.SystemLogMiddleware("[TOLINK]批量解绑角色权限List"), rolePermissionController.DeleteRolePermissionList)

		auth.POST("/tolink/userrole/list", middleware.VerifyMiddleware("userrole:list", 2), userRoleController.List)
		auth.POST("/tolink/userrole/add", middleware.VerifyMiddleware("userrole:add", 2), middleware.SystemLogMiddleware("[TOLINK]绑定用户角色"), userRoleController.AddUserRole)
		auth.POST("/tolink/userrole/delete", middleware.VerifyMiddleware("userrole:delete", 2), middleware.SystemLogMiddleware("[TOLINK]解绑用户角色"), userRoleController.DeleteUserRole)

		auth.POST("/tolink/corp/list", middleware.VerifyMiddleware("corp:list", 2), corpController.FindCorp)
		auth.POST("/tolink/corp/add", middleware.VerifyMiddleware("corp:add", 2), middleware.SystemLogMiddleware("[TOLINK]添加部门"), corpController.AddCorp)
		auth.POST("/tolink/corp/delete", middleware.VerifyMiddleware("corp:delete", 2), middleware.SystemLogMiddleware("[TOLINK]删除部门"), corpController.DeleteCorp)
		auth.POST("/tolink/corp/update", middleware.VerifyMiddleware("corp:update", 2), middleware.SystemLogMiddleware("[TOLINK]修改部门"), corpController.UpdateCorp)
		auth.POST("/tolink/corp/export", middleware.VerifyMiddleware("corp:export", 1), middleware.SystemLogMiddleware("[TOLINK]导出部门"), corpController.ExportExcel)
		auth.POST("/tolink/corp/import", middleware.VerifyMiddleware("corp:import", 1), middleware.SystemLogMiddleware("[TOLINK]导入部门"), corpController.ImportExcel)

		auth.POST("/tolink/syslog/list", middleware.VerifyMiddleware("syslog:list", 2), syslogController.Find)

		// 同步平台登录信息 - 采集云 -> 业务云
		auth.POST("/tolink/synclogin/tobusiness", middleware.VerifyMiddleware("synclogin:tobusiness", 1), middleware.SystemLogMiddleware("[TOLINK]同步登录信息到业务云"), syncPlatLoginController.SyncToBusiness)
		// 同步平台登录信息 - 采集云 -> 云组态
		auth.POST("/tolink/synclogin/todynalt", middleware.VerifyMiddleware("synclogin:todynalt", 1), middleware.SystemLogMiddleware("[TOLINK]同步登录信息到云组态"), syncPlatLoginController.SyncToDynalt)

		/**************************** 业务云权限管理 ****************************/
		auth.POST("/business/user/roles", middleware.AuthWithPlatMiddleware(1), roleController.UserOwnRoles)
		auth.POST("/business/user/permissions", middleware.AuthWithPlatMiddleware(1), permissionController.UserOwnPermissions)

		auth.POST("/business/role/list", middleware.VerifyMiddleware("role:list", 1), roleController.Roles)
		auth.POST("/business/role/add", middleware.VerifyMiddleware("role:add", 1), middleware.SystemLogMiddleware("[BUSINESS]添加角色"), roleController.AddRole)
		auth.POST("/business/role/delete", middleware.VerifyMiddleware("role:delete", 1), middleware.SystemLogMiddleware("[BUSINESS]删除角色"), roleController.DeleteRole)
		auth.POST("/business/role/update", middleware.VerifyMiddleware("role:update", 1), middleware.SystemLogMiddleware("[BUSINESS]修改角色"), roleController.UpdateRole)

		auth.POST("/business/permission/list", middleware.VerifyMiddleware("permission:list", 1), permissionController.Permissions)
		auth.POST("/business/permission/add", middleware.VerifyMiddleware("permission:add", 1), middleware.SystemLogMiddleware("[BUSINESS]添加权限"), permissionController.Create)
		auth.POST("/business/permission/update", middleware.VerifyMiddleware("permission:update", 1), middleware.SystemLogMiddleware("[BUSINESS]修改权限"), permissionController.Update)
		auth.POST("/business/permission/delete", middleware.VerifyMiddleware("permission:delete", 1), middleware.SystemLogMiddleware("[BUSINESS]删除权限"), permissionController.Delete)

		auth.POST("/business/rolepermission/list", middleware.VerifyMiddleware("rolepermission:list", 1), rolePermissionController.List)
		auth.POST("/business/rolepermission/permissions", middleware.VerifyMiddleware("rolepermission:permissions", 1), rolePermissionController.Permissions)
		auth.POST("/business/rolepermission/add", middleware.VerifyMiddleware("rolepermission:add", 1), middleware.SystemLogMiddleware("[BUSINESS]绑定角色权限"), rolePermissionController.AddRolePermission)
		auth.POST("/business/rolepermission/delete", middleware.VerifyMiddleware("rolepermission:delete", 1), middleware.SystemLogMiddleware("[BUSINESS]解绑角色权限"), rolePermissionController.DeleteRolePermission)
		auth.POST("/business/rolepermission/batchadd", middleware.VerifyMiddleware("rolepermission:batchadd", 1), middleware.SystemLogMiddleware("[BUSINESS]批量绑定角色权限"), rolePermissionController.BatchAddRolePermissions)
		auth.POST("/business/rolepermission/batchdelete", middleware.VerifyMiddleware("rolepermission:batchdelete", 1), middleware.SystemLogMiddleware("[BUSINESS]批量解绑角色权限"), rolePermissionController.BatchDeleteRolePermissions)
		auth.POST("/business/rolepermission/batchdeletelist", middleware.VerifyMiddleware("rolepermission:batchdeletelist", 1), middleware.SystemLogMiddleware("[BUSINESS]批量解绑角色权限List"), rolePermissionController.DeleteRolePermissionList)

		auth.POST("/business/userrole/list", middleware.VerifyMiddleware("userrole:list", 1), userRoleController.List)
		auth.POST("/business/userrole/add", middleware.VerifyMiddleware("userrole:add", 1), middleware.SystemLogMiddleware("[BUSINESS]绑定用户角色"), userRoleController.AddUserRole)
		auth.POST("/business/userrole/delete", middleware.VerifyMiddleware("userrole:delete", 1), middleware.SystemLogMiddleware("[BUSINESS]解绑用户角色"), userRoleController.DeleteUserRole)

		auth.POST("/business/corp/list", middleware.VerifyMiddleware("corp:list", 1), corpController.FindCorp)
		auth.POST("/business/corp/add", middleware.VerifyMiddleware("corp:add", 1), middleware.SystemLogMiddleware("[BUSINESS]添加部门"), corpController.AddCorp)
		auth.POST("/business/corp/delete", middleware.VerifyMiddleware("corp:delete", 1), middleware.SystemLogMiddleware("[BUSINESS]删除部门"), corpController.DeleteCorp)
		auth.POST("/business/corp/update", middleware.VerifyMiddleware("corp:update", 1), middleware.SystemLogMiddleware("[BUSINESS]修改部门"), corpController.UpdateCorp)
		auth.POST("/business/corp/export", middleware.VerifyMiddleware("corp:export", 1), middleware.SystemLogMiddleware("[BUSINESS]导出部门"), corpController.ExportExcel)
		auth.POST("/business/corp/import", middleware.VerifyMiddleware("corp:import", 1), middleware.SystemLogMiddleware("[BUSINESS]导入部门"), corpController.ImportExcel)

		auth.POST("/business/syslog/list", middleware.VerifyMiddleware("syslog:list", 1), syslogController.Find)

		/* 用户设备关注/控制授权，数据直接保存到设备表的用户ID数组中。 */
		auth.POST("/business/userconcernequipment/add", middleware.VerifyMiddleware("user:update", 1), userEquipmentPermissionController.AddConcern)
		auth.POST("/business/userconcernequipment/delete", middleware.VerifyMiddleware("user:update", 1), userEquipmentPermissionController.DeleteConcern)
		auth.POST("/business/userconcernequipment/find", middleware.VerifyMiddleware("user:findpage", 1), userEquipmentPermissionController.FindConcern)
		auth.POST("/business/usercontrolequipment/add", middleware.VerifyMiddleware("user:update", 1), userEquipmentPermissionController.AddControl)
		auth.POST("/business/usercontrolequipment/delete", middleware.VerifyMiddleware("user:update", 1), userEquipmentPermissionController.DeleteControl)
		auth.POST("/business/usercontrolequipment/find", middleware.VerifyMiddleware("user:findpage", 1), userEquipmentPermissionController.FindControl)

		// 同步平台登录信息 - 业务云 -> 采集云
		auth.POST("/business/synclogin/totolink", middleware.VerifyMiddleware("synclogin:totolink", 1), middleware.SystemLogMiddleware("[BUSINESS]同步登录信息到采集云"), syncPlatLoginController.SyncToTolink)
		// 同步平台登录信息 - 业务云 -> 云组态
		auth.POST("/business/synclogin/todynalt", middleware.VerifyMiddleware("synclogin:todynalt", 1), middleware.SystemLogMiddleware("[BUSINESS]同步登录信息到云组态"), syncPlatLoginController.SyncToDynalt)
	}

	return router
}
