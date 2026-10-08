package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	businessmodel "dicloud/internal/business/model"
	tolinkmodel "dicloud/internal/tolink/model"
	usermodel "dicloud/models/model"
	"dicloud/util/gormutil"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const streamBatchImportMaxRows = 1000

type streamBatchImportRow struct {
	ExcelRow   int
	StreamName string
	StreamCode string
	GatewayID  string
	TtdcADR    string
}

type streamBatchImportPoint struct {
	Name     string
	DataID   string
	DataType int
}

type streamBatchImportPlan struct {
	Row       streamBatchImportRow
	Gateway   tolinkmodel.Gateway
	Equipment businessmodel.Equipment
	Points    []streamBatchImportPoint
}

type streamBatchImportIssue struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type streamBatchImportResult struct {
	Row      int    `json:"row"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	StreamID uint   `json:"streamId,omitempty"`
	ApiKey   string `json:"apiKey,omitempty"`
}

type streamBatchImportResponse struct {
	Total        int                       `json:"total"`
	SuccessCount int                       `json:"successCount"`
	FailureCount int                       `json:"failureCount"`
	Issues       []streamBatchImportIssue  `json:"issues,omitempty"`
	Results      []streamBatchImportResult `json:"results,omitempty"`
}

func newStreamBatchImportTemplate() (*excelize.File, error) {
	f := excelize.NewFile()
	textStyle, err := f.NewStyle(&excelize.Style{NumFmt: 49})
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	sheet := f.GetSheetName(0)
	if err := f.SetSheetName(sheet, "导入数据"); err != nil {
		_ = f.Close()
		return nil, err
	}
	sheet = "导入数据"
	for i, value := range []string{"数据池名称", "电池ID", "网关ID", "TTDC地址"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, value); err != nil {
			_ = f.Close()
			return nil, err
		}
	}
	if err := f.SetColWidth(sheet, "A", "D", 24); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := f.SetCellStyle(sheet, "A2", "D1001", textStyle); err != nil {
		_ = f.Close()
		return nil, err
	}

	exampleSheet := "填写示例"
	if _, err := f.NewSheet(exampleSheet); err != nil {
		_ = f.Close()
		return nil, err
	}
	exampleRows := [][]any{
		{"数据池名称", "电池ID", "网关ID", "TTDC地址"},
		{"TTDC数据池示例", "BATTERY-001", "GATEWAY-001", "01"},
	}
	for rowIndex, row := range exampleRows {
		for colIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
			if err := f.SetCellValue(exampleSheet, cell, value); err != nil {
				_ = f.Close()
				return nil, err
			}
		}
	}
	if err := f.SetCellValue(exampleSheet, "A4", "示例仅供参考，请将真实数据填写到“导入数据”工作表；电池ID和网关ID必须在系统中存在。"); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := f.SetColWidth(exampleSheet, "A", "D", 24); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := f.SetCellStyle(exampleSheet, "A2", "D1001", textStyle); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

// BatchImportTemplate returns an empty import sheet and one TTDC example sheet.
func (s StreamCtrler) BatchImportTemplate(ctx *gin.Context) {
	f, err := newStreamBatchImportTemplate()
	if err != nil {
		ctx.Status(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", `attachment; filename="stream-batch-import-template.xlsx"`)
	if err := f.Write(ctx.Writer); err != nil {
		ctx.Status(http.StatusInternalServerError)
	}
}

// BatchImport validates the complete workbook first, then writes every row in one transaction.
func (s StreamCtrler) BatchImport(ctx *gin.Context) {
	dryRun := strings.EqualFold(strings.TrimSpace(ctx.Query("dryRun")), "true")
	file, err := ctx.FormFile("file")
	if err != nil {
		response.Fail(ctx, "获取上传文件失败", gin.H{"issues": []streamBatchImportIssue{{Row: 0, Field: "file", Message: err.Error()}}})
		return
	}
	if file.Size <= 0 || file.Size > 20*1024*1024 || !strings.HasSuffix(strings.ToLower(file.Filename), ".xlsx") {
		response.Fail(ctx, "请上传不超过20MB的xlsx文件", nil)
		return
	}
	tmp, err := os.CreateTemp("", "stream-batch-import-*.xlsx")
	if err != nil {
		response.Fail(ctx, "保存上传文件失败", nil)
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err = ctx.SaveUploadedFile(file, tmpPath); err != nil {
		_ = tmp.Close()
		response.Fail(ctx, "保存上传文件失败", nil)
		return
	}
	_ = tmp.Close()
	f, err := excelize.OpenFile(tmpPath)
	if err != nil {
		response.Fail(ctx, "打开Excel失败", nil)
		return
	}
	defer f.Close()
	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		response.Fail(ctx, "读取Excel失败", nil)
		return
	}
	inputRows, issues := parseStreamBatchImportRows(rows)
	if len(issues) > 0 {
		respondStreamBatchImportIssues(ctx, f, file.Filename, len(inputRows), issues, dryRun)
		return
	}
	plans, issues := buildStreamBatchImportPlans(ctx, inputRows)
	if len(issues) > 0 {
		respondStreamBatchImportIssues(ctx, f, file.Filename, len(inputRows), issues, dryRun)
		return
	}

	if dryRun {
		response.Success(ctx, "校验通过", gin.H{"total": len(plans), "successCount": 0, "failureCount": 0, "results": previewStreamBatchImportResults(plans)})
		return
	}

	result, err := commitStreamBatchImport(ctx, plans)
	if err != nil {
		response.Fail(ctx, "批量导入失败，已全部回滚", gin.H{"total": len(plans), "issues": []streamBatchImportIssue{{Row: 0, Field: "database", Message: err.Error()}}})
		return
	}
	response.Success(ctx, "批量导入成功", gin.H{"total": result.Total, "successCount": result.SuccessCount, "failureCount": 0, "results": result.Results})
}

func respondStreamBatchImportIssues(ctx *gin.Context, f *excelize.File, originalFilename string, total int, issues []streamBatchImportIssue, dryRun bool) {
	if !dryRun {
		response.Fail(ctx, "导入校验失败", gin.H{"total": total, "issues": issues})
		return
	}
	if err := writeStreamBatchImportIssues(f, issues); err != nil {
		response.Fail(ctx, "生成校验结果文件失败", nil)
		return
	}
	name := streamBatchImportErrorFilename(originalFilename)
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="stream-batch-import-errors.xlsx"; filename*=UTF-8''%s`, url.QueryEscape(name)))
	if err := f.Write(ctx.Writer); err != nil {
		ctx.Status(http.StatusInternalServerError)
	}
}

