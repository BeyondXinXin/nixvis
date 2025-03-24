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
	logParser := storage.NewNginxLogParser(repository)
	summary := storage.NewSummary(repository)

	logrus.Info("****** 初始扫描 ******")
	performMaintenance(logParser, summary)

	logrus.Info("****** 启动HTTP服务器 ******")
	r := setupCORS(summary)
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
	go runMaintenanceScheduler(ctx, logParser, summary)

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

// runMaintenanceScheduler 定时任务调度器
func runMaintenanceScheduler(
	ctx context.Context,
	parser *storage.NginxLogParser,
	summary *storage.Summary) {

	// 定时扫描 - 每5分钟一次
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logrus.Info("****** 开始执行维护任务 ******")
			performMaintenance(parser, summary)
		case <-ctx.Done():
			return
		}
	}
}

// performMaintenance 执行维护任务
func performMaintenance(
	parser *storage.NginxLogParser,
	summary *storage.Summary) {

	logrus.Info("1. 开始扫描Nginx日志")
	parser.ScanNginxLogs()
	logrus.Info("1. Nginx日志扫描完成")

	logrus.Info("2. 开始更新统计数据")
	summary.UpdateStats()
	logrus.Info("2. 统计数据更新完成")
}

// setupCORS 配置跨域中间件
func setupCORS(summary *storage.Summary) *gin.Engine {
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
	web.SetupRoutes(r, summary)

	return r
}
