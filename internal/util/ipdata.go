package util

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

//go:embed data/ip2region.xdb
var ipDataFiles embed.FS

// ExtractIPRegionDB 从嵌入的文件系统中提取 IP2Region 数据库
func ExtractIPRegionDB() (string, error) {
	// 确保数据目录存在
	if _, err := os.Stat(DataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(DataDir, 0755); err != nil {
			return "", err
		}
	}

	// 目标文件路径
	dbPath := filepath.Join(DataDir, "ip2region.xdb")

	// 检查文件是否已存在
	if _, err := os.Stat(dbPath); err == nil {
		logrus.Info("IP2Region 数据库已存在，跳过提取")
		return dbPath, nil
	}

	// 从嵌入文件系统读取数据
	data, err := fs.ReadFile(ipDataFiles, "data/ip2region.xdb")
	if err != nil {
		return "", err
	}

	// 写入文件
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		return "", err
	}

	logrus.Info("IP2Region 数据库已成功提取")
	return dbPath, nil
}
