package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	bloomutil "dicloud/util/bloom"
	"fmt"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
)

var handleSTRLock sync.Mutex

/* 数据处理格式 %s %s %s */
func HandleTypeStrSwitchTrigger(msgOrg *primitive.MessageExt) {

	handleSTRLock.Lock()
	defer handleSTRLock.Unlock()

	// 获取数据, 转换数据类型
	msg := string(msgOrg.Body)

	gatewayID, dataID, value, triggerTimestamp, dbName, ok := parseTriggerPointText(msg)
	if !ok {
		return
	}
	now := time.Now().Unix()
	// TODO: acording to data mark. distingush plat data.
	ctx := &gin.Context{}
	if len(dbName) > 0 {
		ctx.Set("dbName", dbName)
	}
	if triggerTimestamp > 0 {
		now = triggerTimestamp
	}

	// Public TTDC string rules are stored in general_trigger_config. Keep the
	// existing trigger_config path below unchanged for ordinary string rules.
	HandleGeneralStringTriggerData(ctx, gatewayID, dataID, value, now)

	// 查缓存, 判断是否有对应配置
	if !bloomutil.LocalBloomFilter.Test([]byte(gatewayID + " " + dataID)) {
		// 数据一定不存在, 不查库
		// log.Println("[TRIGGER] bloom judge not exist!")
		return
	}

	// 数据可能存在(hash误差), 查库
	triggerDao := dao.NewTriggerDao(ctx)
	config := model.TriggerConfig{
		Marktrigger: gatewayID + " " + dataID,
	}

	configList, res := triggerDao.ExistMark(config)
	if !res {
		return
	}

	fmt.Println("[HandleTypeStrSwitchTrigger] 48 check value: ", value)
	for idx := range configList {

		// 修改为新的数据结构
		if len(configList[idx].StrMatchs) == 0 {
			continue
		}

		// result 0: 未触发 1:触发状态
		result := 0
		for index3 := range configList[idx].StrMatchs {

			if len(configList[idx].StrMatchs[index3].MatchStr) > 0 {
				switch value {
				case configList[idx].StrMatchs[index3].MatchStr:
					result = 1
					if configList[idx].StrMatchs[index3].ReplaceCodeFlag {
						// 替换报警代码, 替换统计代码
						configList[idx].Code = configList[idx].StrMatchs[idx].ReplaceCode
						configList[idx].AlarmUser.AlarmCode = configList[idx].StrMatchs[idx].ReplaceCode
						if len(configList[idx].StatisticConfig.StatisticCode) > 0 {
							configList[idx].StatisticConfig.StatisticCode = configList[idx].StrMatchs[idx].ReplaceCode
						}
					}
				}

				if result == 1 {
					break
				}
			} else {
				// 默认就是报警数据, 直接报警
				result = 1
				break
			}
		}

		/** 触发配置 */
		unit := configList[idx]

		// 将上次的触发(字符串code) 恢复
		// 恢复不统计
		if len(unit.LastStrMatch) > 0 && unit.Code != unit.LastStrMatch {
			var lastUnit = model.TriggerConfig{
				ID:           unit.ID,
				SiteId:       unit.SiteId,       // 站点ID
				EquipmentId:  unit.EquipmentId,  // 站点ID
				TriggerType:  unit.TriggerType,  // 触发器类型 0: 报警触发器, 1: 统计触发器 2: 报警+统计...
				TriggerModel: unit.TriggerModel, // 触发器处理模型/模式(区分浮点型,字符串) 0: 浮点型, 1: 字符串
				Code:         unit.LastStrMatch, // 触发代码 - 上次触发的代码
				ApiKey:       unit.ApiKey,       // 数据池key
				LabelName:    unit.LabelName,    // 数据池数据点名称
				LabelId:      unit.LabelId,      // 数据池数据点ID
				Cycle:        unit.Cycle,        // 触发模式 0: 表示单次触发模式; 非0: 表示周期触发模式,且值为触发抑制周期(周期时间内数据不触发不推送-过滤)
				Timer:        unit.Timer,        // 报警时间 0: 未报警,正常; 非0: 表示第一次报警状态时间记录
				Record:       unit.Record,       // 触发器上次触发状态 0: 上次未报警 非0: 上一次报警, 上次报警时间戳
				LastStrMatch: unit.LastStrMatch, // 上一次报警str记录
				// Marktrigger:  unit.Marktrigger,  // 去除, 恢复不更新
				Formula: unit.Formula, // 数据池数据点处理公式
			}

			// 直接恢复 - 其中的一个报警
			normalSingleOrCycleJudge(ctx, lastUnit, 0, now, value, triggerDao)

			// 可以设置unit为第一次触发 TODO: 防止同个触发器逻辑未转出
			// 可不操作, 本次报警专为恢复数据
			// 下次再报警, 即报警; 忽略一次即时即刻报警
			// unit.Record = 0
		}

		unit.LastStrMatch = unit.Code
		normalSingleOrCycleJudge(ctx, unit, int64(result), now, value, triggerDao)
	}
}
