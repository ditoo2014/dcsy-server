package dao

import (
	"fmt"
	"testing"
	"time"
)

func TestSelectEquipmentHealthRealtimeReadingPrefersLatestPrefixedPoint(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	fields := map[string]interface{}{
		"alg_soh":          formatRealtimeSOH(99, now.Add(-time.Second)),
		"01_alg_soh":       formatRealtimeSOH(85, now.Add(-20*time.Second)),
		"02_alg_soh":       formatRealtimeSOH(70, now.Add(-5*time.Second)),
		"01_alg_soh_extra": formatRealtimeSOH(100, now),
	}

	reading, valid := selectEquipmentHealthRealtimeReading(fields)
	if !valid {
		t.Fatal("expected a valid realtime reading")
	}
	if reading.SOH != 70 || reading.Timestamp != now.Add(-5*time.Second).Unix() {
		t.Fatalf("reading=%+v, want latest prefixed SOH 70", reading)
	}
}

func TestSelectEquipmentHealthRealtimeReadingFallsBackToLegacyPoint(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	fields := map[string]interface{}{
		"01_alg_soh": fmt.Sprintf("0 70 %d", now.Unix()),
		"alg_soh":    formatRealtimeSOH(88, now.Add(-time.Second)),
	}

	reading, valid := selectEquipmentHealthRealtimeReading(fields)
	if !valid || reading.SOH != 88 {
		t.Fatalf("reading=%+v valid=%t, want legacy SOH 88", reading, valid)
	}
}

func TestParseEquipmentHealthRealtimeReadingRejectsInvalidData(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	tests := []struct {
		name string
		raw  string
	}{
		{name: "missing", raw: ""},
		{name: "bad format", raw: "1 90"},
		{name: "invalid status", raw: fmt.Sprintf("0 90 %d", now.Unix())},
		{name: "not number", raw: fmt.Sprintf("1 invalid %d", now.Unix())},
		{name: "negative", raw: fmt.Sprintf("1 -1 %d", now.Unix())},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if reading, valid := parseEquipmentHealthRealtimeReading(test.raw); valid {
				t.Fatalf("unexpected valid reading: %+v", reading)
			}
		})
	}
}

func TestParseEquipmentHealthRealtimeReadingAcceptsCachedValueRegardlessOfAge(t *testing.T) {
	timestamp := time.Unix(1_700_000_000, 0)
	reading, valid := parseEquipmentHealthRealtimeReading(formatRealtimeSOH(98, timestamp))
	if !valid || reading.SOH != 98 || reading.Timestamp != timestamp.Unix() {
		t.Fatalf("reading=%+v valid=%t", reading, valid)
	}
}

func TestClassifyEquipmentHealthSOHBoundaries(t *testing.T) {
	tests := []struct {
		soh  float64
		want string
	}{
		{soh: 90.1, want: "SOH优"},
		{soh: 90, want: "SOH良"},
		{soh: 80, want: "SOH良"},
		{soh: 79.9, want: "SOH中"},
		{soh: 60, want: "SOH中"},
		{soh: 59.9, want: "SOH差"},
	}
	for _, test := range tests {
		if got := classifyEquipmentHealthSOH(test.soh); got != test.want {
			t.Fatalf("SOH %v classified as %s, want %s", test.soh, got, test.want)
		}
	}
}

func TestEquipmentHealthSOHDataIDMatching(t *testing.T) {
	tests := map[string]bool{
		"alg_soh":          true,
		"01_alg_soh":       true,
		"0A_ALG_SOH":       true,
		"01_alg_soh_extra": false,
		"pack_soh":         false,
		"alg_soc":          false,
	}
	for dataID, want := range tests {
		if got := isEquipmentHealthSOHDataID(dataID); got != want {
			t.Fatalf("dataID %q matched=%t, want %t", dataID, got, want)
		}
	}
}

func formatRealtimeSOH(soh float64, timestamp time.Time) string {
	return fmt.Sprintf("1 %v %d", soh, timestamp.Unix())
}
