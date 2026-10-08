package battery

import (
	"deal/internal/batterylog"
	"deal/models/enum"
	"deal/pkg/cache"
	"deal/util/rocketutil"
	"deal/util/tdengine"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

const (
	batterySQLNumericValueTemplate = "('%s', '%s', %s, %d)"
	batterySQLStringValueTemplate  = "('%s', '%s', '%s', %d)"
	batteryProcessedMessageTTL     = 7 * 24 * time.Hour
	batteryRedisKeyPrefix          = "xc:"
)

var batteryRealtimeUpdateScript = redis.NewScript(`
local updated = {}
for i = 1, #ARGV, 2 do
    local field = ARGV[i]
    local incoming = ARGV[i + 1]
    local current = redis.call('HGET', KEYS[1], field)
    local incomingTimestamp = string.match(incoming, ' (%d+)$')
    local currentTimestamp = nil
    if current then
        currentTimestamp = string.match(current, ' (%d+)$')
    end
    if not current or not currentTimestamp or not incomingTimestamp or tonumber(incomingTimestamp) >= tonumber(currentTimestamp) then
        redis.call('HSET', KEYS[1], field, incoming)
        table.insert(updated, field)
    end
end
return updated
`)

var (
	batteryRealtimeScriptUnsupported atomic.Bool
	batteryRealtimeFallbackLogOnce   sync.Once
)

func WriteParsed(ctx *gin.Context, data ParsedData) error {
	if data.GatewayID == "" {
		return fmt.Errorf("gateway ID is empty")
	}
	debugLogf("[BATTERY-WRITE] begin gatewayID=%s numericPoints=%d stringPoints=%d\n", data.GatewayID, len(data.Points), len(data.StrPoints))
	updatedFields, err := writeRealtime(ctx, data)
	if err != nil {
		return fmt.Errorf("write Redis realtime data: %w", err)
	}
	debugLogf("[BATTERY-WRITE] Redis updated gatewayID=%s fields=%d\n", data.GatewayID, len(updatedFields))
	if err := writeHistory(ctx, data); err != nil {
		return fmt.Errorf("write TDengine history data: %w", err)
	}
	debugLogf("[BATTERY-WRITE] TDengine history written gatewayID=%s\n", data.GatewayID)
	if err := publishTriggers(ctx, data, updatedFields); err != nil {
		return fmt.Errorf("publish trigger messages: %w", err)
	}
	return nil
}

func writeRealtime(ctx *gin.Context, data ParsedData) (map[string]struct{}, error) {
	fields := make(map[string]interface{}, len(data.Points)+len(data.StrPoints))
	for _, point := range data.Points {
		fields[point.DataID] = fmt.Sprintf("%d %s %d", point.Status, formatPointValue(point.Value), point.TS.Unix())
	}
	for _, point := range data.StrPoints {
		fields[point.DataID] = fmt.Sprintf("%d %s %d", point.Status, point.Value, point.TS.Unix())
	}
	if len(fields) == 0 {
		return map[string]struct{}{}, nil
	}

	client := cache.GetRedisClient(ctx)
	if client == nil {
		return nil, fmt.Errorf("Redis client is nil")
	}
	fieldNames := make([]string, 0, len(fields))
	for field := range fields {
		fieldNames = append(fieldNames, field)
	}
	sort.Strings(fieldNames)
	args := make([]interface{}, 0, len(fieldNames)*2)
	for _, field := range fieldNames {
		args = append(args, field, fields[field])
	}
	key := batteryRedisKeyPrefix + "gateway:" + data.GatewayID
	if batteryRealtimeScriptUnsupported.Load() {
		return writeRealtimeFallback(client, key, fields, fieldNames)
	}
	result, err := batteryRealtimeUpdateScript.Run(client, []string{key}, args...).Result()
	if err != nil {
		if isRedisScriptUnavailable(err) {
			batteryRealtimeScriptUnsupported.Store(true)
			batteryRealtimeFallbackLogOnce.Do(func() {
				batterylog.Printf("[BATTERY-WRITE] Redis EVALSHA unavailable, using WATCH/HGET/HSET fallback: %v\n", err)
			})
			return writeRealtimeFallback(client, key, fields, fieldNames)
		}
		return nil, err
	}
	return updatedFieldSet(result)
}

func isRedisScriptUnavailable(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "evalsha") &&
		(strings.Contains(message, "unknown command") || strings.Contains(message, "not supported") || strings.Contains(message, "disabled"))
}

