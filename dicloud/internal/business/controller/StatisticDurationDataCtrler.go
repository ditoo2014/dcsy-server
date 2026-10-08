package controller

import (
	businessdao "dicloud/internal/business/dao"
	businessmodel "dicloud/internal/business/model"
	"dicloud/internal/business/service"
	usermodel "dicloud/models/model"
	"dicloud/util/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type IStatisticDurationDataCtrler interface {
	StartStatisticTask(ctx *gin.Context)
}

type StatisticDurationDataCtrler struct {
}

func (r StatisticDurationDataCtrler) StartStatisticTask(ctx *gin.Context) {

	var (
		startTimeStr string
		endTimeStr   string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 时间范围
	startTimeStr = ctx.Query("startTime")
	endTimeStr = ctx.Query("endTime")
	startTime, _ := strconv.Atoi(startTimeStr)
	endTime, _ := strconv.Atoi(endTimeStr)
	if endTime == 0 || endTime <= startTime {
		response.Fail(ctx, "请指定合理的时间范围", nil)
		return
	}

	// 查询全部
	if userInfo.Isadmin != 1 {
		response.Fail(ctx, "没有权限, 非超管不能操作", nil)
		return
	} else {

		// 根据部门 - 鉴权查寻
		equipmentDao := businessdao.NewEquipmentDAO(ctx)
		rEquipment := businessmodel.Equipment{}
		equipments, total, res := equipmentDao.FindEquipmentAll(rEquipment, 0, 9999)
		if total == 0 || !res {
			response.Success(ctx, "", nil)
			return
		}

		go service.StatisticDurationData(ctx, equipments, int64(startTime), int64(endTime))
		response.Success(ctx, "统计计划执行完成", nil)
	}

}
