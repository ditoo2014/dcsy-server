package controller

import (
	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	"dicloud/internal/tolink/service"
	usermodel "dicloud/models/model"
	"dicloud/pkg/cache"
	"dicloud/util/response"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

/** 时序数据操作 */
type ITimeSequenceDataCtrler interface {
	GetData(ctx *gin.Context)

	DeleteTableAllData(ctx *gin.Context)
	DeleteDataOneByIdTime(ctx *gin.Context)
}

/** 时序数据操作 */
type TimeSequenceDataCtrler struct {
}

// 查询数据
func (ctrler TimeSequenceDataCtrler) GetData(ctx *gin.Context) {

	/* 获取网关信息 */
	gatewayInfoCache, _ := ctx.Get("gatewayInfo")
	gatewayInfo := gatewayInfoCache.(model.Gateway)

	/* 验证信息 */

	/* 获取操作信息 */
	function := ctx.Query("Function")

	/* 提取数据 */
	var reMsg = gin.H{}

	switch function {
	case "intact":
		// 数据列表
		// 显示对应网关，所有通道下，数据点
		var err error
		reMsg, err = service.FindGatewayDatas(ctx, gatewayInfo.GatewayId)
		if err != nil {
			return
		}

	case "countdata":

		// 查询某个数据点的历史数据
		dateConfig := usermodel.Date{}
		ctx.BindQuery(&dateConfig)

		dataConfig := model.GatewayData{}
		ctx.Bind(&dataConfig)

		if 1 > len(dataConfig.DataId) {
			response.Fail(ctx, "数据点ID为空", nil)
			return
		}

		gatewayDataDao := dao.NewGatewayDataDao(ctx)
		dataTem, res := gatewayDataDao.FindOne(model.GatewayData{GatewayId: gatewayInfo.GatewayId, DataId: dataConfig.DataId})

		if !res {
			response.Fail(ctx, "数据点ID不存在", nil)
			return
		}

		if !checkHistoryTable(ctx, function, dataTem, dao.FindExistGateways) {
			return
		}
		count, err := dao.CountGatewayDataByTime(dataTem, dateConfig)
		if nil != err {
			response.Fail(ctx, err.Error(), nil)
			return
		}

		reMsg = gin.H{"data": dataConfig.DataId, "count": count}

	case "historydata":

		// 查询某个数据点的历史数据
		dateConfig := usermodel.Date{}
		ctx.BindQuery(&dateConfig)

		pageConfig := usermodel.Page{}
		ctx.BindQuery(&pageConfig)

		dataConfig := model.GatewayData{}
		ctx.Bind(&dataConfig)

		if 1 > len(dataConfig.DataId) {
			response.Fail(ctx, "数据点ID为空", nil)
			return
		}

		gatewayDataDao := dao.NewGatewayDataDao(ctx)
		dataTem, res := gatewayDataDao.FindOne(model.GatewayData{GatewayId: gatewayInfo.GatewayId, DataId: dataConfig.DataId})

		if !res {
			response.Fail(ctx, "数据点ID不存在", nil)
			return
		}

		if !checkHistoryTable(ctx, function, dataTem, dao.FindExistGateways) {
			return
		}
		formula := ctx.Query("Formula")
		count, times, values, status, err := dao.FindGatewayDataByTime(dataTem, dateConfig, pageConfig, formula, dataTem.DataType == 2)
		if nil != err {
			response.Fail(ctx, err.Error(), nil)
			return
		}

		reMsg = gin.H{"data": dataConfig.DataId, "count": count, "times": times, "values": values, "status": status}

	case "newdata":

		// 查询数据点最新的30条数据
		dataConfig := model.GatewayData{}
		ctx.Bind(&dataConfig)

		if 1 > len(dataConfig.DataId) {
			response.Fail(ctx, "数据点ID为空", nil)
			return
		}

		gatewayDataDao := dao.NewGatewayDataDao(ctx)
		dataTem, res := gatewayDataDao.FindOne(model.GatewayData{GatewayId: gatewayInfo.GatewayId, DataId: dataConfig.DataId})

		if !res {
			response.Fail(ctx, "数据点ID不存在", nil)
			return
		}

		if !checkHistoryTable(ctx, function, dataTem, dao.FindExistGateways) {
			return
		}
		formula := ctx.Query("Formula")
		times, values, status, err := dao.GetDataNew(dataTem, formula, dataTem.DataType == 2)
		if nil != err {
			response.Fail(ctx, err.Error(), nil)
			return
		}

		reMsg = gin.H{"data": dataConfig.DataId, "times": times, "values": values, "status": status}

	case "realtimedata":

		// 查询redis获取数据点的最新数据
		dataList := []string{}
		ctx.Bind(&dataList)

		var (
			_sta  int64
			_val  string
			_time int64
		)

		points := []string{}
		values := []any{}
		times := []int64{}
		status := []int64{}

		for idx := range dataList {
			result := cache.HMGet(ctx, "gateway:"+gatewayInfo.GatewayId, dataList[idx])
			if len(result) > 0 {
				if _, ok := result[0].(string); ok {

					// 字符串数据兼容
					gatewayID := []rune(gatewayInfo.GatewayId)
					if gatewayInfo.Model == "CNC-V" || string(gatewayID[len(gatewayID)-3:]) == "STR" {

						arr := strings.Split(result[0].(string), " ")
						if len(arr) == 3 {

							time, _ := strconv.ParseInt(arr[2], 10, 64)
							stat, _ := strconv.ParseInt(arr[0], 10, 64)
							if arr[1] == "" {
								arr[1] = "nil"
							}
							values = append(values, arr[1])
							points = append(points, dataList[idx])
							times = append(times, time)
							status = append(status, stat)

						} else {
							time, _ := strconv.ParseInt(arr[len(arr)-1], 10, 64)
							stat, _ := strconv.ParseInt(arr[0], 10, 64)
							valStr := ""
							for i := 1; i <= len(arr)-2; i++ {
								if i == 1 {
									valStr = arr[1]
									continue
								}
								valStr = valStr + "," + arr[i]
							}
							values = append(values, valStr)
							points = append(points, dataList[idx])
							times = append(times, time)
							status = append(status, stat)
						}
					} else {

						arr := strings.Split(result[0].(string), " ")

						_time, _ = strconv.ParseInt(arr[len(arr)-1], 10, 64)
						_sta, _ = strconv.ParseInt(arr[0], 10, 64)

						if len(arr[1:len(arr)-1]) > 1 {
							_val = strings.Join(arr[1:len(arr)-1], " ")
						} else {
							_val = strings.Join(arr[1:len(arr)-1], "")
						}

						values = append(values, _val)
						points = append(points, dataList[idx])
						times = append(times, _time)
						status = append(status, _sta)

					}
				}
			}
		}

		reMsg = gin.H{"points": points, "times": times, "values": values, "status": status}

	default:

		response.Fail(ctx, "功能指令错误", nil)
		return
	}

	/* 返回成功 */

	response.Success(ctx, "查找成功", reMsg)
}

// 数量、历史明细和最新值使用同一表检查；数据库故障不能伪装成正常的空数据。
// 查询函数作为参数传入，便于在不连接真实数据库的情况下验证响应协议。
func checkHistoryTable(ctx *gin.Context, function string, point model.GatewayData, findGateways func() ([]string, error)) bool {
	gateways, err := findGateways()
	if err != nil {
		response.Fail(ctx, "查询历史数据表失败："+err.Error(), nil)
		return false
	}
	for _, gatewayID := range gateways {
		if strings.EqualFold(gatewayID, point.GatewayId) {
			return true
		}
	}

	// 已配置数据点但尚未产生历史表时，明确返回零条，保留原有返回字段。
	empty := gin.H{"data": point.DataId}
	if function != "newdata" {
		empty["count"] = int64(0)
	}
	if function != "countdata" {
		empty["times"] = []int64{}
		empty["values"] = []interface{}{}
		empty["status"] = []int64{}
	}
	response.Success(ctx, "无数据", empty)
	return false
}

// 删除时序数据库表
func (ctrler TimeSequenceDataCtrler) DeleteTableAllData(ctx *gin.Context) {

	var (
		gatewayInfo model.Gateway //目标网关信息
	)

	/* 获取网关信息 */
	{
		gatewayInfoCache, _ := ctx.Get("gatewayInfo")
		gatewayInfo = gatewayInfoCache.(model.Gateway)
	}

	/* 验证信息 */

	if len(gatewayInfo.GatewayId) == 0 {
		response.Fail(ctx, "非法数据", nil)
		return
	}

	/* 采集数据出库 */
	{
		err := dao.DeleteDBTable(gatewayInfo)
		if nil != err {
			response.Fail(ctx, err.Error(), nil)
			return
		}
	}

	/* 返回成功 */

	response.Success(ctx, "操作成功", nil)
}

// 删除influxdb中一个数据点的一个值
func (ctrler TimeSequenceDataCtrler) DeleteDataOneByIdTime(ctx *gin.Context) {

	var (
		gatewayInfo model.Gateway //目标网关信息
	)

	/* 获取网关信息 */
	{
		gatewayInfoCache, _ := ctx.Get("gatewayInfo")
		gatewayInfo = gatewayInfoCache.(model.Gateway)
	}

	/* 验证信息 */

	//add 2024.01.25.
	dataId := ctx.Query("dataId")
	timeTem := ctx.Query("timeTem")

	/* 采集数据出库 */
	{
		err := dao.DeleteDBOneByIdTime(model.GatewayData{GatewayId: gatewayInfo.GatewayId, DataId: dataId}, timeTem)
		if nil != err {
			response.Fail(ctx, err.Error(), nil)
			return
		} else {

			//尝试更新redis数据
			// cache.HMSet(gatewayInfo.GatewayId, map[string]interface{}{
			// 	dataId: fmt.Sprintf("%d %f %d",
			// 		status, values, times,
			// 	),
			// })
			// fmt.Printf("line 388 check: %v\n", cache.HMGet(gatewayInfo.GatewayId, dataId))
		}
	}

	/* 返回成功 */

	response.Success(ctx, "操作成功", nil)
}
