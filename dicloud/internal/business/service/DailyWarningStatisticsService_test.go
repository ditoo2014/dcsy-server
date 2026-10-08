package service

import (
	commonmodel "dicloud/internal/common/model"
	"testing"
	"time"
)

func warningStatisticTime(year int, month time.Month, day, hour int) int64 {
	return time.Date(year, month, day, hour, 0, 0, 0, time.Local).Unix()
}

func TestBuildDailyWarningStatisticsCountsEveryLifecycleByDay(t *testing.T) {
	records := []commonmodel.AlarmDataRecord{
		{EquipmentId: 1, AlarmCode: "SOC过低预警", AlarmStatus: 1, BeginTime: warningStatisticTime(2026, time.September, 7, 1)},
		{EquipmentId: 1, AlarmCode: "soc过低", AlarmStatus: 0, BeginTime: warningStatisticTime(2026, time.September, 7, 2)},
		{EquipmentId: 2, AlarmCode: "SOH过低预警", BeginTime: warningStatisticTime(2026, time.September, 7, 3)},
		{EquipmentId: 3, AlarmCode: "其他预警", BeginTime: warningStatisticTime(2026, time.September, 7, 4)},
		{EquipmentId: 3, AlarmCode: "其他预警", BeginTime: warningStatisticTime(2026, time.September, 8, 1)},
	}

	result := BuildDailyWarningStatistics(records)
	if len(result) != 2 {
		t.Fatalf("expected two daily results, got %d: %#v", len(result), result)
	}

	first := result[0]
	if first.MarkDate != "2026-09-07" || first.TotalWarningCount != 4 || first.SOCWarningCount != 2 || first.SOHWarningCount != 1 {
		t.Fatalf("unexpected first day result: %#v", first)
	}

	second := result[1]
	if second.MarkDate != "2026-09-08" || second.TotalWarningCount != 1 || second.SOCWarningCount != 0 || second.SOHWarningCount != 0 {
		t.Fatalf("unexpected second day result: %#v", second)
	}
}

func TestBuildDailyWarningStatisticsIgnoresInvalidRecords(t *testing.T) {
	result := BuildDailyWarningStatistics([]commonmodel.AlarmDataRecord{
		{EquipmentId: 0, AlarmCode: "SOC过低", BeginTime: warningStatisticTime(2026, time.September, 7, 1)},
		{EquipmentId: 1, AlarmCode: "SOH过低", BeginTime: 0},
	})
	if len(result) != 0 {
		t.Fatalf("expected no results, got %#v", result)
	}
}
