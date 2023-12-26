package util

import (
	"log"
	"path"
	"runtime"
	"strings"

	"github.com/spf13/viper"
)

var (
	RootPath string //项目根目录
)

func init() {
	RootPath = path.Dir(GetCurrentPath()+"..") + "/" //项目根目录
}

// 获取当前函数所在go代码的路径
func GetCurrentPath() string {
	_, filename, _, _ := runtime.Caller(1) //1表示当前函数，2表示调用本函数的函数，3...依次类推
	return path.Dir(filename)
}

func CreateConfigReader(fileName string) *viper.Viper {
	config := viper.New()
	config.AddConfigPath(RootPath + "config/") // 文件所在目录
	arr := strings.Split(fileName, ".")
	if len(arr) != 2 {
		log.Fatalf("fileName must have two parts which splited by dot")
	}
	configFile, configType := arr[0], arr[1]
	config.SetConfigName(configFile) // 文件名
	config.SetConfigType(configType) // 文件类型
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("could not found config file %s", fileName)
		} else {
			log.Fatalf("parse config file failed: %s", err.Error())
		}
	}
	return config
}
