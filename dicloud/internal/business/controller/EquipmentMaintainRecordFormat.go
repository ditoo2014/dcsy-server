package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dicloud/internal/business/model"

	"github.com/xuri/excelize/v2"
)

var maintainOperationLocation = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("中国标准时间", 8*60*60)
	}
	return location
}()

// maintainEquipmentIdentifier 与列表使用同一兼容规则：有编码用编码，未编码的旧设备才使用名称。
func maintainEquipmentIdentifier(code, name string) string {
	code = strings.TrimSpace(code)
	if code != "" && code != "-" {
		return code
	}
	return strings.TrimSpace(name)
}

// validateMaintainImportHeader 严格核对原运维模板的六列；允许 Excel 产生的空尾列。
func validateMaintainImportHeader(rows [][]string) error {
	if len(rows) == 0 {
		return errors.New("导入文件不能为空")
	}
	header, ok := normalizeMaintainDataRow(rows[0])
	if !ok {
		return errors.New("运维导入表头必须包含六列")
	}
	for index, expected := range model.EquipmentMaintainRecordHeader {
		if strings.TrimSpace(header[index]) != expected {
			return fmt.Errorf("运维导入表头错误，第 %d 列应为%s", index+1, expected)
		}
	}
	return nil
}

// validateMaintainEmbeddedPictures 允许已有图片引用的导出表再次导入；无引用的嵌入图片明确提示暂不支持。
func validateMaintainEmbeddedPictures(workbook *excelize.File, sheet string, rows [][]string) error {
	for index, row := range rows {
		if index == 0 {
			continue
		}
		cell := fmt.Sprintf("D%d", index+1)
		pictures, err := workbook.GetPictures(sheet, cell)
		if err != nil {
			return fmt.Errorf("读取第 %d 行嵌入图片失败", index+1)
		}
		if len(pictures) > 0 {
			normalized, ok := normalizeMaintainDataRow(row)
			if !ok || legalMaintainImageReference(normalized[3]) == "" {
				return fmt.Errorf("第 %d 行包含嵌入图片但没有合法图片引用；请填写有效图片地址后再导入", index+1)
			}
		}
	}
	return nil
}

// normalizeMaintainDataRow 补齐 Excel 省略的空尾列，同时拒绝第七列及以后出现有效内容。
func normalizeMaintainDataRow(row []string) ([]string, bool) {
	if len(row) > len(model.EquipmentMaintainRecordHeader) {
		for _, value := range row[len(model.EquipmentMaintainRecordHeader):] {
			if strings.TrimSpace(value) != "" {
				return nil, false
			}
		}
	}
	if len(row) > len(model.EquipmentMaintainRecordHeader) {
		row = row[:len(model.EquipmentMaintainRecordHeader)]
	}
	if len(row) < len(model.EquipmentMaintainRecordHeader) {
		padded := make([]string, len(model.EquipmentMaintainRecordHeader))
		copy(padded, row)
		row = padded
	}
	return row, true
}

// readMaintainImportRows 保留其他列的显示格式，只读取运维时间列的原始数字。
// 这样电池编码的前导零不会丢失，日期也不会受 9/21/26 等本地化显示格式影响。
func readMaintainImportRows(workbook *excelize.File, sheet string) ([][]string, error) {
	rows, err := workbook.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	properties, err := workbook.GetWorkbookProps()
	if err != nil {
		return nil, err
	}
	date1904 := properties.Date1904 != nil && *properties.Date1904
	for index := 1; index < len(rows); index++ {
		if isEmptyMaintainImportRow(rows[index]) {
			continue
		}
		raw, err := workbook.GetCellValue(sheet, fmt.Sprintf("E%d", index+1), excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, fmt.Errorf("读取第 %d 行运维时间失败: %w", index+1, err)
		}
		number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			continue
		}
		timestamp, err := parseMaintainNumericTime(number, date1904)
		if err != nil {
			return nil, fmt.Errorf("第 %d 行运维时间无效: %w", index+1, err)
		}
		for len(rows[index]) < 5 {
			rows[index] = append(rows[index], "")
		}
		rows[index][4] = formatMaintainOperationTime(timestamp)
	}
	return rows, nil
}

func isEmptyMaintainImportRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

