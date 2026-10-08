package controller

import (
	"strings"
	"testing"
	"time"

	"dicloud/internal/business/model"
	usermodel "dicloud/models/model"

	"github.com/xuri/excelize/v2"
)

func TestMaintainOperationTimeRejectsInvalidAndOutOfRange(t *testing.T) {
	for _, value := range []string{
		"", "NaN", "+Inf", "-1", "0", "20260921", "99999999", "2958466", "999",
		"253402272000", "253402272000000", "99999999999999999999999999",
		"1789956000.5", "2026-02-29", "2026-13-01", "2026-09-21 25:01:00", "10000-01-01",
	} {
		if timestamp, err := parseMaintainOperationTime(value); err == nil {
			t.Fatalf("invalid time %q accepted as %d", value, timestamp)
		}
	}
}

func TestMaintainOperationTimeSupportsExcelLongRangeAndUnix(t *testing.T) {
	for _, test := range []struct{ input, expected string }{
		{"2958465.5", "9999-12-31 12:00:00"},
		{"1789956000", "2026-09-21 10:00:00"},
		{"1789956000123", "2026-09-21 10:00:00"},
		{"2026-09-21T02:00:00Z", "2026-09-21 10:00:00"},
	} {
		timestamp, err := parseMaintainOperationTime(test.input)
		if err != nil || formatMaintainOperationTime(timestamp) != test.expected {
			t.Fatalf("input %q gives %q, err=%v", test.input, formatMaintainOperationTime(timestamp), err)
		}
	}
}

func TestReadMaintainImportRowsUsesRawDateOnlyAndWorkbookCalendar(t *testing.T) {
	for _, date1904 := range []bool{false, true} {
		t.Run(map[bool]string{false: "1900", true: "1904"}[date1904], func(t *testing.T) {
			workbook := excelize.NewFile()
			defer workbook.Close()
			if err := workbook.SetWorkbookProps(&excelize.WorkbookPropsOptions{Date1904: &date1904}); err != nil {
				t.Fatal(err)
			}
			header := append([]string(nil), model.EquipmentMaintainRecordHeader...)
			if err := workbook.SetSheetRow("Sheet1", "A1", &header); err != nil {
				t.Fatal(err)
			}
			base := time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)
			if date1904 {
				base = time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC)
			}
			date := time.Date(2026, time.September, 21, 10, 0, 0, 0, time.UTC)
			serial := date.Sub(base).Hours() / 24
			row := []any{7, "站点", "故障描述", "", serial, ""}
			if err := workbook.SetSheetRow("Sheet1", "A2", &row); err != nil {
				t.Fatal(err)
			}
			codeFormat, dateFormat := "000000", "m/d/yy h:mm"
			for _, cell := range []struct{ name, format string }{{"A2", codeFormat}, {"E2", dateFormat}} {
				style, err := workbook.NewStyle(&excelize.Style{CustomNumFmt: &cell.format})
				if err != nil {
					t.Fatal(err)
				}
				if err := workbook.SetCellStyle("Sheet1", cell.name, cell.name, style); err != nil {
					t.Fatal(err)
				}
			}
			rows, err := readMaintainImportRows(workbook, "Sheet1")
			if err != nil || len(rows) != 2 || rows[1][0] != "000007" || rows[1][4] != "2026-09-21 10:00:00" {
				t.Fatalf("normalized rows=%v, err=%v", rows, err)
			}
		})
	}
}

func TestReadMaintainImportRowsRejectsBadNumericDateWithRow(t *testing.T) {
	workbook := excelize.NewFile()
	defer workbook.Close()
	header := append([]string(nil), model.EquipmentMaintainRecordHeader...)
	if err := workbook.SetSheetRow("Sheet1", "A1", &header); err != nil {
		t.Fatal(err)
	}
	row := []any{"BAT-1", "站点", "故障", "", 20260921}
	if err := workbook.SetSheetRow("Sheet1", "A2", &row); err != nil {
		t.Fatal(err)
	}
	if _, err := readMaintainImportRows(workbook, "Sheet1"); err == nil || !strings.Contains(err.Error(), "第 2 行") {
		t.Fatalf("invalid numeric date not identified: %v", err)
	}
}

func TestMaintainImportSkipsBlankRowsAndRejectsUnsafeImage(t *testing.T) {
	rows := [][]string{
		model.EquipmentMaintainRecordHeader,
		nil,
		{" ", "", "\t"},
		{"BAT-1", "站点", "非法图片", "javascript:alert(1)", "2026-09-21 10:00:00"},
		{"BAT-1", "站点", "空尾列正常", "/go/images/file/a.png", "2026-09-21 10:00:00"},
	}
	records, duplicates, failed := parseMaintainImportRowsDetailed(rows, map[string]int{"BAT-1": 1}, usermodel.User{ID: 2}, 0)
	if len(records) != 1 || duplicates != 0 || failed != 1 || records[0].MaintainTitle != "空尾列正常" {
		t.Fatalf("records=%+v duplicate=%d fail=%d", records, duplicates, failed)
	}
}

func TestMaintainEmbeddedPictureRejectsNonemptyUnsafeReference(t *testing.T) {
	workbook := excelize.NewFile()
	defer workbook.Close()
	if err := workbook.AddPictureFromBytes("Sheet1", "D2", &excelize.Picture{Extension: ".png", File: maintainTestImage(t)}); err != nil {
		t.Fatal(err)
	}
	for _, reference := range []string{"javascript:bad", "https:missing-host.png", `..\outside.png`, "//unexpected-host/image.png"} {
		rows := [][]string{model.EquipmentMaintainRecordHeader, {"BAT-1", "站点", "故障", reference, "2026-09-21 10:00:00"}}
		if err := validateMaintainEmbeddedPictures(workbook, "Sheet1", rows); err == nil {
			t.Fatalf("unsafe embedded picture reference accepted: %q", reference)
		}
	}
}
