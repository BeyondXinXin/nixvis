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
	logrus.Info("Application started successfully")
	defer logrus.Info("Application shutting down")

	// 初始化数据
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

	// 启动定时任务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runMaintenanceScheduler(ctx, logParser, summary, repository)

	// 启动HTTP服务器
	r := setupCORS(summary) // 配置跨域中间件
	srv := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: r,
	}
	go func() {
		logrus.Info("Starting the server...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithField("error", err).Error("Failed to start the server")
		}
	}()

	// 等待程序退出
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	<-shutdownSignal

	// 取消上下文将会通知所有后台任务退出
	cancel()

	// 给后台任务一些时间来完成清理
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(), 3*time.Second)
	defer shutdownCancel()
	<-shutdownCtx.Done()

	repository.Close()
	logrus.Info("Clean shutdown completed")
}

// 定时任务调度器
func runMaintenanceScheduler(
	ctx context.Context,
	parser *storage.NginxLogParser,
	summary *storage.Summary,
	repo *storage.Repository) {

	logrus.Info("启动Nginx日志扫描任务")

	// 初始扫描
	performMaintenance(parser, summary, repo)

	// 定时扫描 - 每2分钟一次
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			performMaintenance(parser, summary, repo)
		case <-ctx.Done():
			logrus.Info("停止Nginx日志扫描任务")
			return
		}
	}
}

// 执行维护任务
func performMaintenance(
	parser *storage.NginxLogParser,
	summary *storage.Summary,
	repo *storage.Repository) {

	logrus.Info("开始扫描Nginx日志")

	// 1. 清理过期数据
	if err := repo.CleanupOldData(); err != nil {
		logrus.Errorf("清理过期数据失败: %v", err)
	}

	// 2. 扫描日志
	if err := parser.ScanNginxLogs(); err != nil {
		logrus.Errorf("扫描Nginx日志失败: %v", err)
		return
	}

	// 3. 生成统计数据
	if err := summary.UpdateStats(); err != nil {
		logrus.Errorf("生成统计数据失败: %v", err)
		return
	}
}

// 配置跨域中间件
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

	logrus.Info("Successfully initialized the server")

	return r
}
