package dao

import (
	commonmodel "dicloud/internal/common/model"
	"dicloud/internal/tolink/model"
	usermodel "dicloud/models/model"
	"dicloud/util"
	"dicloud/util/tdengine"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// 计数查询
func CountGatewayDataByTime(gtwData model.GatewayData, dateTem usermodel.Date) (count int64, err error) {

	// TDengine数据库连接
	taos := tdengine.GetTDengine()

	// count 查询sql封装
	sqlStr := fmt.Sprintf("select count(1) as total from %s.`%s` where id = '%s' AND ts >= %d000 AND ts <= %d000", tdengine.DBNAME,
		gtwData.GatewayId, gtwData.DataId, dateTem.Datestart, dateTem.Dateend)

	// 查询总条数
	rows, err := taos.Query(sqlStr)
	if err != nil {
		log.Println("[TDengineDao] FindGatewayDataByTime fail: ", err, "\n sql: ", sqlStr)
		return
	}
	defer rows.Close()
	return scanHistoryCount(rows)
}

// 根据时间查询
func FindGatewayDataByTime(gtwData model.GatewayData, dateTem usermodel.Date, pageTem usermodel.Page, formula string, strType bool) (count int64, times []int64, values []interface{}, status []int64, err error) {
	// 无记录时也返回数组，避免调用方将 null 误判为响应缺失。
	times, values, status = []int64{}, []interface{}{}, []int64{}

	// TDengine数据库连接
	taos := tdengine.GetTDengine()

	// count 查询sql封装
	sqlStr := fmt.Sprintf("select count(1) as total from %s.`%s` where id = '%s' AND ts >= %d000 AND ts <= %d000", tdengine.DBNAME,
		gtwData.GatewayId, gtwData.DataId, dateTem.Datestart, dateTem.Dateend)

	// 查询总条数
	rows, err := taos.Query(sqlStr)
	if err != nil {
		log.Println("[TDengineDao] FindGatewayDataByTime fail: ", err, "\n sql: ", sqlStr)
		return
	}
	defer rows.Close()
	count, err = scanHistoryCount(rows)
	if err != nil {
		return
	}

	// 分页查询数据 - sql封装
	// 字段顺序固定, 如果不固定, 写出详细的查询字段
	sqlStr = fmt.Sprintf("select ts,id,val,status from %s.`%s` where id = '%s' AND ts >= %d000 AND ts <= %d000 ORDER BY ts DESC limit %d offset %d", tdengine.DBNAME,
		gtwData.GatewayId, gtwData.DataId, dateTem.Datestart, dateTem.Dateend, pageTem.Pagesize, (pageTem.Pagenum-1)*pageTem.Pagesize)

	if strType {
		// 字符串类型数据查询
		sqlStr = fmt.Sprintf("select ts,id,val_str,status from %s.`%s` where id = '%s' AND ts >= %d000 AND ts <= %d000 ORDER BY ts DESC limit %d offset %d", tdengine.DBNAME,
			gtwData.GatewayId, gtwData.DataId, dateTem.Datestart, dateTem.Dateend, pageTem.Pagesize, (pageTem.Pagenum-1)*pageTem.Pagesize)
	}

	// 查询
	rows, err = taos.Query(sqlStr)
	if err != nil {
		log.Println("[TDengineDao] FindGatewayDataByTime fail: ", err, "\n sql: ", sqlStr)
		return
	}
	defer rows.Close()
	var strCheck = false
	if strings.HasSuffix(gtwData.GatewayId, "STR") || strType {
		strCheck = true
	}
	for rows.Next() {
		// 字符串数据
		if strCheck {
			var r = commonmodel.STRTimeSequenceData{}
			err = rows.Scan(&r.Timestamp, &r.Id, &r.Value, &r.Status)
			if err != nil {
				err = fmt.Errorf("读取历史字符串数据失败：%w", err)
				return
			}
			times = append(times, r.Timestamp.Unix())
			values = append(values, r.Value)
			status = append(status, int64(r.Status))
			continue
		}
		var r = commonmodel.NormalTimeSequenceData{}
		err = rows.Scan(&r.Timestamp, &r.Id, &r.Value, &r.Status)
		if err != nil {
			err = fmt.Errorf("读取历史数值数据失败：%w", err)
			return
		}

		//进行公式计算
		data, err := strconv.ParseFloat(fmt.Sprint(r.Value), 0)
		if err == nil {
			if _res, err := util.ArithmeticNew(data, formula); err == nil {
				r.Value = _res
			}
		}
		times = append(times, r.Timestamp.Unix())
		values = append(values, r.Value)
		status = append(status, int64(r.Status))
		// fmt.Println("ts = ", r.Timestamp, ", id = "+r.Id+", val = ", r.Value, ", status = ", r.Status)
	}

	// 迭代中断也属于查询失败，不返回不完整的历史曲线。
	err = rows.Err()
	return
}

// 获取时间之前前的最新数据
func GetLastDataByTime(gtwData model.GatewayData, endTime int64, formula string, strType bool) (times []int64, values []interface{}, status []int64, err error) {

	// TDengine数据库连接
	taos := tdengine.GetTDengine()

	// 分页查询数据 - sql封装
	sqlStr := fmt.Sprintf("select ts,id,val,status from %s.`%s` where id = '%s' AND ts <= %d000 ORDER BY ts DESC limit %d", tdengine.DBNAME,
		gtwData.GatewayId, gtwData.DataId, endTime, 1)

	if strType {
		sqlStr = fmt.Sprintf("select ts,id,val_str,status from %s.`%s` where id = '%s' AND ts <= %d000 ORDER BY ts DESC limit %d", tdengine.DBNAME,
			gtwData.GatewayId, gtwData.DataId, endTime, 1)
	}
	// 查询
	rows, err := taos.Query(sqlStr)
	if err != nil {
		log.Println("[TDengineDao] GetDataNew fail: ", err, "\n sql: ", sqlStr)
		return
	}
	defer rows.Close()
	var strCheck = false
	if string(gtwData.GatewayId[len(gtwData.GatewayId)-3:]) == "STR" || strType {
		strCheck = true
	}
	for rows.Next() {
		// 字符串数据
		if strCheck {
			var r = commonmodel.STRTimeSequenceData{}
			err := rows.Scan(&r.Timestamp, &r.Id, &r.Value, &r.Status)
			if err != nil {
				fmt.Println("scan TDengine data error:\n", err)
			}
			times = append(times, r.Timestamp.Unix())
			values = append(values, r.Value)
			status = append(status, int64(r.Status))
			continue
		}
		var r = commonmodel.NormalTimeSequenceData{}
		err := rows.Scan(&r.Timestamp, &r.Id, &r.Value, &r.Status)
		if err != nil {
			fmt.Println("scan TDengine data error:\n", err)
		}

		//进行公式计算
		data, err := strconv.ParseFloat(fmt.Sprint(r.Value), 0)
		if err == nil {
			if _res, err := util.ArithmeticNew(data, formula); err == nil {
				r.Value = _res
			}
		}

		times = append(times, r.Timestamp.Unix())
		values = append(values, r.Value)
		status = append(status, int64(r.Status))
	}

	return
}

// 查询最新30条数据
func GetDataNew(gtwData model.GatewayData, formula string, strType bool) (times []int64, values []interface{}, status []int64, err error) {
	times, values, status = []int64{}, []interface{}{}, []int64{}

	// TDengine数据库连接
	taos := tdengine.GetTDengine()

	// 分页查询数据 - sql封装
	sqlStr := fmt.Sprintf("select ts,id,val,status from %s.`%s` where id = '%s' ORDER BY ts DESC limit 30", tdengine.DBNAME,
		gtwData.GatewayId, gtwData.DataId)

	if strType {
		sqlStr = fmt.Sprintf("select ts,id,val_str,status from %s.`%s` where id = '%s' ORDER BY ts DESC limit 30", tdengine.DBNAME,
			gtwData.GatewayId, gtwData.DataId)
	}

	// 查询
	rows, err := taos.Query(sqlStr)
	if err != nil {
		log.Println("[TDengineDao] GetDataNew fail: ", err, "\n sql: ", sqlStr)
		return
	}
	defer rows.Close()
	var strCheck = false
	if strings.HasSuffix(gtwData.GatewayId, "STR") || strType {
		strCheck = true
	}
	for rows.Next() {
		// 字符串数据
		if strCheck {
			var r = commonmodel.STRTimeSequenceData{}
			err = rows.Scan(&r.Timestamp, &r.Id, &r.Value, &r.Status)
			if err != nil {
				err = fmt.Errorf("读取最新字符串数据失败：%w", err)
				return
			}
			times = append(times, r.Timestamp.Unix())
			values = append(values, r.Value)
			status = append(status, int64(r.Status))
			continue
		}
		var r = commonmodel.NormalTimeSequenceData{}
		err = rows.Scan(&r.Timestamp, &r.Id, &r.Value, &r.Status)
		if err != nil {
			err = fmt.Errorf("读取最新数值数据失败：%w", err)
			return
		}

		//进行公式计算
		data, err := strconv.ParseFloat(fmt.Sprint(r.Value), 0)
		if err == nil {
			if _res, err := util.ArithmeticNew(data, formula); err == nil {
				r.Value = _res
			}
		}

		times = append(times, r.Timestamp.Unix())
		values = append(values, r.Value)
		status = append(status, int64(r.Status))
		// fmt.Println("ts = ", r.ts, ", id = "+r.id+", val = ", r.val, ", status = ", r.status)
	}

	err = rows.Err()
	return
}

// 通过固定的间隔时间,查询固定间隔时间的数据  []int64
func GetDataHistoryPassTimeList(gtwData model.GatewayData, dateTem usermodel.Date, pageTem usermodel.Page, formula string, timeLength string, timeType string, strType bool) (count int64, times []string, values []interface{}, status []int64, err error) {
	// 空数据是正常结果，存储故障不是；统一返回数组并完整透传查询/读取错误。
	times, values, status = []string{}, []interface{}{}, []int64{}
	tdEngine := tdengine.GetTDengine()
	if tdEngine == nil {
		err = errors.New("时序数据库尚未初始化")
		return
	}
	if gtwData.GatewayId == "" || gtwData.DataId == "" {
		err = errors.New("历史数据的网关或测点不能为空")
		return
	}

	if len(timeLength) == 0 {
		timeLength = "10"
	}
	interval, parseErr := strconv.ParseInt(timeLength, 10, 64)
	if parseErr != nil || interval <= 0 {
		err = errors.New("历史查询间隔必须为正整数")
		return
	}

	//整理查询的time()参数;
	switch timeType {
	case "minute":

		timeLength = timeLength + "m"

	case "hour":

		timeLength = timeLength + "h"

	case "day":
		timeLength = timeLength + "d"
	case "", "second":

		timeLength = timeLength + "s"
	default:
		err = errors.New("历史查询间隔单位无效")
		return
	}

	cmd := fmt.Sprintf("select count(1) from (select count(1) from %s.`%s` where id='%s' and ts >= %d000 and ts <= %d000 interval(%s, auto))", tdengine.DBNAME,
		gtwData.GatewayId,
		gtwData.DataId,
		// 中国时区转0时时区
		dateTem.Datestart, dateTem.Dateend,
		timeLength,
	)

	rows, err := tdEngine.Query(cmd)
	if err != nil {
		err = fmt.Errorf("查询间隔历史数量失败：%w", err)
		return
	}
	count, err = scanHistoryCount(rows)
	rows.Close()
	if err != nil {
		return
	}

	// STR检查
	var strCheck = false
	if strings.HasSuffix(gtwData.GatewayId, "STR") || strType {
		strCheck = true
	}

	/* 获取数据  status,value顺序符合后面idx*/

	cmd = fmt.Sprintf("select first(ts) ts,first(id),first(val),first(status) from %s.`%s` where id='%s' and ts >= %d000 and ts <= %d000 interval(%s, auto) ORDER BY ts DESC limit %d offset %d", tdengine.DBNAME,
		gtwData.GatewayId,
		gtwData.DataId,
		// 中国时区转0时时区
		dateTem.Datestart, dateTem.Dateend,
		timeLength,
		pageTem.Pagesize, (pageTem.Pagenum-1)*pageTem.Pagesize)

	if strType {
		cmd = fmt.Sprintf("select first(ts) ts,first(id),first(val_str),first(status) from %s.`%s` where id='%s' and ts >= %d000 and ts <= %d000 interval(%s, auto) ORDER BY ts DESC limit %d offset %d", tdengine.DBNAME,
			gtwData.GatewayId,
			gtwData.DataId,
			// 中国时区转0时时区
			dateTem.Datestart, dateTem.Dateend,
			timeLength,
			pageTem.Pagesize, (pageTem.Pagenum-1)*pageTem.Pagesize)
	}

	rows, err = tdEngine.Query(cmd)
	if err != nil {
		err = fmt.Errorf("查询间隔历史明细失败：%w", err)
		return
	}
	// 转换为北京时间 (UTC+8)
	location, locationErr := time.LoadLocation("Asia/Shanghai")
	if locationErr != nil {
		location = time.FixedZone("CST", 8*60*60)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			ts  string
			id  string
			val float64
			sta int
		)

		// 字符串数据
		if strCheck {
			var r = commonmodel.STRTimeSequenceData{}
			err = rows.Scan(&r.Timestamp, &r.Id, &r.Value, &r.Status)
			if err != nil {
				err = fmt.Errorf("读取间隔历史字符串失败：%w", err)
				return
			}

			localTime := r.Timestamp.In(location)
			times = append(times, localTime.String())
			values = append(values, r.Value)
			status = append(status, int64(r.Status))
			continue
		}

		err = rows.Scan(&ts, &id, &val, &sta)
		if err != nil {
			err = fmt.Errorf("读取间隔历史数值失败：%w", err)
			return
		}

		//进行公式计算
		if calculated, formulaErr := util.ArithmeticNew(val, formula); formulaErr == nil {
			val = calculated
		}

		parsedTime, parseErr := time.Parse(time.RFC3339Nano, ts)
		if parseErr != nil {
			err = fmt.Errorf("解析间隔历史采集时间失败：%w", parseErr)
			return
		}
		localTime := parsedTime.In(location)
		times = append(times, localTime.String())
		values = append(values, val)
		status = append(status, int64(sta))
	}

	if rowErr := rows.Err(); rowErr != nil {
		err = fmt.Errorf("遍历间隔历史明细失败：%w", rowErr)
	}
	return
}

