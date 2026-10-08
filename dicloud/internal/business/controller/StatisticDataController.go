package controller

import (
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	statisticservice "dicloud/internal/business/service"
	commondao "dicloud/internal/common/dao"
	commonmodel "dicloud/internal/common/model"
	"dicloud/internal/common/service"
	usermodel "dicloud/models/model"
	normalutil "dicloud/util/normal"
	"dicloud/util/response"
	"io"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type IStatisticDataCtrler interface {
	FindStatisticData(ctx *gin.Context)                   // 查询统计数据 - 全部数据查询
	CountStatisticData(ctx *gin.Context)                  // 简单计算统计数据 - 加起来
	FindStatisticDataByLevelCorp(ctx *gin.Context)        // 查询同级部门数据, 指定code
	FindStatisticDataMakrListByCorpCode(ctx *gin.Context) // 查询部门代码数据, 返回列表
	FindDailyAlarmStatistics(ctx *gin.Context)            // 查询大屏每日累计告警数
	FindDailyWarningStatistics(ctx *gin.Context)          // 查询大屏每日累计预警数
	FindChargeDischargeDailyStatistics(ctx *gin.Context)  // 查询充放电日统计，缺失时同步补算
}

type StatisticDataCtrler struct {
}

// 统计数据查询 - 全部数据查询
func (ctrler StatisticDataCtrler) FindStatisticData(ctx *gin.Context) {

	var (
		pageStr      string
		pageSizeStr  string
		filterFunc   string
		startTimeStr string
		endTimeStr   string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 查询统计数据
	var rStstdTem commonmodel.StatisticData
	ctx.Bind(&rStstdTem)

	statisticDao := commondao.NewStatisticDataDao(ctx)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(page, pageSize)
	// 时间范围
	startTimeStr = ctx.Query("startTime")
	endTimeStr = ctx.Query("endTime")
	startTime, _ := strconv.Atoi(startTimeStr)
	endTime, _ := strconv.Atoi(endTimeStr)
	if endTime == 0 {
		endTime = int(time.Now().Unix())
	}

	/* 鉴权查询 */
	if userInfo.Isadmin == 1 {
		dataList, total, res := statisticDao.FindStatisticData(rStstdTem, offset, limit, startTime, endTime)
		if !res {
			response.Success(ctx, "success", nil)
			return
		}
		response.Success(ctx, "success", gin.H{"data": dataList, "total": total})
		return

	} else {

		filterFunc = ctx.Query("filterFunc")

		/* 用户关注设备 - 查询 */
		if len(filterFunc) == 0 || filterFunc == "ConE" {
			// 默认: concern equipment

			var siteIds []int
			if rStstdTem.SiteId == 0 {
				// 鉴权查询 - 根据用户id查询
				// 根据设备关注用户 - 查询用户关注设备
				equipmentDao := dao.NewEquipmentDAO(ctx)
				eqs, _, _ := equipmentDao.FindEquipmentByUser(model.Equipment{}, int(userInfo.ID), offset, limit)
				for _, v := range eqs {
					siteIds = append(siteIds, int(v.SiteId))
				}
				siteIds = normalutil.Unique(siteIds)
			} else {
				siteIds = append(siteIds, rStstdTem.SiteId)
			}

			// 鉴权查寻
			datas, total, res := statisticDao.FindStatisticDataBySite(rStstdTem, offset, limit, startTime, endTime, siteIds)
			if !res {
				response.Success(ctx, "", nil)
				return
			}

			response.Success(ctx, "success", gin.H{"data": datas, "total": total})
			return
		}

		/* 用户绑定部门 - 部门绑定设备 - 查询 */
		if filterFunc == "BinC" {
			// binded corp

			// 查询用户部门
			var corpIds []int
			for _, v := range userInfo.CorpIds {
				corpIds = append(corpIds, int(v))
			}
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			if len(corpIds) > 0 {
				// 根据部门 - 鉴权查寻
				datas, total, res := statisticDao.FindStatisticDataByCorp(rStstdTem, offset, limit, startTime, endTime, corpIds)
				if !res {
					response.Success(ctx, "success", nil)
					return
				}
				response.Success(ctx, "success", gin.H{"data": datas, "total": total})
				return
			}

			response.Success(ctx, "success", gin.H{"total": 0})
			return
		}
	}

	response.Success(ctx, "查询成功", nil)
}

// 统计查询数据 - 一般用于查询最新累计数据/一段时间增量, 并累加
func (ctrler StatisticDataCtrler) CountStatisticData(ctx *gin.Context) {

	var (
		pageStr      string
		pageSizeStr  string
		filterFunc   string
		startTimeStr string
		endTimeStr   string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 查询统计数据
	var rStstdTem commonmodel.StatisticData
	ctx.Bind(&rStstdTem)

	statisticDao := commondao.NewStatisticDataDao(ctx)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(page, pageSize)

	// 如果查累计数据 - 查最新的一条就行了
	if rStstdTem.StatisticValueType == 0 {
		// 把各个code的最新情况查出即可 - 本身就是累计
		limit = 1
	}

	// 时间范围
	startTimeStr = ctx.Query("startTime")
	endTimeStr = ctx.Query("endTime")
	startTime, _ := strconv.Atoi(startTimeStr)
	endTime, _ := strconv.Atoi(endTimeStr)
	if endTime == 0 {
		endTime = int(time.Now().Unix())
	}

	var resMsg = gin.H{}
	if len(rStstdTem.CodeList) > 0 {

		var (
			codeList []string
			nameList []string
			markList []string
			endStsdT []int64
			stsdVal  []float64
		)

		for _, v := range rStstdTem.CodeList {
			// 获取数据
			var dataList []commonmodel.StatisticData
			var res bool
			// 查询指定数据
			rStstdTem.StatisticCode = v

			/* 鉴权查询 */
			if userInfo.Isadmin == 1 {

				dataList, _, res = statisticDao.FindStatisticData(rStstdTem, offset, limit, startTime, endTime)
				if !res {
					response.Success(ctx, "success", nil)
					continue
				}
			} else {

				filterFunc = ctx.Query("filterFunc")

				/* 用户关注设备 - 查询 */
				if len(filterFunc) == 0 || filterFunc == "ConE" {
					// 默认: concern equipment

					var siteIds []int
					if rStstdTem.SiteId == 0 {
						// 鉴权查询 - 根据用户id查询
						// 根据设备关注用户 - 查询用户关注设备
						equipmentDao := dao.NewEquipmentDAO(ctx)
						eqs, _, _ := equipmentDao.FindEquipmentByUser(model.Equipment{}, int(userInfo.ID), offset, limit)
						for _, v := range eqs {
							siteIds = append(siteIds, int(v.SiteId))
						}
						siteIds = normalutil.Unique(siteIds)
					} else {
						siteIds = append(siteIds, rStstdTem.SiteId)
					}

					// 鉴权查寻
					dataList, _, res = statisticDao.FindStatisticDataBySite(rStstdTem, offset, limit, startTime, endTime, siteIds)
					if !res {
						continue
					}
				}

				/* 用户绑定部门 - 部门绑定设备 - 查询 */
				if filterFunc == "BinC" {
					// binded corp

					var corpIds []int
					// 查询用户部门
					for _, v := range userInfo.CorpIds {
						corpIds = append(corpIds, int(v))
					}
					// 子部门查询
					corpIds = service.FindAllSonCorp(ctx, corpIds)

					// 根据部门 - 鉴权查寻
					dataList, _, res = statisticDao.FindStatisticDataByCorp(rStstdTem, offset, limit, startTime, endTime, corpIds)
					if !res {
						continue
					}
				}
			}

			// 简单计算
			var countAll = float64(0)
			if len(dataList) > 0 {
				for _, dataTem := range dataList {
					countAll = countAll + dataTem.StatisticValue
				}
				// 最新的统计时间
				codeList = append(codeList, dataList[0].StatisticCode)
				nameList = append(nameList, dataList[0].StatisticName)
				markList = append(markList, dataList[0].MarkDate)
				endStsdT = append(endStsdT, dataList[0].EndStatisticTime)
				stsdVal = append(stsdVal, countAll)
			}
		}

		resMsg = gin.H{"code": codeList, "mark": markList, "name": nameList, "endTime": endStsdT, "value": stsdVal}
	}

	response.Success(ctx, "查询成功", resMsg)
}

// 统计查询数据 - 部门的筛选方式 - 查增量数据适用
func (ctrler StatisticDataCtrler) FindStatisticDataByLevelCorp(ctx *gin.Context) {

	var (
		pageStr      string
		pageSizeStr  string
		startTimeStr string
		endTimeStr   string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 查询统计数据
	var rStstdTem commonmodel.StatisticData
	ctx.Bind(&rStstdTem)

	statisticDao := commondao.NewStatisticDataDao(ctx)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(page, pageSize)

	// 时间范围
	startTimeStr = ctx.Query("startTime")
	endTimeStr = ctx.Query("endTime")
	startTime, _ := strconv.Atoi(startTimeStr)
	endTime, _ := strconv.Atoi(endTimeStr)
	if endTime == 0 {
		endTime = int(time.Now().Unix())
	}

	// 查询用户部门 - 非管理员
	var userCorpIds []int
	if userInfo.Isadmin != 1 {
		for _, v := range userInfo.CorpIds {
			userCorpIds = append(userCorpIds, int(v))
		}
		// 子部门查询
		userCorpIds = service.FindAllSonCorp(ctx, userCorpIds)
	}

	if len(rStstdTem.CorpList) > 0 {

		var resultList []map[string]interface{}

		for _, corpIdTem := range rStstdTem.CorpList {

			// 部门鉴权
			if userInfo.Isadmin != 1 {
				// 鉴权部门
				if len(userCorpIds) > 0 {
					idExist := -1
					for _, ID := range userCorpIds {
						if ID == corpIdTem {
							idExist = 1
							break
						}
					}
					if idExist == -1 {
						log.Println("[BUSINESS] statistic ctrler not exist corp: ", corpIdTem)
						response.Fail(ctx, "非法部门, 未授权", nil)
						return
					}
				} else {
					log.Println("[BUSINESS] statistic ctrler not boot corp for user: ", corpIdTem)
					response.Fail(ctx, "非法部门, 未授权", nil)
					return
				}
			}

			// 单独部门操作
			var res bool
			var corpIds []int
			corpIds = append(corpIds, int(corpIdTem))

			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			if len(rStstdTem.CodeList) > 0 {
				for _, codeTem := range rStstdTem.CodeList {

					var (
						codeList []string
						nameList []string
						markList []string
						endStsdT []int64
						stsdVal  []float64
					)
					mapTem := make(map[string]interface{})

					// 获取数据
					var dataList []commonmodel.StatisticData
					// 查询指定数据
					rStstdTem.StatisticCode = codeTem

					/* 鉴权查询 - 部门 */
					// 根据部门 - 鉴权查寻
					dataList, _, res = statisticDao.FindStatisticDataByCorp(rStstdTem, offset, limit, startTime, endTime, corpIds)
					if !res {
						continue
					}

					// 简单计算
					var countAll = float64(0)
					if len(dataList) > 0 {
						for _, dataTem := range dataList {
							countAll = countAll + dataTem.StatisticValue
						}
						// 最新的统计时间
						codeList = append(codeList, dataList[0].StatisticCode)
						nameList = append(nameList, dataList[0].StatisticName)
						markList = append(markList, dataList[0].MarkDate)
						endStsdT = append(endStsdT, dataList[0].EndStatisticTime)
						stsdVal = append(stsdVal, countAll)
					}

					mapTem["code"] = codeList
					mapTem["name"] = nameList
					mapTem["mark"] = markList
					mapTem["endTime"] = endStsdT
					mapTem["value"] = stsdVal
					mapTem["corpId"] = corpIdTem
					resultList = append(resultList, mapTem)
				}
			}
		}

		response.Success(ctx, "查询成功", gin.H{"data": resultList})
		return
	}

	response.Success(ctx, "查询成功", nil)
}

// 统计查询数据 - 部门的筛选方式 - 查存量变化适用
func (ctrler StatisticDataCtrler) FindStatisticDataMakrListByCorpCode(ctx *gin.Context) {

	var (
		pageStr      string
		pageSizeStr  string
		startTimeStr string
		endTimeStr   string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 查询统计数据
	var rStstdTem commonmodel.StatisticData
	ctx.Bind(&rStstdTem)

	statisticDao := commondao.NewStatisticDataDao(ctx)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(page, pageSize)

	// 时间范围
	startTimeStr = ctx.Query("startTime")
	endTimeStr = ctx.Query("endTime")
	startTime, _ := strconv.Atoi(startTimeStr)
	endTime, _ := strconv.Atoi(endTimeStr)
	if endTime == 0 {
		endTime = int(time.Now().Unix())
	}

	// 查询用户部门 - 非管理员
	var userCorpIds []int
	if userInfo.Isadmin != 1 {
		for _, v := range userInfo.CorpIds {
			userCorpIds = append(userCorpIds, int(v))
		}
		// 子部门查询
		userCorpIds = service.FindAllSonCorp(ctx, userCorpIds)
	}

	if len(rStstdTem.CorpList) > 0 {

		var resultList []map[string]interface{}

		for _, corpIdTem := range rStstdTem.CorpList {

			// 部门鉴权
			if userInfo.Isadmin != 1 {
				// 鉴权部门
				if len(userCorpIds) > 0 {
					idExist := -1
					for _, ID := range userCorpIds {
						if ID == corpIdTem {
							idExist = 1
							break
						}
					}
					if idExist == -1 {
						log.Println("[BUSINESS] statistic ctrler not exist corp: ", corpIdTem)
						response.Fail(ctx, "非法部门, 未授权", nil)
						return
					}
				} else {
					log.Println("[BUSINESS] statistic ctrler not boot corp for user: ", corpIdTem)
					response.Fail(ctx, "非法部门, 未授权", nil)
					return
				}
			}

			// 单独部门操作
			var res bool
			var corpIds []int
			corpIds = append(corpIds, int(corpIdTem))

			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			if len(rStstdTem.CodeList) > 0 {
				for _, codeTem := range rStstdTem.CodeList {

					var (
						markList []string
						stsdVal  []float64
					)
					mapTem := make(map[string]interface{})

					// 获取数据
					var dataList []commonmodel.StatisticData
					// 查询指定数据
					rStstdTem.StatisticCode = codeTem

					/* 鉴权查询 - 部门 */
					// 根据部门 - 鉴权查寻
					dataList, _, res = statisticDao.FindStatisticDataByCorp(rStstdTem, offset, limit, startTime, endTime, corpIds)
					if !res {
						continue
					}

					// 简单计算 - 相加每个统计周期的数据 - markDate
					if len(dataList) > 0 {
						valueMapTem := make(map[string]float64)
						for _, dataTem := range dataList {
							// 相加同一天的数据 - 细化数据
							val := dataTem.StatisticValue
							if valueMapTem[dataTem.MarkDate] > 0 {
								val = val + valueMapTem[dataTem.MarkDate]
							}
							valueMapTem[dataTem.MarkDate] = val
						}

						if len(valueMapTem) > 0 {
							for kTem, valTem := range valueMapTem {
								// 最新的统计时间
								markList = append(markList, kTem)
								stsdVal = append(stsdVal, valTem)
							}
						}
						mapTem["mark"] = markList
						mapTem["value"] = stsdVal
						// mapTem["dataMap"] = valueMapTem
					}

					mapTem["code"] = codeTem
					mapTem["corpId"] = corpIdTem
					resultList = append(resultList, mapTem)
				}
			}
		}

		response.Success(ctx, "查询成功", gin.H{"data": resultList})
		return
	}

	response.Success(ctx, "查询成功", nil)
}

// FindDailyAlarmStatistics counts every alarm lifecycle on its start date.
func (ctrler StatisticDataCtrler) FindDailyAlarmStatistics(ctx *gin.Context) {
	var request struct {
		CorpList []int `json:"corpList"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Fail(ctx, "请求参数错误", nil)
		return
	}

	startTime, err := strconv.ParseInt(ctx.Query("startTime"), 10, 64)
	if err != nil || startTime <= 0 {
		response.Fail(ctx, "开始时间错误", nil)
		return
	}
	endTime, err := strconv.ParseInt(ctx.Query("endTime"), 10, 64)
	if err != nil || endTime <= 0 || endTime < startTime {
		response.Fail(ctx, "结束时间错误", nil)
		return
	}

	userInfoCache, exists := ctx.Get("userInfo")
	if !exists {
		response.Fail(ctx, "用户信息不存在", nil)
		return
	}
	userInfo := userInfoCache.(usermodel.User)

	if userInfo.Isadmin != 1 {
		userCorpIds := make([]int, 0, len(userInfo.CorpIds))
		for _, corpID := range userInfo.CorpIds {
			userCorpIds = append(userCorpIds, int(corpID))
		}
		authorizedCorpIds := service.FindAllSonCorp(ctx, userCorpIds)
		authorized := make(map[int]struct{}, len(authorizedCorpIds))
		for _, corpID := range authorizedCorpIds {
			authorized[corpID] = struct{}{}
		}
		for _, corpID := range request.CorpList {
			if _, ok := authorized[corpID]; !ok {
				response.Fail(ctx, "非法部门, 未授权", nil)
				return
			}
		}
	}

	corpIds := service.FindAllSonCorp(ctx, request.CorpList)
	equipmentIds := service.FindAllCorpEquipmentId(ctx, corpIds)
	if len(equipmentIds) == 0 {
		response.Success(ctx, "查询成功", gin.H{"data": []statisticservice.DailyAlarmStatistics{}})
		return
	}

	alarmDao := commondao.NewAlarmDataRecordDao(ctx)
	records, err := alarmDao.FindDailyAlarmStatisticRecords(equipmentIds, startTime, endTime)
	if err != nil {
		log.Println("[BUSINESS] find daily alarm statistics fail:", err)
		response.Fail(ctx, "查询告警统计失败", nil)
		return
	}

	response.Success(ctx, "查询成功", gin.H{"data": statisticservice.BuildDailyAlarmStatistics(records)})
}

// FindDailyWarningStatistics counts every warning lifecycle on its start date.
func (ctrler StatisticDataCtrler) FindDailyWarningStatistics(ctx *gin.Context) {
	var request struct {
		CorpList []int `json:"corpList"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		response.Fail(ctx, "请求参数错误", nil)
		return
	}

	startTime, err := strconv.ParseInt(ctx.Query("startTime"), 10, 64)
	if err != nil || startTime <= 0 {
		response.Fail(ctx, "开始时间错误", nil)
		return
	}
	endTime, err := strconv.ParseInt(ctx.Query("endTime"), 10, 64)
	if err != nil || endTime <= 0 || endTime < startTime {
		response.Fail(ctx, "结束时间错误", nil)
		return
	}

	userInfoCache, exists := ctx.Get("userInfo")
	if !exists {
		response.Fail(ctx, "用户信息不存在", nil)
		return
	}
	userInfo := userInfoCache.(usermodel.User)

	if userInfo.Isadmin != 1 {
		userCorpIds := make([]int, 0, len(userInfo.CorpIds))
		for _, corpID := range userInfo.CorpIds {
			userCorpIds = append(userCorpIds, int(corpID))
		}
		authorizedCorpIds := service.FindAllSonCorp(ctx, userCorpIds)
		authorized := make(map[int]struct{}, len(authorizedCorpIds))
		for _, corpID := range authorizedCorpIds {
			authorized[corpID] = struct{}{}
		}
		for _, corpID := range request.CorpList {
			if _, ok := authorized[corpID]; !ok {
				response.Fail(ctx, "非法部门, 未授权", nil)
				return
			}
		}
	}

	corpIds := service.FindAllSonCorp(ctx, request.CorpList)
	equipmentIds := service.FindAllCorpEquipmentId(ctx, corpIds)
	if len(equipmentIds) == 0 {
		response.Success(ctx, "查询成功", gin.H{"data": []statisticservice.DailyWarningStatistics{}})
		return
	}

	alarmDao := commondao.NewAlarmDataRecordDao(ctx)
	records, err := alarmDao.FindDailyWarningStatisticRecords(equipmentIds, startTime, endTime)
	if err != nil {
		log.Println("[BUSINESS] find daily warning statistics fail:", err)
		response.Fail(ctx, "查询预警统计失败", nil)
		return
	}

	response.Success(ctx, "查询成功", gin.H{"data": statisticservice.BuildDailyWarningStatistics(records)})
}

func (ctrler StatisticDataCtrler) FindChargeDischargeDailyStatistics(ctx *gin.Context) {
	var request struct {
		CorpList []int `json:"corpList"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil && err != io.EOF {
		response.Fail(ctx, "请求参数错误", nil)
		return
	}

	startTime, err := strconv.ParseInt(ctx.Query("startTime"), 10, 64)
	if err != nil || startTime <= 0 {
		response.Fail(ctx, "开始时间错误", nil)
		return
	}
	endTime, err := strconv.ParseInt(ctx.Query("endTime"), 10, 64)
	if err != nil || endTime <= 0 || endTime < startTime {
		response.Fail(ctx, "结束时间错误", nil)
		return
	}

	userInfoCache, exists := ctx.Get("userInfo")
	if !exists {
		response.Fail(ctx, "用户信息不存在", nil)
		return
	}
	userInfo, validUser := userInfoCache.(usermodel.User)
	if !validUser {
		response.Fail(ctx, "用户信息无效", nil)
		return
	}

	corpIDs := request.CorpList
	if userInfo.Isadmin != 1 {
		userCorpIDs := make([]int, 0, len(userInfo.CorpIds))
		for _, corpID := range userInfo.CorpIds {
			userCorpIDs = append(userCorpIDs, int(corpID))
		}
		authorizedCorpIDs := service.FindAllSonCorp(ctx, userCorpIDs)
		authorized := make(map[int]struct{}, len(authorizedCorpIDs))
		for _, corpID := range authorizedCorpIDs {
			authorized[corpID] = struct{}{}
		}
		for _, corpID := range corpIDs {
			if _, ok := authorized[corpID]; !ok {
				response.Fail(ctx, "非法部门, 未授权", nil)
				return
			}
		}
		if len(corpIDs) == 0 {
			corpIDs = authorizedCorpIDs
		}
	}
	if len(corpIDs) > 0 {
		corpIDs = service.FindAllSonCorp(ctx, corpIDs)
	}

	equipmentDAO := dao.NewEquipmentDAO(ctx)
	var equipmentCorpIDs []int
	if len(corpIDs) > 0 {
		equipmentCorpIDs = corpIDs
	} else if userInfo.Isadmin != 1 {
		// No authorized departments means no equipment, not the whole tenant.
		equipmentCorpIDs = []int{}
	}
	equipments, err := equipmentDAO.FindEquipmentByCorpIDsExact(equipmentCorpIDs)
	if err != nil {
		log.Println("[BUSINESS] find charge/discharge equipment failed:", err)
		response.Fail(ctx, "查询电池失败", nil)
		return
	}
	statistics, err := statisticservice.FindChargeDischargeDailyStatistics(ctx, equipments, startTime, endTime)
	if err != nil {
		log.Println("[BUSINESS] find charge/discharge statistics failed:", err)
		response.Fail(ctx, "查询充放电统计失败", nil)
		return
	}
	response.Success(ctx, "查询成功", gin.H{
		"data":     statistics.Data,
		"quality":  statistics.Quality,
		"complete": statistics.Complete,
	})
}
