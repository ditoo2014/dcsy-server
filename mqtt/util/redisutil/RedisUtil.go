package redisutil

import (
	"fmt"
	"log"
	"mqtt/util/encryption"

	_redis "github.com/go-redis/redis"
	"github.com/spf13/viper"
)

// new redis util
// 声明一个全局的rdb变量
var RDB *_redis.Client
var status bool

// 初始化连接
func InitialRedis() (err error) {

	// 获取配置参数
	host := viper.GetString("redis.host")
	port := viper.GetString("redis.port")
	pwd := viper.GetString("redis.password")

	if len(pwd) == 0 {
		fmt.Println("[RedisUtil] 请配置密码")
		return fmt.Errorf("Redis 密码未配置")
	}

	// 解密加密数据
	pwd, err = encryption.DefaultPKCS8Decrypt(pwd)
	if err != nil {
		fmt.Println("[RedisUtil] 解析错误")
		return err
	}

	// 初始化配置
	log.Printf("[RedisUtil] %s\n", "创建数据库链接")
	RDB = _redis.NewClient(&_redis.Options{
		Addr:     host + ":" + port,
		Password: pwd,
		DB:       0,
	})

	_, err = RDB.Ping().Result()
	if err != nil {
		status = false
		_ = RDB.Close()
		RDB = nil
		return err
	}
	status = true

	log.Println("[RedisUtil] redis successfully connected")
	return nil
}

func Check() (result bool, err error) {

	if _, err = RDB.Ping().Result(); err != nil {
		status = false
		log.Println("redis check fail", err)
	} else {
		status = true
		result = true
	}

	return
}

// 初始化连接, 并返回
func InitRenterRedisDBByName(dbName string) (curDB *_redis.Client, err error) {

	// 获取配置参数
	host := viper.GetString("redis." + dbName + ".host")
	port := viper.GetString("redis." + dbName + ".port")
	pwd := viper.GetString("redis." + dbName + ".password")

	if len(pwd) == 0 {
		fmt.Println("[RedisUtil] InitRentRedisDBByName 请配置密码")
		return nil, fmt.Errorf("租户 Redis 密码未配置")
	}

	// 解密加密数据
	pwd, err = encryption.DefaultPKCS8Decrypt(pwd)
	if err != nil {
		fmt.Println("[RedisUtil] InitRentRedisDBByName 解析错误")
		return nil, err
	}

	// 初始化配置
	log.Printf("[RedisUtil] InitRentRedisDBByName %s\n", "创建数据库链接")
	curDB = _redis.NewClient(&_redis.Options{
		Addr:     host + ":" + port,
		Password: pwd,
		DB:       0,
	})

	_, err = curDB.Ping().Result()
	if err != nil {
		_ = curDB.Close()
		return nil, err
	}

	return curDB, nil
}
