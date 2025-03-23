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

	"github.com/beyondxinxin/nixvis/internal/util"
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
	statePath string
	states    map[string]LogScanState // 各网站的扫描状态，以网站ID为键
}

// 创建新的日志解析器
func NewNginxLogParser(userRepoPtr *Repository) *NginxLogParser {
	statePath := "./data/nginx_scan_state.json"
	parser := &NginxLogParser{
		repo:      userRepoPtr,
		statePath: statePath,
		states:    make(map[string]LogScanState),
	}
	parser.loadState()
	return parser
}

// 增量扫描Nginx日志文件
func (p *NginxLogParser) ScanNginxLogs() error {
	// 获取所有网站ID
	websiteIDs := util.GetAllWebsiteIDs()

	for _, id := range websiteIDs {
		website, ok := util.GetWebsiteByID(id)
		if !ok {
			logrus.Warnf("找不到ID为 %s 的网站配置", id)
			continue
		}

		logrus.Infof("%s (%s) 开始扫描", website.Name, id)

		// 1. 打开文件并检查状态
		err := p.scanFile(id, website.LogPath)
		if err != nil {
			logrus.Errorf("扫描网站 %s 的日志失败: %v", website.Name, err)
			continue
		}
	}

	// 2. 统一更新并保存所有状态
	if err := p.updateState(); err != nil {
		logrus.Errorf("保存扫描状态失败: %v", err)
	}

	return nil
}

// 加载上次扫描状态
func (p *NginxLogParser) loadState() {
	data, err := os.ReadFile(p.statePath)
	if os.IsNotExist(err) {
		// 状态文件不存在，创建空状态映射
		p.states = make(map[string]LogScanState)
		return
	}

	if err != nil {
		logrus.Errorf("无法读取扫描状态文件: %v", err)
		p.states = make(map[string]LogScanState)
		return
	}

	if err := json.Unmarshal(data, &p.states); err != nil {
		logrus.Errorf("解析扫描状态失败: %v", err)
		p.states = make(map[string]LogScanState)
	}
}

// 打开并扫描日志文件
func (p *NginxLogParser) scanFile(websiteID string, logPath string) error {
	// 打开文件
	file, err := os.Open(logPath)
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
	startOffset := p.determineStartOffset(websiteID, currentSize)

	// 设置读取位置
	_, err = file.Seek(startOffset, 0)
	if err != nil {
		return err
	}

	// 读取并解析日志
	p.parseLogLines(file, websiteID)

	// 更新状态（但不保存）
	state := p.states[websiteID]
	state.LastOffset = currentSize
	state.LastSize = currentSize
	state.LastScan = time.Now()
	p.states[websiteID] = state

	return nil
}

// 确定扫描起始位置
func (p *NginxLogParser) determineStartOffset(websiteID string, currentSize int64) int64 {
	state, ok := p.states[websiteID]
	if !ok {
		// 如果该网站没有扫描记录，创建新状态并从头开始扫描
		p.states[websiteID] = LogScanState{
			LastOffset: 0,
			LastSize:   0,
			LastScan:   time.Now(),
		}
		return 0
	}

	// 检测文件是否被轮转（当前大小小于上次记录的大小）
	if currentSize < state.LastSize {
		logrus.Infof("检测到网站 %s 的日志文件已被轮转，从头开始扫描", websiteID)
		return 0
	}
	return state.LastOffset
}

// 解析日志行
func (p *NginxLogParser) parseLogLines(file *os.File, websiteID string) {
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

		if err := p.repo.BatchInsertLogsForWebsite(websiteID, batch); err != nil {
			logrus.Errorf("批量插入网站 %s 的日志记录失败: %v", websiteID, err)
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
		logrus.Errorf("扫描网站 %s 的文件时出错: %v", websiteID, err)
	}

	// 输出性能统计
	totalElapsed := time.Since(startTime)
	website, _ := util.GetWebsiteByID(websiteID)
	logrus.Infof("%s (%s) 扫描完成 - 新增条数: %d, 耗时: %.2fs",
		website.Name, websiteID, sumEntries, totalElapsed.Seconds())
}

// 更新并保存状态
func (p *NginxLogParser) updateState() error {
	data, err := json.Marshal(p.states)
	if err != nil {
		return err
	}
	// 确保目录存在
	os.MkdirAll("./data", 0755)
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

	pageviewFlag := 0
	// pv过滤条件：status = 200、path 中不含 favicon、sitemap
	if statusCode == 200 && !regexp.MustCompile(`favicon|sitemap`).MatchString(decodedPath) {
		pageviewFlag = 1
	}

	return &NginxLogRecord{
		ID:           0,
		IP:           matches[1],
		PageviewFlag: pageviewFlag,
		Timestamp:    timestamp,
		Method:       matches[4],
		Path:         decodedPath,
		Status:       statusCode,
		BytesSent:    bytesSent,
		Referer:      referPath,
		UserAgent:    matches[9],
	}, nil
}

// GetStatsData 获取统计数据（为兼容性保留，实际使用默认网站ID）
func (s *Summary) GetStatsData() (*StatsData, error) {
	websiteIDs := util.GetAllWebsiteIDs()
	if len(websiteIDs) == 0 {
		return &StatsData{
			Days:        make(map[string]DailyStats),
			LastUpdated: time.Now(),
		}, nil
	}

	// 使用第一个网站ID
	return s.GetStatsDataForWebsite(websiteIDs[1])
}
