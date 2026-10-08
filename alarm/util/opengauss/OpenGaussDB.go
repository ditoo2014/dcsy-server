package opengauss

import (
	"alarm/util/encryption"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "gitee.com/opengauss/openGauss-connector-go-pq"
)

var OpengaussDB *sql.DB
var OpengaussGormDB *gorm.DB

func InitOpenGauss() error {
	// 统一从配置文件读取连接信息，禁止在代码中固化主机、账号和密码。
	return InitOpenGaussDB()
}

func InitOpenGaussDB() error {

	// 获取配置参数
	host := viper.GetString("opengauss.host")
	port := viper.GetString("opengauss.port")
	user := viper.GetString("opengauss.user")
	pwd := viper.GetString("opengauss.password")
	dbname := viper.GetString("opengauss.dbname")

	var err error

	if len(pwd) == 0 {
		log.Println("[OpenGauss] 请配置密码")
		return fmt.Errorf("OpenGauss password is not configured")
	}

	// 解密加密数据
	pwd, err = encryption.DefaultPKCS8Decrypt(pwd)
	if err != nil {
		log.Println("[OpenGauss] 解析错误")
		return err
	}

	log.Printf("[OpenGauss] %s\n", "创建数据库链接")
	psqlInfo := fmt.Sprintf("host=%s port=%s user='%s' password='%s' dbname=%s sslmode=disable", host, port, user, pwd, dbname)
	// 注意！！！这里不要使用:=，我们是给全局变量赋值，然后在main函数中使用全局变量db
	OpengaussDB, err = sql.Open("opengauss", psqlInfo)
	if err != nil {
		log.Println("[OpenGauss] 链接错误")
		return err
	}

	// 设置连接池大小
	OpengaussDB.SetConnMaxLifetime(time.Minute * 5) // 连接最大生命周期
	OpengaussDB.SetMaxOpenConns(50)                 // 最大连接数
	OpengaussDB.SetMaxIdleConns(10)                 // 最大空闲连接数

	// 尝试与数据库建立连接（校验dsn是否正确）
	err = OpengaussDB.Ping()
	if err != nil {
		return err
	}
	return nil
}

func CloseOpenGaussDB() {
	if nil != OpengaussDB {
		defer OpengaussDB.Close()
	}
}

func GetOpenGaussDB() *sql.DB {
	return OpengaussDB
}

// InitGormDB 初始化 GORM 数据库连接并返回 *gorm.DB
func InitGormDB() *gorm.DB {

	// official: https://gorm.io/zh_CN/docs/models.html
	// gorm: https://gorm.io/docs/connecting_to_the_database.html#Customize-Driver-1
	// 查询技巧: https://www.topgoer.com/%E6%95%B0%E6%8D%AE%E5%BA%93%E6%93%8D%E4%BD%9C/gorm/
	if OpengaussDB == nil {
		log.Println("[OpenGauss] database is not initialized")
		return nil
	}

	var err error
	err = OpengaussDB.Ping()
	if err != nil {
		log.Println("[OpenGauss] Gorm 请先初始化链接!")
		return nil
	}

	OpengaussGormDB, err = gorm.Open(
		postgres.New(postgres.Config{
			// 明确标识底层使用 OpenGauss 驱动，避免 GORM 迁移器误按 pgx 注入协议参数。
			DriverName: "opengauss",
			Conn:       OpengaussDB,
		}),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})

	if err != nil {
		log.Printf("[OpenGauss] Failed to connect to database: %v", err)
		return nil
	}

	log.Println("[OpenGauss] GORM database successfully connected")
	return OpengaussGormDB
}

func GetGormDB(ctx *gin.Context) *gorm.DB {
	dbNameAny, exist := ctx.Get("dbName")
	if exist && len(fmt.Sprint(dbNameAny)) > 0 {
		// 转换数据
		dbName := dbNameAny.(string)
		// 获取租户数据源
		return GetRenterDB(dbName)
	}
	// 不要在此处初始化, 各个程序获得的引用不一样; 请保证获取引用的时候, 已经完成初始化
	return OpengaussGormDB
}

// GetDefaultGormDB returns the process-wide default datasource. TTDC battery
// configuration must not follow tenant switching from an HTTP context.
func GetDefaultGormDB() *gorm.DB {
	return OpengaussGormDB
}

func Check() error {
	// 尝试与数据库建立连接（校验dsn是否正确）
	err := OpengaussDB.Ping()
	if err != nil {
		return err
	}
	return nil
}
