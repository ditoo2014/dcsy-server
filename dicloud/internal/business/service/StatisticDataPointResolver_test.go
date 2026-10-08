package service

import (
	"dicloud/internal/tolink/model"
	"testing"
)

func TestResolveStatisticPoint(t *testing.T) {
	tests := []struct {
		name     string
		points   []model.GatewayData
		legacyID string
		want     string
	}{
		{
			name: "exact legacy ID wins",
			points: []model.GatewayData{
				{DataId: "02_tt_fangdiandianliang", Name: "02_累计放电量"},
				{DataId: "tt_fangdiandianliang", Name: "累计放电量"},
			},
			legacyID: "tt_fangdiandianliang",
			want:     "tt_fangdiandianliang",
		},
		{
			name: "address-prefixed ID",
			points: []model.GatewayData{
				{DataId: "02_tt_chongdiandianliang", Name: "02_累计充电量"},
			},
			legacyID: "tt_chongdiandianliang",
			want:     "02_tt_chongdiandianliang",
		},
		{
			name: "point name fallback",
			points: []model.GatewayData{
				{DataId: "custom_discharge_total", Name: "累计放电容量"},
			},
			legacyID: "tt_fangdiandianliang",
			want:     "custom_discharge_total",
		},
		{
			name: "address-prefixed SOH",
			points: []model.GatewayData{
				{DataId: "01_alg_soh", Name: "01_SOH"},
			},
			legacyID: "alg_soh",
			want:     "01_alg_soh",
		},
		{
			name: "unrelated point",
			points: []model.GatewayData{
				{DataId: "01_pack_voltage", Name: "组电压"},
			},
			legacyID: "tt_fangdiandianliang",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveStatisticPoint(tt.points, tt.legacyID); got != tt.want {
				t.Fatalf("resolveStatisticPoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveStatisticPointsIncludesAllAddressCandidates(t *testing.T) {
	points := []model.GatewayData{
		{DataId: "02_tt_fangdiandianliang", Name: "02_累计放电量"},
		{DataId: "01_tt_fangdiandianliang", Name: "01_累计放电量"},
	}
	want := []string{"01_tt_fangdiandianliang", "02_tt_fangdiandianliang"}
	got := resolveStatisticPoints(points, "tt_fangdiandianliang")
	if len(got) != len(want) {
		t.Fatalf("resolveStatisticPoints() length = %d, want %d: %v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("resolveStatisticPoints()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}
