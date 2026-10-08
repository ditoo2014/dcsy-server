package service

import (
	businessdao "dicloud/internal/business/dao"
	businessmodel "dicloud/internal/business/model"
	commonmodel "dicloud/internal/common/model"
	"testing"
	"time"
)

func TestBuildChargeDischargeDaysUsesNaturalDaysAndRealtimeToday(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 9, 1, 11, 30, 0, 0, location)
	start := time.Date(2026, 8, 26, 0, 0, 0, 0, location).Unix()
	end := now.Unix()

	days, err := buildChargeDischargeDays(start, end, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 7 {
		t.Fatalf("day count = %d, want 7", len(days))
	}
	if days[0].MarkDate != "2026-08-26" || days[0].Realtime {
		t.Fatalf("unexpected first day: %+v", days[0])
	}
	last := days[len(days)-1]
	if last.MarkDate != "2026-09-01" || !last.Realtime {
		t.Fatalf("unexpected realtime day: %+v", last)
	}
	wantCutoff := time.Date(2026, 8, 26, 23, 59, 59, 0, location).Unix()
	if days[0].Cutoff != wantCutoff {
		t.Fatalf("cutoff = %d, want %d", days[0].Cutoff, wantCutoff)
	}
}

func TestChargeDischargeCodeForPointSupportsAddressPrefixes(t *testing.T) {
	tests := []struct {
		dataID   string
		name     string
		dataName string
		want     string
	}{
		{dataID: "01_tt_fangdiandianliang", want: ChargeDischargeCodeDischarge},
		{dataID: "02_tt_chongdiandianliang", want: ChargeDischargeCodeCharge},
		{dataID: "custom", dataName: "03_累计充电容量", want: ChargeDischargeCodeCharge},
		{dataID: "01_alg_soh", name: "SOH", want: ""},
	}
	for _, test := range tests {
		if got := chargeDischargeCodeForPoint(test.dataID, test.name, test.dataName); got != test.want {
			t.Fatalf("code for %q = %q, want %q", test.dataID, got, test.want)
		}
	}
}

func TestAggregateChargeDischargeStatisticsUsesRealtimeAndHistoryCarryForward(t *testing.T) {
	days := []chargeDischargeDay{
		{MarkDate: "2026-08-31"},
		{MarkDate: "2026-09-01", Realtime: true},
	}
	equipments := []businessmodel.Equipment{{ID: 1}, {ID: 2}}
	points := map[int]map[string][]chargeDischargePoint{
		1: {
			ChargeDischargeCodeDischarge: {{EquipmentID: 1, GatewayID: "g1", DataID: "01_tt_fangdiandianliang", Code: ChargeDischargeCodeDischarge}},
			ChargeDischargeCodeCharge:    {{EquipmentID: 1, GatewayID: "g1", DataID: "01_tt_chongdiandianliang", Code: ChargeDischargeCodeCharge}},
		},
		2: {
			ChargeDischargeCodeDischarge: {{EquipmentID: 2, GatewayID: "g2", DataID: "02_tt_fangdiandianliang", Code: ChargeDischargeCodeDischarge}},
			ChargeDischargeCodeCharge:    {{EquipmentID: 2, GatewayID: "g2", DataID: "02_tt_chongdiandianliang", Code: ChargeDischargeCodeCharge}},
		},
	}
	records := map[string]commonmodel.StatisticData{}
	for _, record := range []commonmodel.StatisticData{
		{EquipmentId: 1, MarkDate: "2026-08-31", StatisticCode: ChargeDischargeCodeDischarge, StatisticValue: 10},
		{EquipmentId: 1, MarkDate: "2026-08-31", StatisticCode: ChargeDischargeCodeCharge, StatisticValue: 20},
		{EquipmentId: 2, MarkDate: "2026-08-31", StatisticCode: ChargeDischargeCodeDischarge, StatisticValue: 30},
		{EquipmentId: 2, MarkDate: "2026-08-31", StatisticCode: ChargeDischargeCodeCharge, StatisticValue: 40},
	} {
		records[chargeDischargeRecordKey(record.EquipmentId, record.MarkDate, record.StatisticCode)] = record
	}
	realtime := map[string]businessdao.ChargeDischargeRealtimeReading{
		"g1\x0001_tt_fangdiandianliang":  {Value: 11, Timestamp: 100},
		"g1\x0001_tt_chongdiandianliang": {Value: 22, Timestamp: 100},
	}

	result := aggregateChargeDischargeStatistics(days, equipments, points, records, realtime)
	if got := result.Data[0].Value; got[0] != 40 || got[1] != 41 {
		t.Fatalf("discharge values = %v, want [40 41]", got)
	}
	if got := result.Data[1].Value; got[0] != 60 || got[1] != 62 {
		t.Fatalf("charge values = %v, want [60 62]", got)
	}
	if quality := result.Quality[1]; !quality.Complete || quality.RealtimeCount != 2 || quality.CarriedCount != 2 {
		t.Fatalf("unexpected realtime quality: %+v", quality)
	}
}

func TestBuildChargeDischargeDaysRejectsMoreThanMaximum(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, location)
	start := now.AddDate(0, 0, -31).Unix()
	if _, err := buildChargeDischargeDays(start, now.Unix(), now); err == nil {
		t.Fatal("expected range limit error")
	}
}
