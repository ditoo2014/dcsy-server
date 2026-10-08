package service

import (
	commonmodel "dicloud/internal/common/model"
	"testing"
	"time"
)

func alarmStatisticTime(year int, month time.Month, day, hour int) int64 {
	return time.Date(year, month, day, hour, 0, 0, 0, time.Local).Unix()
}

func TestBuildDailyAlarmStatisticsCountsEveryLifecycleByDay(t *testing.T) {
	records := []commonmodel.AlarmDataRecord{
		{EquipmentId: 1, AlarmCode: "放电过流", AlarmStatus: 1, BeginTime: alarmStatisticTime(2026, time.August, 30, 1)},
		{EquipmentId: 1, AlarmCode: "放电过流保护", AlarmStatus: 0, BeginTime: alarmStatisticTime(2026, time.August, 30, 2)},
		{EquipmentId: 2, AlarmCode: "放电过流告警", BeginTime: alarmStatisticTime(2026, time.August, 30, 3)},
		{EquipmentId: 1, AlarmCode: "充电高温告警", BeginTime: alarmStatisticTime(2026, time.August, 30, 4)},
		{EquipmentId: 1, AlarmCode: "充电高温告警", BeginTime: alarmStatisticTime(2026, time.August, 30, 5)},
		{EquipmentId: 3, AlarmCode: "放电高温", BeginTime: alarmStatisticTime(2026, time.August, 30, 6)},
		{EquipmentId: 1, AlarmCode: "放电过流", BeginTime: alarmStatisticTime(2026, time.August, 31, 1)},
		{EquipmentId: 4, AlarmCode: "其他告警", BeginTime: alarmStatisticTime(2026, time.August, 31, 2)},
	}

	result := BuildDailyAlarmStatistics(records)
	if len(result) != 2 {
		t.Fatalf("expected two daily results, got %d: %#v", len(result), result)
	}

	first := result[0]
	if first.MarkDate != "2026-08-30" || first.DischargeOvercurrentCount != 3 || first.ChargeHighTemperatureCount != 2 || first.DischargeHighTemperatureCount != 1 || first.TotalAlarmCount != 6 {
		t.Fatalf("unexpected first day result: %#v", first)
	}

	second := result[1]
	if second.MarkDate != "2026-08-31" || second.DischargeOvercurrentCount != 1 || second.TotalAlarmCount != 2 {
		t.Fatalf("unexpected second day result: %#v", second)
	}
}

func TestBuildDailyAlarmStatisticsIgnoresInvalidRecords(t *testing.T) {
	result := BuildDailyAlarmStatistics([]commonmodel.AlarmDataRecord{
		{EquipmentId: 0, AlarmCode: "放电过流", BeginTime: alarmStatisticTime(2026, time.August, 30, 1)},
		{EquipmentId: 1, AlarmCode: "放电过流", BeginTime: 0},
	})
	if len(result) != 0 {
		t.Fatalf("expected no results, got %#v", result)
	}
}