// 删除数据库中 一个数据点的所有数据
func DeleteDBOneById(gtwData model.GatewayData) error {
	// TDengine数据库连接
	taos := tdengine.GetTDengine()

	// 删除一个数据点的历史值 - sql封装
	sqlStr := fmt.Sprintf("delete from %s.`%s` where id = '%s'", tdengine.DBNAME,
		gtwData.GatewayId, gtwData.DataId)
	// 删除
	_, err := taos.Exec(sqlStr)
	if err != nil {
		log.Println("[TDengineDao] DeleteDBOne fail: ", err, "\n sql: ", sqlStr)
		return err
	}

	return nil
}

// 删除整张表
func DeleteDBTable(gtwData model.Gateway) error {
	// TDengine数据库连接
	taos := tdengine.GetTDengine()

	// 删除整张表 - sql封装
	sqlStr := fmt.Sprintf("drop table %s.`%s`", tdengine.DBNAME,
		gtwData.GatewayId)
	// 删除
	_, err := taos.Exec(sqlStr)
	if err != nil {
		log.Println("[TDengineDao] DeleteDBAll fail: ", err, "\n sql: ", sqlStr)
		return err
	}

	return nil
}

// 删除一个数据点的时间点的数据
func DeleteDBOneByIdTime(gtwData model.GatewayData, tm string) error {
	// TDengine数据库连接
	taos := tdengine.GetTDengine()

	// 删除一个数据点的时间点的数据 - sql封装
	sqlStr := fmt.Sprintf("delete from %s.`%s` where id = '%s' AND ts = %s000", tdengine.DBNAME,
		gtwData.GatewayId, gtwData.DataId, tm)
	// 删除
	_, err := taos.Exec(sqlStr)
	if err != nil {
		log.Println("[TDengineDao] DeleteDBAll fail: ", err, "\n sql: ", sqlStr)
		return err
	}

	return nil
}

