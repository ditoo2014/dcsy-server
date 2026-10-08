package service

import (
	"dicloud/pkg/cache"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RealTimeDataHashGet 获取实时数据
func RealTimeDataHashGet(ctx *gin.Context, gatewayID string, dataId string) (status int, value string, times int64) {

	val, err := cache.GetRedisClient(ctx).HGet("xc:gateway:"+gatewayID, dataId).Result()
	if err != nil {
		return
	}

	arrStr := strings.Split(val, " ")

	if len(arrStr) < 3 {
		return 1, "-999999", time.Now().Unix()
	}
	timeStr := arrStr[len(arrStr)-1:]
	status, _ = strconv.Atoi(arrStr[0])
	times, _ = strconv.ParseInt(strings.Join(timeStr, ""), 10, 64)

	if len(arrStr[1:len(arrStr)-1]) > 1 {
		value = strings.Join(arrStr[1:len(arrStr)-1], " ")
	} else {
		value = strings.Join(arrStr[1:len(arrStr)-1], "")
	}
	return
}

// HashGetAllGtwLive 网关状态
func HashGetAllGtwLive(ctx *gin.Context, key string) (result map[string]string) {

	result, err := cache.GetRedisClient(ctx).HGetAll(key).Result()
	if err != nil {
		// panic(err)
		log.Println("Redis HGetAll操作失败", err)
		return
	}

	return
}
