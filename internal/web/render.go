package web

import (
	"html/template"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// 加载模板文件
func LoadTemplates(router *gin.Engine) {
	// 添加自定义函数
	funcMap := template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"humanizeBytes": func(bytes int64) string {
			return humanize.Bytes(uint64(bytes))
		},
		"add": func(a, b int) int {
			return a + b
		},
	}

	// 配置模板
	templatesDir := "./data/templates"
	router.SetFuncMap(funcMap)

	// 加载所有模板
	templates, err := filepath.Glob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		logrus.Fatalf("无法加载模板: %v", err)
	}

	router.LoadHTMLFiles(templates...)
	logrus.Infof("已加载 %d 个模板文件", len(templates))
}
