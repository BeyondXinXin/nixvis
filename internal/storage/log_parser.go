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
	"strings"
	"time"

	"github.com/beyondxinxin/nixvis/internal/netparser"
	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/sirupsen/logrus"
)

var (
	nginxLogPattern = regexp.MustCompile(`^(\S+) - (\S+) \[([^\]]+)\] "(\S+) ([^"]+) HTTP\/\d\.\d" (\d+) (\d+) "([^"]*)" "([^"]*)"`)
	lastCleanupDate = ""
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

type LogScanState struct {
	Files map[string]FileState `json:"files"` // 每个文件的状态
}

type FileState struct {
	LastOffset int64 `json:"last_offset"`
	LastSize   int64 `json:"last_size"`
}

type LogParser struct {
	repo      *Repository
	statePath string
	states    map[string]LogScanState // 各网站的扫描状态，以网站ID为键
}

// NewLogParser 创建新的日志解析器
func NewLogParser(userRepoPtr *Repository) *LogParser {
	statePath := filepath.Join(util.DataDir, "nginx_scan_state.json")
	parser := &LogParser{
		repo:      userRepoPtr,
		statePath: statePath,
		states:    make(map[string]LogScanState),
	}
	parser.loadState()
	netparser.InitPVFilters()
	return parser
}

// loadState 加载上次扫描状态
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

// updateState 更新并保存状态
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

// CleanOldLogs 清理45天前的日志数据
func (p *LogParser) CleanOldLogs() error {
	today := time.Now().Format("2006-01-02")
	currentHour := time.Now().Hour()

	shouldClean := lastCleanupDate == "" || (currentHour == 2 && lastCleanupDate != today)

	if !shouldClean {
		return nil
	}

	err := p.repo.CleanOldLogs()
	if err != nil {
		return err
	}

	lastCleanupDate = today

	return nil
}

// ScanNginxLogs 增量扫描Nginx日志文件
func (p *LogParser) ScanNginxLogs() []ParserResult {
	// 获取所有网站ID
	websiteIDs := util.GetAllWebsiteIDs()
	parserResults := make([]ParserResult, len(websiteIDs))

	for i, id := range websiteIDs {
		startTime := time.Now()

		website, _ := util.GetWebsiteByID(id)
		parserResult := EmptyParserResult(website.Name, id)

		logPath := website.LogPath
		if strings.Contains(logPath, "*") {
			matches, err := filepath.Glob(logPath)
			if err != nil {
				errstr := "解析日志路径模式 " + logPath + " 失败: " + err.Error()
				parserResult.Success = false
				parserResult.Error = errors.New(errstr)
			} else if len(matches) == 0 {
				errstr := "日志路径模式 " + logPath + " 未匹配到任何文件"
				parserResult.Success = false
				parserResult.Error = errors.New(errstr)
			} else {
				for _, matchPath := range matches {
					p.scanSingleFile(id, matchPath, &parserResult)
				}
			}
		} else {
			p.scanSingleFile(id, logPath, &parserResult)
		}

		parserResult.Duration = time.Since(startTime)
		parserResults[i] = parserResult
	}

	// 2. 更新并保存状态
	p.updateState()

	return parserResults
}

// scanSingleFile 扫描单个日志文件
func (p *LogParser) scanSingleFile(
	websiteID string, logPath string, parserResult *ParserResult) {
	// 打开文件
	file, err := os.Open(logPath)
	if err != nil {
		logrus.Errorf("无法打开日志文件 %s: %v", logPath, err)
		return
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		logrus.Errorf("无法获取文件信息 %s: %v", logPath, err)
		return
	}

	// 确定扫描起始位置
	currentSize := fileInfo.Size()
	startOffset := p.determineStartOffset(websiteID, logPath, currentSize)

	// 设置读取位置
	_, err = file.Seek(startOffset, 0)
	if err != nil {
		logrus.Errorf("无法设置文件读取位置 %s: %v", logPath, err)
		return
	}

	// 读取并解析日志
	entriesCount := p.parseLogLines(file, websiteID, parserResult)

	// 更新文件状态
	p.updateFileState(websiteID, logPath, currentSize)

	if entriesCount > 0 {
		logrus.Infof("网站 %s 的日志文件 %s 扫描完成，解析了 %d 条记录",
			websiteID, logPath, entriesCount)
	}
}

// updateFileState 更新文件状态
func (p *LogParser) updateFileState(
	websiteID string, filePath string, currentSize int64) {
	state, ok := p.states[websiteID]
	if !ok {
		state = LogScanState{
			Files: make(map[string]FileState),
		}
	}

	if state.Files == nil {
		state.Files = make(map[string]FileState)
	}

	fileState := FileState{
		LastOffset: currentSize,
		LastSize:   currentSize,
	}

	state.Files[filePath] = fileState
	p.states[websiteID] = state
}

// determineStartOffset 确定扫描起始位置
func (p *LogParser) determineStartOffset(
	websiteID string, filePath string, currentSize int64) int64 {

	state, ok := p.states[websiteID]
	if !ok { // 网站没有扫描记录，创建新状态
		p.states[websiteID] = LogScanState{
			Files: make(map[string]FileState),
		}
		return 0
	}

	if state.Files == nil {
		state.Files = make(map[string]FileState)
		p.states[websiteID] = state
		return 0
	}

	fileState, ok := state.Files[filePath]
	if !ok {
		return 0
	}

	// 文件是否被轮转
	if currentSize < fileState.LastSize {
		logrus.Infof("检测到网站 %s 的日志文件 %s 已被轮转，从头开始扫描", websiteID, filePath)
		return 0
	}

	return fileState.LastOffset
}

// parseLogLines 解析日志行并返回解析的记录数
func (p *LogParser) parseLogLines(
	file *os.File, websiteID string, parserResult *ParserResult) int {
	scanner := bufio.NewScanner(file)
	entriesCount := 0

	// 批量插入相关
	const batchSize = 100
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
		entriesCount++
		parserResult.TotalEntries++ // 累加到总结果中，而非赋值

		if len(batch) >= batchSize {
			processBatch()
		}
	}

	processBatch() // 处理剩余的记录

	if err := scanner.Err(); err != nil {
		logrus.Errorf("扫描网站 %s 的文件时出错: %v", websiteID, err)
	}

	return entriesCount // 返回当前文件的日志条数
}

// parseNginxLogLine 解析单行Nginx日志
func (p *LogParser) parseNginxLogLine(line string) (*NginxLogRecord, error) {
	matches := nginxLogPattern.FindStringSubmatch(line)

	if len(matches) < 10 {
		return nil, errors.New("日志格式不匹配")
	}

	timestamp, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[3])
	if err != nil {
		return nil, err
	}

	cutoffTime := time.Now().AddDate(0, 0, -31)
	if timestamp.Before(cutoffTime) {
		return nil, errors.New("日志超过30天")
	}

	decodedPath, err := url.QueryUnescape(matches[5])
	if err != nil {
		decodedPath = matches[5]
	}
	statusCode, _ := strconv.Atoi(matches[6])
	bytesSent, _ := strconv.Atoi(matches[7])
	referPath, err := url.QueryUnescape(matches[8])
	if err != nil {
		referPath = matches[8]
	}

	pageviewFlag := netparser.ShouldCountAsPageView(statusCode, decodedPath, matches[1])
	domesticLocation, globalLocation, _ := netparser.GetIPLocation(matches[1])
	browser, os, device := netparser.ParseUserAgent(matches[9])

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

// EmptyParserResult 生成空结果
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
