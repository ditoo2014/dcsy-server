package battery

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestFormatPointValue(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{name: "integer", value: 16, want: "16"},
		{name: "one decimal", value: 51.9, want: "51.9"},
		{name: "two decimals", value: -3.62, want: "-3.62"},
		{name: "three decimals", value: 3.243, want: "3.243"},
		{name: "zero", value: 0, want: "0"},
		{name: "negative zero", value: math.Copysign(0, -1), want: "0"},
		{name: "small decimal", value: 0.000001, want: "0.000001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatPointValue(tt.value); got != tt.want {
				t.Fatalf("formatPointValue(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestBuildHistoryRowsUseLegacyFourColumnTables(t *testing.T) {
	ts := time.Date(2026, 8, 7, 12, 30, 45, 0, time.Local)
	numeric := buildNumericHistoryRows([]Point{
		{DataID: "cell01", Value: 3.243, Status: 1, TS: ts},
		{DataID: "cell02", Value: 16, Status: 1, TS: ts},
	})
	wantNumeric := "('2026-08-07 12:30:45', 'cell01', 3.243, 1), ('2026-08-07 12:30:45', 'cell02', 16, 1)"
	if numeric != wantNumeric {
		t.Fatalf("numeric rows = %q, want %q", numeric, wantNumeric)
	}

	stringRows := buildStringHistoryRows([]PointStr{
		{DataID: "risk", Value: "don't charge", Status: 1, TS: ts},
	})
	wantString := "('2026-08-07 12:30:45', 'risk', 'don''t charge', 1)"
	if stringRows != wantString {
		t.Fatalf("string rows = %q, want %q", stringRows, wantString)
	}
}

func TestUpdatedFieldSet(t *testing.T) {
	fields, err := updatedFieldSet([]interface{}{"cell01", "soc"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["cell01"]; !ok {
		t.Fatal("cell01 is missing from updated fields")
	}
	if _, ok := fields["soc"]; !ok {
		t.Fatal("soc is missing from updated fields")
	}

	if _, err := updatedFieldSet(int64(2)); err == nil {
		t.Fatal("unexpected Redis result type must return an error")
	}
}

func TestShouldReplaceRealtimeValue(t *testing.T) {
	tests := []struct {
		name        string
		current     string
		hasCurrent  bool
		incoming    string
		wantReplace bool
	}{
		{name: "missing current", incoming: "1 3.2 100", wantReplace: true},
		{name: "newer timestamp", current: "1 3.2 99", hasCurrent: true, incoming: "1 3.2 100", wantReplace: true},
		{name: "same timestamp", current: "1 3.2 100", hasCurrent: true, incoming: "1 3.3 100", wantReplace: true},
		{name: "older timestamp", current: "1 3.2 100", hasCurrent: true, incoming: "1 3.3 99", wantReplace: false},
		{name: "invalid current timestamp", current: "1 3.2 now", hasCurrent: true, incoming: "1 3.3 100", wantReplace: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldReplaceRealtimeValue(tt.current, tt.hasCurrent, tt.incoming); got != tt.wantReplace {
				t.Fatalf("shouldReplaceRealtimeValue() = %t, want %t", got, tt.wantReplace)
			}
		})
	}
}

func TestIsRedisScriptUnavailable(t *testing.T) {
	if !isRedisScriptUnavailable(fmt.Errorf(`ERR unknown command "evalsha"`)) {
		t.Fatal("evalsha unknown-command error should trigger fallback")
	}
	if isRedisScriptUnavailable(fmt.Errorf("ERR wrong number of arguments for 'evalsha' command")) {
		t.Fatal("script argument errors should not trigger fallback")
	}
}

func TestBatteryTriggerMessageUsesDedicatedTagWithoutDatasource(t *testing.T) {
	message := batteryTriggerMessage("865619085125863", "01_pack_soc", "98.5", 1787537313)
	if got := string(message.Body); got != "865619085125863 01_pack_soc 98.5 1787537313" {
		t.Fatalf("battery trigger body = %q", got)
	}
	if got := message.GetTags(); got != "trigger_battery" {
		t.Fatalf("battery trigger tag = %q", got)
	}
}
