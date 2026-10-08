package controller

import (
	"encoding/json"
	"testing"

	businessmodel "dicloud/internal/business/model"
)

func TestBatchEquipmentStreamIDReadsHistoricalBinding(t *testing.T) {
	details, _ := json.Marshal(map[string]any{"streamId": 99})
	if got := batchEquipmentStreamID(businessmodel.Equipment{EquipmentDetails: details}); got != 99 {
		t.Fatalf("expected stream ID 99, got %d", got)
	}
	legacyDetails, _ := json.Marshal(map[string]any{"streamID": "100"})
	if got := batchEquipmentStreamID(businessmodel.Equipment{EquipmentDetails: legacyDetails}); got != 100 {
		t.Fatalf("expected legacy stream ID 100, got %d", got)
	}
}

func TestWriteStreamBatchImportIssuesAggregatesErrorsByRow(t *testing.T) {
	f, err := newStreamBatchImportTemplate()
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	defer f.Close()
	if err := f.SetSheetRow("导入数据", "A2", &[]any{"池A", "BAT-01", "GW-01", "01"}); err != nil {
		t.Fatalf("set input row: %v", err)
	}
	issues := []streamBatchImportIssue{
		{Row: 2, Field: "数据池名称", Message: "数据库中已存在"},
		{Row: 2, Field: "电池ID", Message: "数据库中已存在数据池"},
		{Row: 3, Field: "网关ID", Message: "网关不存在"},
	}
	if err := writeStreamBatchImportIssues(f, issues); err != nil {
		t.Fatalf("write issues: %v", err)
	}
	if got, _ := f.GetCellValue("导入数据", "E1"); got != "校验错误" {
		t.Fatalf("expected error header, got %q", got)
	}
	if got, _ := f.GetCellValue("导入数据", "E2"); got != "数据池名称：数据库中已存在；电池ID：数据库中已存在数据池" {
		t.Fatalf("unexpected row 2 error: %q", got)
	}
	if got, _ := f.GetCellValue("导入数据", "E3"); got != "网关ID：网关不存在" {
		t.Fatalf("unexpected row 3 error: %q", got)
	}
	if err := writeStreamBatchImportIssues(f, []streamBatchImportIssue{{Row: 2, Field: "电池ID", Message: "设备不存在"}}); err != nil {
		t.Fatalf("rewrite issues: %v", err)
	}
	if got, _ := f.GetCellValue("导入数据", "E2"); got != "电池ID：设备不存在" {
		t.Fatalf("expected existing error column to be refreshed, got %q", got)
	}
	if got, _ := f.GetCellValue("导入数据", "E3"); got != "" {
		t.Fatalf("expected stale row error to be cleared, got %q", got)
	}
	if got, _ := f.GetCellValue("导入数据", "F1"); got != "" {
		t.Fatalf("expected error column to be reused, got extra header %q", got)
	}
}

func TestStreamBatchImportTemplateContainsOneTtdcExample(t *testing.T) {
	f, err := newStreamBatchImportTemplate()
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) != 2 || sheets[0] != "导入数据" || sheets[1] != "填写示例" {
		t.Fatalf("unexpected sheets: %#v", sheets)
	}
	importRows, err := f.GetRows("导入数据")
	if err != nil {
		t.Fatalf("read import sheet: %v", err)
	}
	if len(importRows) != 1 {
		t.Fatalf("expected import sheet to contain only its header, got %#v", importRows)
	}
	exampleRows, err := f.GetRows("填写示例")
	if err != nil {
		t.Fatalf("read example sheet: %v", err)
	}
	if len(exampleRows) < 2 || len(exampleRows[1]) != 4 || exampleRows[1][3] != "01" {
		t.Fatalf("unexpected TTDC example: %#v", exampleRows)
	}
	for _, cell := range []struct {
		sheet string
		cell  string
	}{
		{sheet: "导入数据", cell: "B2"},
		{sheet: "导入数据", cell: "C2"},
		{sheet: "导入数据", cell: "D2"},
		{sheet: "填写示例", cell: "B2"},
		{sheet: "填写示例", cell: "C2"},
		{sheet: "填写示例", cell: "D2"},
	} {
		styleID, err := f.GetCellStyle(cell.sheet, cell.cell)
		if err != nil {
			t.Fatalf("read %s!%s style: %v", cell.sheet, cell.cell, err)
		}
		style, err := f.GetStyle(styleID)
		if err != nil {
			t.Fatalf("read %s!%s style definition: %v", cell.sheet, cell.cell, err)
		}
		if style.NumFmt != 49 {
			t.Fatalf("expected %s!%s to use text format, got numFmt %d", cell.sheet, cell.cell, style.NumFmt)
		}
	}
}

func TestParseStreamBatchImportRowsRejectsFileDuplicates(t *testing.T) {
	rows := [][]string{
		{"数据池名称", "电池ID", "网关ID", "TTDC地址"},
		{"池A", "BAT-01", "GW-01", ""},
		{"池A", "BAT-02", "GW-02", ""},
		{"池B", "BAT-01", "GW-03", ""},
	}
	parsed, issues := parseStreamBatchImportRows(rows)
	if len(parsed) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(parsed))
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 duplicate issues, got %d: %#v", len(issues), issues)
	}
}

func TestParseStreamBatchImportRowsRequiresHeaders(t *testing.T) {
	_, issues := parseStreamBatchImportRows([][]string{{"数据池名称", "电池ID"}, {"池A", "BAT-01"}})
	if len(issues) != 1 || issues[0].Field != "网关ID" {
		t.Fatalf("expected missing gateway header, got %#v", issues)
	}
}

func TestBatchLabelResolverStripsAddressAndUsesDynamicNames(t *testing.T) {
	resolver := newBatchLabelResolver("TTDC")
	if got := resolver(stripBatchTtdcPrefix("01_电池总压数据", "01")); got != "组电压" {
		t.Fatalf("expected mapped label, got %q", got)
	}
	if got := resolver(stripBatchTtdcPrefix("01_单体电池 3 电压", "01")); got != "电压3" {
		t.Fatalf("expected dynamic label, got %q", got)
	}
	if got := stripBatchTtdcPrefix("01_环境欠温保护", "01"); got != "环境欠温保护" {
		t.Fatalf("expected address prefix stripped, got %q", got)
	}
}
