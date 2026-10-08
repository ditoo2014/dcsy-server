package service

import (
	commonmodel "dicloud/internal/common/model"
	"sort"
	"time"
)

const (
	AlarmCategoryDischargeOvercurrent     = "放电过流"
	AlarmCategoryChargeHighTemperature    = "充电高温"
	AlarmCategoryDischargeHighTemperature = "放电高温"
)

var dashboardAlarmCodeAliases = map[string]string{
	"放电过流":   AlarmCategoryDischargeOvercurrent,
	"放电过流告警": AlarmCategoryDischargeOvercurrent,
	"放电电流告警": AlarmCategoryDischargeOvercurrent,
	"放电过流保护": AlarmCategoryDischargeOvercurrent,
	"充电高温":   AlarmCategoryChargeHighTemperature,
	"充电高温告警": AlarmCategoryChargeHighTemperature,
	"放电高温":   AlarmCategoryDischargeHighTemperature,
	"放电高温告警": AlarmCategoryDischargeHighTemperature,
}

type DailyAlarmStatistics struct {
	MarkDate                      string `json:"markDate"`
	TotalAlarmCount               int    `json:"totalAlarmCount"`
	DischargeOvercurrentCount     int    `json:"dischargeOvercurrentCount"`
	ChargeHighTemperatureCount    int    `json:"chargeHighTemperatureCount"`
	DischargeHighTemperatureCount int    `json:"dischargeHighTemperatureCount"`
}

// BuildDailyAlarmStatistics counts every alarm lifecycle on its start date.
// TotalAlarmCount includes all alarm codes, while category counts only include
// the alarm-code aliases displayed by the dashboard.
func BuildDailyAlarmStatistics(records []commonmodel.AlarmDataRecord) []DailyAlarmStatistics {
	dayStatistics := make(map[string]DailyAlarmStatistics)

	for _, record := range records {
		if record.EquipmentId <= 0 || record.BeginTime <= 0 {
			continue
		}

		markDate := time.Unix(record.BeginTime, 0).Format("2006-01-02")
		item := dayStatistics[markDate]
		item.MarkDate = markDate
		item.TotalAlarmCount++
		switch dashboardAlarmCodeAliases[record.AlarmCode] {
		case AlarmCategoryDischargeOvercurrent:
			item.DischargeOvercurrentCount++
		case AlarmCategoryChargeHighTemperature:
			item.ChargeHighTemperatureCount++
		case AlarmCategoryDischargeHighTemperature:
			item.DischargeHighTemperatureCount++
		}
		dayStatistics[markDate] = item
	}

	markDates := make([]string, 0, len(dayStatistics))
	for markDate := range dayStatistics {
		markDates = append(markDates, markDate)
	}
	sort.Strings(markDates)

	result := make([]DailyAlarmStatistics, 0, len(markDates))
	for _, markDate := range markDates {
		result = append(result, dayStatistics[markDate])
	}

	return result
}