func FindExistGateways() (id []string, err error) {
	// TDengine数据库连接
	taos := tdengine.GetTDengine()
	if taos == nil {
		return nil, errors.New("时序数据库尚未初始化")
	}

	// 查所有表 - sql封装
	sqlStr := "show tables;"
	rows, err := taos.Query(sqlStr)
	if err != nil {
		log.Println("[TDengineDao] FindExistGateways fail: ", err, "\n sql: ", sqlStr)
		return
	}
	defer rows.Close()
	return scanGatewayTableNames(rows)
}

// 时序查询只依赖行读取能力，回归测试不需要连接真实数据库。
type timeSequenceRows interface {
	Columns() ([]string, error)
	Next() bool
	Scan(...interface{}) error
	Err() error
}

// SHOW TABLES 在不同 TDengine 版本中返回的列数不同，按列名定位表名并接收其余列。
func scanGatewayTableNames(rows timeSequenceRows) ([]string, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("读取历史表字段失败：%w", err)
	}
	if len(columns) == 0 {
		return nil, errors.New("历史表查询未返回字段")
	}
	tableNameIndex := 0 // 兼容旧版只有第一列表名的返回格式。
	for i, column := range columns {
		if strings.EqualFold(column, "table_name") || strings.EqualFold(column, "name") {
			tableNameIndex = i
			break
		}
	}
	ids := []string{}
	for rows.Next() {
		var gatewayID string
		values := make([]interface{}, len(columns))
		destinations := make([]interface{}, len(columns))
		for i := range destinations {
			destinations[i] = &values[i]
		}
		destinations[tableNameIndex] = &gatewayID
		if err := rows.Scan(destinations...); err != nil {
			return nil, fmt.Errorf("读取历史表名称失败：%w", err)
		}
		ids = append(ids, gatewayID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历历史表失败：%w", err)
	}
	return ids, nil
}

