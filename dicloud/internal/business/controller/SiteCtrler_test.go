package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"dicloud/internal/business/model"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func siteImportTestRows(dataRows ...[]string) [][]string {
	header := append([]string(nil), model.SiteExportHeader...)
	return append([][]string{header}, dataRows...)
}

func TestValidateSiteImportRowsParsesSite(t *testing.T) {
	rows := siteImportTestRows([]string{"站点甲", "SITE-001", "部门甲", "地址甲", "120.1", "30.2"})
	sites, issues := validateSiteImportRows(rows, map[string]int{"部门甲": 12})
	if len(issues) != 0 || len(sites) != 1 {
		t.Fatalf("sites=%+v issues=%+v", sites, issues)
	}
	if sites[0].SiteName != "站点甲" || sites[0].SiteSerialNumber != "SITE-001" || sites[0].CorpId != 12 {
		t.Fatalf("parsed site = %+v", sites[0])
	}
	var details map[string]any
	if err := json.Unmarshal(sites[0].SiteDetails, &details); err != nil {
		t.Fatalf("unmarshal site details: %v", err)
	}
	if details["addr"] != "地址甲" || details["longitude"] != "120.1" || details["latitude"] != "30.2" {
		t.Fatalf("site details = %+v", details)
	}
}

func TestValidateSiteImportRowsHandlesShortRowsAndRequiredFields(t *testing.T) {
	rows := siteImportTestRows([]string{"", "SITE-NO-NAME"}, []string{"站点乙"})
	sites, issues := validateSiteImportRows(rows, nil)
	if len(sites) != 0 {
		t.Fatalf("invalid rows were parsed: %+v", sites)
	}
	if len(issues) != 2 || issues[0].Row != 2 || !strings.Contains(issues[0].Message, "站点名称不能为空") {
		t.Fatalf("issues = %+v", issues)
	}
	if issues[1].Row != 3 || !strings.Contains(issues[1].Message, "站址编码不能为空") {
		t.Fatalf("short-row issues = %+v", issues)
	}
}

func TestValidateSiteImportRowsRequiresExpectedHeader(t *testing.T) {
	_, issues := validateSiteImportRows([][]string{{"名称", "编码"}}, nil)
	if len(issues) != 1 || issues[0].Row != 1 || !strings.Contains(issues[0].Message, "表头必须为") {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestValidateSiteImportRowsRejectsDuplicateCode(t *testing.T) {
	rows := siteImportTestRows(
		[]string{"站点甲", "SITE-001"},
		[]string{"站点乙", "SITE-001"},
	)
	sites, issues := validateSiteImportRows(rows, nil)
	if len(sites) != 1 {
		t.Fatalf("site count = %d, want 1 valid row retained for diagnostics", len(sites))
	}
	if len(issues) != 1 || issues[0].Row != 3 || !strings.Contains(issues[0].Message, "与第2行重复") {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestValidateSiteImportRowsRejectsTrimmedDuplicateName(t *testing.T) {
	rows := siteImportTestRows(
		[]string{"站点甲", "SITE-001"},
		[]string{"  站点甲  ", "SITE-002"},
	)
	sites, issues := validateSiteImportRows(rows, nil)
	if len(sites) != 1 {
		t.Fatalf("site count = %d, want 1 valid row retained for diagnostics", len(sites))
	}
	if len(issues) != 1 || issues[0].Row != 3 || !strings.Contains(issues[0].Message, "站点名称与第2行重复") {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestSiteImportExistingCodeIssuesRejectsDatabaseCodeAtFirstRow(t *testing.T) {
	rows := siteImportTestRows(
		[]string{"站点甲", "SITE-EXISTS"},
		[]string{"站点乙", "SITE-EXISTS"},
	)
	sites, _ := validateSiteImportRows(rows, nil)
	issues := siteImportExistingCodeIssues(sites, rows, func(code string) bool {
		return code == "SITE-EXISTS"
	})
	if len(issues) != 1 || issues[0].Row != 2 || issues[0].Message != "站址编码已存在，禁止导入" {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestSiteImportExistingNameIssuesRejectsDatabaseName(t *testing.T) {
	rows := siteImportTestRows([]string{"站点甲", "SITE-001"})
	sites, issues := validateSiteImportRows(rows, nil)
	if len(issues) != 0 || len(sites) != 1 {
		t.Fatalf("initial validation returned sites=%+v issues=%+v", sites, issues)
	}

	issues = siteImportExistingNameIssues(sites, rows, func(name string) bool {
		return name == "站点甲"
	})
	if len(issues) != 1 || issues[0].Row != 2 || issues[0].Message != "站点名称已存在，禁止导入" {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestMergeSiteImportIssuesCombinesMessagesByRow(t *testing.T) {
	issues := mergeSiteImportIssues([]siteImportIssue{
		{Row: 2, Message: "站点名称已存在，禁止导入"},
		{Row: 2, Message: "站址编码已存在，禁止导入"},
	})
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "站点名称已存在") || !strings.Contains(issues[0].Message, "站址编码已存在") {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestWriteSiteImportIssuesUsesColumnGAndRedText(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	issues := []siteImportIssue{{Row: 1, Message: "表头错误"}, {Row: 3, Message: "站址编码已存在，禁止导入"}}
	if err := writeSiteImportIssues(f, "Sheet1", issues); err != nil {
		t.Fatal(err)
	}
	assertSiteImportCell(t, f, "G1", "校验错误信息")
	assertSiteImportCell(t, f, "G2", "表头错误")
	assertSiteImportCell(t, f, "G3", "站址编码已存在，禁止导入")
	for _, cell := range []string{"G2", "G3"} {
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

func TestSiteImportErrorFilename(t *testing.T) {
	if got := siteImportErrorFilename("站点导入模板.xlsx"); got != "站点导入模板_校验失败.xlsx" {
		t.Fatalf("error filename = %q", got)
	}
}

func TestWriteSiteWorkbookReturnsExcelStream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetCellValue("Sheet1", "A1", "站点名称"); err != nil {
		t.Fatalf("set workbook cell: %v", err)
	}

	fileName := "站点信息.xlsx"
	writeSiteWorkbook(ctx, fileName, f, http.StatusOK, "导出成功")
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
	assertSiteImportCell(t, exported, "A1", "站点名称")
}

func assertSiteImportCell(t *testing.T, f *excelize.File, cell, want string) {
	t.Helper()
	got, err := f.GetCellValue("Sheet1", cell)
	if err != nil {
		t.Fatalf("get %s: %v", cell, err)
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", cell, got, want)
	}
}
