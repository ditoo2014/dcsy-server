package controller

import (
	"dicloud/internal/business/model"
	"dicloud/util/response"
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

func (ctrler SiteCtrler) ImportTemplate(ctx *gin.Context) {
	f, err := newImportTemplate(model.SiteExportHeader)
	if err != nil {
		response.Fail(ctx, "生成站点模板失败", nil)
		return
	}
	defer f.Close()
	writeSiteWorkbook(ctx, "站点信息导入模板.xlsx", f, 200, "模板下载成功")
}

func (ctrler WarehouseCtrler) ImportTemplate(ctx *gin.Context) {
	f, err := newImportTemplate(model.WarehouseExportHeader)
	if err != nil {
		response.Fail(ctx, "生成库房模板失败", nil)
		return
	}
	defer f.Close()
	writeWarehouseWorkbook(ctx, "库房信息导入模板.xlsx", f, 200, "模板下载成功")
}
