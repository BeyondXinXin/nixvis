package util

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	LogFileName       = "nixvis.log"
	LogBackupFileName = "nixvis_backup.log"
	LogMaxSize        = 5 * 1024 * 1024 // 5MB
)

var (
	logFileHandle *os.File
	logMutex      sync.Mutex
)

type CustomFormatter struct{}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	level := entry.Level.String()
	message := entry.Message
	logLine := fmt.Sprintf("%s %s %s", timestamp, level, message)
	if len(entry.Data) > 0 {
		// 获取所有键并排序
		keys := make([]string, 0, len(entry.Data))
		for k := range entry.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// 按排序后的顺序打印字段
		for _, k := range keys {
			logLine += fmt.Sprintf(" [%s=%v]", k, entry.Data[k])
		}
	}
	logLine += "\n"

	return []byte(logLine), nil
}

// configureLogging 配置日志
func ConfigureLogging() {
	logrus.SetFormatter(&CustomFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	cfg := ReadConfig()

	switch cfg.System.LogDestination {
	case "stdout":
		logrus.SetOutput(os.Stdout)
	case "file":
		logMutex.Lock()
		defer logMutex.Unlock()

		logPath := filepath.Join(DataDir, LogFileName)
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			logrus.SetOutput(os.Stdout)
			logrus.WithError(err).Error("无法打开日志文件,降级到stdout输出")
			return
		}
		logFileHandle = logFile
		logFile.WriteString("\n\n\n")
		logrus.SetOutput(logFile)
		logrus.Info("日志系统已初始化")
	default:
		logrus.SetOutput(os.Stdout)
	}
}

// RotateLogFile 轮转日志文件
func RotateLogFile() error {
	cfg := ReadConfig()
	if cfg.System.LogDestination != "file" {
		return nil
	}

	logPath := filepath.Join(DataDir, LogFileName)
	info, err := os.Stat(logPath)
	if err != nil || info.Size() <= LogMaxSize {
		return nil
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	if logFileHandle != nil {
		logFileHandle.Close()
		logFileHandle = nil
	}

	backupPath := filepath.Join(DataDir, LogBackupFileName)
	os.Remove(backupPath)
	renameErr := os.Rename(logPath, backupPath)

	logFileHandle, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	logrus.SetOutput(logFileHandle)

	if renameErr != nil {
		logrus.WithError(renameErr).Warn("日志轮转失败但继续使用原文件")
	} else {
		logrus.Info("日志文件已轮转")
	}

	return nil
}

// CloseLogFile 关闭日志文件
func CloseLogFile() {
	logMutex.Lock()
	defer logMutex.Unlock()

	if logFileHandle != nil {
		logFileHandle.Close()
		logFileHandle = nil
	}
}