// 计数读取错误必须透传，不能静默变成零条。
func scanHistoryCount(rows timeSequenceRows) (int64, error) {
	var count int64
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, fmt.Errorf("读取历史数据数量失败：%w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("遍历历史数据数量失败：%w", err)
	}
	return count, nil
}

func FindGatewayNewTime(id []string) (times []int64, err error) {
	// TDengine数据库连接
	taos := tdengine.GetTDengine()

	if len(id) > 0 {
		for _, gatewayId := range id {

			// 查询一条数据 查询sql封装
			sqlStr := fmt.Sprintf("select * from %s.`%s` ORDER BY ts DESC limit 1", tdengine.DBNAME, gatewayId)
			rows, err := taos.Query(sqlStr)
			if err != nil {
				log.Println("[TDengineDao] FindGatewayNewTime one data fail: ", err, "\n sql: ", sqlStr)
				return times, err
			}
			defer rows.Close()
			// 一次数据
			var strCheck = false
			if string(gatewayId[len(gatewayId)-3:]) == "STR" {
				strCheck = true
			}
			for rows.Next() {
				// 字符串数据
				if strCheck {
					var r = commonmodel.STRTimeSequenceData{}
					err := rows.Scan(&r.Timestamp, &r.Id, &r.Value, &r.Status)
					if err != nil {
						fmt.Println("scan TDengine data error:\n", err)
					}
					times = append(times, r.Timestamp.Unix())
					continue
				}
				var r = commonmodel.NormalTimeSequenceData{}
				err := rows.Scan(&r.Timestamp, &r.Id, &r.Value, &r.Status)
				if err != nil {
					fmt.Println("scan TDengine data error:\n", err)
					// 有问题, 跳过此条数据
					return times, err
				}
				times = append(times, r.Timestamp.Unix())
			}
		}
	}

	return
}
