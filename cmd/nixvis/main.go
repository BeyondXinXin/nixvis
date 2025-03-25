package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beyondxinxin/nixvis/internal/storage"
	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/beyondxinxin/nixvis/internal/web"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/sirupsen/logrus"
)

func main() {
	// 初始化配置
	util.InitDir()
	util.ConfigureLogging()
	cfg := util.ReadConfig()
	logrus.Info("------ 服务启动成功 ------")
	defer logrus.Info("------ 服务已安全关闭 ------")

	logrus.Info("****** 初始化数据 ******")
	repository, err := storage.NewRepository()
	if err != nil {
		logrus.WithField("error", err).Error("Failed to create database file")
		return
	}
	if err := repository.Init(); err != nil {
		logrus.WithField("error", err).Error("Failed to create tables")
		return
	}
	logParser := storage.NewLogParser(repository)
	stats := storage.NewStatsManager(repository)
	urlStatsMgr := storage.NewURLStatsManager(repository)

	logrus.Info("****** 初始扫描 ******")
	executePeriodicTasks(logParser, stats, urlStatsMgr)

	// 测试查询
	websiteIDs := util.GetAllWebsiteIDs()
	for _, id := range websiteIDs {

		timeRangeList := []string{"today", "last7days", "last30days"}
		for _, timeRange := range timeRangeList {
			queryStart := time.Now()

			// 获取统计数据
			urlStats, _ := urlStatsMgr.GetTopURLs(id, timeRange, 30)

			// 构建格式化的URL统计信息
			// var formattedStats []string
			// for i, stat := range urlStats {
			// 	formattedStats = append(formattedStats,
			// 		fmt.Sprintf("\n  [%d] URL: %s, PV: %d, UV: %d, Traffic: %d bytes",
			// 			i+1, stat.URL, stat.PV, stat.UV, stat.Traffic))
			// }

			queryDuration := time.Since(queryStart)
			// 合并打印
			logrus.WithFields(logrus.Fields{
				"websiteID": id,
				"timeRange": timeRange,
				"count":     len(urlStats),
				"duration":  queryDuration,
			}).Infof("")
		}
	}

	logrus.Info("****** 启动HTTP服务器 ******")
	r := setupCORS(stats)
	srv := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			logrus.WithField("error", err).Error("Failed to start the server")
		}
	}()
	logrus.Info("服务器初始化成功")

	logrus.Info("****** 启动维护任务 ******")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runPeriodicTaskScheduler(ctx, logParser, stats, urlStatsMgr)

	// 等待程序退出
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	<-shutdownSignal

	logrus.Info("****** 开始关闭服务器 ******")

	// 取消上下文将会通知所有后台任务退出
	logrus.Info("停止维护任务")
	cancel()

	// 给后台任务一些时间来完成清理
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(), 3*time.Second)
	defer shutdownCancel()
	<-shutdownCtx.Done()

	logrus.Info("关闭数据库")
	repository.Close()

}

// setupCORS 配置跨域中间件
func setupCORS(stats *storage.StatsManager) *gin.Engine {
	gin.DefaultWriter = logrus.StandardLogger().Writer()
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 设置Web路由
	web.SetupRoutes(r, stats)

	return r
}

// runPeriodicTaskScheduler 运行周期性任务（每5分钟）
func runPeriodicTaskScheduler(
	ctx context.Context,
	parser *storage.LogParser,
	stats *storage.StatsManager,
	urlStatsMgr *storage.URLStatsManager) {

	// 定时扫描 - 每5分钟一次
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logrus.Info("****** 开始执行维护任务 ******")
			executePeriodicTasks(parser, stats, urlStatsMgr)
		case <-ctx.Done():
			return
		}
	}
}

// executePeriodicTasks 执行周期性任务
func executePeriodicTasks(
	parser *storage.LogParser,
	stats *storage.StatsManager,
	urlStatsMgr *storage.URLStatsManager) {

	logrus.Info("开始扫描Nginx日志")
	parser.ScanNginxLogs()
	logrus.Info("Nginx日志扫描完成")

	// logrus.Info("2. 开始更新统计数据")
	// stats.UpdateStats()
	// logrus.Info("统计数据更新完成")

	// logrus.Info("3. 开始更新URL访问统计数据")
	// urlStatsMgr.UpdateURLStats()
	// logrus.Info("URL访问统计数据更新完成")
}
