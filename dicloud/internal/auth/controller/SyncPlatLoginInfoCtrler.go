package controller

import (
	"dicloud/internal/auth/client"
	"dicloud/models/model"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
)

type ISyncPlatLoginInfoCtrler interface {
	SyncToTolink(ctx *gin.Context)
	SyncToBusiness(ctx *gin.Context)
	SyncToDynalt(ctx *gin.Context)
}

type SyncPlatLoginInfoCtrler struct {
}

// 同步登录信息到采集云
func (ctrler SyncPlatLoginInfoCtrler) SyncToTolink(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	resp, err := client.SyncToTolink(ctx, &userInfo)
	if err != nil || resp.Code != 200 {
		response.Fail(ctx, resp.Message, nil)
		return
	}

	response.Success(ctx, "同步成功", nil)
}

// 同步登录信息到业务云
func (ctrler SyncPlatLoginInfoCtrler) SyncToBusiness(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	resp, err := client.SyncToBusiness(ctx, &userInfo)
	if err != nil || resp.Code != 200 {
		response.Fail(ctx, resp.Message, nil)
		return
	}

	response.Success(ctx, "同步成功", nil)
}

// 同步登录信息到云组态
func (ctrler SyncPlatLoginInfoCtrler) SyncToDynalt(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	resp, err := client.SyncToDynalt(ctx, &userInfo)
	if err != nil || resp.Code != 200 {
		response.Fail(ctx, resp.Message, nil)
		return
	}

	response.Success(ctx, "同步成功", nil)
}
