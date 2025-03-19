package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	nginxLogPattern = regexp.MustCompile(`^(\S+) - (\S+) \[([^\]]+)\] "(\S+) ([^"]+) HTTP\/\d\.\d" (\d+) (\d+) "([^"]*)" "([^"]*)"`)
)

// 保存日志扫描的状态
type LogScanState struct {
	LastOffset int64     `json:"last_offset"`
	LastSize   int64     `json:"last_size"`
	LastScan   time.Time `json:"last_scan"`
}

// 用于处理Nginx日志
type NginxLogParser struct {
	repo      *Repository
	logPath   string
	statePath string
	state     LogScanState
}

// 创建新的日志解析器
func NewNginxLogParser(userRepoPtr *Repository, logPath string) *NginxLogParser {
	statePath := "./data/nginx_scan_state.json"
	parser := &NginxLogParser{
		repo:      userRepoPtr,
		logPath:   logPath,
		statePath: statePath,
	}
	parser.loadState()
	return parser
}

// 增量扫描Nginx日志文件
func (p *NginxLogParser) ScanNginxLogs() error {
	// 1. 打开文件并检查状态
	err := p.scanFile()
	if err != nil {
		return err
	}

	// 2. 更新内部状态并保存
	if err := p.updateState(); err != nil {
		logrus.Errorf("保存扫描状态失败: %v", err)
	}

	return nil
}

// 加载上次扫描状态
func (p *NginxLogParser) loadState() {
	data, err := os.ReadFile(p.statePath)
	if os.IsNotExist(err) {
		p.state = LogScanState{
			LastOffset: 0,
			LastSize:   0,
			LastScan:   time.Now(),
		}
		return
	}

	if err != nil {
		logrus.Errorf("无法读取扫描状态文件: %v", err)
		return
	}

	if err := json.Unmarshal(data, &p.state); err != nil {
		logrus.Errorf("解析扫描状态失败: %v", err)
		p.state = LogScanState{
			LastOffset: 0,
			LastSize:   0,
			LastScan:   time.Now(),
		}
	}
}

// 打开并扫描日志文件
func (p *NginxLogParser) scanFile() error {
	// 打开文件
	file, err := os.Open(p.logPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	// 确定扫描起始位置
	currentSize := fileInfo.Size()
	startOffset := p.determineStartOffset(currentSize)

	// 设置读取位置
	_, err = file.Seek(startOffset, 0)
	if err != nil {
		return err
	}

	// 读取并解析日志
	p.parseLogLines(file)

	// 更新状态（但不保存）
	p.state.LastOffset = currentSize
	p.state.LastSize = currentSize

	return nil
}

// 确定扫描起始位置
func (p *NginxLogParser) determineStartOffset(currentSize int64) int64 {
	// 检测文件是否被轮转（当前大小小于上次记录的大小）
	if currentSize < p.state.LastSize {
		logrus.Info("检测到日志文件已被轮转，从头开始扫描")
		return 0
	}
	return p.state.LastOffset
}

// 解析日志行
func (p *NginxLogParser) parseLogLines(file *os.File) {
	scanner := bufio.NewScanner(file)
	sumEntries := 0
	startTime := time.Now()

	// 批量插入相关
	const batchSize = 100 // 可以调整批量大小
	batch := make([]NginxLogRecord, 0, batchSize)

	// 处理一批数据
	processBatch := func() {
		if len(batch) == 0 {
			return
		}

		if err := p.repo.BatchInsertLogs(batch); err != nil {
			logrus.Errorf("批量插入日志记录失败: %v", err)
		}

		batch = batch[:0] // 清空批次但保留容量
	}

	// 逐行处理
	for scanner.Scan() {
		line := scanner.Text()
		entry, err := p.parseNginxLogLine(line)
		if err != nil {
			continue
		}
		batch = append(batch, *entry)
		sumEntries++
		if len(batch) >= batchSize {
			processBatch()
		}
	}

	processBatch() // 处理剩余的记录

	if err := scanner.Err(); err != nil {
		logrus.Errorf("扫描文件时出错: %v", err)
	}

	// 输出性能统计
	totalElapsed := time.Since(startTime)
	logrus.Infof("日志处理完成 - 总条数: %d, 总耗时: %.2fs",
		sumEntries, totalElapsed.Seconds())
}

// 更新并保存状态
func (p *NginxLogParser) updateState() error {
	p.state.LastScan = time.Now()
	p.state.LastScan = time.Now()
	data, err := json.Marshal(p.state)
	if err != nil {
		return err
	}
	return os.WriteFile(p.statePath, data, 0644)
}

// 解析单行Nginx日志
func (p *NginxLogParser) parseNginxLogLine(line string) (*NginxLogRecord, error) {
	matches := nginxLogPattern.FindStringSubmatch(line)

	if matches == nil || len(matches) < 10 {
		return nil, errors.New("日志格式不匹配")
	}

	timestamp, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[3])
	if err != nil {
		return nil, err
	}

	statusCode, _ := strconv.Atoi(matches[6])
	bytesSent, _ := strconv.Atoi(matches[7])
	decodedPath, err := url.QueryUnescape(matches[5])
	if err != nil {
		decodedPath = matches[5]
	}
	referPath, err := url.QueryUnescape(matches[8])
	if err != nil {
		referPath = matches[8]
	}

	return &NginxLogRecord{
		ID:        0,
		IP:        matches[1],
		UserID:    matches[2],
		Timestamp: timestamp,
		Method:    matches[4],
		Path:      decodedPath,
		Status:    statusCode,
		BytesSent: bytesSent,
		Referer:   referPath,
		UserAgent: matches[9],
		CreatedAt: time.Now(),
	}, nil
}
