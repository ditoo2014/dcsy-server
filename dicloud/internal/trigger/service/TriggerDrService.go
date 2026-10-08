// dtdp协议 实时触发器

package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	"dicloud/util"
	bloomutil "dicloud/util/bloom"
	"encoding/json"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
)

var handleDRLock sync.Mutex

/* 数据格式 gatewayID{datastr} */
func HandleDrTrigger(msgOrg *primitive.MessageExt) {

	handleDRLock.Lock()
	defer handleDRLock.Unlock()

	// 获取数据, 转换数据类型
	msg := string(msgOrg.Body)

	gatewayID, dataStr := "", ""

	/* 格式解析 */

	for i, length := 0, len(msg)-1; i < length; i++ {
		if msg[i] == '{' {
			gatewayID = msg[:i]
			dataStr = msg[i:]
			break
		}
	}

	if gatewayID == "" || dataStr == "" {
		return
	}

	/* 数据解析 */

	var PackData model.PackDtdpData

	err := json.Unmarshal([]byte(dataStr), &PackData)
	if nil != err {
		return
	}

	var now int64
	var value float64

	if PackData.T == 0 {
		now = time.Now().Unix()
	} else {
		now = PackData.T
	}

	// TODO: acording to data mark. distingush plat data.
	ctx := &gin.Context{}
	if len(PackData.DbName) > 0 {
		ctx.Set("dbName", PackData.DbName)
	}

	/** trigger dao 初始化 */
	triggerDao := dao.NewTriggerDao(ctx)

	for idx := range PackData.Dl {

		if 0 >= PackData.Dl[idx].S {
			continue
		}

		value = PackData.Dl[idx].V

		// 通用触发器触发判断 - 触发
		HandleGeneralTriggerData(ctx, gatewayID, PackData.Dl[idx].ID, value, now)

		// 查缓存, 判断是否有对应配置
		if !bloomutil.LocalBloomFilter.Test([]byte(gatewayID + " " + PackData.Dl[idx].ID)) {
			// 数据一定不存在, 不查库
			// log.Println("[TRIGGER] bloom judge not exist!")
			return
		}

		config := model.TriggerConfig{
			Marktrigger: gatewayID + " " + PackData.Dl[idx].ID,
		}

		configList, res := triggerDao.ExistMark(config)
		if !res {
			continue
		}

		for idx := range configList {

			// 修改为新的数据结构
			if nil != err || len(configList[idx].Logics) == 0 {
				continue
			}

			value, err = util.ArithmeticNew(value, configList[idx].Formula)
			if nil != err {
				continue
			}

			result := 1
			for idx2 := range configList[idx].Logics {

				// 逻辑判断符号/值
				logic := configList[idx].Logics[idx2].Logic
				obj3 := configList[idx].Logics[idx2].TriggerValue

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

			// 触发配置
			unit := configList[idx]

			switch {
			/* 单次报警模式 */
			case unit.Cycle == 0:
				switch {
				////// 上一次正常
				case unit.Record == 0:
					switch result {
					//// 此次正常
					case 0:
						{
							// {不执行操作}

							// add 20251210. 增加触发持续时间
							if unit.SustainTime != 0 {
								// 恢复持续时间, 避免未重置
								triggerDao.SetSustainTimer(unit, 0)
							}
						}
					//// 此次触发
					case 1:
						{

							// add 20251210. 增加触发持续时间
							if unit.SustainTime != 0 {

								if unit.SustainRecord == 0 {
									// 只记录
									triggerDao.SetSustainTimer(unit, now)
									// 直接跳过, 进入下次判断
									continue
								} else {
									if (now - unit.SustainRecord) >= unit.SustainTime {
										// 不为0, 且达到持续时间. 报警

										// 存储参数
										// 计时器		-	当前时间
										err = triggerDao.SetRecordTimer(unit, now, -1)
										if nil != err {
											continue
										}

										// 发送
										SendMsg(ctx, unit, value, now, 0)
									}

									// now时间未达到标准的时候
									continue
								}
							}

							// 存储参数
							// 计时器		-	当前时间
							err = triggerDao.SetRecordTimer(unit, now, -1)
							if nil != err {
								continue
							}

							// 打包信息 {触发开始}
							// 开始时间		-	当前时间
							// 结束时间		-	无
							// 代码			-	触发器编码
							// 值			-	当前值
							SendMsg(ctx, unit, value, now, 0)
							return
						}
					}
				////// 上一次触发
				case unit.Record != 0:
					switch result {
					//// 此次正常
					case 0:
						{

							// add 20251210. 增加触发持续时间
							if unit.SustainTime != 0 {
								// 恢复持续时间记录
								triggerDao.SetSustainTimer(unit, 0)
							}

							// 存储参数
							// 计时器		-	置 0
							err = triggerDao.SetRecordTimer(unit, 0, now)
							if nil != err {
								continue
							}

							// 打包信息 {触发结束}
							// 开始时间		-	记录点
							// 结束时间		-	当前时间
							// 代码			-	触发器编码
							// 值			-	当前值
							SendMsg(ctx, unit, value, unit.Record, now)
							return
						}
					//// 此次触发
					case 1:
						{
							// {不执行操作}
						}
					}
				}
			/* 周期报警模式 */
			case unit.Cycle != 0:
				switch {
				////// 上一次正常
				case unit.Timer == 0:
					switch {
					//// 此次正常
					case result == 0:
						{
							// fmt.Println("一直正常")

							// {不执行操作}

							// add 20251210. 增加触发持续时间
							if unit.SustainTime != 0 {
								// 恢复持续时间, 避免未重置
								triggerDao.SetSustainTimer(unit, 0)
							}
						}
					//// 此次触发
					case result == 1:
						{
							// fmt.Println("第一次不正常")

							// 打包信息 {无}

							// add 20251210. 增加触发持续时间
							if unit.SustainTime != 0 {

								if unit.SustainRecord == 0 {
									// 只记录
									triggerDao.SetSustainTimer(unit, now)

									// 直接跳过, 进入下次判断
									continue
								} else {
									if (now - unit.SustainRecord) >= unit.SustainTime {
										// 不为0, 且达到持续时间. 报警

										// 存储参数
										// 计时器		-	当前时间
										// 设置 record - now, 下一次直接周期循环 - 不进入record == 0
										err = triggerDao.SetRecordTimer(unit, now, now)
										if nil != err {
											continue
										}

										// 发送
										SendMsg(ctx, unit, value, now, 0)
									}

									// now时间未达到标准的时候
									continue
								}
							}

							// 存储参数
							// 记录点		-	置 0
							// 计时器		-	装载当前时间
							// 设置 record - now, 下一次直接周期循环 - 不进入record == 0
							err = triggerDao.SetRecordTimer(unit, now, now)
							if nil != err {
								continue
							}
							// 发送-无

							// 发送
							SendMsg(ctx, unit, value, now, 0)
						}
					}
				////// 上一次触发
				case unit.Timer != 0:
					switch {
					//// 此次正常 && 记录点为 0
					case result == 0 && unit.Record == 0:
						{
							// fmt.Println("不正常，但持续时间不够，恢复正常")

							// 打包信息 {无}

							// add 20251210. 增加触发持续时间
							if unit.SustainTime != 0 {
								// 恢复持续时间记录
								triggerDao.SetSustainTimer(unit, 0)
							}

							// 存储参数
							// 计时器		-	置 0
							err = triggerDao.SetRecordTimer(unit, -1, 0)
							if nil != err {
								continue
							}

							SendMsg(ctx, unit, value, unit.Record, now)
							continue
						}
					//// 此次正常 && 记录点不为 0
					case result == 0 && unit.Record != 0:
						{
							// fmt.Println("不正常，持续时间足够，现恢复正常")

							// add 20251210. 增加触发持续时间
							if unit.SustainTime != 0 {
								// 恢复持续时间记录
								triggerDao.SetSustainTimer(unit, 0)
							}

							// 存储参数
							// 记录点		-	置 0
							// 计时器		-	置 0
							err = triggerDao.SetRecordTimer(unit, 0, 0)
							if nil != err {
								continue
							}

							// 打包信息 {触发结束}
							// 开始时间		-	记录点
							// 结束时间		-	当前时间
							// 代码			-	触发器编码
							// 值			-	当前值
							SendMsg(ctx, unit, value, unit.Record, now)
							return
						}
					//// 此次触发 && 记录点为 0 && 持续时间小于周期
					case result == 1 && unit.Record == 0 && unit.Cycle > now-unit.Timer:
						{
							// fmt.Println("不正常，持续时间不够，现依旧不正常", now-unit.Timer)

							// {不执行操作}
						}
					//// 此次触发 && 记录点为 0 && 持续时间大于周期
					case result == 1 && unit.Record == 0 && unit.Cycle <= now-unit.Timer:
						{
							// fmt.Println("不正常，持续时间第一次够，报警")

							// 存储参数
							// 记录点		-	当前时间
							// 计时器		-	当前时间
							// err = triggerDao.SetRecordTimer(unit, now, now)
							// if nil != err {
							// 	continue
							// }

							// // 打包信息 {触发开始}
							// // 开始时间		-	当前时间
							// // 结束时间		-	0
							// // 代码			-	触发器编码
							// // 值			-	当前值
							// SendMsg(ctx, unit, value, now, 1)
							// return
						}
					//// 此次触发 && 记录点不为 0 && 持续时间小于周期
					case result == 1 && unit.Record != 0 && unit.Cycle > now-unit.Timer:
						{
							// fmt.Println("不正常，持续时间足够，现等待下一次报警")

							// {不执行操作}
						}
					//// 此次触发 && 记录点不为 0 && 持续时间大于周期
					case result == 1 && unit.Record == 0 && unit.Cycle <= now-unit.Timer:
						{
							// fmt.Println("不正常，持续时间足够，执行阶段性报警")

							// 存储参数
							// 计时器		-	装载当前时间
							err = triggerDao.SetRecordTimer(unit, -1, now)
							if nil != err {
								continue
							}

							// 打包信息 {触发持续}
							// 开始时间		-	当前时间
							// 结束时间		-	1
							// 代码			-	触发器编码
							// 值			-	当前值
							SendMsg(ctx, unit, value, now, 1)
							return
						}
					}
				}
			}
		}
	}
}
