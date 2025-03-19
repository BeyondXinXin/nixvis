package util

import (
	"fmt"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
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
	if cfg.System.LogDestination == "stdout" {
		logrus.SetOutput(os.Stdout)
	} else if cfg.System.LogDestination == "file" {
		logFile, err := os.OpenFile("./data/log.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			logrus.SetOutput(logFile)
		} else {
			panic(err)
		}
		if _, err := logrus.StandardLogger().Out.Write([]byte("\n\n\n")); err != nil {
			panic(err)
		}
		logrus.Info("Starting Application")
	}

}
