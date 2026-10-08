package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	"log"

	"github.com/gin-gonic/gin"
)

// 通用, 单次触发周期触发, 通用触发器判断
func normalSingleOrCycleGeneralJudge(ctx *gin.Context, unit model.GeneralTriggerConfig, unitMarktrigger model.GeneralMarktrigger, result, now int64, value interface{}, triggerDao *dao.GeneralTriggerDao) {

	var err error

	switch {
	/* 单次报警模式 */
	case unit.Cycle == 0:
		switch {
		////// 上一次正常
		case unitMarktrigger.Record == 0:
			switch result {
			//// 此次正常
			case 0:
				{
					// {不执行操作}

					// add 20251210. 增加触发持续时间
					if unit.SustainTime != 0 {
						// 恢复持续时间, 避免未重置
						triggerDao.SetSustainTimer(unitMarktrigger, 0)
					}
				}
			//// 此次触发
			case 1:
				{

					// add 20251210. 增加触发持续时间
					if unit.SustainTime != 0 {

						if unitMarktrigger.SustainRecord == 0 {
							// 只记录
							triggerDao.SetSustainTimer(unitMarktrigger, now)
							// 直接跳过, 进入下次判断
							return
						} else {
							if (now - unitMarktrigger.SustainRecord) >= unit.SustainTime {
								// 不为0, 且达到持续时间. 报警

								// 存储参数
								// 计时器		-	当前时间
								err = triggerDao.SetRecordTimer(unitMarktrigger, now, -1)
								if nil != err {
									return
								}

								// 发送
								SendGeneralTriggerMsg(ctx, &unit, &unitMarktrigger, value, now, 0)
							}

							// now时间未达到标准的时候
							return
						}
					}

					// 存储参数
					// 计时器		-	当前时间
					err = triggerDao.SetRecordTimer(unitMarktrigger, now, -1)
					if nil != err {
						log.Println("[NormalSingleOrCycleJudge] err: ", err)
						return
					}

					// 打包信息 {触发开始}
					// 开始时间		-	当前时间
					// 结束时间		-	无
					// 代码			-	触发器编码
					// 值			-	当前值
					SendGeneralTriggerMsg(ctx, &unit, &unitMarktrigger, value, now, 0)
					return
				}
			}
		////// 上一次触发
		case unitMarktrigger.Record != 0:
			switch result {
			//// 此次正常
			case 0:
				{

					// add 20251210. 增加触发持续时间
					if unit.SustainTime != 0 {
						// 恢复持续时间记录
						triggerDao.SetSustainTimer(unitMarktrigger, 0)
					}

					// 存储参数
					// 计时器		-	当前时间
					err = triggerDao.SetRecordTimer(unitMarktrigger, 0, now)
					if nil != err {
						log.Println("[NormalSingleOrCycleJudge] err: ", err)
						return
					}

					// 打包信息 {触发结束}
					// 开始时间		-	记录点
					// 结束时间		-	当前时间
					// 代码			-	触发器编码
					// 值			-	当前值
					SendGeneralTriggerMsg(ctx, &unit, &unitMarktrigger, value, unitMarktrigger.Record, now)
					return
				}
			//// 此次触发
			case 1:
				{
					// {不执行操作}

					// 如果是统计触发器, 做个兼容 - TODO:后续分离业务, 只规定触发器类型 - 不根据业务区分
					if unit.TriggerType == 1 {
						// 最新值, 记录开始时间, 最新时间
						SendGeneralTriggerMsg(ctx, &unit, &unitMarktrigger, value, unitMarktrigger.Record, now)
					}

				}
			}
		}
	/* 周期报警模式 */
	case unit.Cycle != 0:
		switch {
		////// 上一次正常
		case unitMarktrigger.Timer == 0:
			switch {
			//// 此次正常
			case result == 0:
				{
					// fmt.Println("一直正常")

					// {不执行操作}

					// add 20251210. 增加触发持续时间
					if unit.SustainTime != 0 {
						// 恢复持续时间, 避免未重置
						triggerDao.SetSustainTimer(unitMarktrigger, 0)
					}
				}
			//// 此次触发
			case result == 1:
				{
					// fmt.Println("第一次不正常")

					// 打包信息 {无}

					// add 20251210. 增加触发持续时间
					if unit.SustainTime != 0 {

						if unitMarktrigger.SustainRecord == 0 {
							// 只记录
							triggerDao.SetSustainTimer(unitMarktrigger, now)

							// 直接跳过, 进入下次判断
							return
						} else {
							if (now - unitMarktrigger.SustainRecord) >= unit.SustainTime {
								// 不为0, 且达到持续时间. 报警

								// 存储参数
								// 计时器		-	当前时间
								// 设置 record - now, 下一次直接周期循环 - 不进入record == 0
								err = triggerDao.SetRecordTimer(unitMarktrigger, now, now)
								if nil != err {
									return
								}

								// 发送
								SendGeneralTriggerMsg(ctx, &unit, &unitMarktrigger, value, now, 0)
							}

							// now时间未达到标准的时候
							return
						}
					}

					// 存储参数
					// 记录点		-	置 0
					// 计时器		-	装载当前时间
					// 设置 record - now, 下一次直接周期循环 - 不进入record == 0
					err = triggerDao.SetRecordTimer(unitMarktrigger, now, now)
					if nil != err {
						log.Println("[NormalSingleOrCycleJudge] err: ", err)
						return
					}
					// 发送-无

					// 发送
					SendGeneralTriggerMsg(ctx, &unit, &unitMarktrigger, value, now, 0)
				}
			}
		////// 上一次触发
		case unitMarktrigger.Timer != 0:
			switch {
			//// 此次正常 && 记录点为 0
			case result == 0 && unitMarktrigger.Record == 0:
				{
					// fmt.Println("不正常，但持续时间不够，恢复正常")

					// 打包信息 {无}

					// add 20251210. 增加触发持续时间
					if unit.SustainTime != 0 {
						// 恢复持续时间记录
						triggerDao.SetSustainTimer(unitMarktrigger, 0)
					}

					// 存储参数
					// 计时器		-	置 0
					err = triggerDao.SetRecordTimer(unitMarktrigger, -1, 0)
					if nil != err {
						log.Println("[NormalSingleOrCycleJudge] err: ", err)
						return
					}

					SendGeneralTriggerMsg(ctx, &unit, &unitMarktrigger, value, unitMarktrigger.Record, now)
					return
				}
			//// 此次正常 && 记录点不为 0
			case result == 0 && unitMarktrigger.Record != 0:
				{
					// fmt.Println("不正常，持续时间足够，现恢复正常")

					// add 20251210. 增加触发持续时间
					if unit.SustainTime != 0 {
						// 恢复持续时间记录
						triggerDao.SetSustainTimer(unitMarktrigger, 0)
					}

					// 存储参数
					// 记录点		-	置 0
					// 计时器		-	置 0
					err = triggerDao.SetRecordTimer(unitMarktrigger, 0, 0)
					if nil != err {
						log.Println("[NormalSingleOrCycleJudge] err: ", err)
						return
					}

					// 打包信息 {触发结束}
					// 开始时间		-	记录点
					// 结束时间		-	当前时间
					// 代码			-	触发器编码
					// 值			-	当前值
					SendGeneralTriggerMsg(ctx, &unit, &unitMarktrigger, value, unitMarktrigger.Record, now)
					return
				}
			//// 此次触发 && 记录点为 0 && 持续时间小于周期
			case result == 1 && unitMarktrigger.Record == 0 && unit.Cycle > now-unitMarktrigger.Timer:
				{
					// fmt.Println("不正常，持续时间不够，现依旧不正常", now-unit.Timer)

					// {不执行操作}
				}
			//// 此次触发 && 记录点为 0 && 持续时间大于周期
			case result == 1 && unitMarktrigger.Record == 0 && unit.Cycle <= now-unitMarktrigger.Timer:
				{
					// fmt.Println("不正常，持续时间第一次够，报警")

					// 存储参数
					// 记录点		-	当前时间
					// 计时器		-	当前时间
					// err = triggerDao.SetRecordTimer(unit, now, now)
					// if nil != err {
					// 	log.Println("[NormalSingleOrCycleJudge] err: ", err)
					// 	return
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
			case result == 1 && unitMarktrigger.Record != 0 && unit.Cycle > now-unitMarktrigger.Timer:
				{
					// fmt.Println("不正常，持续时间足够，现等待下一次报警")

					// {不执行操作}
				}
			//// 此次触发 && 记录点不为 0 && 持续时间大于周期
			case result == 1 && unitMarktrigger.Record != 0 && unit.Cycle <= now-unitMarktrigger.Timer:
				{
					// fmt.Println("不正常，持续时间足够，执行阶段性报警")

					// 存储参数
					// 计时器		-	装载当前时间
					err = triggerDao.SetRecordTimer(unitMarktrigger, -1, now)
					if nil != err {
						log.Println("[NormalSingleOrCycleJudge] err: ", err)
						return
					}

					// 打包信息 {触发持续}
					// 开始时间		-	当前时间
					// 结束时间		-	1
					// 代码			-	触发器编码
					// 值			-	当前值
					SendGeneralTriggerMsg(ctx, &unit, &unitMarktrigger, value, now, 1)
					return
				}
			}
		}
	}

}
