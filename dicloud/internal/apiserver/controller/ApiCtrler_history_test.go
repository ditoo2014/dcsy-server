package controller

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 直接检查控制器真实响应逻辑，不连接数据库、消息队列或任何外部服务。
func TestHistoryPassTimeResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name string
		err  error
		code float64
	}{
		{name: "空历史仍然成功", code: 200},
		{name: "存储失败明确报错", err: errors.New("private-database-connection-error"), code: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			writeHistoryPassTimeResponse(ctx, "point", 0, nil, nil, nil, test.err)
			var body map[string]interface{}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["code"] != test.code {
				t.Fatalf("响应错误: %s", recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "private-database") {
				t.Fatal("不应向浏览器返回内部数据库连接细节")
			}
			if test.err != nil && body["return"] != nil {
				t.Fatal("失败响应不应携带伪装成成功的空历史数据")
			}
		})
	}
}

func TestHistoryBatchResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name    string
		results []HistoryPassTimeBatchResult
		code    float64
	}{
		{name: "全部正常", results: []HistoryPassTimeBatchResult{{LabelId: "point"}}, code: 200},
		{name: "部分失败", results: []HistoryPassTimeBatchResult{{LabelId: "point"}, {LabelId: "failed", Error: "查询失败"}}, code: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			writeHistoryBatchResponse(ctx, test.results)
			var body map[string]interface{}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["code"] != test.code {
				t.Fatalf("响应错误: %s", recorder.Body.String())
			}
		})
	}
}
