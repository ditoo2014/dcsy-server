package cache

import (
	"dicloud/util/redisutil"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

var (
	RDB    *redis.Client
	prefix = "xc:%s"
)

func NewRedis() {
	RDB = redisutil.RDB
}

func Set(ctx *gin.Context, key string, data string) error {
	key = fmt.Sprintf(prefix, key)
	return GetRedisClient(ctx).Set(key, data, 0).Err()
}

// SetWithExpiration atomically writes a value and its TTL.
func SetWithExpiration(ctx *gin.Context, key string, data string, expiration time.Duration) error {
	key = fmt.Sprintf(prefix, key)
	return GetRedisClient(ctx).Set(key, data, expiration).Err()
}

func Get(ctx *gin.Context, key string) (string, error) {
	key = fmt.Sprintf(prefix, key)
	result, err := GetRedisClient(ctx).Get(key).Result()
	if err != nil {
		return "", err
	}
	return result, err
}

func Incr(ctx *gin.Context, key string) (int64, error) {
	key = fmt.Sprintf(prefix, key)
	return GetRedisClient(ctx).Incr(key).Result()
}

func TTL(ctx *gin.Context, key string) (time.Duration, error) {
	key = fmt.Sprintf(prefix, key)
	return GetRedisClient(ctx).TTL(key).Result()
}

func HMSet(ctx *gin.Context, key string, data map[string]interface{}) error {

	key = fmt.Sprintf(prefix, key)
	// fmt.Println("[CACHE] cache:", key)
	return GetRedisClient(ctx).HMSet(key, data).Err()
}

func HMGet(ctx *gin.Context, key string, fields ...string) (result []interface{}) {

	var err error

	key = fmt.Sprintf(prefix, key)
	if result, err = GetRedisClient(ctx).HMGet(key, fields...).Result(); err != nil {
		result = nil
		log.Println("[Cache] HMGet get fail")
	}

	return
}

func HDel(ctx *gin.Context, key string, fields ...string) bool {

	key = fmt.Sprintf(prefix, key)

	result, err := GetRedisClient(ctx).HDel(key, fields...).Result()
	if err != nil {
		return false
	}

	return int(result) == len(fields)
}

func ExpireKey(ctx *gin.Context, key string, timeLength time.Duration) (result []interface{}) {

	var err error

	key = fmt.Sprintf(prefix, key)
	if err = GetRedisClient(ctx).Expire(key, timeLength).Err(); err != nil {
		result = nil
		log.Println("[Cache] ExpireKey get fail")
	}

	return
}

func HashGetAll(ctx *gin.Context, key string) (result map[string]string) {

	key = fmt.Sprintf(prefix, key)
	result, err := GetRedisClient(ctx).HGetAll(key).Result()
	if err != nil {
		log.Println("[Cache] HMGetAll get fail")
	}

	return
}

// unlink在PikiwiDB中没生效 - TODO: 待完善
func Unlink(ctx *gin.Context, key string) (result int64) {

	key = fmt.Sprintf(prefix, key)
	result, err := GetRedisClient(ctx).Unlink(key).Result()
	if err != nil {
		log.Println("[Cache] Unlink fail")
	}

	return
}

// Del批量操作会锁库, 尽量用unlink
func Del(ctx *gin.Context, key string) (result int64) {

	key = fmt.Sprintf(prefix, key)
	result, err := GetRedisClient(ctx).Del(key).Result()
	if err != nil {
		log.Println("[Cache] Unlink fail")
	}

	return
}

// Delete removes a key and reports Redis failures to the caller.
func Delete(ctx *gin.Context, key string) error {
	key = fmt.Sprintf(prefix, key)
	return GetRedisClient(ctx).Del(key).Err()
}
