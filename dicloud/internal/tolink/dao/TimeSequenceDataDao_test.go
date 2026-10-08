package dao

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"reflect"
	"testing"
	"time"

	"dicloud/internal/tolink/model"
	usermodel "dicloud/models/model"
	"dicloud/util/tdengine"
)

// 内存驱动只模拟 database/sql 返回行，所有测试均不访问配置文件或真实服务。
type historyTestResult struct {
	columns []string
	values  [][]driver.Value
	err     error
	nextErr error
	index   int
}

func (r *historyTestResult) Columns() []string { return r.columns }
func (r *historyTestResult) Close() error      { return nil }
func (r *historyTestResult) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		if r.nextErr != nil {
			return r.nextErr
		}
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

type historyTestConnector struct {
	results []*historyTestResult
	index   int
}

func (c *historyTestConnector) Connect(context.Context) (driver.Conn, error) { return c, nil }
func (c *historyTestConnector) Driver() driver.Driver                        { return nil }
func (c *historyTestConnector) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("测试不支持预处理")
}
func (c *historyTestConnector) Close() error { return nil }
func (c *historyTestConnector) Begin() (driver.Tx, error) {
	return nil, errors.New("测试不支持事务")
}
func (c *historyTestConnector) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	if c.index >= len(c.results) {
		return nil, errors.New("出现非预期数据库查询")
	}
	result := c.results[c.index]
	c.index++
	return result, result.err
}

func useHistoryTestDB(t *testing.T, results ...*historyTestResult) {
	t.Helper()
	previous := tdengine.TDengineDB
	testDB := sql.OpenDB(&historyTestConnector{results: results})
	tdengine.TDengineDB = testDB
	t.Cleanup(func() {
		tdengine.TDengineDB = previous
		_ = testDB.Close()
	})
}

func TestFindExistGatewaysColumnCompatibility(t *testing.T) {
	for _, test := range []struct {
		name    string
		columns []string
		values  [][]driver.Value
		want    []string
	}{
		{"旧版单列", []string{"table_name"}, [][]driver.Value{{"GW01"}}, []string{"GW01"}},
		{"新版多列", []string{"table_name", "created_time", "columns"}, [][]driver.Value{{[]byte("GW01"), time.Now(), int64(4)}, {"GW02", nil, int64(4)}}, []string{"GW01", "GW02"}},
		{"按字段名定位", []string{"db_name", "TABLE_NAME", "columns"}, [][]driver.Value{{"demo", "GW01", int64(4)}}, []string{"GW01"}},
		{"旧版首列兼容", []string{"legacy_name", "meta"}, [][]driver.Value{{"GW01", "ignored"}}, []string{"GW01"}},
		{"空数据库", []string{"table_name", "columns"}, nil, []string{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			useHistoryTestDB(t, &historyTestResult{columns: test.columns, values: test.values})
			got, err := FindExistGateways()
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("表名解析错误：got=%v, want=%v, err=%v", got, test.want, err)
			}
		})
	}
}

func TestFindExistGatewaysRejectsReadFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		result historyTestResult
	}{
		{"查询失败", historyTestResult{err: errors.New("查询中断")}},
		{"名称解析失败", historyTestResult{columns: []string{"table_name"}, values: [][]driver.Value{{nil}}}},
		{"迭代失败", historyTestResult{columns: []string{"table_name"}, values: [][]driver.Value{{"GW01"}}, nextErr: errors.New("连接中断")}},
		{"无返回字段", historyTestResult{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			useHistoryTestDB(t, &test.result)
			got, err := FindExistGateways()
			if err == nil || len(got) != 0 {
				t.Fatalf("读取失败不应返回成功或部分表名：got=%v, err=%v", got, err)
			}
		})
	}
}

func TestHistoryCountRejectsReadFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		result historyTestResult
	}{
		{"数量转换失败", historyTestResult{columns: []string{"total"}, values: [][]driver.Value{{"invalid"}}}},
		{"数量迭代失败", historyTestResult{columns: []string{"total"}, values: [][]driver.Value{{int64(12)}}, nextErr: errors.New("连接中断")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			useHistoryTestDB(t, &test.result)
			if _, err := CountGatewayDataByTime(model.GatewayData{GatewayId: "GW01", DataId: "point"}, usermodel.Date{}); err == nil {
				t.Fatal("读取数量失败被静默忽略")
			}
		})
	}
}

func TestHistoryDataRowsAndErrors(t *testing.T) {
	timestamp := time.Unix(1_790_000_000, 0)
	for _, test := range []struct {
		name    string
		value   driver.Value
		strType bool
		empty   bool
		nextErr error
		wantErr bool
	}{
		{name: "数值历史", value: float64(3.332)},
		{name: "字符串历史", value: "正常", strType: true},
		{name: "无历史记录", empty: true},
		{name: "数值解析失败", value: "不是数值", wantErr: true},
		{name: "字符串解析失败", value: nil, strType: true, wantErr: true},
		{name: "明细迭代失败", value: float64(3.332), nextErr: errors.New("连接中断"), wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows := [][]driver.Value{{timestamp, "point", test.value, int64(1)}}
			count := int64(1)
			if test.empty {
				rows, count = nil, 0
			}
			useHistoryTestDB(t,
				&historyTestResult{columns: []string{"total"}, values: [][]driver.Value{{count}}},
				&historyTestResult{columns: []string{"ts", "id", "val", "status"}, values: rows, nextErr: test.nextErr},
			)
			gotCount, times, values, status, err := FindGatewayDataByTime(model.GatewayData{GatewayId: "GW01", DataId: "point"}, usermodel.Date{}, usermodel.Page{Pagenum: 1, Pagesize: 30}, "", test.strType)
			if (err != nil) != test.wantErr {
				t.Fatalf("错误处理不符：err=%v, wantErr=%v", err, test.wantErr)
			}
			if !test.wantErr {
				if gotCount != count || len(times) != int(count) || len(values) != int(count) || len(status) != int(count) {
					t.Fatalf("历史数据返回错误：count=%d times=%v values=%v status=%v", gotCount, times, values, status)
				}
				if times == nil || values == nil || status == nil {
					t.Fatal("空历史应返回空数组而非 null")
				}
			}
		})
	}
}

func TestHistoryLatestDataRejectsReadFailures(t *testing.T) {
	for _, test := range []struct {
		name    string
		value   driver.Value
		strType bool
		nextErr error
	}{
		{name: "数值解析失败", value: "不是数值"},
		{name: "字符串解析失败", value: nil, strType: true},
		{name: "最新值迭代失败", value: float64(3.332), nextErr: errors.New("连接中断")},
	} {
		t.Run(test.name, func(t *testing.T) {
			useHistoryTestDB(t, &historyTestResult{
				columns: []string{"ts", "id", "val", "status"},
				values:  [][]driver.Value{{time.Now(), "point", test.value, int64(1)}},
				nextErr: test.nextErr,
			})
			_, _, _, err := GetDataNew(model.GatewayData{GatewayId: "GW01", DataId: "point"}, "", test.strType)
			if err == nil {
				t.Fatal("最新历史值读取失败被静默忽略")
			}
		})
	}
}
