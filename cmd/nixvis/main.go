package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/beyondxinxin/nixvis/internal/storage"
	"github.com/beyondxinxin/nixvis/internal/util"

	"github.com/sirupsen/logrus"
)

func main() {
	// 初始化配置
	util.InitDir()
	util.ConfigureLogging()
	util.ReadConfig()
	logrus.Info("Application started successfully")
	defer logrus.Info("Application shutting down")

	// 初始化数据
	repository, err := storage.NewRepository()
	if err != nil {
		logrus.WithField("error", err).
			Error("Failed to initialize the database")
		return
	}
	repository.Init()

	// 创建Nginx日志解析器
	logParser := storage.NewNginxLogParser(
		repository, "./data/blog.beyondxin.top.log")

	// 启动定时扫描
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动定时任务
	go runMaintenanceScheduler(ctx, logParser, repository)

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
	repo *storage.Repository) {

	logrus.Info("启动Nginx日志扫描任务")

	// 初始扫描
	performMaintenance(parser, repo)

	// 定时扫描 - 每2分钟一次
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			performMaintenance(parser, repo)
		case <-ctx.Done():
			logrus.Info("停止Nginx日志扫描任务")
			return
		}
	}
}

// 执行维护任务
func performMaintenance(
	parser *storage.NginxLogParser,
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
	// TODO: 添加生成统计数据的逻辑

	// 4. 生成网页
	// TODO: 添加生成网页的逻辑

}
