package util

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// InitViperFileOnly 仅从当前工作目录下的 configs 目录读取配置，不使用环境变量覆盖。
// 认证服务使用该入口，确保服务器上的 auth.yml 是唯一配置来源。
func InitViperFileOnly(filename string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	configFile := filepath.Join(workDir, "configs", filename+".yml")
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		log.Println("配置文件读取失败", err)
		return err
	}

	return nil
}
