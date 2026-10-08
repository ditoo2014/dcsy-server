package controller

import (
	"dicloud/internal/apiserver/service"
	tolinkdao "dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	usermodel "dicloud/models/model"
	"dicloud/util/response"
	"log"
	"sync"

	"github.com/gin-gonic/gin"
)

type IApiCtrler interface {
	GetLabel(ctx *gin.Context)
	GetData(ctx *gin.Context)
	GetLive(ctx *gin.Context)
	GetApiListDataAll(ctx *gin.Context)
}

type ApiCtrler struct {
}

func (a ApiCtrler) GetLive(ctx *gin.Context) {

	response.Success(ctx, "执行成功", gin.H{"result": service.HashGetAllGtwLive(ctx, "gtw_live")})
}

func (a ApiCtrler) GetData(ctx *gin.Context) {

	/* 获取流信息 */

	streamInfoCache, _ := ctx.Get("streamInfo")
	streamInfo := streamInfoCache.(model.Stream)

	function := ctx.Query("Function")

	//获取当前数据池的全部标签
	streamDao := tolinkdao.NewStreamDAO(ctx)

	label, err := streamDao.GetLabel(streamInfo)
	if err != nil {
		response.Fail(ctx, "系统出错", gin.H{"error": err})
		return
	}

	msg := gin.H{}
	switch function {
	case "status":

		var (
			time   int64
			status int
		)
		for _, v := range label {
			status, _, time = service.RealTimeDataHashGet(ctx, v.GatewayId, v.DataId)
			if status < 0 {
				break
			}
		}
		msg = gin.H{"time": time, "status": status}

	case "formula":
		//以前的功能模块，暂时未使用

	case "realtime":

		var (
			labelIdList []string
			statusList  []int
			valueList   []string
			timeList    []int64
		)

		if err := ctx.Bind(&labelIdList); err != nil {
			response.Fail(ctx, "获取参数失败", gin.H{"error": err})
			return
		}

		for _, s := range labelIdList {
			for _, v := range label {
				if s == v.LabelId {
					status, value, times := service.RealTimeDataHashGet(ctx, v.GatewayId, v.DataId)
					statusList = append(statusList, status)
					valueList = append(valueList, value)
					timeList = append(timeList, times)
				}
			}
		}
		msg = gin.H{"labels": labelIdList, "status": statusList, "valueList": valueList, "timeList": timeList}

	case "pagetime":

		// 基础数据, 查询参数封装
		dateConfig := usermodel.Date{}
		ctx.BindQuery(&dateConfig)

		pageConfig := usermodel.Page{}
		ctx.BindQuery(&pageConfig)

		var labelUnit model.StreamLabelUnit
		if err := ctx.Bind(&labelUnit); err != nil {
			response.Fail(ctx, "参数错误", nil)
			return
		}

		// 获取具体网关信息
		labelUnit = streamDao.GetLabelUnitByLabelId(labelUnit.LabelId)

		// 查询数据
		// count, times, values, status, err2 := service.GetHistoryData(labelUnit, DateStart, DateEnd, PageSize, Pagenum)
		count, times, values, status, err2 := tolinkdao.FindGatewayDataByTime(
			model.GatewayData{GatewayId: labelUnit.GatewayId, DataId: labelUnit.DataId},
			dateConfig, pageConfig, labelUnit.Formula, labelUnit.DataType == "2",
		)
		if err2 != nil {
			writeHistoryQueryFailure(ctx, err2)
			return
		}
		msg = gin.H{"label": labelUnit.LabelId, "count": count, "times": times, "values": values, "status": status}

	case "pagetimelist":

		// 基础数据, 查询参数封装
		dateConfig := usermodel.Date{}
		ctx.BindQuery(&dateConfig)

		pageConfig := usermodel.Page{}
		ctx.BindQuery(&pageConfig)

		list := []string{}
		ctx.Bind(&list)

		var (
			resultList []map[string]interface{}
		)
		var labelUnit model.StreamLabelUnit

		for idx := range list {

			hisData := make(map[string]interface{})

			label := model.StreamLabelUnit{LabelId: list[idx]}
			labelUnit = streamDao.GetLabelUnitByLabelId(label.LabelId)

			// count, times, values, status, err2 := service.GetHistoryData(labelUnit, DateStart, DateEnd, PageSize, Pagenum)
			count, times, values, status, err2 := tolinkdao.FindGatewayDataByTime(
				model.GatewayData{GatewayId: labelUnit.GatewayId, DataId: labelUnit.DataId},
				dateConfig, pageConfig, labelUnit.Formula, labelUnit.DataType == "2",
			)
			if err2 != nil {
				log.Println("[API] illegal label")
				writeHistoryQueryFailure(ctx, err2)
				return
			}

			hisData["label"] = label.LabelId
			hisData["count"] = count
			hisData["times"] = times
			hisData["values"] = values
			hisData["status"] = status

			resultList = append(resultList, hisData)
		}

		msg = gin.H{"data": resultList}

	case "countdata":

		// 基础数据, 查询参数封装
		dateConfig := usermodel.Date{}
		ctx.BindQuery(&dateConfig)

		var labelUnit model.StreamLabelUnit
		if err := ctx.Bind(&labelUnit); err != nil {
			response.Fail(ctx, "参数错误", nil)
			return
		}
		labelUnit = streamDao.GetLabelUnitByLabelId(labelUnit.LabelId)

		// count, err := service.GetDataCount(labelUnit, DateStart, DateEnd)
		count, err := tolinkdao.CountGatewayDataByTime(
			model.GatewayData{GatewayId: labelUnit.GatewayId, DataId: labelUnit.DataId},
			dateConfig,
		)
		if err != nil {
			response.Fail(ctx, "系统错误", gin.H{"error": err})
			return
		}
		msg = gin.H{"count": count}

	case "newdata":

		var labelUnit model.StreamLabelUnit
		if err := ctx.Bind(&labelUnit); err != nil {
			response.Fail(ctx, "参数错误", nil)
			return
		}
		labelUnit = streamDao.GetLabelUnitByLabelId(labelUnit.LabelId)
		//获取最新的30条数据，来自influxdb
		// times, values, status, err2 := service.GetNewData(labelUnit)
		times, values, status, err2 := tolinkdao.GetDataNew(
			model.GatewayData{GatewayId: labelUnit.GatewayId, DataId: labelUnit.DataId},
			labelUnit.Formula, labelUnit.DataType == "2",
		)
		if err2 != nil {
			response.Fail(ctx, "系统错误", gin.H{"error": err2})
			return
		}
		msg = gin.H{"times": times, "values": values, "status": status}

	case "historydatapasstime":

		// 基础数据, 查询参数封装
		dateConfig := usermodel.Date{}
		ctx.BindQuery(&dateConfig)

		pageConfig := usermodel.Page{}
		ctx.BindQuery(&pageConfig)

		//时间值,时间长度
		timeLength := ctx.Query("timeLength")
		//时间类型(单位:second,minute,hour),秒,分钟,小时
		timeType := ctx.Query("timeType")

		var labelUnit model.StreamLabelUnit
		if err := ctx.Bind(&labelUnit); err != nil {
			response.Fail(ctx, "参数错误", nil)
			return
		}
		if len(labelUnit.LabelId) == 0 {
			response.Fail(ctx, "参数为空", nil)
			return
		}
		labelUnit = streamDao.GetLabelUnitByLabelId(labelUnit.LabelId)
		if len(labelUnit.DataId) == 0 {
			response.Fail(ctx, "无效数据", nil)
			return
		}

		// Change to common dao
		count, times, values, status, err2 := tolinkdao.GetDataHistoryPassTimeList(
			model.GatewayData{GatewayId: labelUnit.GatewayId, DataId: labelUnit.DataId},
			dateConfig, pageConfig, labelUnit.Formula,
			timeLength, timeType, labelUnit.DataType == "2",
		)
		writeHistoryPassTimeResponse(ctx, labelUnit.LabelId, count, times, values, status, err2)
		return

	case "historydatapasstimelist":
		var request HistoryPassTimeBatchRequest
		if err := ctx.Bind(&request); err != nil || len(request.Requests) == 0 {
			response.Fail(ctx, "参数错误", nil)
			return
		}

		labelMap := make(map[string]model.StreamLabelUnit, len(label))
		for _, labelItem := range label {
			labelMap[labelItem.LabelId] = labelItem
		}

		resultList := make([]HistoryPassTimeBatchResult, len(request.Requests))
		var waitGroup sync.WaitGroup
		for index, requestItem := range request.Requests {
			waitGroup.Add(1)
			go func(resultIndex int, item HistoryPassTimeBatchItem) {
				defer waitGroup.Done()
				result := HistoryPassTimeBatchResult{LabelId: item.LabelId}
				labelUnit, exist := labelMap[item.LabelId]
				if !exist || len(labelUnit.DataId) == 0 {
					result.Error = "无效数据"
					resultList[resultIndex] = result
					return
				}

				pageNum := item.PageNum
				if pageNum <= 0 {
					pageNum = 1
				}
				pageSize := item.PageSize
				if pageSize <= 0 {
					pageSize = 10000
				}
				timeLength := item.TimeLength
				if len(timeLength) == 0 {
					timeLength = "30"
				}
				timeType := item.TimeType
				if len(timeType) == 0 {
					timeType = "second"
				}

				count, times, values, status, queryErr := tolinkdao.GetDataHistoryPassTimeList(
					model.GatewayData{GatewayId: labelUnit.GatewayId, DataId: labelUnit.DataId},
					usermodel.Date{Datestart: item.DateStart, Dateend: item.DateEnd},
					usermodel.Page{Pagenum: pageNum, Pagesize: pageSize},
					labelUnit.Formula, timeLength, timeType, labelUnit.DataType == "2",
				)
				result.Count = count
				result.Times = times
				result.Values = values
				result.Status = status
				if queryErr != nil {
					log.Printf("[API] 批量历史数据查询失败: %v", queryErr)
					result.Error = "历史数据查询失败，请检查历史存储服务"
				}
				resultList[resultIndex] = result
			}(index, requestItem)
		}
		waitGroup.Wait()
		writeHistoryBatchResponse(ctx, resultList)
		return

	default:
		response.Fail(ctx, "功能指令错误", nil)
		return

	}
	response.Success(ctx, "success", msg)
}