func streamBatchImportErrorFilename(originalFilename string) string {
	base := filepath.Base(strings.TrimSpace(originalFilename))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if base == "" || base == "." {
		base = "数据池批量导入"
	}
	return base + "_校验失败.xlsx"
}

func writeStreamBatchImportIssues(f *excelize.File, issues []streamBatchImportIssue) error {
	sheet := f.GetSheetName(0)
	if sheet == "" {
		return errors.New("Excel缺少工作表")
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return err
	}
	maxCol, errorCol, maxRow := 0, 0, len(rows)
	for rowIndex, row := range rows {
		if len(row) > maxCol {
			maxCol = len(row)
		}
		if rowIndex == 0 {
			for colIndex, value := range row {
				if strings.TrimSpace(value) == "校验错误" {
					errorCol = colIndex + 1
					break
				}
			}
		}
	}
	if errorCol == 0 {
		errorCol = maxCol + 1
		if errorCol < 1 {
			errorCol = 1
		}
	}
	grouped := make(map[int][]string)
	seen := make(map[int]map[string]bool)
	for _, issue := range issues {
		row := issue.Row
		if row <= 1 {
			row = 2
		}
		message := strings.TrimSpace(issue.Message)
		if field := strings.TrimSpace(issue.Field); field != "" {
			message = field + "：" + message
		}
		if message == "" {
			continue
		}
		if seen[row] == nil {
			seen[row] = make(map[string]bool)
		}
		if !seen[row][message] {
			seen[row][message] = true
			grouped[row] = append(grouped[row], message)
		}
		if row > maxRow {
			maxRow = row
		}
	}
	headerCell, _ := excelize.CoordinatesToCellName(errorCol, 1)
	if err := f.SetCellValue(sheet, headerCell, "校验错误"); err != nil {
		return err
	}
	if errorCol > 1 {
		previousHeader, _ := excelize.CoordinatesToCellName(errorCol-1, 1)
		if styleID, styleErr := f.GetCellStyle(sheet, previousHeader); styleErr == nil && styleID > 0 {
			if err := f.SetCellStyle(sheet, headerCell, headerCell, styleID); err != nil {
				return err
			}
		}
	}
	if maxRow < 2 {
		maxRow = 2
	}
	startCell, _ := excelize.CoordinatesToCellName(errorCol, 2)
	endCell, _ := excelize.CoordinatesToCellName(errorCol, maxRow)
	if err := f.SetCellValue(sheet, startCell, ""); err != nil {
		return err
	}
	for row := 2; row <= maxRow; row++ {
		cell, _ := excelize.CoordinatesToCellName(errorCol, row)
		if err := f.SetCellValue(sheet, cell, strings.Join(grouped[row], "；")); err != nil {
			return err
		}
	}
	errorStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "F56C6C"},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		NumFmt:    49,
	})
	if err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, startCell, endCell, errorStyle); err != nil {
		return err
	}
	columnName, _ := excelize.ColumnNumberToName(errorCol)
	return f.SetColWidth(sheet, columnName, columnName, 48)
}

