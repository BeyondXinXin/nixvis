package web

import (
	"net/http"

	"github.com/beyondxinxin/nixvis/internal/storage"
	"github.com/gin-gonic/gin"
)

// 初始化Web路由
func SetupRoutes(router *gin.Engine, summary *storage.Summary) {
	// 初始化模板引擎
	LoadTemplates(router)

	// 首页路由 - 显示访问统计
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "NixVis - Nginx访问统计",
		})
	})

	// 静态文件服务
	router.Static("/static", "./data/static")

	router.GET("/api/stats-data", func(c *gin.Context) {
		statsData, _ := summary.GetStatsData()
		c.JSON(http.StatusOK, statsData)
	})
}
