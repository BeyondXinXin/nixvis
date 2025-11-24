package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/beyondxinxin/nixvis/internal/netparser"
	"github.com/beyondxinxin/nixvis/internal/stats"
	"github.com/beyondxinxin/nixvis/internal/storage"
	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/beyondxinxin/nixvis/internal/web"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/sirupsen/logrus"
)

func main() {
	if util.ProcessCliCommands() {
		os.Exit(0)
	}

	// 初始化日志、配置
	util.ConfigureLogging()
	defer util.CloseLogFile()

	logrus.Info("------ 服务启动成功 ------")
	// 在 main 函数中的适当位置添加
	logrus.Infof("构建时间: %s, Git提交: %s", util.BuildTime, util.GitCommit)
	defer logrus.Info("------ 服务已安全关闭 ------")

	// 初始化数据库
	err := netparser.InitIPGeoLocation()
	if err != nil {
		return
	}
	repository, err := initRepository()
	if err != nil {
		return
	}

	logParser := storage.NewLogParser(repository)
	statsFactory := stats.NewStatsFactory(repository)
	defer repository.Close()

	// 初始扫描
	initScan(logParser)

	// 启动HTTP服务器
	startHTTPServer(statsFactory)

	// 启动维护任务
	startPeriodicTaskScheduler(logParser)
}

// 初始化数据
func initRepository() (*storage.Repository, error) {
	logrus.Info("****** 1 初始化数据 ******")
	repository, err := storage.NewRepository()
	if err != nil {
		logrus.WithField("error", err).Error("Failed to create database file")
		return repository, err
	}

	if err := repository.Init(); err != nil {
		logrus.WithField("error", err).Error("Failed to create tables")
		return repository, err
	}

	return repository, nil
}

// 初始扫描
func initScan(parser *storage.LogParser) {
	logrus.Info("****** 2 初始扫描 ******")
	executePeriodicTasks(parser)
}

// 启动HTTP服务器
func startHTTPServer(statsFactory *stats.StatsFactory) {
	logrus.Info("****** 3 启动HTTP服务器 ******")
	cfg := util.ReadConfig()
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
	logrus.Infof("服务器已启动，监听地址: %s", cfg.Server.Port)
}

// setupCORS 配置跨域中间件
func setupCORS(statsFactory *stats.StatsFactory) *gin.Engine {

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		status := c.Writer.Status()

		if status >= 400 {
			logrus.Warnf("HTTP %d %s %s %s %v",
				status, c.Request.Method, path, c.ClientIP(), duration)
		} else if strings.HasPrefix(path, "/api/") && duration > 100*time.Millisecond {
			logrus.Warnf("高延迟 %s %s %d %s %v",
				c.Request.Method, path, status, c.ClientIP(), duration)
		}
	})

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

// 启动维护任务
func startPeriodicTaskScheduler(logParser *storage.LogParser) {
	logrus.Info("****** 4 启动维护任务 ******")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runPeriodicTaskScheduler(ctx, logParser)

	// 等待程序退出
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	<-shutdownSignal

	logrus.Info("开始关闭服务 ......")

	cancel() // 取消上下文将会通知所有后台任务退出

	// 给后台任务一些时间来完成清理
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()
	<-shutdownCtx.Done()
}

// runPeriodicTaskScheduler 运行周期性任务
func runPeriodicTaskScheduler(
	ctx context.Context, parser *storage.LogParser) {

	cfg := util.ReadConfig()
	interval := util.ParseInterval(cfg.System.TaskInterval, 5*time.Minute)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	iteration := 0

	for {
		select {
		case <-ticker.C:
			iteration++
			logrus.WithFields(logrus.Fields{"iteration": iteration}).Info("定期任务开始")
			executePeriodicTasks(parser)
		case <-ctx.Done():
			return
		}
	}
}

// executePeriodicTasks 执行周期性任务
func executePeriodicTasks(parser *storage.LogParser) {

	{ // 1 日志轮转
		if err := util.RotateLogFile(); err != nil {
			logrus.WithError(err).Warn("日志轮转失败")
		}
	}

	{ // 2 清理旧数据
		if err := parser.CleanOldLogs(); err != nil {
			logrus.WithError(err).Warn("清理数据库中过期日志数据失败")
		}
	}

	{ // 3 Nginx日志扫描
		startTime := time.Now()
		results := parser.ScanNginxLogs()
		totalDuration := time.Since(startTime)

		totalEntries := 0
		successCount := 0

		for _, result := range results {
			if result.WebName == "" {
				continue
			}

			totalEntries += result.TotalEntries

			if result.Success {
				successCount++
				if result.TotalEntries > 0 {
					logrus.Infof("网站 %s (%s) 扫描完成: %d 条记录, 耗时 %.2fs",
						result.WebName, result.WebID, result.TotalEntries, result.Duration.Seconds())
				}
			} else {
				logrus.Warnf("网站 %s (%s) 扫描失败: %s",
					result.WebName, result.WebID, result.Error)
			}
		}

		if totalEntries > 0 {
			logrus.Infof("Nginx日志扫描完成: %d/%d 个站点成功, 共 %d 条记录, 总耗时 %.2fs",
				successCount, len(results), totalEntries, totalDuration.Seconds())
		}
	}
}