// writeRealtimeFallback preserves the script's timestamp ordering on Redis
// deployments where Lua commands are disabled by policy or unsupported.
func writeRealtimeFallback(client *redis.Client, key string, fields map[string]interface{}, fieldNames []string) (map[string]struct{}, error) {
	for attempt := 0; attempt < 3; attempt++ {
		updated := make(map[string]struct{}, len(fieldNames))
		err := client.Watch(func(tx *redis.Tx) error {
			current, err := tx.HMGet(key, fieldNames...).Result()
			if err != nil {
				return err
			}

			updates := make(map[string]interface{}, len(fieldNames))
			for index, field := range fieldNames {
				incoming := fmt.Sprint(fields[field])
				currentValue, exists := redisStringValue(current[index])
				if !shouldReplaceRealtimeValue(currentValue, exists, incoming) {
					continue
				}
				updates[field] = incoming
				updated[field] = struct{}{}
			}
			if len(updates) == 0 {
				return nil
			}

			_, err = tx.TxPipelined(func(pipe redis.Pipeliner) error {
				for field, value := range updates {
					pipe.HSet(key, field, value)
				}
				return nil
			})
			return err
		}, key)
		if err == redis.TxFailedErr {
			continue
		}
		if err != nil {
			return nil, err
		}
		return updated, nil
	}
	return nil, fmt.Errorf("Redis realtime fallback transaction failed after retries: %w", redis.TxFailedErr)
}

func redisStringValue(value interface{}) (string, bool) {
	if value == nil {
		return "", false
	}
	return fmt.Sprint(value), true
}

func shouldReplaceRealtimeValue(current string, hasCurrent bool, incoming string) bool {
	if !hasCurrent {
		return true
	}
	currentTimestamp, currentOK := realtimeValueTimestamp(current)
	incomingTimestamp, incomingOK := realtimeValueTimestamp(incoming)
	return !currentOK || !incomingOK || incomingTimestamp >= currentTimestamp
}

func realtimeValueTimestamp(value string) (int64, bool) {
	separator := strings.LastIndexByte(value, ' ')
	if separator < 0 || separator == len(value)-1 {
		return 0, false
	}
	timestamp, err := strconv.ParseInt(value[separator+1:], 10, 64)
	return timestamp, err == nil
}

func writeHistory(ctx *gin.Context, data ParsedData) error {
	if rows := buildNumericHistoryRows(data.Points); rows != "" {
		if err := sqlBatteryBatchWrite(ctx, data.GatewayID, rows, false); err != nil {
			return err
		}
	}
	if rows := buildStringHistoryRows(data.StrPoints); rows != "" {
		if err := sqlBatteryBatchWrite(ctx, data.GatewayID+"STR", rows, true); err != nil {
			return err
		}
	}
	return nil
}

func publishTriggers(ctx *gin.Context, data ParsedData, updatedFields map[string]struct{}) error {
	messages := make([]*primitive.Message, 0, len(data.Points)+len(data.StrPoints))
	for _, point := range data.Points {
		if point.Status <= 0 {
			continue
		}
		if _, updated := updatedFields[point.DataID]; !updated {
			continue
		}
		messages = append(messages, batteryTriggerMessage(data.GatewayID, point.DataID, formatPointValue(point.Value), point.TS.Unix()))
	}
	for _, point := range data.StrPoints {
		if point.Status <= 0 {
			continue
		}
		if _, updated := updatedFields[point.DataID]; !updated {
			continue
		}
		messages = append(messages, batteryTriggerMessage(data.GatewayID, point.DataID, point.Value, point.TS.Unix()))
	}
	if len(messages) == 0 {
		debugLogf("[BATTERY-TRIGGER] no trigger messages gatewayID=%s\n", data.GatewayID)
		return nil
	}
	published := rocketutil.BatchPubSync(ctx, messages)
	debugLogf("[BATTERY-TRIGGER] published gatewayID=%s topic=%s messages=%d result=%t\n", data.GatewayID, enum.XC_DEAL_TRIGGER_QUEUE, len(messages), published)
	if !published {
		return fmt.Errorf("RocketMQ batch publish returned false")
	}
	return nil
}

func batteryTriggerMessage(gatewayID, dataID, value string, timestamp int64) *primitive.Message {
	body := fmt.Sprintf("%s %s %s %d", gatewayID, dataID, value, timestamp)
	return rocketutil.FormatMsg(enum.XC_DEAL_TRIGGER_QUEUE, body, enum.TAG_TRIGGER_BATTERY, gatewayID, 0)
}

