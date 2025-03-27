package util

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

const (
	DataDir    = "./nixvis_data"
	ConfigFile = "./nixvis_config.json"
)

// HandleAppConfig 处理应用程序配置初始化和命令行参数
func HandleAppConfig() bool {
	// 命令行参数
	genConfig := flag.Bool("gen-config", false, "生成配置文件并退出")
	flag.Parse()

	// 初始化目录
	dirs := []string{
		DataDir,
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "初始化目录失败: %v\n", err)
				return true
			}
		}
	}

	// 检查配置文件是否存在
	_, err := os.Stat(ConfigFile)
	configExists := err == nil

	// 生成配置文件
	if *genConfig || !configExists {
		err := writeDefaultConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成配置文件失败: %v\n", err)
		} else {
			fmt.Println("配置文件已生成: " + ConfigFile)
			fmt.Println("请编辑配置文件后再启动服务")
		}
		return true
	}

	// 验证配置文件是否完整有效
	if err := validateConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "配置文件验证失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "请修正配置问题后重新启动服务\n")
		return true
	}

	// 不需要退出，继续运行
	return false
}

// writeDefaultConfig 写入默认配置
func writeDefaultConfig() error {
	// 默认配置内容
	configJson := `{
  "system": {
    "logDestination": "file"
  },
  "server": {
    "Port": ":8088"
  },
  "websites": [
    {
      "name": "示例网站",
      "logPath": "./weblog_eg/example.log"
    }
  ]
}`
	return os.WriteFile(ConfigFile, []byte(configJson), 0644)
}

// validateConfig 验证配置文件是否完整有效
func validateConfig() error {
	// 读取配置
	cfg, err := ReadRawConfig()
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 检查是否至少有一个网站配置
	if len(cfg.Websites) == 0 {
		return fmt.Errorf("配置文件缺少网站配置，至少需要配置一个网站")
	}

	// 检查每个日志文件是否存在
	var missingLogs []string
	for _, site := range cfg.Websites {
		if site.LogPath == "" {
			missingLogs = append(missingLogs, fmt.Sprintf("'%s' (缺少日志文件路径配置)", site.Name))
			continue
		}

		// 检查日志文件是否存在
		if _, err := os.Stat(site.LogPath); os.IsNotExist(err) {
			missingLogs = append(missingLogs, fmt.Sprintf("'%s' (%s)", site.Name, site.LogPath))
		}
	}

	// 如果有缺失的日志文件，返回错误
	if len(missingLogs) > 0 {
		errMsg := "以下网站的日志文件不存在:\n"
		for _, missing := range missingLogs {
			errMsg += " - " + missing + "\n"
		}
		return errors.New(errMsg)
	}

	return nil
}