func parseStreamBatchImportRows(rows [][]string) ([]streamBatchImportRow, []streamBatchImportIssue) {
	if len(rows) < 2 {
		return nil, []streamBatchImportIssue{{Row: 1, Field: "header", Message: "Excel至少需要一行表头和一行数据"}}
	}
	index := make(map[string]int, len(rows[0]))
	for i, raw := range rows[0] {
		key := strings.TrimSpace(raw)
		if key != "" {
			index[key] = i
		}
	}
	for _, header := range []string{"数据池名称", "电池ID", "网关ID", "TTDC地址"} {
		if _, ok := index[header]; !ok {
			return nil, []streamBatchImportIssue{{Row: 1, Field: header, Message: "缺少必需列"}}
		}
	}
	var out []streamBatchImportRow
	var issues []streamBatchImportIssue
	for i, cells := range rows[1:] {
		excelRow := i + 2
		values := func(header string) string {
			pos := index[header]
			if pos >= len(cells) {
				return ""
			}
			return strings.TrimSpace(cells[pos])
		}
		name, code, gatewayID, adr := values("数据池名称"), values("电池ID"), values("网关ID"), strings.ToUpper(values("TTDC地址"))
		if name == "" && code == "" && gatewayID == "" && adr == "" {
			continue
		}
		row := streamBatchImportRow{ExcelRow: excelRow, StreamName: name, StreamCode: code, GatewayID: gatewayID, TtdcADR: adr}
		for field, value := range map[string]string{"数据池名称": name, "电池ID": code, "网关ID": gatewayID} {
			if value == "" {
				issues = append(issues, streamBatchImportIssue{Row: excelRow, Field: field, Message: "不能为空"})
			}
		}
		out = append(out, row)
		if len(out) > streamBatchImportMaxRows {
			issues = append(issues, streamBatchImportIssue{Row: excelRow, Field: "file", Message: fmt.Sprintf("最多导入%d行", streamBatchImportMaxRows)})
			break
		}
	}
	if len(out) == 0 && len(issues) == 0 {
		issues = append(issues, streamBatchImportIssue{Row: 1, Field: "file", Message: "没有可导入的数据"})
	}
	checkDuplicateRows(out, &issues, "数据池名称", func(row streamBatchImportRow) string { return row.StreamName })
	checkDuplicateRows(out, &issues, "电池ID", func(row streamBatchImportRow) string { return row.StreamCode })
	return out, issues
}