func updatedFieldSet(result interface{}) (map[string]struct{}, error) {
	values, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected Redis script result type %T", result)
	}
	fields := make(map[string]struct{}, len(values))
	for _, value := range values {
		field, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected Redis field type %T", value)
		}
		fields[field] = struct{}{}
	}
	return fields, nil
}

// formatPointValue emits the shortest non-exponential decimal representation.
// The protocol formula determines the value precision, so no extra scale setting is needed.
func formatPointValue(value float64) string {
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func buildNumericHistoryRows(points []Point) string {
	rows := make([]string, 0, len(points))
	for _, point := range points {
		rows = append(rows, fmt.Sprintf(
			batterySQLNumericValueTemplate,
			point.TS.Format("2006-01-02 15:04:05"),
			escapeSQL(point.DataID),
			formatPointValue(point.Value),
			point.Status,
		))
	}
	return strings.Join(rows, ", ")
}

func buildStringHistoryRows(points []PointStr) string {
	rows := make([]string, 0, len(points))
	for _, point := range points {
		rows = append(rows, fmt.Sprintf(
			batterySQLStringValueTemplate,
			point.TS.Format("2006-01-02 15:04:05"),
			escapeSQL(point.DataID),
			escapeSQL(point.Value),
			point.Status,
		))
	}
	return strings.Join(rows, ", ")
}

func escapeSQL(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func sqlBatteryBatchWrite(ctx *gin.Context, tableID, valuesSQL string, stringValue bool) error {
	db := tdengine.GetTDengine()
	if db == nil {
		return fmt.Errorf("TDengine db is nil")
	}

	tableName := quoteTDIdentifier(tableID)
	databaseName := quoteTDIdentifier(tdengine.DBNAME)
	result := cache.HMGet(ctx, "gateway:"+tableID, "createtable")
	if len(result) > 0 {
		if _, ok := result[0].(string); !ok {
			valueType := "float"
			if stringValue {
				valueType = "VARCHAR(255)"
			}
			createSQL := fmt.Sprintf(
				"CREATE TABLE IF NOT EXISTS %s.%s (ts TIMESTAMP, id binary(64) PRIMARY KEY, val %s, status int);",
				databaseName,
				tableName,
				valueType,
			)
			if _, err := db.Exec(createSQL); err != nil {
				batterylog.Printf("[BATTERY] create TDengine table failed, tableID=%s err=%v\n", tableID, err)
				return fmt.Errorf("create table %s: %w", tableID, err)
			}
			if err := cache.HMSet(ctx, "gateway:"+tableID, map[string]interface{}{
				"createtable": true,
			}); err != nil {
				return fmt.Errorf("cache table creation for %s: %w", tableID, err)
			}
		}
	}

	insertSQL := fmt.Sprintf("INSERT INTO %s.%s VALUES %s", databaseName, tableName, valuesSQL)
	if _, err := db.Exec(insertSQL); err != nil {
		batterylog.Printf("[BATTERY] insert TDengine table failed, tableID=%s err=%v\n", tableID, err)
		if strings.Contains(err.Error(), "Table does not exist") {
			cache.HDel(ctx, "gateway:"+tableID, "createtable")
		}
		return fmt.Errorf("insert table %s: %w", tableID, err)
	}
	debugLogf("[BATTERY-TDENGINE] insert success tableID=%s stringValue=%t\n", tableID, stringValue)
	return nil
}

func isMessageProcessed(ctx *gin.Context, messageID string) (bool, error) {
	if messageID == "" {
		return false, nil
	}
	if cache.GetRedisClient(ctx) == nil {
		return false, fmt.Errorf("Redis client is nil")
	}
	_, err := cache.Get(ctx, "battery:processed:"+messageID)
	if err == nil {
		return true, nil
	}
	if err == redis.Nil {
		return false, nil
	}
	return false, err
}

func markMessageProcessed(ctx *gin.Context, messageID string) error {
	if messageID == "" {
		return nil
	}
	if cache.GetRedisClient(ctx) == nil {
		return fmt.Errorf("Redis client is nil")
	}
	return cache.SetWithExpiration(ctx, "battery:processed:"+messageID, "1", batteryProcessedMessageTTL)
}

func quoteTDIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}
