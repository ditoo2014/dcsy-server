package dao

import (
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"dicloud/internal/tolink/model"
	usermodel "dicloud/models/model"
	"dicloud/util/tdengine"
)

// 复用内存 database/sql 驱动覆盖真实 DAO，禁止访问真实 TDengine。
func TestIntervalHistoryRejectsStorageFailures(t *testing.T) {
	for _, test := range []struct {
		name    string
		results []*historyTestResult
		strType bool
	}{
		{"计数查询失败", []*historyTestResult{{err: errors.New("count failed")}}, false},
		{"计数解析失败", []*historyTestResult{{columns: []string{"count"}, values: [][]driver.Value{{"invalid"}}}}, false},
		{"计数迭代失败", []*historyTestResult{{columns: []string{"count"}, values: [][]driver.Value{{int64(1)}}, nextErr: errors.New("count interrupted")}}, false},
		{"明细查询失败", []*historyTestResult{{columns: []string{"count"}, values: [][]driver.Value{{int64(1)}}}, {err: errors.New("data failed")}}, false},
		{"数值解析失败", []*historyTestResult{{columns: []string{"count"}, values: [][]driver.Value{{int64(1)}}}, {columns: []string{"ts", "id", "val", "status"}, values: [][]driver.Value{{time.Now(), "point", "invalid", int64(1)}}}}, false},
		{"字符串解析失败", []*historyTestResult{{columns: []string{"count"}, values: [][]driver.Value{{int64(1)}}}, {columns: []string{"ts", "id", "val", "status"}, values: [][]driver.Value{{time.Now(), "point", nil, int64(1)}}}}, true},
		{"明细迭代失败", []*historyTestResult{{columns: []string{"count"}, values: [][]driver.Value{{int64(1)}}}, {columns: []string{"ts", "id", "val", "status"}, values: [][]driver.Value{{time.Now(), "point", float64(3.3), int64(1)}}, nextErr: errors.New("data interrupted")}}, false},
		{"时间解析失败", []*historyTestResult{{columns: []string{"count"}, values: [][]driver.Value{{int64(1)}}}, {columns: []string{"ts", "id", "val", "status"}, values: [][]driver.Value{{"not-a-time", "point", float64(3.3), int64(1)}}}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			useHistoryTestDB(t, test.results...)
			_, _, _, _, err := GetDataHistoryPassTimeList(model.GatewayData{GatewayId: "GW01", DataId: "point"}, usermodel.Date{}, usermodel.Page{Pagenum: 1, Pagesize: 30}, "", "5", "minute", test.strType)
			if err == nil {
				t.Fatal("历史存储错误不能伪装成成功的空数据")
			}
			if strings.HasPrefix(test.name, "计数") && !strings.Contains(err.Error(), "数量") {
				t.Fatalf("计数失败应立即返回原始计数错误，不能继续查询并覆盖错误：%v", err)
			}
		})
	}
}

func TestIntervalHistoryNormalAndEmptyData(t *testing.T) {
	for _, test := range []struct {
		name    string
		value   driver.Value
		strType bool
		empty   bool
	}{
		{name: "数值历史", value: float64(3.332)},
		{name: "字符串历史", value: "正常", strType: true},
		{name: "空历史", empty: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			count := int64(1)
			rows := [][]driver.Value{{time.Unix(1_790_000_000, 0), "point", test.value, int64(1)}}
			if test.empty {
				count, rows = 0, nil
			}
			useHistoryTestDB(t,
				&historyTestResult{columns: []string{"count"}, values: [][]driver.Value{{count}}},
				&historyTestResult{columns: []string{"ts", "id", "val", "status"}, values: rows},
			)
			// 短网关 ID 也不能在 STR 后缀检查时发生切片越界。
			gotCount, times, values, status, err := GetDataHistoryPassTimeList(model.GatewayData{GatewayId: "G", DataId: "point"}, usermodel.Date{}, usermodel.Page{Pagenum: 1, Pagesize: 30}, "", "5", "minute", test.strType)
			if err != nil || gotCount != count || len(times) != int(count) || len(values) != int(count) || len(status) != int(count) {
				t.Fatalf("历史响应错误: count=%d times=%v values=%v status=%v error=%v", gotCount, times, values, status, err)
			}
			if times == nil || values == nil || status == nil {
				t.Fatal("空历史必须返回数组")
			}
		})
	}
}

func TestIntervalHistoryUninitializedStorage(t *testing.T) {
	previous := tdengine.TDengineDB
	tdengine.TDengineDB = nil
	t.Cleanup(func() { tdengine.TDengineDB = previous })
	if _, _, _, _, err := GetDataHistoryPassTimeList(model.GatewayData{GatewayId: "GW01", DataId: "point"}, usermodel.Date{}, usermodel.Page{}, "", "5", "second", false); err == nil {
		t.Fatal("时序库未初始化时应返回错误而不是崩溃")
	}
}
