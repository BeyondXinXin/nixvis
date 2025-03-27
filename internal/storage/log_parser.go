package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/mileusna/useragent"
	"github.com/sirupsen/logrus"
)

var (
	nginxLogPattern = regexp.MustCompile(`^(\S+) - (\S+) \[([^\]]+)\] "(\S+) ([^"]+) HTTP\/\d\.\d" (\d+) (\d+) "([^"]*)" "([^"]*)"`)
)

// 解析结果
type ParserResult struct {
	WebName      string
	WebID        string
	TotalEntries int
	Duration     time.Duration
	Success      bool
	Error        error
}

// 保存日志扫描的状态
type LogScanState struct {
	LastOffset int64     `json:"last_offset"`
	LastSize   int64     `json:"last_size"`
	LastScan   time.Time `json:"last_scan"`
}

// 用于处理Nginx日志
type LogParser struct {
	repo      *Repository
	statePath string
	states    map[string]LogScanState // 各网站的扫描状态，以网站ID为键
}

// 创建新的日志解析器
func NewLogParser(userRepoPtr *Repository) *LogParser {
	statePath := filepath.Join(util.DataDir, "nginx_scan_state.json")
	parser := &LogParser{
		repo:      userRepoPtr,
		statePath: statePath,
		states:    make(map[string]LogScanState),
	}
	parser.loadState()
	return parser
}

// 增量扫描Nginx日志文件
func (p *LogParser) ScanNginxLogs() []ParserResult {
	// 获取所有网站ID
	websiteIDs := util.GetAllWebsiteIDs()
	parserResults := make([]ParserResult, len(websiteIDs))

	for i, id := range websiteIDs {
		startTime := time.Now()

		website, _ := util.GetWebsiteByID(id)
		parserResult := EmptyParserResult(website.Name, id)

		p.scanFile(id, website.LogPath, &parserResult)

		parserResult.Duration = time.Since(startTime)
		parserResults[i] = parserResult
	}

	// 2. 更新并保存状态
	p.updateState()

	return parserResults
}

// 生成空结果
func EmptyParserResult(name, id string) ParserResult {
	return ParserResult{
		WebName:      name,
		WebID:        id,
		TotalEntries: 0,
		Duration:     0,
		Success:      true,
		Error:        nil,
	}
}

// 加载上次扫描状态
func (p *LogParser) loadState() {
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
func (p *LogParser) scanFile(websiteID string, logPath string, parserResult *ParserResult) {
	// 打开文件
	file, err := os.Open(logPath)
	if err != nil {
		parserResult.Success = false
		parserResult.Error = err
		return
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		parserResult.Success = false
		parserResult.Error = err
		return
	}

	// 确定扫描起始位置
	currentSize := fileInfo.Size()
	startOffset := p.determineStartOffset(websiteID, currentSize)

	// 设置读取位置
	_, err = file.Seek(startOffset, 0)
	if err != nil {
		parserResult.Success = false
		parserResult.Error = err
		return
	}

	// 读取并解析日志
	p.parseLogLines(file, websiteID, parserResult)

	// 更新状态（但不保存）
	state := p.states[websiteID]
	state.LastOffset = currentSize
	state.LastSize = currentSize
	state.LastScan = time.Now()
	p.states[websiteID] = state
}

// 确定扫描起始位置
func (p *LogParser) determineStartOffset(websiteID string, currentSize int64) int64 {
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
func (p *LogParser) parseLogLines(file *os.File, websiteID string, parserResult *ParserResult) {
	scanner := bufio.NewScanner(file)
	sumEntries := 0

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

	parserResult.TotalEntries = sumEntries
}

// 更新并保存状态
func (p *LogParser) updateState() {
	data, err := json.Marshal(p.states)
	if err != nil {
		logrus.Errorf("保存扫描状态失败: %v", err)
		return
	}

	if err := os.WriteFile(p.statePath, data, 0644); err != nil {
		logrus.Errorf("保存扫描状态失败: %v", err)
	}
}

// 解析单行Nginx日志
func (p *LogParser) parseNginxLogLine(line string) (*NginxLogRecord, error) {
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

	userAgent := useragent.Parse(matches[9])
	var browser, os, device string

	if userAgent.Bot {
		browser = "蜘蛛"
		os = "蜘蛛"
		device = "蜘蛛"
	} else {
		if userAgent.Name != "" {
			browser = userAgent.Name
		} else {
			browser = "桌面设备"
		}

		if userAgent.OS != "" {
			os = userAgent.OS
		} else {
			os = "其他"
		}

		if userAgent.Mobile {
			device = "手机"
		} else if userAgent.Tablet {
			device = "平板"
		} else if userAgent.Desktop {
			device = "桌面设备"
		} else {
			device = "其他"
		}
	}

	pageviewFlag := 0

	// pv过滤条件：
	// status = 200
	// path 中不含 favicon、sitemap、rss、robots.txt
	// 开头不含 /_nuxt
	if statusCode == 200 &&
		!regexp.MustCompile(`favicon|sitemap|rss|robots.txt`).MatchString(decodedPath) &&
		!regexp.MustCompile(`^/_nuxt`).MatchString(decodedPath) {
		pageviewFlag = 1
	}

	domesticLocation, globalLocation, _ := util.GetIPLocation(matches[1])

	return &NginxLogRecord{
		ID:               0,
		IP:               matches[1],
		PageviewFlag:     pageviewFlag,
		Timestamp:        timestamp,
		Method:           matches[4],
		Url:              decodedPath,
		Status:           statusCode,
		BytesSent:        bytesSent,
		Referer:          referPath,
		UserBrowser:      browser,
		UserOs:           os,
		UserDevice:       device,
		DomesticLocation: domesticLocation,
		GlobalLocation:   globalLocation,
	}, nil
}
