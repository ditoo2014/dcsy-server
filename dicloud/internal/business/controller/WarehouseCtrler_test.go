package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"dicloud/internal/business/model"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func warehouseImportTestRows(dataRows ...[]string) [][]string {
	header := append([]string(nil), model.WarehouseExportHeader...)
	return append([][]string{header}, dataRows...)
}

func TestValidateWarehouseImportRowsRejectsDuplicateCode(t *testing.T) {
	rows := warehouseImportTestRows(
		[]string{"库房甲", "WH-001", "地址甲"},
		[]string{"库房乙", "WH-001", "地址乙"},
	)

	warehouses, issues := validateWarehouseImportRows(rows, nil)
	if len(warehouses) != 1 {
		t.Fatalf("warehouse count = %d, want 1 valid row retained for diagnostics", len(warehouses))
	}
	if len(issues) != 1 || issues[0].Row != 3 || !strings.Contains(issues[0].Message, "与第2行重复") {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestValidateWarehouseImportRowsRejectsTrimmedDuplicateName(t *testing.T) {
	rows := warehouseImportTestRows(
		[]string{"库房甲", "WH-001"},
		[]string{"  库房甲  ", "WH-002"},
	)

	warehouses, issues := validateWarehouseImportRows(rows, nil)
	if len(warehouses) != 1 {
		t.Fatalf("warehouse count = %d, want 1 valid row retained for diagnostics", len(warehouses))
	}
	if len(issues) != 1 || issues[0].Row != 3 || !strings.Contains(issues[0].Message, "库房名称与第2行重复") {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestValidateWarehouseImportRowsHandlesShortRowsAndRequiredFields(t *testing.T) {
	rows := warehouseImportTestRows([]string{"", "WH-EMPTY-NAME"})

	warehouses, issues := validateWarehouseImportRows(rows, nil)
	if len(warehouses) != 0 {
		t.Fatalf("invalid row was parsed: %+v", warehouses)
	}
	if len(issues) != 1 || issues[0].Row != 2 || !strings.Contains(issues[0].Message, "库房名称不能为空") {
		t.Fatalf("issues = %+v", issues)
	}

	rows = warehouseImportTestRows([]string{"库房丙"})
	_, issues = validateWarehouseImportRows(rows, nil)
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "库房编码不能为空") {
		t.Fatalf("short-row issues = %+v", issues)
	}
}

func TestValidateWarehouseImportRowsRequiresExpectedHeader(t *testing.T) {
	rows := [][]string{{"名称", "编码"}}
	_, issues := validateWarehouseImportRows(rows, nil)
	if len(issues) != 1 || issues[0].Row != 1 || !strings.Contains(issues[0].Message, "表头必须为") {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestWarehouseImportExistingCodeIssuesRejectDatabaseCode(t *testing.T) {
	rows := warehouseImportTestRows([]string{"库房甲", "WH-EXISTS"})
	warehouses, issues := validateWarehouseImportRows(rows, nil)
	if len(issues) != 0 || len(warehouses) != 1 {
		t.Fatalf("initial validation returned warehouses=%+v issues=%+v", warehouses, issues)
	}

	issues = warehouseImportExistingCodeIssues(warehouses, rows, func(code string) bool {
		return code == "WH-EXISTS"
	})
	if len(issues) != 1 || issues[0].Row != 2 || issues[0].Message != "库房编码已存在，禁止导入" {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestWarehouseImportExistingCodeIssuesUsesFirstDuplicateRow(t *testing.T) {
	rows := warehouseImportTestRows(
		[]string{"库房甲", "WH-EXISTS"},
		[]string{"库房乙", "WH-EXISTS"},
	)
	warehouses, _ := validateWarehouseImportRows(rows, nil)

	issues := warehouseImportExistingCodeIssues(warehouses, rows, func(string) bool { return true })
	if len(issues) != 1 || issues[0].Row != 2 {
		t.Fatalf("database duplicate issues = %+v, want row 2", issues)
	}
}

func TestWarehouseImportExistingNameIssuesRejectsDatabaseName(t *testing.T) {
	rows := warehouseImportTestRows([]string{"库房甲", "WH-001"})
	warehouses, issues := validateWarehouseImportRows(rows, nil)
	if len(issues) != 0 || len(warehouses) != 1 {
		t.Fatalf("initial validation returned warehouses=%+v issues=%+v", warehouses, issues)
	}

	issues = warehouseImportExistingNameIssues(warehouses, rows, func(name string) bool {
		return name == "库房甲"
	})
	if len(issues) != 1 || issues[0].Row != 2 || issues[0].Message != "库房名称已存在，禁止导入" {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestMergeWarehouseImportIssuesCombinesMessagesByRow(t *testing.T) {
	issues := mergeWarehouseImportIssues([]warehouseImportIssue{
		{Row: 2, Message: "库房名称已存在，禁止导入"},
		{Row: 2, Message: "库房编码已存在，禁止导入"},
	})
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "库房名称已存在") || !strings.Contains(issues[0].Message, "库房编码已存在") {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestWriteWarehouseWorkbookReturnsExcelStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetCellValue("Sheet1", "A1", "库房名称"); err != nil {
		t.Fatalf("set workbook cell: %v", err)
	}

	fileName := "库房信息.xlsx"
	writeWarehouseWorkbook(ctx, fileName, f, http.StatusOK, "导出成功")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	wantContentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if got := recorder.Header().Get("Content-Type"); got != wantContentType {
		t.Fatalf("content type = %q, want %q", got, wantContentType)
	}
	wantEncodedName := "filename*=UTF-8''" + url.PathEscape(fileName)
	if got := recorder.Header().Get("Content-Disposition"); !strings.Contains(got, wantEncodedName) {
		t.Fatalf("content disposition = %q, want it to contain %q", got, wantEncodedName)
	}

	exported, err := excelize.OpenReader(bytes.NewReader(recorder.Body.Bytes()))
	if err != nil {
		t.Fatalf("open exported workbook: %v", err)
	}
	defer exported.Close()
	if got, err := exported.GetCellValue("Sheet1", "A1"); err != nil || got != "库房名称" {
		t.Fatalf("exported A1 = %q, err = %v", got, err)
	}
}

func TestWriteWarehouseImportIssuesUsesLastColumnAndRedText(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	issues := []warehouseImportIssue{{Row: 1, Message: "表头错误"}, {Row: 3, Message: "库房编码已存在，禁止导入"}}
	if err := writeWarehouseImportIssues(f, "Sheet1", issues); err != nil {
		t.Fatal(err)
	}

	assertWarehouseImportCell(t, f, "I1", "校验错误信息")
	assertWarehouseImportCell(t, f, "I2", "表头错误")
	assertWarehouseImportCell(t, f, "I3", "库房编码已存在，禁止导入")
	for _, cell := range []string{"I2", "I3"} {
		styleID, err := f.GetCellStyle("Sheet1", cell)
		if err != nil {
			t.Fatalf("get %s style: %v", cell, err)
		}
		style, err := f.GetStyle(styleID)
		if err != nil {
			t.Fatalf("read %s style: %v", cell, err)
		}
		if style.Font == nil || !strings.EqualFold(style.Font.Color, "FF0000") {
			t.Fatalf("%s font color = %#v, want FF0000", cell, style.Font)
		}
	}
}

func TestWarehouseImportErrorFilename(t *testing.T) {
	if got := warehouseImportErrorFilename("库房导入模板.xlsx"); got != "库房导入模板_校验失败.xlsx" {
		t.Fatalf("error filename = %q", got)
	}
}

func assertWarehouseImportCell(t *testing.T, f *excelize.File, cell, want string) {
	t.Helper()
	got, err := f.GetCellValue("Sheet1", cell)
	if err != nil {
		t.Fatalf("get %s: %v", cell, err)
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", cell, got, want)
	}
}
