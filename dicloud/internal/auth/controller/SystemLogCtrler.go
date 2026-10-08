package controller

import (
	"dicloud/internal/common/dao"
	usermodel "dicloud/models/model"
	normalutil "dicloud/util/normal"
	"dicloud/util/response"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ISystemLogController interface {
	Find(ctx *gin.Context)
}

type SystemLogController struct {
}

func (ctrler SystemLogController) Find(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 绑定信息
	var rSysLog usermodel.SystemLog
	ctx.Bind(&rSysLog)

	// 根据当前接口所属平台查询日志，业务云为1，采集云为2。
	if rSysLog.PlatId == 0 && userInfo.PlatId > 0 {
		rSysLog.PlatId = userInfo.PlatId
	}

	var (
		pageStr      string
		pageSizeStr  string
		startTimeStr string
		endTimeStr   string
	)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page <= 0 {
		page = 1
	}
	offset, limit := normalutil.GetPageOffsetLimit(page, pageSize)
	pageSize = limit

	// 时间范围
	startTimeStr = ctx.Query("startTime")
	endTimeStr = ctx.Query("endTime")
	startTime, _ := strconv.Atoi(startTimeStr)
	endTime, _ := strconv.Atoi(endTimeStr)
	if endTime == 0 {
		endTime = int(time.Now().Unix())
	}

	systemlogDao := dao.NewSystemLogDao(ctx)

	dataList, total, res := systemlogDao.Find(rSysLog, offset, limit, startTime, endTime)
	if !res {
		response.Fail(ctx, "查询失败", nil)
		return
	}

	response.Success(ctx, "查询成功", gin.H{
		"data":     dataList,
		"total":    total,
		"pageNum":  page,
		"pageSize": pageSize,
	})
}
