package cache

import (
	"deal/util/redisutil"
	"fmt"
	"log"
	"sync"

	"github.com/gin-gonic/gin"
	_redis "github.com/go-redis/redis"
)

var (
	// 租户的数据库连接
	RedisRentDBs = make(map[string]*_redis.Client)
	renterDBMu   sync.RWMutex
)

// 根据租户名, 获取租户数据源
func GetRenterDB(dbName string) *_redis.Client {

	// 从既有数据中获取连接
	renterDBMu.RLock()
	curDB := RedisRentDBs[dbName]
	renterDBMu.RUnlock()
	if curDB != nil {
		return curDB
	}
	return InitRenterDB(dbName)
}

func InitRenterDB(dbName string) *_redis.Client {
	renterDBMu.Lock()
	defer renterDBMu.Unlock()

	if RedisRentDBs[dbName] != nil {
		return RedisRentDBs[dbName]
	}

	curDB, err := redisutil.InitRenterRedisDBByName(dbName)
	if err == nil {
		RedisRentDBs[dbName] = curDB
		return curDB
	}

	log.Println("[RedisCache] InitRentDB Fail")
	return nil
}

func GetRedisClient(ctx *gin.Context) *_redis.Client {
	dbNameAny, exist := ctx.Get("dbName")
	if exist && len(fmt.Sprint(dbNameAny)) > 0 {
		// 转换数据
		dbName := dbNameAny.(string)
		// 获取租户数据源
		return GetRenterDB(dbName)
	}

	return RDB
}
