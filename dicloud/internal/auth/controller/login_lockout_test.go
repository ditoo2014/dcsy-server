package controller

import (
	"testing"
	"time"
)

func TestNormalizeLoginSecurityConfig(t *testing.T) {
	tests := []struct {
		name                  string
		maxFailures           int
		lockMinutes           int
		failureWindowMinutes  int
		wantMaxFailures       int
		wantLockDuration      time.Duration
		wantFailWindowMinutes time.Duration
	}{
		{
			name:                  "defaults when config is empty",
			wantMaxFailures:       5,
			wantLockDuration:      15 * time.Minute,
			wantFailWindowMinutes: 15 * time.Minute,
		},
		{
			name:                  "cap max failures to five",
			maxFailures:           6,
			lockMinutes:           20,
			failureWindowMinutes:  30,
			wantMaxFailures:       5,
			wantLockDuration:      20 * time.Minute,
			wantFailWindowMinutes: 30 * time.Minute,
		},
		{
			name:                  "lock duration must be greater than ten minutes",
			maxFailures:           3,
			lockMinutes:           10,
			failureWindowMinutes:  0,
			wantMaxFailures:       3,
			wantLockDuration:      15 * time.Minute,
			wantFailWindowMinutes: 15 * time.Minute,
		},
		{
			name:                  "allow stricter failure threshold",
			maxFailures:           4,
			lockMinutes:           11,
			failureWindowMinutes:  12,
			wantMaxFailures:       4,
			wantLockDuration:      11 * time.Minute,
			wantFailWindowMinutes: 12 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeLoginSecurityConfig(tt.maxFailures, tt.lockMinutes, tt.failureWindowMinutes)
			if got.maxFailures != tt.wantMaxFailures {
				t.Fatalf("maxFailures = %d, want %d", got.maxFailures, tt.wantMaxFailures)
			}
			if got.lockDuration != tt.wantLockDuration {
				t.Fatalf("lockDuration = %s, want %s", got.lockDuration, tt.wantLockDuration)
			}
			if got.failWindow != tt.wantFailWindowMinutes {
				t.Fatalf("failWindow = %s, want %s", got.failWindow, tt.wantFailWindowMinutes)
			}
		})
	}
}

func TestBuildLoginLockMessage(t *testing.T) {
	if got := buildLoginLockMessage(10*time.Minute + time.Second); got != "账号已锁定，请11分钟后再试" {
		t.Fatalf("message = %q", got)
	}
	if got := buildLoginLockMessage(0); got != "账号已锁定，请稍后再试" {
		t.Fatalf("message = %q", got)
	}
}
