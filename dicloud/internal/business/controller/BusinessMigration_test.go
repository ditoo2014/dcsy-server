package controller

import (
	"bytes"
	"dicloud/internal/business/model"
	"dicloud/internal/common/grpc/tolink_grpc"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// 不连接真实采集服务，覆盖断连、空响应和业务失败的全部错误分支。
func TestGeneralTriggerFailureResponse(t *testing.T) {
	for _, tc := range []struct {
		name   string
		resp   *tolink_grpc.TolinkGeneralTriggerResponse
		err    error
		failed bool
	}{
		{"连接失败", nil, errors.New("测试连接失败"), true},
		{"空响应", nil, nil, true},
		{"业务失败", &tolink_grpc.TolinkGeneralTriggerResponse{Code: 400, Message: "配置无效"}, nil, true},
		{"空错误说明", &tolink_grpc.TolinkGeneralTriggerResponse{Code: 400}, nil, true},
		{"正常结果", &tolink_grpc.TolinkGeneralTriggerResponse{Code: 200}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			if failed := generalTriggerFailed(ctx, tc.resp, tc.err); failed != tc.failed {
				t.Fatalf("失败标记不符：%v", failed)
			}
			if !tc.failed {
				if recorder.Body.Len() != 0 {
					t.Fatal("正常响应不应写入错误体")
				}
				return
			}
			var result struct {
				Code    int    `json:"code"`
				Message string `json:"msg"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != 200 || result.Code != 400 || result.Message == "" {
				t.Fatalf("错误响应不完整：%s", recorder.Body.String())
			}
		})
	}
}

// 模板应直接下载且与真正导入使用同一套表头，长编号按文本保存。
func TestBusinessImportTemplateDownload(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler gin.HandlerFunc
		headers []string
	}{
		{"站点", (SiteCtrler{}).ImportTemplate, model.SiteExportHeader},
		{"库房", (WarehouseCtrler{}).ImportTemplate, model.WarehouseExportHeader},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			tc.handler(ctx)
			if recorder.Code != 200 || !strings.Contains(recorder.Header().Get("Content-Type"), "spreadsheetml") {
				t.Fatalf("未返回表格文件：%d", recorder.Code)
			}
			if !strings.Contains(recorder.Header().Get("Content-Disposition"), "filename*=UTF-8''") {
				t.Fatal("未提供可解码的中文文件名")
			}
			f, err := excelize.OpenReader(bytes.NewReader(recorder.Body.Bytes()))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			rows, err := f.GetRows("Sheet1")
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 1 || !reflect.DeepEqual(rows[0], tc.headers) {
				t.Fatalf("模板表头错误：%v", rows)
			}
			styleID, err := f.GetCellStyle("Sheet1", "B2")
			if err != nil {
				t.Fatal(err)
			}
			style, err := f.GetStyle(styleID)
			if err != nil || style.NumFmt != 49 {
				t.Fatal("编码未设置为文本格式")
			}
		})
	}
}