func checkDuplicateRows(rows []streamBatchImportRow, issues *[]streamBatchImportIssue, field string, value func(streamBatchImportRow) string) {
	seen := map[string]int{}
	for _, row := range rows {
		key := value(row)
		if key == "" {
			continue
		}
		if first, ok := seen[key]; ok {
			*issues = append(*issues, streamBatchImportIssue{Row: row.ExcelRow, Field: field, Message: fmt.Sprintf("与第%d行重复", first)})
		} else {
			seen[key] = row.ExcelRow
		}
	}
}

func buildStreamBatchImportPlans(ctx *gin.Context, rows []streamBatchImportRow) ([]streamBatchImportPlan, []streamBatchImportIssue) {
	db := gormutil.GetGormDB(ctx, "")
	if db == nil {
		return nil, []streamBatchImportIssue{{Row: 0, Field: "database", Message: "数据库未初始化"}}
	}
	names, codes := make([]string, 0, len(rows)), make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.StreamName)
		codes = append(codes, row.StreamCode)
	}
	var existing []tolinkmodel.Stream
	if err := db.Where("name IN ? OR stream_code IN ?", names, codes).Find(&existing).Error; err != nil {
		return nil, []streamBatchImportIssue{{Row: 0, Field: "database", Message: err.Error()}}
	}
	var issues []streamBatchImportIssue
	for _, item := range existing {
		for _, row := range rows {
			if item.Name == row.StreamName {
				issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "数据池名称", Message: "数据库中已存在"})
			}
			if item.StreamCode == row.StreamCode {
				issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "电池ID", Message: "数据库中已存在数据池"})
			}
		}
	}
	plans := make([]streamBatchImportPlan, 0, len(rows))
	for _, row := range rows {
		var gateway tolinkmodel.Gateway
		if err := db.Where("gateway_id = ?", row.GatewayID).First(&gateway).Error; err != nil {
			issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "网关ID", Message: "网关不存在"})
			continue
		}
		if gateway.Model == tolinkmodel.TTDC {
			if row.TtdcADR == "" {
				issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "TTDC地址", Message: "TTDC网关必须填写地址"})
				continue
			}
			if _, ok := tolinkmodel.NormalizeTtdcADR(row.TtdcADR); !ok {
				issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "TTDC地址", Message: "必须是两位十六进制地址"})
				continue
			}
		}
		var equipment businessmodel.Equipment
		if err := db.Where("equipment_code = ?", row.StreamCode).First(&equipment).Error; err != nil {
			issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "电池ID", Message: "设备不存在"})
			continue
		}
		bound, err := batchEquipmentHasLiveStreamBinding(db, equipment)
		if err != nil {
			issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "电池ID", Message: err.Error()})
			continue
		}
		if bound {
			issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "电池ID", Message: "设备已绑定数据池"})
			continue
		}
		var points []streamBatchImportPoint
		if gateway.Model == tolinkmodel.TTDC {
			var modules []tolinkmodel.GatewayModuleTtdc
			if err := db.Model(&tolinkmodel.GatewayModuleTtdc{}).Preload("TtdcConfig").Preload("ModuleDatas").Where("gateway_id = ?", row.GatewayID).Find(&modules).Error; err != nil {
				issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "网关ID", Message: err.Error()})
				continue
			}
			for _, module := range modules {
				if strings.EqualFold(strings.TrimSpace(module.TtdcConfig.ModuleAdr), row.TtdcADR) {
					for _, p := range module.ModuleDatas {
						if strings.TrimSpace(p.DataId) != "" && strings.TrimSpace(p.Name) != "" {
							points = append(points, streamBatchImportPoint{Name: p.Name, DataID: p.DataId, DataType: p.DataType})
						}
					}
				}
			}
		} else {
			var gatewayPoints []tolinkmodel.GatewayData
			if err := db.Where("gateway_id = ?", row.GatewayID).Find(&gatewayPoints).Error; err != nil {
				issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "网关ID", Message: err.Error()})
				continue
			}
			for _, p := range gatewayPoints {
				if strings.TrimSpace(p.DataId) != "" && strings.TrimSpace(p.Name) != "" {
					points = append(points, streamBatchImportPoint{Name: p.Name, DataID: p.DataId, DataType: p.DataType})
				}
			}
		}
		if len(points) == 0 {
			issues = append(issues, streamBatchImportIssue{Row: row.ExcelRow, Field: "数据点", Message: "网关未找到可用数据点"})
			continue
		}
		plans = append(plans, streamBatchImportPlan{Row: row, Gateway: gateway, Equipment: equipment, Points: dedupeBatchImportPoints(points)})
	}
	return plans, issues
}

