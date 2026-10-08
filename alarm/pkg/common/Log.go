package common

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// logo
const Logo = `
* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *

	 ______      _____     ____   _____         ____     __    __   ______    
	(_  __ \    (_   _)   / ___) (_   _)       / __ \    ) )  ( (  (_  __ \   
	  ) ) \ \     | |    / /       | |        / /  \ \  ( (    ) )   ) ) \ \  
	 ( (   ) )    | |   ( (        | |       ( ()  () )  ) )  ( (   ( (   ) ) 
	  ) )  ) )    | |   ( (        | |   __  ( ()  () ) ( (    ) )   ) )  ) ) 
	 / /__/ /    _| |__  \ \___  __| |___) )  \ \__/ /   ) \__/ (   / /__/ /  
	(______/    /_____(   \____) \________/    \____/    \______/  (______/   


* * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * * *
`

func InitLog(logfile string) {

	path := viper.GetString(logfile)
	if path == "" {
		path = filepath.Join("logs", "service.log")
	}
	// 日志目录随项目配置自动创建，避免依赖固定磁盘路径。
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		panic(err)
	}

	logFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
		panic(err)
	}

	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)

	fmt.Println("\x1b[31m" + Logo + "\x1b[0m")
	log.Println("\x1b[31m" + Logo + "\x1b[0m")

}

func LogErr(msg string, err error) {

	log.Printf("[%s] %s\n", msg, err)
}

func LogError(msg string) {

	log.Printf("[%s] \n", msg)
}

// add 0912.
func LogInfo(msg string, data interface{}) {

	log.Printf("[%s] %s\n", msg, data)
}
