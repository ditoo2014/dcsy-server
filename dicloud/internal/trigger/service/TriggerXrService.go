// xrdp协议 实时触发器
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

var handleXRLock sync.Mutex

/* 数据格式 x */
func HandleXrTrigger(msgOrg *primitive.MessageExt) {

	handleXRLock.Lock()
	defer handleXRLock.Unlock()

	// 获取数据, 转换数据类型
	msg := string(msgOrg.Body)

	gatewayID, dataStr, format := "", "", ""

	/* 格式解析 */

	for i, length := 0, len(msg)-1; i < length; i++ {
		if msg[i] == '{' {
			gatewayID = msg[:i-1]
			format = msg[i-1 : i]
			dataStr = msg[i:]
			break
		}
	}

	if format != "J" {
		// 格式不对
		return
	}

	if gatewayID == "" || dataStr == "" {
		// 网关id和msg解析不对
		return
	}

	/* 数据解析 */

	var PackData model.PackMcdpData

	err := json.Unmarshal([]byte(dataStr), &PackData)
	if nil != err {
		// json解析不对
		return
	}
	// TODO: acording to data mark. distingush plat data.
	ctx := &gin.Context{}

	t, err2 := time.ParseInLocation("2006-01-02 15:04:05", PackData.B.Ts, time.Local)
	if nil != err2 {
		// 时间戳不对
		return
	}

	now := t.Unix()

	/** trigger dao 初始化 */
	triggerDao := dao.NewTriggerDao(ctx)

	for idx := range PackData.B.Dvl {

		for index2, obj2 := range PackData.B.Dvl[idx].Dl {

			switch value := obj2.(type) {
			case float64:
				{

					// 查缓存, 判断是否有对应配置
					if !bloomutil.LocalBloomFilter.Test([]byte(gatewayID + " " + index2)) {
						// 数据一定不存在, 不查库
						// log.Println("[TRIGGER] bloom judge not exist!")
						return
					}

					config := model.TriggerConfig{
						Marktrigger: gatewayID + " " + index2,
					}

					configList, res := triggerDao.ExistMark(config)
					if !res {
						continue
					}

					for idx := range configList {

						// 触发配置
						unit := configList[idx]

						value, err = util.ArithmeticNew(value, configList[idx].Formula)
						if nil != err {
							return
						}

						result := 1
						for index3 := range unit.Logics {

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
									}
								//// 此次触发
								case 1:
									{
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
										SendMsg(ctx, unit, value, unit.Timer, now)

										// 存储参数
										// 计时器		-	置 0
										err = triggerDao.SetRecordTimer(unit, 0, -1)
										if nil != err {
											continue
										}

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
									}
								//// 此次触发
								case result == 1:
									{
										// fmt.Println("第一次不正常")

										// 打包信息 {无}

										// 存储参数
										// 记录点		-	置 0
										// 计时器		-	装载当前时间
										err = triggerDao.SetRecordTimer(unit, 0, now)
										if nil != err {
											return
										}
										// 发送-无
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

										// 存储参数
										// 计时器		-	置 0
										err = triggerDao.SetRecordTimer(unit, -1, 0)
										if nil != err {
											return
										}
									}
								//// 此次正常 && 记录点不为 0
								case result == 0 && unit.Record == 0:
									{
										// fmt.Println("不正常，持续时间足够，现恢复正常")

										// 打包信息 {触发结束}
										// 开始时间		-	记录点
										// 结束时间		-	当前时间
										// 代码			-	触发器编码
										// 值			-	当前值
										SendMsg(ctx, unit, value, unit.Timer, now)

										// 存储参数
										// 记录点		-	置 0
										// 计时器		-	置 0
										err = triggerDao.SetRecordTimer(unit, 0, 0)
										if nil != err {
											return
										}

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

										// 打包信息 {触发开始}
										// 开始时间		-	当前时间
										// 结束时间		-	0
										// 代码			-	触发器编码
										// 值			-	当前值
										SendMsg(ctx, unit, value, now, 0)

										// 存储参数
										// 记录点		-	当前时间
										// 计时器		-	当前时间
										err = triggerDao.SetRecordTimer(unit, now, now)
										if nil != err {
											return
										}

										return
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

										// 打包信息 {触发持续}
										// 开始时间		-	当前时间
										// 结束时间		-	1
										// 代码			-	触发器编码
										// 值			-	当前值
										SendMsg(ctx, unit, value, now, 1)

										// 存储参数
										// 计时器		-	装载当前时间
										err = triggerDao.SetRecordTimer(unit, -1, now)
										if nil != err {
											return
										}

										return
									}
								}
							}
						}
					}
				}

			default:
				continue
			}

		}
	}
}