func dedupeBatchImportPoints(points []streamBatchImportPoint) []streamBatchImportPoint {
	seen := map[string]bool{}
	out := make([]streamBatchImportPoint, 0, len(points))
	for _, p := range points {
		key := p.DataID + "::" + p.Name
		if !seen[key] {
			seen[key] = true
			out = append(out, p)
		}
	}
	return out
}

func previewStreamBatchImportResults(plans []streamBatchImportPlan) []streamBatchImportResult {
	result := make([]streamBatchImportResult, 0, len(plans))
	for _, plan := range plans {
		result = append(result, streamBatchImportResult{Row: plan.Row.ExcelRow, Status: "ready", Message: fmt.Sprintf("%d个数据点", len(plan.Points))})
	}
	return result
}

func commitStreamBatchImport(ctx *gin.Context, plans []streamBatchImportPlan) (streamBatchImportResponse, error) {
	db := gormutil.GetGormDB(ctx, "")
	if db == nil {
		return streamBatchImportResponse{}, errors.New("数据库未初始化")
	}
	userInfoAny, _ := ctx.Get("userInfo")
	userInfo, _ := userInfoAny.(usermodel.User)
	admin := userInfo.TrueName
	if userInfo.Isadmin == 1 && admin == "" {
		admin = "无"
	}
	result := streamBatchImportResponse{Total: len(plans)}
	err := db.Transaction(func(tx *gorm.DB) error {
		// Lock all devices in a stable order before checking or changing bindings.
		lockOrder := make([]int, len(plans))
		for i := range lockOrder {
			lockOrder[i] = i
		}
		sort.Slice(lockOrder, func(i, j int) bool {
			return plans[lockOrder[i]].Equipment.ID < plans[lockOrder[j]].Equipment.ID
		})
		for _, planIndex := range lockOrder {
			var locked businessmodel.Equipment
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", plans[planIndex].Equipment.ID).First(&locked).Error; err != nil {
				return fmt.Errorf("第%d行设备锁定失败: %w", plans[planIndex].Row.ExcelRow, err)
			}
			bound, err := batchEquipmentHasLiveStreamBinding(tx, locked)
			if err != nil {
				return fmt.Errorf("第%d行设备绑定状态检查失败: %w", plans[planIndex].Row.ExcelRow, err)
			}
			if bound {
				return fmt.Errorf("第%d行设备已绑定数据池", plans[planIndex].Row.ExcelRow)
			}
		}
		for i := range plans {
			var exists int64
			if err := tx.Model(&tolinkmodel.Stream{}).Where("name = ? OR stream_code = ?", plans[i].Row.StreamName, plans[i].Row.StreamCode).Count(&exists).Error; err != nil {
				return err
			}
			if exists > 0 {
				return fmt.Errorf("第%d行数据池名称或电池ID已存在", plans[i].Row.ExcelRow)
			}
		}
		streams := make([]tolinkmodel.Stream, len(plans))
		for i, plan := range plans {
			streams[i] = tolinkmodel.Stream{Admin: admin, Name: plan.Row.StreamName, ApiKey: strings.ReplaceAll(uuid.NewString(), "-", ""), CreateUserId: int(userInfo.ID), StreamCode: plan.Row.StreamCode}
		}
		if err := tx.Create(&streams).Error; err != nil {
			return err
		}
		var labels []tolinkmodel.StreamLabelUnit
		var details []businessmodel.DataDetails
		for i, plan := range plans {
			var detailMap map[string]any
			if len(plan.Equipment.EquipmentDetails) > 0 {
				_ = json.Unmarshal(plan.Equipment.EquipmentDetails, &detailMap)
			}
			if detailMap == nil {
				detailMap = map[string]any{}
			}
			detailMap["streamId"] = streams[i].ID
			encoded, _ := json.Marshal(detailMap)
			if err := tx.Model(&businessmodel.Equipment{}).Where("id = ?", plan.Equipment.ID).Updates(map[string]any{"api_key": streams[i].ApiKey, "equipment_details": encoded}).Error; err != nil {
				return err
			}
			resolver := newBatchLabelResolver(plan.Gateway.Model)
			for idx, point := range plan.Points {
				labelID := strings.ReplaceAll(uuid.NewString(), "-", "")
				labelName := resolver(stripBatchTtdcPrefix(point.Name, plan.Row.TtdcADR))
				labels = append(labels, tolinkmodel.StreamLabelUnit{StreamId: streams[i].ID, LabelId: labelID, Name: labelName, GatewayName: plan.Gateway.Name, GatewayId: plan.Gateway.GatewayId, DataName: point.Name, DataId: point.DataID, Formula: "x", Unformula: "x", DataType: strconv.Itoa(normalizeBatchDataType(point.DataType))})
				details = append(details, businessmodel.DataDetails{EquipmentId: int(plan.Equipment.ID), Name: labelName, LabelId: labelID, LabelName: labelName, ContentDataDetails: []byte(`{}`), SortNum: idx + 1})
			}
		}
		if err := tx.Create(&labels).Error; err != nil {
			return err
		}
		ids := make([]int, 0, len(plans))
		for _, plan := range plans {
			ids = append(ids, int(plan.Equipment.ID))
		}
		if err := tx.Where("equipment_id IN ?", ids).Delete(&businessmodel.DataDetails{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&details).Error; err != nil {
			return err
		}
		for i := range streams {
			result.Results = append(result.Results, streamBatchImportResult{Row: plans[i].Row.ExcelRow, Status: "success", StreamID: streams[i].ID, ApiKey: streams[i].ApiKey})
		}
		return nil
	})
	if err != nil {
		return streamBatchImportResponse{}, err
	}
	result.SuccessCount = len(plans)
	return result, nil
}