// 数据库错误必须与“成功查询到零条记录”区分，避免前端把故障渲染成空曲线。
// 详细存储错误只写服务日志，不将连接信息等内部细节直接返回给浏览器。
func writeHistoryQueryFailure(ctx *gin.Context, err error) {
	log.Printf("[API] 历史数据查询失败: %v", err)
	response.Fail(ctx, "历史数据查询失败，请检查历史存储服务", nil)
}

func writeHistoryPassTimeResponse(ctx *gin.Context, labelID string, count int64, times []string, values []interface{}, status []int64, queryErr error) {
	if queryErr != nil {
		writeHistoryQueryFailure(ctx, queryErr)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"label": labelID, "count": count, "times": times, "values": values, "status": status})
}

// 批量请求保留每个测点的结果，但任何测点失败都不能将整个响应标记为成功。
func writeHistoryBatchResponse(ctx *gin.Context, results []HistoryPassTimeBatchResult) {
	for _, result := range results {
		if result.Error != "" {
			response.Fail(ctx, "部分历史数据查询失败，请检查历史存储服务", gin.H{"data": results})
			return
		}
	}
	response.Success(ctx, "查询成功", gin.H{"data": results})
}

func (a ApiCtrler) GetLabel(ctx *gin.Context) {
	/* 获取流信息 */

	streamInfoCache, _ := ctx.Get("streamInfo")
	streamInfo := streamInfoCache.(model.Stream)

	streamDao := tolinkdao.NewStreamDAO(ctx)

	list, err := streamDao.StreamLabels(streamInfo)
	if err != nil {
		response.Fail(ctx, "查询失败", gin.H{"error": err})
		return
	}
	response.Success(ctx, "获取成功", gin.H{"labelList": list.List})
}

