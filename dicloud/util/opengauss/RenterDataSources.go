package opengauss

import (
	"database/sql"
	"dicloud/util/encryption"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// 租户的数据库连接
	OpengaussRenterDBs = make(map[string]*gorm.DB)
	renterDBsMu        sync.RWMutex
)

// 根据租户名, 获取租户数据源
func GetRenterDB(dbName string) *gorm.DB {
	if db := getRenterDB(dbName); db != nil {
		return db
	}
	return InitRenterDB(dbName)
}

func InitRenterDB(dbName string) *gorm.DB {
	if db := getRenterDB(dbName); db != nil {
		return db
	}

	err := InitRenterOpenGaussDB(dbName)
	if err == nil {
		return getRenterDB(dbName)
	}

	log.Println("[RenterOpenGauss] InitRentDB Fail")
	return nil
}

// 初始化ORM连接, 并保存租户
func InitRenterOpenGaussDB(dbName string) error {
	renterDBsMu.Lock()
	defer renterDBsMu.Unlock()
	if OpengaussRenterDBs[dbName] != nil {
		return nil
	}

	// 获取配置参数
	host := viper.GetString("opengauss." + dbName + ".host")
	port := viper.GetString("opengauss." + dbName + ".port")
	user := viper.GetString("opengauss." + dbName + ".user")
	pwd := viper.GetString("opengauss." + dbName + ".password")
	dbname := viper.GetString("opengauss." + dbName + ".dbname")

	var err error

	if len(pwd) == 0 {
		log.Println("[RenterOpenGauss] 请配置密码 ", "init renter "+dbName+" fail")
		return fmt.Errorf("renter OpenGauss password is not configured: %s", dbName)
	}

	// 解密加密数据
	pwd, err = encryption.DefaultPKCS8Decrypt(pwd)
	if err != nil {
		log.Println("[RenterOpenGauss] 解析错误")
		return err
	}

	log.Printf("[RenterOpenGauss] %s\n", "创建数据库链接")
	psqlInfo := fmt.Sprintf("host=%s port=%s user='%s' password='%s' dbname=%s sslmode=disable", host, port, user, pwd, dbname)
	// 注意！！！这里不要使用:=，我们是给全局变量赋值，然后在main函数中使用全局变量db
	curOGDB, err := sql.Open("opengauss", psqlInfo)
	if err != nil {
		log.Println("[RenterOpenGauss] 链接错误")
		return err
	}

	// 设置连接池大小
	curOGDB.SetConnMaxLifetime(time.Minute * 5) // 连接最大生命周期
	curOGDB.SetMaxOpenConns(50)                 // 最大连接数
	curOGDB.SetMaxIdleConns(10)                 // 最大空闲连接数

	// 尝试与数据库建立连接（校验dsn是否正确）
	err = curOGDB.Ping()
	if err != nil {
		return err
	}

	curOGGormDB, err := gorm.Open(
		postgres.New(postgres.Config{
			// 租户库与默认库使用同一 OpenGauss 驱动标识，保证自动迁移行为一致。
			DriverName: "opengauss",
			Conn:       curOGDB,
		}),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})

	if err != nil {
		log.Printf("[RenterOpenGauss] Failed to connect to database: %v", err)
		return err
	}

	// 存储不同租户的数据库连接
	OpengaussRenterDBs[dbName] = curOGGormDB

	return nil
}

func getRenterDB(dbName string) *gorm.DB {
	renterDBsMu.RLock()
	db := OpengaussRenterDBs[dbName]
	renterDBsMu.RUnlock()
	return db
}

func RenterDBNames() []string {
	renterDBsMu.RLock()
	names := make([]string, 0, len(OpengaussRenterDBs))
	for name := range OpengaussRenterDBs {
		names = append(names, name)
	}
	renterDBsMu.RUnlock()
	sort.Strings(names)
	return names
}

func ConfiguredRenterDBNames() []string {
	settings := viper.GetStringMap("opengauss")
	names := make([]string, 0)
	for name := range settings {
		if viper.GetString("opengauss."+name+".dbname") != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
