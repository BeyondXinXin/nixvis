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
func SetupRoutes(
	router *gin.Engine,
	statsFactory *storage.StatsFactory) {

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

	// 查询接口
	router.GET("/api/stats/:type", func(c *gin.Context) {
		statsType := c.Param("type")
		websiteID := c.Query("id")
		timeRange := c.Query("timeRange")

		// 构建查询对象
		query := storage.StatsQuery{
			WebsiteID:  websiteID,
			TimeRange:  timeRange,
			ViewType:   c.DefaultQuery("viewType", ""),
			ExtraParam: make(map[string]interface{}),
		}

		// 检查必要参数
		if websiteID == "" || timeRange == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "缺少必要参数: id 和 timeRange",
			})
			return
		}

		// 根据统计类型设置额外参数
		switch statsType {
		case "url":
		case "referer":
		case "browser":
		case "os":
		case "device":
			if limitStr := c.Query("limit"); limitStr != "" {
				limit := 10 // 默认值
				if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil {
					query.ExtraParam["limit"] = limit
				}
			} else {
				query.ExtraParam["limit"] = 10
			}
		case "timeseries":
			if query.ViewType == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "时间序列查询需要 viewType 参数",
				})
				return
			}
		case "overall":
			// 总体统计不需要额外参数
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("不支持的统计类型: %s", statsType),
			})
			return
		}

		// 执行查询
		result, err := statsFactory.QueryStats(statsType, query)
		if err != nil {
			logrus.WithError(err).Errorf("查询统计数据[%s]失败", statsType)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("查询失败: %v", err),
			})
			return
		}

		c.JSON(http.StatusOK, result)
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
