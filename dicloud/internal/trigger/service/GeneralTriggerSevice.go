package service

// 通用触发器, 不区分设备数据, 直接通过设备编码(ApiKey)发送触发信息
// 只需要匹配数据点ID(DataId)即可
// 固定数据点ID/数据点名称触发器

import (
	"dicloud/internal/common/dao"
	"dicloud/util"
	bloomutil "dicloud/util/bloom"
	"fmt"

	"github.com/gin-gonic/gin"
)

// var handleGeneralTriggerLock sync.Mutex

/* 数据处理格式 %s %s %f */
func HandleGeneralTriggerData(ctx *gin.Context, gatewayId, dataId string, value float64, now int64) {

	// handleGeneralTriggerLock.Lock()
	// defer handleGeneralTriggerLock.Unlock()

	marktriggerStr := gatewayId + " " + dataId

	// "863434088202178 204959682640355328"
	if gatewayId == "863434088202178" && dataId == "204959682640355328" {
		fmt.Println("line 32 check gatewayID: ", gatewayId, " , dataID: ", dataId)
	}

	// 查缓存, 判断是否有对应配置
	if !bloomutil.BusinessGeneralTriggerBloomFilter.Test([]byte(marktriggerStr)) {
		// 数据一定不存在, 不查库
		// log.Println("[TRIGGER] bloom judge not exist!")
		return
	}
	fmt.Println("msg pass general trigger bloom: ", gatewayId, " dataId: ", dataId)

	// 数据可能存在(hash误差), 查库
	generaltriggerDao := dao.NewGeneralTriggerDao(ctx)

	configList, res := generaltriggerDao.FindByMarktrigger(marktriggerStr)
	if !res {
		return
	}

	for idx := range configList {
		// 基础信息校验
		if len(configList[idx].Logics) == 0 {
			continue
		}

		for idx2 := range configList[idx].Marktriggers {
			// 每个触发标记, 匹配一个设备, 进行设备报警
			if configList[idx].Marktriggers[idx2].Marktrigger == marktriggerStr {
				// 匹配触发器成功

				/** 数据计算 */
				value, err := util.ArithmeticNew(value, configList[idx].Formula)
				if nil != err {
					continue
				}
				/** 触发判断 */
				result := 1
				for index3 := range configList[idx].Logics {

					// 逻辑判断符号/值
					logic := configList[idx].Logics[index3].Logic
					obj3 := configList[idx].Logics[index3].TriggerValue

					switch logic {
					case ">":
						if value <= obj3 {
							result = 0
						}
					case "<":
						if value >= obj3 {
							result = 0
						}
					case ">=":
						if value < obj3 {
							result = 0
						}
					case "<=":
						if value > obj3 {
							result = 0
						}
					case "=":
						if value != obj3 {
							result = 0
						}
					}

					if result == 0 {
						break
					}
				}

				/** 触发配置 */
				unit := configList[idx]
				unitMarktrigger := configList[idx].Marktriggers[idx2]

				/** 触发处理 */
				normalSingleOrCycleGeneralJudge(ctx, unit, unitMarktrigger, int64(result), now, value, generaltriggerDao)
			}
		}
	}
}
