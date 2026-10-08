package schedule

import (
	"dicloud/pkg/cache"
	"time"

	"github.com/gin-gonic/gin"
)

var BUSINESS_SCHEDULE_BASE_LOCK = "business:schedule:baselock"

// 业务云定时任务锁, 避免多个定时任务; 生成重复数据
func BaseScheduleLock() bool {
	/* 默认平台数据 - multi renter(多租户) - ######################################################### */
	ctx := &gin.Context{}

	// 尝试获取锁
	ok, err := cache.GetRedisClient(ctx).SetNX(BUSINESS_SCHEDULE_BASE_LOCK, "1", 3*time.Second).Result()

	if !ok || err != nil {
		return false
	}

	return true
}
