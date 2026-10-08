package util

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

func InitViper(filename string) error {

	// 获取工作路径
	workDir, _ := os.Getwd()

	// 设置配置文件名
	viper.SetConfigName(filename)
	viper.SetConfigType("yml")
	viper.AddConfigPath(workDir + "/configs")

	// 环境变量优先于文件配置，格式为“服务名_配置路径”。
	// 例如 MQTT_MQTT_PASSWORD、DEAL_OPENGAUSS_PASSWORD。
	viper.SetEnvPrefix(strings.ToUpper(filename))
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 启用配置
	err := viper.ReadInConfig()
	if err != nil {
		log.Println("配置文件读取失败", err)
		return err
	}

	return nil
}
