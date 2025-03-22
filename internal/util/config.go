package util

import (
	"encoding/json"
	"os"
)

var (
	globalConfig *Config
)

type Config struct {
	System SystemConfig `json:"system"`
	Server ServerConfig `json:"server"`
}

type SystemConfig struct {
	LogDestination string `json:"logDestination"`
}

type ServerConfig struct {
	Port string `json:"Port"`
}

// ReadConfig 读取配置文件并返回微信配置
func ReadConfig() *Config {
	if globalConfig != nil {
		return globalConfig
	}

	// 读取文件内容
	bytes, err := os.ReadFile("./data/config.json")
	if err != nil {
		panic(err)
	}

	cfg := &Config{}
	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		panic(err)
	}

	globalConfig = cfg
	return globalConfig
}
