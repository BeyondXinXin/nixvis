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
	router.GET("/logs", func(c *gin.Context) {
		c.HTML(http.StatusOK, "logs.html", gin.H{
			"title": "NixVis - 访问日志查看",
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
		params := make(map[string]string)
		for key, values := range c.Request.URL.Query() {
			if len(values) > 0 {
				params[key] = values[0]
			}
		}

		query, err := statsFactory.BuildQueryFromRequest(statsType, params)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
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
