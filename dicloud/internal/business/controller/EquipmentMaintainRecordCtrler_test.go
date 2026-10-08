package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	usermodel "dicloud/models/model"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func TestMaintainRecordFilterPreservesTimeAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/?startTime=10&endTime=90", strings.NewReader(`{"equipmentId":0,"equipmentCode":"BAT","equipmentTypeName":"钠电池","minimumCount":3,"startTime":2,"isAdmin":true,"corpIDs":[999]}`))
	filter, err := bindMaintainFilter(ctx, 100)
	if err != nil || filter.StartTime != 10 || filter.EndTime != 90 || filter.EquipmentCode != "BAT" || filter.EquipmentTypeName != "钠电池" || filter.MinimumCount != 3 {
		t.Fatalf("filter=%+v, err=%v", filter, err)
	}
	if filter.IsAdmin || len(filter.CorpIDs) != 0 {
		t.Fatal("request body supplied trusted authorization fields")
	}
}

func TestMaintainRecordFilterRejectsInvalidValues(t *testing.T) {
	for _, test := range []struct{ path, body string }{
		{"/?startTime=100&endTime=10", `{}`},
		{"/?startTime=bad", `{}`},
		{"/", `{"minimumCount":-1}`},
		{"/", `{"equipmentId":-1}`},
		{"/", `{"equipmentId":"bad"}`},
	} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
		if _, err := bindMaintainFilter(ctx, 100); err == nil {
			t.Fatalf("invalid query accepted: %+v", test)
		}
	}
}

func TestMaintainRecordWorkbookStreamsRowsAndEmptyHeader(t *testing.T) {
	for _, rows := range [][]dao.MaintainRecordRow{
		{},
		{{EquipmentMaintainRecord: model.EquipmentMaintainRecord{MaintainContent: "更换连接线", MaintainDetails: json.RawMessage(`{"create_time":"2026-09-21 10:00:00","bs":"业务云上报"}`)}, EquipmentCode: "BAT-1", SiteSerialNumber: "SITE-1"}},
	} {
		f, err := buildMaintainWorkbook(rows)
		if err != nil {
			t.Fatal(err)
		}
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		writeMaintainWorkbook(ctx, f)
		f.Close()
		if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
			t.Fatalf("invalid Excel response headers: %+v", recorder.Header())
		}
		if !strings.Contains(recorder.Header().Get("Content-Disposition"), "filename*=UTF-8''") {
			t.Fatal("missing UTF-8 download filename")
		}
		exported, err := excelize.OpenReader(bytes.NewReader(recorder.Body.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		gotRows, err := exported.GetRows("Sheet1")
		exported.Close()
		if err != nil || len(gotRows) != len(rows)+1 || gotRows[0][0] != model.EquipmentMaintainRecordHeader[0] {
			t.Fatalf("unexpected rows=%+v err=%v", gotRows, err)
		}
		if len(rows) > 0 && (gotRows[1][0] != "BAT-1" || gotRows[1][2] != "更换连接线") {
			t.Fatalf("exported values=%+v", gotRows[1])
		}
	}
}

func TestMaintainRecordImportRejectsShortAndUnauthorizedRows(t *testing.T) {
	rows := [][]string{
		model.EquipmentMaintainRecordHeader,
		{"BAT-1"},
		{"BAT-2", "SITE", "故障", "", "2026-09-21 10:00:00", "标识"},
		{" BAT-1 ", "SITE", "故障", "", "2026-09-21 10:00:00", "标识"},
	}
	user := usermodel.User{ID: 7, Username: "操作员"}
	records, failed := parseMaintainImportRows(rows, map[string]int{"BAT-1": 1}, user, 123)
	if failed != 2 || len(records) != 1 || records[0].EquipmentId != 1 || records[0].MaintainUserId != 7 || records[0].CreateTime != 1789956000 {
		t.Fatalf("records=%+v failed=%d", records, failed)
	}
}

func TestMaintainImportStrictHeaderDateAndDuplicateValidation(t *testing.T) {
	if err := validateMaintainImportHeader([][]string{{"电池ID", "站址名称", "故障描述", "图片", "运维时间", "标识", ""}}); err != nil {
		t.Fatalf("empty trailing column should be accepted: %v", err)
	}
	if err := validateMaintainImportHeader([][]string{{"电池ID", "站址名称", "故障描述"}}); err == nil {
		t.Fatal("short header accepted")
	}
	rows := [][]string{
		model.EquipmentMaintainRecordHeader,
		{"BAT-1", "SITE", "更换连接线", "/go/images/file/a.png", "2026年9月21日 10:00", "业务云"},
		{"BAT-1", "SITE", "更换连接线", "/go/images/file/a.png", "2026-09-21 10:00:00", "业务云"},
		{"BAT-1", "SITE", "另一故障", "", "45200.5", ""},
	}
	records, duplicateCount, failCount := parseMaintainImportRowsDetailed(rows, map[string]int{"BAT-1": 1}, usermodel.User{ID: 2, Username: "用户"}, 9)
	if len(records) != 2 || duplicateCount != 1 || failCount != 0 {
		t.Fatalf("records=%+v duplicate=%d fail=%d", records, duplicateCount, failCount)
	}
	if records[0].MaintainTitle != "更换连接线" || records[0].CreateTime <= 0 || records[0].MaintainDetails == nil {
		t.Fatalf("import fields not normalized: %+v", records[0])
	}
}

func TestMaintainExportUsesTitleAndCreateTimeWithLegacyFallback(t *testing.T) {
	details, _ := json.Marshal(map[string]any{"create_time": "2024-01-02 03:04:05", "bs": "旧数据"})
	f, err := buildMaintainWorkbook([]dao.MaintainRecordRow{{
		EquipmentMaintainRecord: model.EquipmentMaintainRecord{MaintainTitle: "故障标题", MaintainContent: "原始内容", CreateTime: 1704164645, MaintainDetails: details},
		EquipmentCode:           "BAT-1", SiteSerialNumber: "SITE-1",
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows("Sheet1")
	if err != nil || len(rows) != 2 || rows[1][2] != "故障标题" || rows[1][4] != "2024-01-02 11:04:05" {
		t.Fatalf("export row=%v err=%v", rows, err)
	}
}

func TestMaintainEmbeddedPictureRequiresReference(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetRow("Sheet1", "A1", &[]any{"电池ID", "站址名称", "故障描述", "图片", "运维时间", "标识"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("Sheet1", "A2", &[]any{"BAT-1", "SITE", "故障", "", "2026-09-21 10:00:00", ""}); err != nil {
		t.Fatal(err)
	}
	if err := validateMaintainEmbeddedPictures(f, "Sheet1", [][]string{{"电池ID", "站址名称", "故障描述", "图片", "运维时间", "标识"}, {"BAT-1", "SITE", "故障", "", "2026-09-21 10:00:00", ""}}); err != nil {
		t.Fatalf("workbook without embedded picture rejected: %v", err)
	}
	if err := f.AddPictureFromBytes("Sheet1", "D2", &excelize.Picture{Extension: ".png", File: maintainTestImage(t)}); err != nil {
		t.Fatal(err)
	}
	if err := validateMaintainEmbeddedPictures(f, "Sheet1", [][]string{{"电池ID", "站址名称", "故障描述", "图片", "运维时间", "标识"}, {"BAT-1", "SITE", "故障", "", "2026-09-21 10:00:00", ""}}); err == nil {
		t.Fatal("embedded picture without reference accepted")
	}
}
