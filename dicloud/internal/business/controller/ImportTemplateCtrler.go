package controller

import (
	"dicloud/internal/business/model"
	"dicloud/util/response"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// newImportTemplate 复用实际导入表头，避免模板和导入校验不一致。
func newImportTemplate(headers []string) (*excelize.File, error) {
	f := excelize.NewFile()
	for i, title := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue("Sheet1", cell, title); err != nil {
			_ = f.Close()
			return nil, err
		}
	}
	lastColumn, _ := excelize.ColumnNumberToName(len(headers))
	_ = f.SetColWidth("Sheet1", "A", lastColumn, 20)
	// 编码和电话按文本保存，保留前导零及长数字。
	if style, err := f.NewStyle(&excelize.Style{NumFmt: 49}); err == nil {
		_ = f.SetCellStyle("Sheet1", "A2", lastColumn+"1001", style)
	}
	return f, nil
}

// addImportTemplateExamples 单独提供填写参考，避免示例被第一个工作表的导入逻辑写入数据库。
func addImportTemplateExamples(f *excelize.File, headers []string, examples [][]string, notes []string) error {
	const sheet = "填写示例"
	index, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}
	rows := append([][]string{headers}, examples...)
	for rowIndex, row := range rows {
		for columnIndex, value := range row {
			cell, err := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			if err != nil {
				return err
			}
			if err := f.SetCellStr(sheet, cell, value); err != nil {
				return err
			}
		}
	}
	lastColumn, err := excelize.ColumnNumberToName(len(headers))
	if err != nil {
		return err
	}
	if err := f.SetColWidth(sheet, "A", lastColumn, 24); err != nil {
		return err
	}
	// 地址及部门名称较长，适当加宽示例页，实际导入页的布局保持不变。
	for columnIndex, title := range headers {
		width := 24.0
		switch title {
		case "详细地址", "库房地址":
			width = 52
		case "部门", "所属公司":
			width = 28
		}
		column, _ := excelize.ColumnNumberToName(columnIndex + 1)
		if err := f.SetColWidth(sheet, column, column, width); err != nil {
			return err
		}
	}
	// 示例统一为文本，长编码、电话和日期不会被 Excel 自动转换。
	exampleStyle, err := f.NewStyle(&excelize.Style{NumFmt: 49, Font: &excelize.Font{Color: "FF0000"}})
	if err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A2", fmt.Sprintf("%s%d", lastColumn, len(rows)), exampleStyle); err != nil {
		return err
	}
	noteStyle, err := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{WrapText: true, Vertical: "center"}})
	if err != nil {
		return err
	}
	for noteIndex, note := range notes {
		row := len(rows) + 2 + noteIndex
		firstCell, lastCell := fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastColumn, row)
		if err := f.MergeCell(sheet, firstCell, lastCell); err != nil {
			return err
		}
		if err := f.SetCellStr(sheet, firstCell, note); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, firstCell, lastCell, noteStyle); err != nil {
			return err
		}
		if err := f.SetRowHeight(sheet, row, 32); err != nil {
			return err
		}
	}
	// 打开文件时先看到示例；工作表顺序不变，后端仍只导入第一个 Sheet1。
	f.SetActiveSheet(index)
	return nil
}

func (ctrler SiteCtrler) ImportTemplate(ctx *gin.Context) {
	f, err := newImportTemplate(model.SiteExportHeader)
	if err != nil {
		response.Fail(ctx, "生成站点模板失败", nil)
		return
	}
	defer f.Close()
	if err := addImportTemplateExamples(f, model.SiteExportHeader, [][]string{
		{"示例站点一", "000000000000000001", "请填写已有部门名称", "四川省成都市示例路1号", "104.066541", "30.572269"},
		{"示例站点二", "000000000000000002", "请填写已有部门名称", "四川省成都市示例路2号", "104.067541", "30.573269"},
	}, []string{
		"填写说明：红色内容仅为示例，请将真实数据填写到第一个“Sheet1”工作表，从第2行开始；本页不参与导入。",
		"站点名称、站址编码不能为空，且不能与文件中其他行或平台已有站点重复。部门请填写平台中已有的完整部门名称。",
		"请保留表头及列顺序；编码按文本填写，保留前导零。经度、纬度按十进制度填写，不要带单位。",
	}); err != nil {
		response.Fail(ctx, "生成站点模板失败", nil)
		return
	}
	writeSiteWorkbook(ctx, "站点信息导入模板.xlsx", f, 200, "模板下载成功")
}

func (ctrler WarehouseCtrler) ImportTemplate(ctx *gin.Context) {
	f, err := newImportTemplate(model.WarehouseExportHeader)
	if err != nil {
		response.Fail(ctx, "生成库房模板失败", nil)
		return
	}
	defer f.Close()
	if err := addImportTemplateExamples(f, model.WarehouseExportHeader, [][]string{
		{"示例库房一", "KF000001", "四川省成都市示例路1号", "示例管理员甲", "13800000000", "请填写已有部门名称", "104.066541", "30.572269"},
		{"示例库房二", "KF000002", "四川省成都市示例路2号", "示例管理员乙", "13900000000", "请填写已有部门名称", "104.067541", "30.573269"},
	}, []string{
		"填写说明：红色内容仅为示例，请将真实数据填写到第一个“Sheet1”工作表，从第2行开始；本页不参与导入。",
		"库房名称、库房编号不能为空，且不能与文件中其他行或平台已有库房重复。部门请填写平台中已有的完整部门名称。",
		"请保留表头及列顺序；编号、管理员电话按文本填写。示例电话请替换为真实管理员电话，经纬度按十进制度填写。",
	}); err != nil {
		response.Fail(ctx, "生成库房模板失败", nil)
		return
	}
	writeWarehouseWorkbook(ctx, "库房信息导入模板.xlsx", f, 200, "模板下载成功")
}