func normalizeBatchDataType(value int) int {
	if value == 2 {
		return 2
	}
	return 1
}

// batchEquipmentHasLiveStreamBinding only blocks bindings that still point to
// an existing stream. Deleted streams may leave historical fields on the
// equipment row; those fields are intentionally overwritten by the import.
func batchEquipmentHasLiveStreamBinding(db *gorm.DB, equipment businessmodel.Equipment) (bool, error) {
	if db == nil {
		return false, errors.New("数据库未初始化")
	}
	apiKey := strings.TrimSpace(equipment.ApiKey)
	if apiKey != "" && apiKey != "-" {
		var count int64
		if err := db.Model(&tolinkmodel.Stream{}).Where("api_key = ?", apiKey).Count(&count).Error; err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	streamIDNumber := batchEquipmentStreamID(equipment)
	if streamIDNumber == 0 {
		return false, nil
	}
	var count int64
	if err := db.Model(&tolinkmodel.Stream{}).Where("id = ?", streamIDNumber).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func batchEquipmentStreamID(equipment businessmodel.Equipment) uint64 {
	var details map[string]any
	if len(equipment.EquipmentDetails) == 0 || json.Unmarshal(equipment.EquipmentDetails, &details) != nil {
		return 0
	}
	streamID, ok := details["streamId"]
	if !ok {
		streamID = details["streamID"]
	}
	streamIDText := strings.TrimSpace(fmt.Sprint(streamID))
	if number, ok := streamID.(float64); ok {
		streamIDText = strconv.FormatUint(uint64(number), 10)
	}
	streamIDNumber, err := strconv.ParseUint(streamIDText, 10, 64)
	if err != nil {
		return 0
	}
	return streamIDNumber
}

func stripBatchTtdcPrefix(name, adr string) string {
	name, adr = strings.TrimSpace(name), strings.TrimSpace(adr)
	if adr != "" {
		prefix := adr + "_"
		if strings.HasPrefix(strings.ToUpper(name), strings.ToUpper(prefix)) {
			return name[len(prefix):]
		}
	}
	return name
}

func newBatchLabelResolver(gatewayModel string) func(string) string {
	if !strings.EqualFold(strings.TrimSpace(gatewayModel), tolinkmodel.TTDC) {
		return func(value string) string { return strings.TrimSpace(value) }
	}
	mapping := map[string]string{
		"SOH": "SOH", "SOC": "SOC", "N(电芯温度数量)": "温度采集数",
		"1号单体SOC": "1号单体SOC", "2号单体SOC": "2号单体SOC", "3号单体SOC": "3号单体SOC", "4号单体SOC": "4号单体SOC",
		"5号单体SOC": "5号单体SOC", "6号单体SOC": "6号单体SOC", "7号单体SOC": "7号单体SOC", "8号单体SOC": "8号单体SOC",
		"9号单体SOC": "9号单体SOC", "10号单体SOC": "10号单体SOC", "11号单体SOC": "11号单体SOC", "12号单体SOC": "12号单体SOC",
		"13号单体SOC": "13号单体SOC", "14号单体SOC": "14号单体SOC", "15号单体SOC": "15号单体SOC", "16号单体SOC": "16号单体SOC",
		"压差": "压差", "温差": "温差", "MOS温度数据": "MOS温度", "热失控风险等级": "热失控风险等级",
		"M(单体电池数量)": "单体电池支数", "环境温度数据": "环境温度", "放电时长": "放电时长", "电池循环次数": "循环次数",
		"电池总压数据": "组电压", "电池电流数据": "组电流", "电池组荷电状态(SOC)": "电池组SOC", "电池组健康状态(SOH)": "电池组SOH",
		"最高单体电池电压": "最高单体电池电压", "最高电压单体编号": "最高电压单体编号", "最低单体电池电压": "最低单体电池电压",
		"最低电压单体编号": "最低电压单体编号", "最高单体电池温度": "最高单体电池温度", "最低单体电池温度": "最低单体电池温度",
		"Link电压": "Link电压", "剩余放电时长": "剩余放电时长", "剩余充电时长": "剩余充电时长", "电池额定容量": "电池额定容量",
		"年": "年", "月": "月", "日": "日", "时": "时", "分": "分", "秒": "秒", "累计充电容量": "累计充电量",
		"累计放电容量": "累计放电量", "PTC温度": "PTC温度", "软件版本号": "软件版本号",
	}
	return func(value string) string {
		key := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "（", "("), "）", ")")
		key = strings.Join(strings.Fields(key), "")
		key = strings.ReplaceAll(key, " ", "")
		if mapped := mapping[key]; mapped != "" {
			return mapped
		}
		if match := regexp.MustCompile(`^单体电池(\d+)电压$`).FindStringSubmatch(key); len(match) == 2 {
			return "电压" + match[1]
		}
		if match := regexp.MustCompile(`^电芯温度(\d+)数据$`).FindStringSubmatch(key); len(match) == 2 {
			return "温度" + match[1]
		}
		return key
	}
}
