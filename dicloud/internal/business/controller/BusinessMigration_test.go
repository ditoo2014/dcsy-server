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

// 核对下载文件与实际导入兼容，并确保示例页不会混入真实导入数据。
func TestBusinessImportTemplateDownload(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler gin.HandlerFunc
		headers []string
	}{
		{"站点", (SiteCtrler{}).ImportTemplate, model.SiteExportHeader},
		{"库房", (WarehouseCtrler{}).ImportTemplate, model.WarehouseExportHeader},
		{"电池", (EquipmentCtrler{}).ImportTemplate, model.EquipmentExportHeader},
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
			if !reflect.DeepEqual(f.GetSheetList(), []string{"Sheet1", "填写示例"}) {
				t.Fatalf("示例改变了导入工作表顺序：%v", f.GetSheetList())
			}
			examples, err := f.GetRows("填写示例")
			if err != nil || len(examples) < 5 || !reflect.DeepEqual(examples[0], tc.headers) {
				t.Fatalf("未提供完整填写示例：%v，错误：%v", examples, err)
			}
			for _, row := range examples[1:3] {
				if len(row) != len(tc.headers) {
					t.Fatalf("示例字段不完整：%v", row)
				}
			}
			if f.GetActiveSheetIndex() != 1 || !strings.Contains(strings.Join(examples[4], ""), "本页不参与导入") {
				t.Fatal("打开模板时未展示示例或未提示填写位置")
			}
			// 模拟用户填写一行后上传，调用真实校验函数，确认只读第一工作表。
			if err := f.SetSheetRow("Sheet1", "A2", &examples[1]); err != nil {
				t.Fatal(err)
			}
			importRows, err := f.GetRows(f.GetSheetName(0))
			if err != nil {
				t.Fatal(err)
			}
			corpMap := map[string]int{"请填写已有部门名称": 99}
			switch tc.name {
			case "站点":
				data, issues := validateSiteImportRows(importRows, corpMap)
				if len(data) != 1 || len(issues) != 0 || data[0].CorpId != 99 || data[0].SiteSerialNumber != examples[1][1] {
					t.Fatalf("示例干扰站点导入或编码失真：%v，%v", data, issues)
				}
			case "库房":
				data, issues := validateWarehouseImportRows(importRows, corpMap)
				if len(data) != 1 || len(issues) != 0 || data[0].CorpId != 99 || data[0].ManagerPhone != examples[1][4] {
					t.Fatalf("示例干扰库房导入或电话失真：%v，%v", data, issues)
				}
			case "电池":
				data, issues := validateEquipmentImportRows(importRows, corpMap)
				if len(data) != 1 || len(issues) != 0 || data[0].CorpId != 99 || data[0].ManufacturerTime != examples[1][3] {
					t.Fatalf("示例干扰电池导入或日期失真：%v，%v", data, issues)
				}
			}
		})
	}
}
