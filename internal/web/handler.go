package web

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"github.com/beyondxinxin/nixvis/internal/storage"
	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// 初始化Web路由
func SetupRoutes(router *gin.Engine, stats *storage.StatsManager) {
	// 初始化模板引擎
	loadTemplates(router)

	// 首页路由 - 显示访问统计
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "NixVis - Nginx访问统计",
		})
	})

	// 静态文件服务
	router.Static("/static", "./data/static")

	// 获取所有网站列表
	router.GET("/api/websites", func(c *gin.Context) {
		websiteIDs := util.GetAllWebsiteIDs()

		websites := make([]map[string]string, 0, len(websiteIDs))
		for _, id := range websiteIDs {
			website, ok := util.GetWebsiteByID(id)
			if !ok {
				continue
			}

			websites = append(websites, map[string]string{
				"id":   id,
				"name": website.Name,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"websites": websites,
		})
	})

	// 获取网站统计数据
	router.GET("/api/stats-data", func(c *gin.Context) {
		// 获取网站ID参数
		websiteID := c.Query("id")
		timeRange := c.Query("timeRange")
		viewType := c.Query("viewType")
		if websiteID == "" || timeRange == "" || viewType == "" {
			errStr := fmt.Sprintf("参数错误: id[%s] timeRange[%s] viewType[%s]",
				websiteID, timeRange, viewType)
			logrus.Error(errStr)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": errStr,
			})
			return
		}

		// 获取该网站的统计数据并返回
		statsResult, err := stats.StatsByWebIDAndTimeRange(websiteID, timeRange, viewType)
		if err != nil {
			errStr := fmt.Sprintf("获取统计数据失败: %v", err)
			logrus.Error(errStr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": errStr,
			})
			return
		}

		c.JSON(http.StatusOK, statsResult)
	})
}

// 加载模板文件
func loadTemplates(router *gin.Engine) {
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
}
