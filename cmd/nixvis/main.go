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
	statsFactory := storage.NewStatsFactory(repository)

	logrus.Info("****** 初始扫描 ******")
	executePeriodicTasks(logParser)

	logrus.Info("****** 启动HTTP服务器 ******")
	r := setupCORS(statsFactory)
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

	go runPeriodicTaskScheduler(ctx, logParser)

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
func setupCORS(statsFactory *storage.StatsFactory) *gin.Engine {

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
	web.SetupRoutes(r, statsFactory)

	return r
}

// runPeriodicTaskScheduler 运行周期性任务（每5分钟）
func runPeriodicTaskScheduler(
	ctx context.Context, parser *storage.LogParser) {

	// 定时扫描 - 每5分钟一次
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logrus.Info("****** 开始执行维护任务 ******")
			executePeriodicTasks(parser)
		case <-ctx.Done():
			return
		}
	}
}

// executePeriodicTasks 执行周期性任务
func executePeriodicTasks(
	parser *storage.LogParser) {

	logrus.Info("开始扫描Nginx日志")
	parser.ScanNginxLogs()
	logrus.Info("Nginx日志扫描完成")

}
