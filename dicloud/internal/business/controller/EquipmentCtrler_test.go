package controller

import (
	"strings"
	"testing"

	"dicloud/internal/business/model"

	"github.com/xuri/excelize/v2"
)

func equipmentImportTestRows(dataRows ...[]string) [][]string {
	header := append([]string(nil), model.EquipmentExportHeader...)
	return append([][]string{header}, dataRows...)
}

func TestValidateEquipmentImportRowsAcceptsFreeTextType(t *testing.T) {
	rows := equipmentImportTestRows([]string{
		"BAT-001", "定制磷酸铁锂-A型", "厂家甲", "2026-08-01", "P001", "48V100Ah", "测试公司", "BMS甲", "V1.0",
	})

	equipments, issues := validateEquipmentImportRows(rows, map[string]int{"测试公司": 12})
	if len(issues) != 0 {
		t.Fatalf("valid row returned issues: %+v", issues)
	}
	if len(equipments) != 1 {
		t.Fatalf("equipment count = %d, want 1", len(equipments))
	}
	if equipments[0].EquipmentTypeName != "定制磷酸铁锂-A型" {
		t.Fatalf("equipment type = %q", equipments[0].EquipmentTypeName)
	}
	if equipments[0].CorpId != 12 {
		t.Fatalf("corp id = %d, want 12", equipments[0].CorpId)
	}
}

func TestValidateEquipmentImportRowsRequiresType(t *testing.T) {
	rows := equipmentImportTestRows([]string{"BAT-001", ""})

	equipments, issues := validateEquipmentImportRows(rows, nil)
	if len(equipments) != 0 {
		t.Fatalf("invalid row was parsed: %+v", equipments)
	}
	if len(issues) != 1 || issues[0].Row != 2 || !strings.Contains(issues[0].Message, "电池类型不能为空") {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestValidateEquipmentImportRowsRejectsDuplicateCode(t *testing.T) {
	rows := equipmentImportTestRows(
		[]string{"BAT-001", "类型甲"},
		[]string{"BAT-001", "类型乙"},
	)

	_, issues := validateEquipmentImportRows(rows, nil)
	if len(issues) != 1 || issues[0].Row != 3 || !strings.Contains(issues[0].Message, "与第2行重复") {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestEquipmentImportExistingCodeIssuesRejectDatabaseCode(t *testing.T) {
	rows := equipmentImportTestRows([]string{"BAT-EXISTS", "类型甲"})
	equipments, issues := validateEquipmentImportRows(rows, nil)
	if len(issues) != 0 || len(equipments) != 1 {
		t.Fatalf("initial validation returned equipments=%+v issues=%+v", equipments, issues)
	}

	issues = equipmentImportExistingCodeIssues(equipments, rows, func(code string) bool {
		return code == "BAT-EXISTS"
	})
	if len(issues) != 1 || issues[0].Row != 2 || issues[0].Message != "电池ID已存在，禁止导入" {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestWriteEquipmentImportIssuesUsesLastColumnAndRedText(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	issues := []equipmentImportIssue{{Row: 1, Message: "表头错误"}, {Row: 3, Message: "电池类型不能为空"}}
	if err := writeEquipmentImportIssues(f, "Sheet1", issues); err != nil {
		t.Fatal(err)
	}

	assertEquipmentImportCell(t, f, "J1", "校验错误信息")
	assertEquipmentImportCell(t, f, "J2", "表头错误")
	assertEquipmentImportCell(t, f, "J3", "电池类型不能为空")
	for _, cell := range []string{"J2", "J3"} {
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

func assertEquipmentImportCell(t *testing.T, f *excelize.File, cell, want string) {
	t.Helper()
	got, err := f.GetCellValue("Sheet1", cell)
	if err != nil {
		t.Fatalf("get %s: %v", cell, err)
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", cell, got, want)
	}
}
