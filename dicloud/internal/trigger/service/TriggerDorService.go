package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	"dicloud/util"
	bloomutil "dicloud/util/bloom"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
)

var handleDORLock sync.Mutex

/* 数据处理格式 %s %s %f */
func HandleDorTrigger(msgOrg *primitive.MessageExt) {

	handleDORLock.Lock()
	defer handleDORLock.Unlock()

	// 获取数据, 转换数据类型
	msg := string(msgOrg.Body)

	gatewayID, dataID, valueText, triggerTimestamp, dbName, ok := parseTriggerPointText(msg)
	if !ok {
		return
	}
	value, err := strconv.ParseFloat(valueText, 64)
	if err != nil {
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

	// 通用触发器触发判断 - 触发
	HandleGeneralTriggerData(ctx, gatewayID, dataID, value, now)

	// 查缓存, 判断是否有对应配置
	if !bloomutil.LocalBloomFilter.Test([]byte(gatewayID + " " + dataID)) {
		// 数据一定不存在, 不查库
		// log.Println("[TRIGGER] bloom judge not exist!")
		return
	}
	fmt.Println("msg pass bloom: ", gatewayID, " dataId: ", dataID)

	// 数据可能存在(hash误差), 查库
	triggerDao := dao.NewTriggerDao(ctx)
	config := model.TriggerConfig{
		Marktrigger: gatewayID + " " + dataID,
	}

	configList, res := triggerDao.ExistMark(config)
	if !res {
		return
	}

	for idx := range configList {

		// 修改为新的数据结构
		if len(configList[idx].Logics) == 0 {
			continue
		}

		value, err := util.ArithmeticNew(value, configList[idx].Formula)
		if nil != err {
			continue
		}

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
								// unit.SetSustainTimer(now)
								triggerDao.SetSustainTimer(unit, now)
								// 直接跳过, 进入下次判断
								continue
							} else {
								if (now - unit.SustainRecord) >= unit.SustainTime {
									// 不为0, 且达到持续时间. 报警

									// 存储参数
									// 计时器		-	当前时间
									// err = unit.SetTimer(now, -1)
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

						// 打包信息 {触发开始}
						// 开始时间		-	当前时间
						// 结束时间		-	无
						// 代码			-	触发器编码
						// 值			-	当前值
						SendMsg(ctx, unit, value, now, 0)

						// 存储参数
						// 计时器		-	当前时间
						err = triggerDao.SetRecordTimer(unit, now, -1)
						if nil != err {
							continue
						}

						return
					}
				}
			////// 上一次触发
			case unit.Record != 0:
				switch result {
				//// 此次正常
				case 0:
					{
						// 打包信息 {触发结束}
						// 开始时间		-	记录点
						// 结束时间		-	当前时间
						// 代码			-	触发器编码
						// 值			-	当前值
						SendMsg(ctx, unit, value, unit.Record, now)

						// 存储参数
						// 计时器		-	当前时间
						err = triggerDao.SetRecordTimer(unit, 0, now)
						if nil != err {
							continue
						}

						return
					}
				//// 此次触发
				case 1:
					{
						// {不执行操作}

						// 如果是统计触发器, 做个兼容 - TODO:后续分离业务, 只规定触发器类型 - 不根据业务区分
						// if unit.TriggerType == 1 {
						// 	// 最新值, 记录开始时间, 最新时间
						// 	SendMsg(ctx, unit, value, unit.Record, now)
						// }

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

						// 发送
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
				case result == 1 && unit.Record != 0 && unit.Cycle <= now-unit.Timer:
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

func parseTriggerPointText(msg string) (gatewayID, dataID, value string, timestamp int64, dbName string, ok bool) {
	fields := strings.Fields(msg)
	if len(fields) < 3 {
		return
	}

	gatewayID = fields[0]
	dataID = fields[1]
	value = fields[2]

	if len(fields) >= 4 {
		if triggerTimestamp, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
			timestamp = triggerTimestamp
			if len(fields) >= 5 {
				dbName = fields[4]
			}
		} else {
			dbName = fields[3]
		}
	}

	ok = true
	return
}
