package service

import (
	"alarm/internal/common/model"
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

type fakeAlarmLockRedis struct {
	setResult bool
	setErr    error
	getResult string
	getErr    error
	delResult int64
	delErr    error
	setKey    string
	setValue  interface{}
	setTTL    time.Duration
	getKey    string
	delKeys   []string
}

func (fake *fakeAlarmLockRedis) SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	fake.setKey = key
	fake.setValue = value
	fake.setTTL = expiration
	return redis.NewBoolResult(fake.setResult, fake.setErr)
}

func (fake *fakeAlarmLockRedis) Get(key string) *redis.StringCmd {
	fake.getKey = key
	return redis.NewStringResult(fake.getResult, fake.getErr)
}

func (fake *fakeAlarmLockRedis) Del(keys ...string) *redis.IntCmd {
	fake.delKeys = keys
	return redis.NewIntResult(fake.delResult, fake.delErr)
}

func useAlarmLockRedis(t *testing.T, client alarmLockRedisClient) {
	t.Helper()
	previous := getAlarmLockRedisClient
	getAlarmLockRedisClient = func(*gin.Context) alarmLockRedisClient { return client }
	t.Cleanup(func() { getAlarmLockRedisClient = previous })
}

func TestAlarmLockUsesTokenAndExpiration(t *testing.T) {
	fake := &fakeAlarmLockRedis{setResult: true, delResult: 1}
	useAlarmLockRedis(t, fake)
	alarm := &model.AlarmMQModel{EquipmentId: 42}

	token, locked := LockAlarmNormalMsg(&gin.Context{}, alarm)
	if !locked || token == "" {
		t.Fatalf("expected lock token, got token=%q locked=%t", token, locked)
	}
	if fake.setKey != "alarm:normal:lock:42" {
		t.Fatalf("unexpected lock key: %s", fake.setKey)
	}
	if fake.setTTL != 30*time.Second {
		t.Fatalf("expected 30 second lock, got %s", fake.setTTL)
	}
	if fake.setValue != token {
		t.Fatalf("expected Redis value to match token")
	}
	fake.getResult = token

	if !UnlockAlarmNormalMsg(&gin.Context{}, alarm, token) {
		t.Fatal("expected token owner to unlock")
	}
	if fake.getKey != fake.setKey {
		t.Fatalf("unexpected lock read key: %s", fake.getKey)
	}
	if len(fake.delKeys) != 1 || fake.delKeys[0] != fake.setKey {
		t.Fatalf("unexpected lock delete keys: %v", fake.delKeys)
	}
}

func TestAlarmLockUnavailableReturnsImmediately(t *testing.T) {
	tests := []struct {
		name   string
		client alarmLockRedisClient
	}{
		{name: "nil client"},
		{name: "lock busy", client: &fakeAlarmLockRedis{}},
		{name: "redis error", client: &fakeAlarmLockRedis{setErr: errors.New("i/o timeout")}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			useAlarmLockRedis(t, test.client)
			token, locked := LockAlarmNormalMsg(&gin.Context{}, &model.AlarmMQModel{EquipmentId: 7})
			if locked || token != "" {
				t.Fatalf("expected unavailable lock, got token=%q locked=%t", token, locked)
			}
		})
	}
}

func TestAlarmUnlockRejectsNonOwner(t *testing.T) {
	fake := &fakeAlarmLockRedis{getResult: "owner-token", delResult: 1}
	useAlarmLockRedis(t, fake)

	if UnlockAlarmNormalMsg(&gin.Context{}, &model.AlarmMQModel{EquipmentId: 9}, "other-token") {
		t.Fatal("expected non-owner unlock to fail")
	}
	if len(fake.delKeys) != 0 {
		t.Fatal("non-owner must not delete the lock")
	}
}

func TestAlarmUnlockCommandFailureLeavesLockForExpiration(t *testing.T) {
	tests := []struct {
		name   string
		client *fakeAlarmLockRedis
	}{
		{name: "get failed", client: &fakeAlarmLockRedis{getErr: errors.New("i/o timeout")}},
		{name: "delete failed", client: &fakeAlarmLockRedis{getResult: "token", delErr: errors.New("unknown command")}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			useAlarmLockRedis(t, test.client)
			if UnlockAlarmNormalMsg(&gin.Context{}, &model.AlarmMQModel{EquipmentId: 9}, "token") {
				t.Fatal("expected unlock failure")
			}
		})
	}
}
