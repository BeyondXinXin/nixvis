package util

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var (
	BuildTime = "unknown"
	GitCommit = "unknown"
)

const (
	DataDir    = "./nixvis_data"
	ConfigFile = "./nixvis_config.json"
)

// HandleAppConfig 处理应用程序配置初始化和命令行参数
func ProcessCliCommands() bool {
	// 命令行参数
	genConfig := flag.Bool("gen-config", false, "生成配置文件并退出")
	cleanApp := flag.Bool("clean", false, "清理nixvis服务、释放端口和删除数据")
	showVer := flag.Bool("v", false, "显示版本信息")
	flag.Parse()

	// 显示版本信息
	if *showVer {
		showVersion()
		return true
	}

	// 清理服务
	if *cleanApp {
		cleanService()
		return true
	}

	// 生成配置文件
	if exit := initConfig(*genConfig); exit {
		return true
	}

	// 验证配置文件是否完整有效
	if exit := validateConfig(); exit {
		return true
	}

	// 初始化目录
	if exit := initDirs(); exit {
		return true
	}

	// 不需要退出，继续运行
	return false
}

// showVersion 显示版本信息
func showVersion() {
	fmt.Printf("构建时间: %s\n", BuildTime)
	fmt.Printf("Git 提交: %s\n", GitCommit)
}

func initConfig(genConfig bool) bool {
	_, err := os.Stat(ConfigFile)
	configExists := err == nil

	if genConfig || !configExists {
		err := writeDefaultConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成配置文件失败: %v\n", err)
		} else {
			fmt.Println("配置文件已生成: " + ConfigFile)
			fmt.Println("请编辑配置文件后再启动服务")
		}
		return true
	}
	return false
}

// initDirs 初始化目录
func initDirs() bool {
	dirs := []string{
		DataDir,
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "初始化目录失败: %v\n", err)
				return true
			}
		}
	}
	return false
}

// writeDefaultConfig 写入默认配置
func writeDefaultConfig() error {
	// 默认配置内容
	configJson := `
{
  "websites": [
    {
      "name": "示例网站1",
      "logPath": "./weblog_eg/blog.beyondxin.top.log"
    },
    {
      "name": "示例网站2",
      "logPath": "./weblog_eg/YiHangPavilion.log"
    }
  ],
  "system": {
    "logDestination": "file",
    "taskInterval": "5m"
  },
  "server": {
    "Port": ":8088"
  },
  "pvFilter": {
    "statusCodeInclude": [
      200
    ],
    "excludePatterns": [
      "favicon.ico$",
      "robots.txt$",
      "sitemap.xml$",
      "\\.(?:js|css|jpg|jpeg|png|gif|svg|webp|woff|woff2|ttf|eot|ico)$",
      "^/api/",
      "^/ajax/",
      "^/health$",
      "^/_(?:nuxt|next)/",
      "rss.xml$",
      "feed.xml$",
      "atom.xml$"
    ]
  }
}
`
	return os.WriteFile(ConfigFile, []byte(configJson), 0644)
}

// validateConfig 验证配置文件是否完整有效
func validateConfig() bool {

	// 读取配置
	cfg, err := ReadRawConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取配置文件失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "请修正配置问题后重新启动服务\n")
		return true
	}

	// 检查是否至少有一个网站配置
	if len(cfg.Websites) == 0 {
		fmt.Fprintf(os.Stderr,
			"读取配置文件失败: 配置文件缺少网站配置，至少需要配置一个网站\n")
		fmt.Fprintf(os.Stderr,
			"请修正配置问题后重新启动服务\n")
		return true
	}

	// 检查每个日志文件是否存在
	var missingLogs []string
	for _, site := range cfg.Websites {
		if site.LogPath == "" {
			missingLogs = append(missingLogs,
				fmt.Sprintf("'%s' (缺少日志文件路径配置)", site.Name))
			continue
		}

		// 检查日志文件是否存在，支持通配符模式
		if strings.Contains(site.LogPath, "*") {
			matches, err := filepath.Glob(site.LogPath)
			if err != nil || len(matches) == 0 {
				missingLogs = append(missingLogs,
					fmt.Sprintf("'%s' (%s - 未找到匹配的文件)",
						site.Name, site.LogPath))
			}
		} else if _, err := os.Stat(site.LogPath); os.IsNotExist(err) {
			// 普通文件路径
			missingLogs = append(missingLogs,
				fmt.Sprintf("'%s' (%s)", site.Name, site.LogPath))
		}
	}

	// 如果有缺失的日志文件，返回错误
	if len(missingLogs) > 0 {
		errMsg := "以下网站的日志文件不存在:\n"
		for _, missing := range missingLogs {
			errMsg += " - " + missing + "\n"
		}

		fmt.Fprintf(os.Stderr, "读取配置文件失败: %v\n", errMsg)
		fmt.Fprintf(os.Stderr, "请修正配置问题后重新启动服务\n")

		return true
	}

	// 检查PV过滤器配置
	if len(cfg.PVFilter.StatusCodeInclude) == 0 {
		fmt.Fprintf(os.Stderr, "配置文件错误: pvFilter.statusCodeInclude 不能为空\n")
		fmt.Fprintf(os.Stderr, "请修正配置问题后重新启动服务\n")
		return true
	}

	if len(cfg.PVFilter.ExcludePatterns) == 0 {
		fmt.Fprintf(os.Stderr, "配置文件错误: pvFilter.excludePatterns 不能为空\n")
		fmt.Fprintf(os.Stderr, "请修正配置问题后重新启动服务\n")
		return true
	}

	return false
}

// cleanService 清理 nixvis 服务、释放端口和删除数据
func cleanService() {
	fmt.Println("开始清理nixvis服务...")

	findAndTerminateProcesses("nixvis")

	// 清理数据目录
	fmt.Println("开始清理 nixvis_data 目录...")
	if err := os.RemoveAll(DataDir); err != nil {
		fmt.Printf("清理数据目录失败: %v\n", err)
	}
	fmt.Println("清理工作完成")
}

// findAndTerminateProcesses 查找并终止指定进程
func findAndTerminateProcesses(processName string) {
	// 获取当前进程和父进程的PID
	skipPID := os.Getpid()
	ppid := os.Getppid()

	// 查找并终止进程
	cmd := exec.Command("pgrep", "-f", processName)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		fmt.Printf("没有发现 %s 进程\n", processName)
		return
	}

	for _, pidStr := range strings.Split(
		strings.TrimSpace(string(output)), "\n") {
		// 解析PID
		pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
		if err != nil || pid == skipPID || pid == ppid {
			continue
		}

		// 终止进程
		if proc, err := os.FindProcess(pid); err == nil {
			fmt.Printf("正在终止进程 (PID: %d)\n", pid)
			proc.Signal(syscall.SIGKILL)
		}
	}
}
