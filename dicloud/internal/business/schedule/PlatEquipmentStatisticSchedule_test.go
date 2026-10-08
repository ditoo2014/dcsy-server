package schedule

import (
	"testing"
	"time"
)

func TestShouldRunDailyStatistic(t *testing.T) {
	tests := []struct {
		name string
		hour int
		min  int
		want bool
	}{
		{name: "before window", hour: 2, min: 29, want: false},
		{name: "window start", hour: 2, min: 30, want: true},
		{name: "later same day", hour: 18, min: 0, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2026, time.August, 31, tt.hour, tt.min, 0, 0, time.Local)
			if got := shouldRunDailyStatistic(now); got != tt.want {
				t.Fatalf("shouldRunDailyStatistic(%s) = %v, want %v", now, got, tt.want)
			}
		})
	}
}