type RequestApiList struct {
	Data []ApiData `json:"data"`
}

type ApiData struct {
	Apikey     string   `json:"apiKey"`
	RequestKey string   `json:"requestKey"`
	LabelList  []string `json:"labelList"`
}

type HistoryPassTimeBatchRequest struct {
	Requests []HistoryPassTimeBatchItem `json:"requests"`
}

type HistoryPassTimeBatchItem struct {
	LabelId    string `json:"labelId"`
	DateStart  uint64 `json:"dateStart"`
	DateEnd    uint64 `json:"dateEnd"`
	PageNum    int64  `json:"pageNum"`
	PageSize   int64  `json:"pageSize"`
	TimeLength string `json:"timeLength"`
	TimeType   string `json:"timeType"`
}

type HistoryPassTimeBatchResult struct {
	LabelId string        `json:"labelId"`
	Count   int64         `json:"count"`
	Times   []string      `json:"times"`
	Values  []interface{} `json:"values"`
	Status  []int64       `json:"status"`
	Error   string        `json:"error,omitempty"`
}

type apiDataCache struct {
	stream   model.Stream
	labelMap map[string]model.StreamLabelUnit
	message  string
}

func (ctrler ApiCtrler) GetApiListDataAll(ctx *gin.Context) {

	params := RequestApiList{}
	ctx.Bind(&params)

	var reMsg = gin.H{}

	//获取当前数据池的全部标签
	streamDao := tolinkdao.NewStreamDAO(ctx)

	if len(params.Data) > 0 {
		var resultList []map[string]interface{}
		dataCache := make(map[string]apiDataCache)
		for _, apikeyStream := range params.Data {

			realtimeMap := make(map[string]interface{})
			realtimeMap["apikey"] = apikeyStream.Apikey
			realtimeMap["requestKey"] = apikeyStream.RequestKey

			// 查询数据池
			if 1 > len(apikeyStream.Apikey) {
				realtimeMap["msg"] = "illegal apikey"
				resultList = append(resultList, realtimeMap)
				log.Println("[ApiCtrler] GetData GetApiListDataAll, apikey is null")
				continue
			}

			cache, cached := dataCache[apikeyStream.Apikey]
			if !cached {
				dbStream, res := streamDao.ExistApiKey(model.Stream{ApiKey: apikeyStream.Apikey})
				if !res {
					cache.message = "apikey not exist"
				} else {
					labelList, err := streamDao.GetLabel(dbStream)
					if err != nil {
						cache.message = "empty label list"
					} else {
						cache.stream = dbStream
						cache.labelMap = make(map[string]model.StreamLabelUnit, len(labelList))
						for _, labelItem := range labelList {
							cache.labelMap[labelItem.LabelId] = labelItem
						}
					}
				}
				dataCache[apikeyStream.Apikey] = cache
			}
			if len(cache.message) > 0 {
				realtimeMap["msg"] = cache.message
				resultList = append(resultList, realtimeMap)
				continue
			}

			var (
				LabelListTem []string
				statusList   []int
				valueList    []string
				timeList     []int64
			)
			for _, s := range apikeyStream.LabelList {
				labelItem, exist := cache.labelMap[s]
				if !exist {
					realtimeMap["msg"] = "none data exist"
					continue
				}
				status, value, times := service.RealTimeDataHashGet(ctx, labelItem.GatewayId, labelItem.DataId)
				statusList = append(statusList, status)
				valueList = append(valueList, value)
				timeList = append(timeList, times)
				LabelListTem = append(LabelListTem, s)
			}

			keyValueList := make(map[string]interface{})
			keyValueList["status"] = statusList
			keyValueList["value"] = valueList
			keyValueList["time"] = timeList
			keyValueList["label"] = LabelListTem

			// 封装单个数据池数据
			realtimeMap["name"] = cache.stream.Name
			realtimeMap["data"] = keyValueList

			resultList = append(resultList, realtimeMap)
		}

		reMsg = gin.H{"data": resultList, "state": "success"}
	} else {
		reMsg = gin.H{"state": "error", "msg": "不正常的请求"}
	}

	response.Success(ctx, "执行成功", reMsg)
}
