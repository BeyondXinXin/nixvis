package util

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"sync"
)

var (
	globalConfig *Config
	websiteIDMap sync.Map
)

type Config struct {
	System   SystemConfig    `json:"system"`
	Server   ServerConfig    `json:"server"`
	Websites []WebsiteConfig `json:"websites"`
	PVFilter PVFilterConfig  `json:"pvFilter"`
}

type WebsiteConfig struct {
	Name    string `json:"name"`
	LogPath string `json:"logPath"`
}

type SystemConfig struct {
	LogDestination string `json:"logDestination"`
}

type ServerConfig struct {
	Port string `json:"Port"`
}

type PVFilterConfig struct {
	StatusCodeInclude []int    `json:"statusCodeInclude"`
	ExcludePatterns   []string `json:"excludePatterns"`
}

// ReadRawConfig 读取配置文件但不初始化全局变量
func ReadRawConfig() (*Config, error) {
	// 读取文件内容
	bytes, err := os.ReadFile(ConfigFile)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// ReadConfig 读取配置文件并返回配置，同时初始化 ID 映射
func ReadConfig() *Config {
	if globalConfig != nil {
		return globalConfig
	}

	// 读取文件内容
	bytes, err := os.ReadFile(ConfigFile)
	if err != nil {
		panic(err)
	}

	cfg := &Config{}
	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		panic(err)
	}

	// 初始化 ID 映射
	for _, website := range cfg.Websites {
		id := GenerateID(website.LogPath)
		websiteIDMap.Store(id, website) // 以 ID 为键存储 WebsiteConfig
	}

	globalConfig = cfg
	return globalConfig
}

// GenerateID 根据输入字符串生成唯一 ID
func GenerateID(input string) string {
	hash := md5.Sum([]byte(input))
	return hex.EncodeToString(hash[:2])
}

// GetWebsiteByID 根据 ID 获取对应的 WebsiteConfig
func GetWebsiteByID(id string) (WebsiteConfig, bool) {
	value, ok := websiteIDMap.Load(id)
	if ok {
		return value.(WebsiteConfig), true
	}
	return WebsiteConfig{}, false
}

// GetAllWebsiteIDs 获取所有网站的 ID 列表
func GetAllWebsiteIDs() []string {
	var ids []string
	websiteIDMap.Range(func(key, value interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})
	return ids
}
