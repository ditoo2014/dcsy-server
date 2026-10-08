package schedule

import (
	"dicloud/internal/common/enum"
	"dicloud/internal/common/service"
	"dicloud/pkg/cache"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// RocketMQ 消息去重, 定时任务, 过时删除
func DedumplicationRocketMQCleanSchedule(serverName string) {

	// 删除 - 30分钟前 到现在的数据
	curTem := time.Now().Unix()
	_, err := cache.GetRedisClient(&gin.Context{}).ZRemRangeByScore(
		fmt.Sprintf(enum.DEDUMPLICATION_ROCKETMQ_MESSAGE, serverName),
		fmt.Sprint(curTem-30*60*60),
		fmt.Sprint(curTem)).
		Result()

	if err != nil {
		log.Println("[COMMON] DedumplicationRocketMSGSchedule clean rocketmq message, err: ", err)
	}

	/* multi renter(多租户) */
	// 处理所有租户的
	dbNameDatas := service.LoadingAllPlat()
	for _, dbTem := range dbNameDatas {
		ctxTem := &gin.Context{}
		ctxTem.Set("dbName", dbTem)

		_, err := cache.GetRedisClient(ctxTem).ZRemRangeByScore(
			fmt.Sprintf(enum.DEDUMPLICATION_ROCKETMQ_MESSAGE, serverName),
			fmt.Sprint(curTem-30*60*60),
			fmt.Sprint(curTem)).
			Result()

		if err != nil {
			log.Println("[COMMON] DedumplicationRocketMSGSchedule clean rocketmq message, err: ", err)
		}
	}

}
