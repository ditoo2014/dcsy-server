package controller

import (
	"dicloud/internal/tolink/service"
	"dicloud/models/model"
	"dicloud/util/response"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type TtdcTemplateController struct{}

func (TtdcTemplateController) FindPage(ctx *gin.Context) {
	var req service.TtdcTemplatePageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	rows, total, err := service.NewTtdcTemplateService(ctx).FindPage(req)
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"data": rows, "total": total})
}

func (TtdcTemplateController) Detail(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Query("id"), 10, 64)
	if err != nil || id == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	item, err := service.NewTtdcTemplateService(ctx).Detail(uint(id))
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"data": item})
}

func (TtdcTemplateController) Options(ctx *gin.Context) {
	groupID, _ := strconv.ParseUint(ctx.Query("groupId"), 10, 64)
	rows, err := service.NewTtdcTemplateService(ctx).Options(uint(groupID), ctx.Query("gatewayId"), ctx.Query("moduleAdr"))
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"data": rows})
}

func (TtdcTemplateController) Profiles(ctx *gin.Context) {
	response.Success(ctx, "查询成功", gin.H{"data": service.NewTtdcTemplateService(ctx).Profiles()})
}

func (TtdcTemplateController) Algorithms(ctx *gin.Context) {
	response.Success(ctx, "查询成功", gin.H{"data": service.AlgorithmKeyOptions()})
}

func (TtdcTemplateController) Save(ctx *gin.Context) {
	var req service.TtdcTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	userID := 0
	if value, ok := ctx.Get("userInfo"); ok {
		if user, ok := value.(model.User); ok {
			userID = int(user.ID)
		}
	}
	item, err := service.NewTtdcTemplateService(ctx).Save(req, userID)
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "保存成功", gin.H{"data": item})
}

func (TtdcTemplateController) Delete(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Query("id"), 10, 64)
	if err != nil || id == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	if err = service.NewTtdcTemplateService(ctx).Delete(uint(id)); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "删除成功", nil)
}

