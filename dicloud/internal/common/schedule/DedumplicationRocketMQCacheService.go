package schedule

import (
	"dicloud/internal/common/enum"
	"dicloud/pkg/cache"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
)

type DeduplicationRocketMQResult int

const (
	DeduplicationUnavailable DeduplicationRocketMQResult = iota
	DeduplicationNew
	DeduplicationDuplicate
)

// DedumplicationRocketMSG marks a RocketMQ message as being processed.
func DedumplicationRocketMSG(ctx *gin.Context, serverName, messageId string) DeduplicationRocketMQResult {
	redisClient := cache.GetRedisClient(ctx)
	if redisClient == nil {
		log.Println("[COMMON] dedumplication rocketmq message fail, redis client is nil, msg: ", messageId)
		return DeduplicationUnavailable
	}

	return dedumplicationRocketMSG(func() (int64, error) {
		return redisClient.ZAdd(
			fmt.Sprintf(enum.DEDUMPLICATION_ROCKETMQ_MESSAGE, serverName),
			redis.Z{
				Score:  float64(time.Now().Unix()) + 30,
				Member: messageId,
			}).Result()
	}, messageId)
}

func ShouldHandleDedumplicationRocketMSG(ctx *gin.Context, serverName, messageId string, reconsumeTimes int32) (handle, retryLater bool) {
	return resolveRocketMessage(DedumplicationRocketMSG(ctx, serverName, messageId), messageId, reconsumeTimes)
}

func resolveRocketMessage(res DeduplicationRocketMQResult, messageId string, reconsumeTimes int32) (handle, retryLater bool) {
	switch res {
	case DeduplicationNew:
		return true, false
	case DeduplicationDuplicate:
		if reconsumeTimes > 0 {
			log.Println("[COMMON] retry rocketmq message duplicate mark exists, continue consume, msg: ", messageId)
			return true, false
		}
		return false, false
	default:
		return false, true
	}
}

// RocketMQ 消息去重服务 - 取消去重
// 兼容重试消息的二次消费
func CancelDedumplicationRocketMSG(ctx *gin.Context, serverName, messageId string) bool {
	redisClient := cache.GetRedisClient(ctx)
	if redisClient == nil {
		log.Println("[COMMON] Cancel dedumplication rocketmq message fail, redis client is nil, msg: ", messageId)
		return false
	}

	setRes, err := redisClient.ZRem(
		fmt.Sprintf(enum.DEDUMPLICATION_ROCKETMQ_MESSAGE, serverName),
		messageId).
		Result()

	if err != nil {
		log.Println("[COMMON] Cancel dedumplication rocketmq message fail, err: ", err, " \nmsg: ", messageId)
		return false
	}
	if setRes == 0 {
		log.Println("[COMMON] Cancel dedumplication rocketmq message, Id: ", messageId)
	}
	return setRes == 1
}

func dedumplicationRocketMSG(add func() (int64, error), messageId string) DeduplicationRocketMQResult {
	setRes, err := add()
	if err != nil {
		log.Println("[COMMON] dedumplication rocketmq message fail, err: ", err, " \nmsg: ", messageId)
		return DeduplicationUnavailable
	}
	if setRes == 0 {
		log.Println("[COMMON] dedumplication rocketmq message, Id: ", messageId)
		return DeduplicationDuplicate
	}
	return DeduplicationNew
}
