package web

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/beyondxinxin/nixvis/internal/stats"
	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// 初始化Web路由
func SetupRoutes(
	router *gin.Engine,
	statsFactory *stats.StatsFactory) {

	// 加载模板
	tmpl, err := LoadTemplates()
	if err != nil {
		logrus.Fatalf("无法加载模板: %v", err)
	}
	router.SetHTMLTemplate(tmpl)

	// 设置静态文件服务
	staticFS, err := GetStaticFS()
	if err != nil {
		logrus.Fatalf("无法加载静态文件: %v", err)
	}

	router.StaticFS("/static", staticFS)
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "NixVis - Nginx访问统计",
		})
	})
	router.GET("/favicon.ico", func(c *gin.Context) {
		data, err := fs.ReadFile(staticFiles, "assets/static/favicon.ico")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "image/x-icon", data)
	})

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
		query := stats.StatsQuery{
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
		case "url", "referer", "browser", "os", "device":
			if limitStr := c.Query("limit"); limitStr != "" {
				limit := 10 // 默认值
				if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil {
					query.ExtraParam["limit"] = limit
				}
			} else {
				query.ExtraParam["limit"] = 10
			}
		case "location":
			if limitStr := c.Query("limit"); limitStr != "" {
				limit := 99 // 默认值
				if _, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil {
					query.ExtraParam["limit"] = limit
				}
			} else {
				query.ExtraParam["limit"] = 99
			}

			if locationType := c.Query("locationType"); locationType != "" {
				query.ExtraParam["locationType"] = locationType
			} else {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "缺少必要参数: locationType",
				})
				return
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
