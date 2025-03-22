package web

import (
	"net/http"
	"time"

	"github.com/beyondxinxin/nixvis/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// 初始化Web路由
func SetupRoutes(router *gin.Engine, summary *storage.Summary) {
	// 初始化模板引擎
	LoadTemplates(router)
	buildTime := time.Now().Unix()

	// 首页路由 - 显示访问统计
	router.GET("/", func(c *gin.Context) {
		statsData, err := summary.GetStatsData()
		if err != nil {
			logrus.Errorf("获取统计数据失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计数据失败"})
			return
		}

		now := time.Now()
		today := now.Format("2006-01-02")

		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":     "NixVis - Nginx访问统计",
			"statsData": statsData,
			"today":     today,
			"buildTime": buildTime,
		})
	})

	// 静态文件服务
	router.Static("/static", "./data/static")
}
