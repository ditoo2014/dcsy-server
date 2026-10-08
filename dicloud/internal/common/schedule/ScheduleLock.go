package schedule

import (
	"dicloud/internal/common/enum"
	"dicloud/pkg/cache"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// RocketMQ消息去重, 定时任务锁
func DedumplicationRocketMQCleanScheduleLock(serverName string) bool {

	/* 默认平台数据 - multi renter(多租户) - ######################################################### */
	ctx := &gin.Context{}

	// 尝试获取锁
	keyTem := fmt.Sprintf(enum.SERVER_ROCKETMQ_DEDUMPLICATION_SCHEDULE, serverName)
	ok, err := cache.GetRedisClient(ctx).SetNX(keyTem, "1", 5*time.Second).Result()

	if !ok || err != nil {
		return false
	}

	return true
}