func (TtdcTemplateController) SetState(ctx *gin.Context) {
	var req struct {
		ID      uint   `json:"id"`
		Field   string `json:"field"`
		Value   bool   `json:"value"`
		Confirm bool   `json:"confirm"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	if err := service.NewTtdcTemplateService(ctx).SetEnabled(req.ID, req.Field, req.Value, req.Confirm); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "状态更新成功", nil)
}

func (TtdcTemplateController) SetManualEnabled(ctx *gin.Context) {
	setTemplateState(ctx, "enabled", false)
}
func (TtdcTemplateController) SetAutoCreateEnabled(ctx *gin.Context) {
	setTemplateState(ctx, "auto_create_enabled", false)
}
func (TtdcTemplateController) SetRuntimeEnabled(ctx *gin.Context) {
	setTemplateState(ctx, "runtime_match_enabled", true)
}

func (TtdcTemplateController) Export(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Query("id"), 10, 64)
	if err != nil || id == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	item, err := service.NewTtdcTemplateService(ctx).Detail(uint(id))
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	f := excelize.NewFile()
	defer f.Close()
	defaultSheet := f.GetSheetName(0)
	infoSheet, pointSheet := "模板信息", "数据点"
	f.NewSheet(infoSheet)
	f.NewSheet(pointSheet)
	_ = f.DeleteSheet(defaultSheet)
	infoHeaders := []string{"groupId", "templateName", "cid2Type", "enabled", "version", "description"}
	writeExcelRow(f, infoSheet, 1, toAnyStrings(infoHeaders))
	writeExcelRow(f, infoSheet, 2, []interface{}{item.GroupID, item.TemplateName, item.CID2Type, boolInt(item.Enabled), item.Version, item.Description})
	writeExcelRow(f, pointSheet, 1, toAnyStrings([]string{"sortNo", "baseDataId", "name", "dataType", "address", "byteCount", "convertType", "bitIndex", "algorithmKey", "transport", "formula", "unformula", "enabled", "description"}))
	for i, point := range item.Points {
		bit := interface{}("")
		if point.BitIndex != nil {
			bit = *point.BitIndex
		}
		writeExcelRow(f, pointSheet, i+2, []interface{}{point.SortNo, point.BaseDataID, point.Name, point.DataType, pointerIntValue(point.Address), pointerIntValue(point.ByteCount), pointerIntValue(point.ConvertType), bit, pointerStringValue(point.AlgorithmKey), point.Transport, point.Formula, point.Unformula, boolInt(point.Enabled), point.Description})
	}
	fileName := item.TemplateName + ".xlsx"
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.QueryEscape(fileName)))
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	if err := f.Write(ctx.Writer); err != nil {
		service.WriteTemplateFailure(ctx, err)
	}
}

func (TtdcTemplateController) Import(ctx *gin.Context) {
	upload, err := ctx.FormFile("file")
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	file, err := upload.Open()
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	defer file.Close()
	book, err := excelize.OpenReader(file)
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	defer book.Close()
	info, err := book.GetRows("模板信息")
	if err != nil || len(info) < 2 {
		service.WriteTemplateFailure(ctx, fmt.Errorf("模板信息Sheet缺少数据"))
		return
	}
	row := info[1]
	groupID := uint(excelInt(ctx.PostForm("groupId")))
	if groupID == 0 {
		groupID = uint(excelInt(excelHeaderCell(info[0], row, "groupId")))
	}
	req := service.TtdcTemplateRequest{
		GroupID: groupID, TemplateName: excelHeaderCell(info[0], row, "templateName"), CID2Type: excelHeaderCell(info[0], row, "cid2Type"),
		Enabled: boolPtr(excelBool(excelHeaderCell(info[0], row, "enabled"), true)), Version: excelInt(excelHeaderCell(info[0], row, "version")),
		Description: excelHeaderCell(info[0], row, "description"),
	}
	pointRows, err := book.GetRows("数据点")
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	for i, values := range pointRows {
		if i == 0 || strings.TrimSpace(excelCell(values, 1)) == "" {
			continue
		}
		var bit *int
		if value := strings.TrimSpace(excelCell(values, 7)); value != "" {
			n := excelInt(value)
			bit = &n
		}
		req.Points = append(req.Points, service.TtdcTemplatePointRequest{SortNo: excelInt(excelCell(values, 0)), BaseDataID: excelCell(values, 1), Name: excelCell(values, 2), DataType: excelInt(excelCell(values, 3)), Address: excelInt(excelCell(values, 4)), ByteCount: excelInt(excelCell(values, 5)), ConvertType: excelInt(excelCell(values, 6)), BitIndex: bit, AlgorithmKey: excelCell(values, 8), Transport: excelCell(values, 9), Formula: excelCell(values, 10), Unformula: excelCell(values, 11), Enabled: excelBool(excelCell(values, 12), true), Description: excelCell(values, 13)})
	}
	userID := 0
	if value, ok := ctx.Get("userInfo"); ok {
		if user, ok := value.(model.User); ok {
			userID = int(user.ID)
		}
	}
	item, err := service.NewTtdcTemplateService(ctx).Save(req, userID)
	if err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "导入成功", gin.H{"data": item})
}

func writeExcelRow(f *excelize.File, sheet string, row int, values []interface{}) {
	cell, _ := excelize.CoordinatesToCellName(1, row)
	_ = f.SetSheetRow(sheet, cell, &values)
}
func toAnyStrings(values []string) []interface{} {
	result := make([]interface{}, len(values))
	for i := range values {
		result[i] = values[i]
	}
	return result
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
func boolPtr(value bool) *bool { return &value }
func pointerIntValue(value *int) interface{} {
	if value == nil {
		return ""
	}
	return *value
}
func pointerStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func excelCell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func excelHeaderCell(headers, row []string, name string) string {
	for index, header := range headers {
		if strings.EqualFold(strings.TrimSpace(header), name) {
			return excelCell(row, index)
		}
	}
	return ""
}
func excelInt(value string) int { n, _ := strconv.Atoi(strings.TrimSpace(value)); return n }
func excelBool(value string, fallback bool) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "是"
}

func setTemplateState(ctx *gin.Context, field string, requireConfirm bool) {
	var req struct {
		ID      uint `json:"id"`
		Value   bool `json:"value"`
		Confirm bool `json:"confirm"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil || req.ID == 0 {
		service.WriteTemplateFailure(ctx, errInvalidTemplateID())
		return
	}
	if requireConfirm && !req.Value && !req.Confirm {
		service.WriteTemplateFailure(ctx, templateConfirmError{})
		return
	}
	if err := service.NewTtdcTemplateService(ctx).SetEnabled(req.ID, field, req.Value, req.Confirm || req.Value); err != nil {
		service.WriteTemplateFailure(ctx, err)
		return
	}
	response.Success(ctx, "状态更新成功", nil)
}

type templateIDError struct{}

func (templateIDError) Error() string { return "模板ID不合法" }
func errInvalidTemplateID() error     { return templateIDError{} }

type templateConfirmError struct{}

func (templateConfirmError) Error() string { return "关闭运行时接收必须确认" }
