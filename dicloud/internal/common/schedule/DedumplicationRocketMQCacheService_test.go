package schedule

import (
	"dicloud/pkg/cache"
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDedumplicationRocketMSGResult(t *testing.T) {
	tests := []struct {
		name     string
		setRes   int64
		err      error
		expected DeduplicationRocketMQResult
	}{
		{name: "new message", setRes: 1, expected: DeduplicationNew},
		{name: "duplicate message", setRes: 0, expected: DeduplicationDuplicate},
		{name: "redis unavailable", err: errors.New("i/o timeout"), expected: DeduplicationUnavailable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			result := dedumplicationRocketMSG(func() (int64, error) {
				calls++
				return test.setRes, test.err
			}, "message-id")

			if result != test.expected {
				t.Fatalf("expected result %d, got %d", test.expected, result)
			}
			if calls != 1 {
				t.Fatalf("expected one Redis call, got %d", calls)
			}
		})
	}
}

func TestShouldHandleDedumplicationRocketMSG(t *testing.T) {
	tests := []struct {
		name           string
		result         DeduplicationRocketMQResult
		reconsumeTimes int32
		handle         bool
		retryLater     bool
	}{
		{name: "new message", result: DeduplicationNew, handle: true},
		{name: "first duplicate", result: DeduplicationDuplicate},
		{name: "redelivered duplicate", result: DeduplicationDuplicate, reconsumeTimes: 1, handle: true},
		{name: "redis unavailable", result: DeduplicationUnavailable, retryLater: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handle, retryLater := resolveRocketMessage(test.result, "message-id", test.reconsumeTimes)
			if handle != test.handle || retryLater != test.retryLater {
				t.Fatalf("expected handle=%t retryLater=%t, got handle=%t retryLater=%t", test.handle, test.retryLater, handle, retryLater)
			}
		})
	}
}

func TestDedumplicationRocketMSGWithNilRedisClient(t *testing.T) {
	previousClient := cache.RDB
	cache.RDB = nil
	t.Cleanup(func() {
		cache.RDB = previousClient
	})

	result := DedumplicationRocketMSG(&gin.Context{}, "trigger", "message-id")
	if result != DeduplicationUnavailable {
		t.Fatalf("expected Redis unavailable, got %d", result)
	}
}
