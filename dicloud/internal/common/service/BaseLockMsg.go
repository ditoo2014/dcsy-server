package service

import (
	"dicloud/internal/common/model"
	"dicloud/pkg/cache"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

var (
	LOCK_ALARM_PATTERN = "alarm:normal:lock:%d" // equipmentId

	LOCK_STATISTIC_PATTERN       = "statistic:normal:%d" // statistic:equipmentId
	LOCK_STATISTIC_FIELD_PATTERN = "field:%s:%d:%d"      // statisticCode, dataSource, valueType
	LOCK_STATISTIC_DAILY_PATTERN = "statistic:daily:lock:%d:%d:%d:%s:%s:%s"
)

const alarmLockExpiration = 30 * time.Second

const statisticDailyLockExpiration = 2 * time.Minute

type alarmLockRedisClient interface {
	SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Get(key string) *redis.StringCmd
	Del(keys ...string) *redis.IntCmd
}

var getAlarmLockRedisClient = func(ctx *gin.Context) alarmLockRedisClient {
	client := cache.GetRedisClient(ctx)
	if client == nil {
		return nil
	}
	return client
}

func LockAlarmNormalMsg(ctx *gin.Context, dataTem *model.AlarmMQModel) (string, bool) {
	client := getAlarmLockRedisClient(ctx)
	if client == nil {
		log.Println("[COMMON] alarm lock fail, redis client is nil")
		return "", false
	}

	keyTem := fmt.Sprintf(LOCK_ALARM_PATTERN, dataTem.EquipmentId)
	token := uuid.NewString()
	setRes, err := client.SetNX(keyTem, token, alarmLockExpiration).Result()
	if err != nil {
		log.Println("[COMMON] alarm lock fail: ", err)
		return "", false
	}
	if !setRes {
		return "", false
	}

	return token, true
}

func UnlockAlarmNormalMsg(ctx *gin.Context, dataTem *model.AlarmMQModel, token string) bool {
	client := getAlarmLockRedisClient(ctx)
	if client == nil || len(token) == 0 {
		return false
	}

	keyTem := fmt.Sprintf(LOCK_ALARM_PATTERN, dataTem.EquipmentId)
	currentToken, err := client.Get(keyTem).Result()
	if err != nil {
		if err != redis.Nil {
			log.Println("[COMMON] read alarm lock fail: ", err)
		}
		return false
	}
	if currentToken != token {
		return false
	}

	deleted, err := client.Del(keyTem).Result()
	if err != nil {
		log.Println("[COMMON] delete alarm lock fail: ", err)
		return false
	}
	return deleted == 1
}

func LockStatisticNormalMsg(ctx *gin.Context, dataTem *model.StatisticData) bool {
	keyTem := fmt.Sprintf(LOCK_STATISTIC_PATTERN, dataTem.EquipmentId)
	fieldTem := fmt.Sprintf(LOCK_STATISTIC_FIELD_PATTERN, dataTem.StatisticCode, dataTem.DataSourceType, dataTem.StatisticValueType)
	setRes, err := cache.GetRedisClient(ctx).HSetNX(keyTem, fieldTem, "1").Result()
	if err != nil {
		return false
	}

	return setRes
}

func UnlockStatisticNormalMsg(ctx *gin.Context, dataTem *model.StatisticData) bool {
	keyTem := fmt.Sprintf(LOCK_STATISTIC_PATTERN, dataTem.EquipmentId)
	fieldTem := fmt.Sprintf(LOCK_STATISTIC_FIELD_PATTERN, dataTem.StatisticCode, dataTem.DataSourceType, dataTem.StatisticValueType)
	setRes, err := cache.GetRedisClient(ctx).HDel(keyTem, fieldTem).Result()
	if err != nil {
		return false
	}

	return setRes == 1
}

func LockStatisticDailyData(ctx *gin.Context, dataTem *model.StatisticData, wait time.Duration) (string, bool) {
	client := cache.GetRedisClient(ctx)
	if client == nil || dataTem == nil {
		return "", false
	}
	key := statisticDailyLockKey(dataTem)
	deadline := time.Now().Add(wait)
	for {
		token := uuid.NewString()
		locked, err := client.SetNX(key, token, statisticDailyLockExpiration).Result()
		if err != nil {
			log.Println("[COMMON] statistic daily lock fail: ", err)
			return "", false
		}
		if locked {
			return token, true
		}
		if wait <= 0 || time.Now().After(deadline) {
			return "", false
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func UnlockStatisticDailyData(ctx *gin.Context, dataTem *model.StatisticData, token string) bool {
	client := cache.GetRedisClient(ctx)
	if client == nil || dataTem == nil || token == "" {
		return false
	}
	const unlockScript = `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	deleted, err := client.Eval(unlockScript, []string{statisticDailyLockKey(dataTem)}, token).Int64()
	if err != nil {
		log.Println("[COMMON] statistic daily unlock fail: ", err)
		return false
	}
	return deleted == 1
}

func statisticDailyLockKey(dataTem *model.StatisticData) string {
	return fmt.Sprintf(LOCK_STATISTIC_DAILY_PATTERN,
		dataTem.EquipmentId,
		dataTem.DataSourceType,
		dataTem.StatisticValueType,
		dataTem.StatisticDateType,
		dataTem.StatisticCode,
		dataTem.MarkDate,
	)
}
