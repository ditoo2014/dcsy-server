package controller

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	"dicloud/internal/common/service"
	usermodel "dicloud/models/model"
	normalutil "dicloud/util/normal"
	"dicloud/util/response"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
)

type IAlarmDataRecordCtrler interface {
	DeleteAlarm(ctx *gin.Context)
	FindAlarm(ctx *gin.Context)
	BatchCountAlarm(ctx *gin.Context)
	BatchAlarmStatus(ctx *gin.Context)
}

type AlarmDataRecordCtrler struct {
}

func (r AlarmDataRecordCtrler) DeleteAlarm(ctx *gin.Context) {

	// 绑定触发器数据
	var alarmDataRecord = model.AlarmDataRecord{}
	ctx.Bind(&alarmDataRecord)

	if alarmDataRecord.ID == 0 {
		response.Fail(ctx, "非法数据", nil)
		return
	}

	alarmDataRecordDao := dao.NewAlarmDataRecordDao(ctx)
	res := alarmDataRecordDao.DeleteRecord(alarmDataRecord)
	if !res {
		response.Fail(ctx, "删除失败", nil)
		return
	}

	response.Success(ctx, "删除成功", nil)
}

func (r AlarmDataRecordCtrler) FindAlarm(ctx *gin.Context) {

	var (
		pageStr      string
		pageSizeStr  string
		startTimeStr string
		endTimeStr   string
	)
	// 绑定触发器数据
	var alarmDataRecord = model.AlarmDataRecord{}
	ctx.Bind(&alarmDataRecord)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(page, pageSize)

	startTimeStr = ctx.Query("startTime")
	endTimeStr = ctx.Query("endTime")
	if (startTimeStr == "") != (endTimeStr == "") {
		response.Fail(ctx, "开始时间和结束时间必须同时传入", nil)
		return
	}
	var startTime, endTime int64
	if startTimeStr != "" {
		var err error
		startTime, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil || startTime <= 0 {
			response.Fail(ctx, "开始时间不合法", nil)
			return
		}
		endTime, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil || endTime <= 0 || endTime < startTime {
			response.Fail(ctx, "结束时间不合法", nil)
			return
		}
	}

	corpId, _ := strconv.Atoi(ctx.Query("corpId"))

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 查询全部
	if userInfo.Isadmin == 1 {

		// 部门筛选
		if corpId > 0 {
			var corpIds []int
			// 单独部门操作
			corpIds = append(corpIds, corpId)
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			// 查询部门包含设备ids
			equipmentIds := service.FindAllCorpEquipmentId(ctx, corpIds)

			alarmDataRecordDao := dao.NewAlarmDataRecordDao(ctx)
			dataList, total, res := alarmDataRecordDao.FindRecordsByEquipment(alarmDataRecord, offset, limit, equipmentIds, startTime, endTime)
			if !res {
				response.Fail(ctx, "查询失败", nil)
				return
			}
			response.Success(ctx, "查询成功", gin.H{"data": dataList, "total": total})

		} else {

			alarmDataRecordDao := dao.NewAlarmDataRecordDao(ctx)
			dataList, total, res := alarmDataRecordDao.FindRecords(alarmDataRecord, offset, limit, startTime, endTime)
			if !res {
				response.Fail(ctx, "查询失败", nil)
				return
			}
			response.Success(ctx, "查询成功", gin.H{"data": dataList, "total": total})

		}
	} else {

		// 查询用户部门
		var corpIds []int
		for _, v := range userInfo.CorpIds {
			corpIds = append(corpIds, int(v))
		}
		// 子部门查询
		corpIds = service.FindAllSonCorp(ctx, corpIds)

		// 鉴权部门
		if corpId > 0 && len(corpIds) > 0 {
			idExist := -1
			for _, ID := range corpIds {
				if ID == corpId {
					idExist = 1
					break
				}
			}
			if idExist == -1 {
				log.Println("[BUSINESS] not exist corp: ", corpId)
				response.Fail(ctx, "非法部门, 未授权", nil)
				return
			}

			// 重新筛选
			corpIds = []int{corpId}
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)
		}

		// 根据部门 - 鉴权查寻
		// 查询部门包含设备ids
		equipmentIds := service.FindAllCorpEquipmentId(ctx, corpIds)

		alarmDataRecordDao := dao.NewAlarmDataRecordDao(ctx)
		dataList, total, res := alarmDataRecordDao.FindRecordsByEquipment(alarmDataRecord, offset, limit, equipmentIds, startTime, endTime)
		if !res {
			response.Fail(ctx, "查询失败", nil)
			return
		}

		response.Success(ctx, "查询成功", gin.H{"data": dataList, "total": total})
	}

}

// 设备报警状态数量查询接口
func (r AlarmDataRecordCtrler) BatchCountAlarm(ctx *gin.Context) {

	// 绑定触发器数据
	var alarmDataRecord = model.AlarmDataRecord{}
	ctx.Bind(&alarmDataRecord)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	if len(alarmDataRecord.EquipmentIds) == 0 {
		response.Success(ctx, "", nil)
		log.Println("[Business] Count alarm record fail.")
		return
	}

	// 查询全部
	if userInfo.Isadmin == 1 {

		alarmDataRecordDao := dao.NewAlarmDataRecordDao(ctx)
		result, res := alarmDataRecordDao.CountRecords(alarmDataRecord)
		if !res {
			response.Fail(ctx, "查询失败", nil)
			return
		}

		response.Success(ctx, "查询成功", gin.H{"data": result})

	} else {

		alarmDataRecordDao := dao.NewAlarmDataRecordDao(ctx)
		result, res := alarmDataRecordDao.CountRecords(alarmDataRecord)
		if !res {
			response.Fail(ctx, "查询失败", nil)
			return
		}

		response.Success(ctx, "查询成功", gin.H{"data": result})
	}

}

// 一次查询设备的实时告警和预警状态。
func (r AlarmDataRecordCtrler) BatchAlarmStatus(ctx *gin.Context) {
	var alarmDataRecord = model.AlarmDataRecord{}
	ctx.Bind(&alarmDataRecord)
	if len(alarmDataRecord.EquipmentIds) == 0 {
		response.Success(ctx, "", gin.H{"data": []model.EquipmentAlarmStatus{}})
		return
	}
	if alarmDataRecord.AlarmStatus == 0 {
		alarmDataRecord.AlarmStatus = 1
	}
	alarmDataRecord.AlarmType = ""

	alarmDataRecordDao := dao.NewAlarmDataRecordDao(ctx)
	result, res := alarmDataRecordDao.CountRecordStatus(alarmDataRecord)
	if !res {
		response.Fail(ctx, "查询失败", nil)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"data": result})
}
