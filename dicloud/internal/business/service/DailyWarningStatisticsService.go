package service

import (
	commonmodel "dicloud/internal/common/model"
	"sort"
	"strings"
	"time"
)

const (
	WarningCategorySOC = "SOC"
	WarningCategorySOH = "SOH"
)

// matchWarningCategory checks if an alarm code belongs to a warning category.
func matchWarningCategory(alarmCode string) string {
	upper := strings.ToUpper(strings.TrimSpace(alarmCode))
	if strings.Contains(upper, WarningCategorySOC) {
		return WarningCategorySOC
	}
	if strings.Contains(upper, WarningCategorySOH) {
		return WarningCategorySOH
	}
	return ""
}

type DailyWarningStatistics struct {
	MarkDate          string `json:"markDate"`
	TotalWarningCount int    `json:"totalWarningCount"`
	SOCWarningCount   int    `json:"socWarningCount"`
	SOHWarningCount   int    `json:"sohWarningCount"`
}

// BuildDailyWarningStatistics counts every warning lifecycle on its start date.
// TotalWarningCount includes all warning codes, while category counts only
// include SOC- and SOH-related warnings displayed by the dashboard.
func BuildDailyWarningStatistics(records []commonmodel.AlarmDataRecord) []DailyWarningStatistics {
	dayStatistics := make(map[string]DailyWarningStatistics)

	for _, record := range records {
		if record.EquipmentId <= 0 || record.BeginTime <= 0 {
			continue
		}

		markDate := time.Unix(record.BeginTime, 0).Format("2006-01-02")
		item := dayStatistics[markDate]
		item.MarkDate = markDate
		item.TotalWarningCount++
		switch matchWarningCategory(record.AlarmCode) {
		case WarningCategorySOC:
			item.SOCWarningCount++
		case WarningCategorySOH:
			item.SOHWarningCount++
		}
		dayStatistics[markDate] = item
	}

	markDates := make([]string, 0, len(dayStatistics))
	for markDate := range dayStatistics {
		markDates = append(markDates, markDate)
	}
	sort.Strings(markDates)

	result := make([]DailyWarningStatistics, 0, len(markDates))
	for _, markDate := range markDates {
		result = append(result, dayStatistics[markDate])
	}

	return result
}
