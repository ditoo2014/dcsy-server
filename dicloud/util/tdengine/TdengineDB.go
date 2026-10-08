package tdengine

import (
	"database/sql"
	"dicloud/util/encryption"
	"fmt"
	"log"

	"github.com/spf13/viper"
	_ "github.com/taosdata/driver-go/v3/taosRestful"

	_ "github.com/taosdata/driver-go/v3/taosWS"
)

var TDengineDB *sql.DB

var urlSql = "%s:%s@tcp(%s:%d)/%s"
var urlRestful = "%s:%s@http(%s:%d)/%s"
var urlWS = "%s:%s@ws(%s:%d)/%s"

var HOST string
var PORT int
var USER string
var PWD string
var DBNAME string

// 初始化TDengine DB
func InitTDengine() (*sql.DB, error) {

	/* 配置数据 */

	var err error
	HOST = viper.GetString("taos.host")
	// PORT = viper.GetString("taos.tcpport")
	PORT = viper.GetInt("taos.webport")
	USER = viper.GetString("taos.user")
	PWD = viper.GetString("taos.password")
	DBNAME = viper.GetString("taos.dbname")

	if len(PWD) == 0 {
		log.Println("[TDengine] 请配置密码")
		return nil, err
	}

	// 解密加密数据
	PWD, err := encryption.DefaultPKCS8Decrypt(PWD)
	if err != nil {
		log.Println("[TDengine] 解析错误")
		return nil, err
	}

	log.Printf("[TDengine] %s\n", "创建数据库链接")
	/* 注意！！！这里不要使用:=，我们是给全局变量赋值，然后在main函数中使用全局变量db */

	/* 使用原生连接(需要taos.h文件的), 要先安装客户端: https://docs.taosdata.com/3.3.4/develop/connect/#! */
	// sql
	// var taosDSN = fmt.Sprintf(urlSql, USER, PWD, HOST, PORT, DBNAME)
	// TDengineDB, err = sql.Open("taosSql", taosDSN)

	// restful
	var taosDSN = fmt.Sprintf(urlRestful, USER, PWD, HOST, PORT, DBNAME)
	TDengineDB, err = sql.Open("taosRestful", taosDSN)

	// web socket
	// var taosDSN = fmt.Sprintf(urlWS, USER, PWD, HOST, PORT, DBNAME)
	// TDengineDB, err = sql.Open("taosWS", taosDSN)

	// fmt.Println("[TDengine] URL:", taosDSN)
	if err != nil {
		fmt.Println("[TDengine] 链接错误", err)
		return nil, err
	}

	// 尝试与数据库建立连接（校验dsn是否正确）
	err = TDengineDB.Ping()
	if err != nil {
		return nil, err
	}

	log.Println("[TDengine] Connected to database successfully.")
	return TDengineDB, nil
}

func CloseTDengine() {
	if nil != TDengineDB {
		defer TDengineDB.Close()
	}
}

func GetTDengine() *sql.DB {
	return TDengineDB
}

func Check() {
	err := TDengineDB.Ping()
	if err != nil {
		log.Println("[TDengine] Check", err)
	}
}

// InitGormDB 初始化 GORM 数据库连接并返回 *gorm.DB
// func InitGormDB() *gorm.DB {
// 	// official: https://gorm.io/zh_CN/docs/models.html
// 	// gorm: https://gorm.io/docs/connecting_to_the_database.html#Customize-Driver-1
// 	// 查询技巧: https://www.topgoer.com/%E6%95%B0%E6%8D%AE%E5%BA%93%E6%93%8D%E4%BD%9C/gorm/
// 	var err error

// 	// var taosDSN = "root:ditoo@XIN0226@tcp(192.168.0.88:6030)/demo"
// 	TDengineGormDB, err = gorm.Open(
// 		mysql.New(mysql.Config{
// 			Conn: TDengineDB,
// 		}),
// 		&gorm.Config{
// 			Logger: logger.Default.LogMode(logger.Info),
// 		})

// 	if err != nil {
// 		log.Printf("[TDengine] Failed to connect to database: %v", err)
// 		return nil
// 	}

// 	log.Println("[TDengine] GORM database successfully connected")
// 	return TDengineGormDB
// }

// func GetGormDB() *gorm.DB {
// 	return TDengineGormDB
// }
