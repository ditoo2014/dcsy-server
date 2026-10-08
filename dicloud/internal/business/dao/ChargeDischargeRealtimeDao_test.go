package dao

import "testing"

func TestParseChargeDischargeRealtimeReading(t *testing.T) {
	reading, valid := ParseChargeDischargeRealtimeReading("1 123.45 1788233400")
	if !valid || reading.Value != 123.45 || reading.Timestamp != 1788233400 {
		t.Fatalf("unexpected reading: %+v valid=%t", reading, valid)
	}
	for _, raw := range []interface{}{
		"0 123.45 1788233400",
		"1 -1 1788233400",
		"1 bad 1788233400",
		"1 123.45 0",
	} {
		if _, ok := ParseChargeDischargeRealtimeReading(raw); ok {
			t.Fatalf("expected invalid realtime value: %v", raw)
		}
	}
}