// parseMaintainNumericTime 在明确范围内区分 Unix 秒、毫秒与 Excel 天数，先校验再转换。
func parseMaintainNumericTime(number float64, date1904 bool) (int64, error) {
	if math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 {
		return 0, errors.New("时间数值必须是有效正数")
	}
	// Unix 数字只接受明确的十位及以上秒数或十三位及以上毫秒数。
	// 20260921 等紧凑日期不能被错误识别为时间戳或 Excel 天数。
	if number >= 1_000_000_000 {
		if math.Trunc(number) != number {
			return 0, errors.New("Unix 时间戳必须是整数秒或整数毫秒")
		}
		seconds := number
		if number >= 1_000_000_000_000 {
			seconds = math.Floor(number / 1000)
		}
		maximum := time.Date(10000, time.January, 1, 0, 0, 0, 0, maintainOperationLocation).Unix() - 1
		if seconds > float64(maximum) {
			return 0, errors.New("运维时间超出有效年份范围")
		}
		return int64(seconds), nil
	}
	if number >= 2_958_466 {
		return 0, errors.New("数字日期格式无效，请使用完整日期文本或有效时间戳")
	}
	date, err := excelize.ExcelDateToTime(number, date1904)
	if err != nil || date.Year() < 1970 || date.Year() > 9999 {
		return 0, errors.New("Excel 运维时间必须在 1970 至 9999 年之间")
	}
	// Excel 时间没有时区含义，按中国本地墙上时间解释，不能把 UTC 时间直接 In 转换。
	local := time.Date(date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute(), date.Second(), 0, maintainOperationLocation)
	if local.Unix() <= 0 {
		return 0, errors.New("运维时间必须晚于 Unix 起点")
	}
	return local.Unix(), nil
}

// parseMaintainOperationTime 支持 Excel 序列日期、Unix 秒和常见中文/ISO 日期文本。
func parseMaintainOperationTime(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("运维时间不能为空")
	}
	if number, err := strconv.ParseFloat(value, 64); err == nil {
		return parseMaintainNumericTime(number, false)
	}
	normalized := strings.NewReplacer("年", "-", "月", "-", "日", "", "/", "-", "T", " ").Replace(value)
	normalized = strings.TrimSpace(normalized)
	layouts := []string{
		"2006-1-2 15:04:05", "2006-1-2 15:04", "2006-1-2",
		"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02",
		time.RFC3339, "2006-01-02 15:04:05.999999999-07:00",
	}
	for _, candidate := range []string{value, normalized} {
		for _, layout := range layouts {
			var parsed time.Time
			var err error
			if strings.Contains(layout, "Z07") || strings.Contains(layout, "-07:00") {
				parsed, err = time.Parse(layout, candidate)
			} else {
				parsed, err = time.ParseInLocation(layout, candidate, maintainOperationLocation)
			}
			if err == nil && parsed.Unix() > 0 {
				return parsed.Unix(), nil
			}
		}
	}
	return 0, fmt.Errorf("无效运维时间: %s", value)
}

func formatMaintainOperationTime(timestamp int64) string {
	if timestamp <= 0 {
		return ""
	}
	return time.Unix(timestamp, 0).In(maintainOperationLocation).Format("2006-01-02 15:04:05")
}

func decodeMaintainDetails(raw json.RawMessage) map[string]any {
	result := make(map[string]any)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &result)
	}
	return result
}

// maintainDetailsTime 兼容旧数据中 create_time 为时间文本、Unix 数字或 JSON 数字字符串的情况。
func maintainDetailsTime(details map[string]any) (int64, bool) {
	value, exists := details["create_time"]
	if !exists || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case string:
		timestamp, err := parseMaintainOperationTime(typed)
		return timestamp, err == nil
	case float64:
		timestamp, err := parseMaintainOperationTime(strconv.FormatFloat(typed, 'f', -1, 64))
		return timestamp, err == nil
	case json.Number:
		timestamp, err := parseMaintainOperationTime(typed.String())
		return timestamp, err == nil
	default:
		return 0, false
	}
}

// legalMaintainImageReference 仅保留安全的相对/HTTP(S)图片引用，禁止脚本和换行注入。
func legalMaintainImageReference(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	text = strings.TrimSpace(text)
	if text == "" || strings.Contains(text, "\\") {
		return ""
	}
	for _, character := range text {
		if character < 32 || character == 127 {
			return ""
		}
	}
	parsed, err := url.Parse(text)
	if err != nil || parsed.Path == "" {
		return ""
	}
	if parsed.Scheme != "" && parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if parsed.User != nil || (parsed.Scheme != "" && parsed.Hostname() == "") || (parsed.Scheme == "" && parsed.Host != "") {
		return ""
	}
	for _, part := range strings.Split(parsed.Path, "/") {
		if part == ".." {
			return ""
		}
	}
	return text
}
