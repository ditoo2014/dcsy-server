package dao

import (
	"alarm/pkg/cache"
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

type ChargeDischargeRealtimeReading struct {
	Value     float64
	Timestamp int64
}

func LoadChargeDischargeRealtimeData(ctx *gin.Context, candidates map[string][]string) (map[string]map[string]ChargeDischargeRealtimeReading, error) {
	result := make(map[string]map[string]ChargeDischargeRealtimeReading, len(candidates))
	if len(candidates) == 0 {
		return result, nil
	}
	redisClient := cache.GetRedisClient(ctx)
	if redisClient == nil {
		return nil, errors.New("Redis client is nil")
	}
	pipeline := redisClient.Pipeline()
	defer pipeline.Close()
	commands := make(map[string]*redis.SliceCmd, len(candidates))
	for gatewayID, dataIDs := range candidates {
		commands[gatewayID] = pipeline.HMGet("xc:gateway:"+gatewayID, dataIDs...)
	}
	_, pipelineErr := pipeline.Exec()
	for gatewayID, command := range commands {
		values, err := command.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			if pipelineErr != nil && !errors.Is(pipelineErr, redis.Nil) {
				return nil, pipelineErr
			}
			return nil, err
		}
		result[gatewayID] = make(map[string]ChargeDischargeRealtimeReading)
		for index, raw := range values {
			if index >= len(candidates[gatewayID]) || raw == nil {
				continue
			}
			reading, valid := ParseChargeDischargeRealtimeReading(raw)
			if valid {
				result[gatewayID][candidates[gatewayID][index]] = reading
			}
		}
	}
	return result, nil
}

func ParseChargeDischargeRealtimeReading(raw interface{}) (ChargeDischargeRealtimeReading, bool) {
	var text string
	switch value := raw.(type) {
	case string:
		text = value
	case []byte:
		text = string(value)
	default:
		return ChargeDischargeRealtimeReading{}, false
	}
	parts := strings.Fields(text)
	if len(parts) < 3 {
		return ChargeDischargeRealtimeReading{}, false
	}
	status, err := strconv.Atoi(parts[0])
	if err != nil || status != 1 {
		return ChargeDischargeRealtimeReading{}, false
	}
	timestamp, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil || timestamp <= 0 {
		return ChargeDischargeRealtimeReading{}, false
	}
	valueText := strings.TrimSpace(strings.Join(parts[1:len(parts)-1], " "))
	value, err := strconv.ParseFloat(valueText, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return ChargeDischargeRealtimeReading{}, false
	}
	return ChargeDischargeRealtimeReading{Value: value, Timestamp: timestamp}, true
}
