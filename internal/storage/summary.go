package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// 统计指标点
type StatsPoint struct {
	PV      int   `json:"pv"`      // 页面浏览量
	UV      int   `json:"uv"`      // 独立访客数
	Traffic int64 `json:"traffic"` // 流量（字节）
}

// 每小时的统计数据
type HourlyStats struct {
	Hour  int        `json:"hour"`
	Stats StatsPoint `json:"stats"`
}

// 每天的统计数据
type DailyStats struct {
	Date   string        `json:"date"`
	Hourly []HourlyStats `json:"hourly"` // 24小时的详细数据
	Total  StatsPoint    `json:"total"`  // 当天汇总数据
}

// 存储在文件中的完整统计数据
type StatsData struct {
	Days        map[string]DailyStats `json:"days"` // 键为日期(2006-01-02)
	LastUpdated time.Time             `json:"last_updated"`
}

// TimeRange 表示统计的时间范围
type TimeRange struct {
	Start time.Time
	End   time.Time
}

type Summary struct {
	repo         *Repository
	currentHour  *HourlyStats // 当前小时所有指标的实时数据
	dataFilePath string
	mutex        sync.RWMutex
}

// NewSummary 创建一个新的 Summary 实例
func NewSummary(userRepoPtr *Repository) *Summary {
	return &Summary{
		repo:         userRepoPtr,
		dataFilePath: filepath.Join("./data", "stats_data.json"),
	}
}

// UpdateStats 更新当前小时的统计数据
func (s *Summary) UpdateStats() error {

	// 1.获取当前小时的统计数据
	now := time.Now()
	stats, err := s.GetHourlyStats(now)
	if err != nil {
		return err
	}

	// 2.更新当前小时统计
	s.mutex.Lock()
	s.currentHour = &HourlyStats{Hour: now.Hour(), Stats: stats}
	s.mutex.Unlock()

	logrus.Infof("当前小时统计数据: PV=%d, UV=%d, Traffic=%d",
		stats.PV, stats.UV, stats.Traffic)

	s.GetStatsData()
	return nil
}

// GetStatsData 获取统计数据、返回完整的统计数据集、在必要时保存到文件
func (s *Summary) GetStatsData() (*StatsData, error) {

	// 加载或创建统计数据
	statsData, err := s.loadStatsDataFromFile()
	if os.IsNotExist(err) || len(statsData.Days) == 0 {
		statsData, err = s.createInitialStatsData()
		if err != nil {
			return nil, fmt.Errorf("创建初始统计数据失败: %v", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("加载统计数据失败: %v", err)
	}

	// 获取或创建今天的数据
	s.mutex.RLock()
	currentHour := s.currentHour
	s.mutex.RUnlock()

	if currentHour == nil {
		return statsData, nil // 还没有当前小时的数据，直接返回
	}

	// 更新今天的数据
	now := time.Now()
	today := now.Format("2006-01-02")
	dailyStats := createOrGetDailyStats(today, statsData.Days)
	dailyStats.Hourly[currentHour.Hour] = *currentHour
	dailyStats.Total = calculateDailyTotal(dailyStats.Hourly)
	statsData.Days[today] = dailyStats

	// 在必要时保存到文件
	if now.After(statsData.LastUpdated) {
		nextHour := now.Add(time.Hour).Truncate(time.Hour)
		statsData.LastUpdated = nextHour

		if err := s.saveStatsDataToFile(statsData); err != nil {
			logrus.Warnf("保存统计数据失败: %v", err)
		}
	}

	return statsData, nil
}

// loadStatsDataFromFile 从文件加载统计数据
func (s *Summary) loadStatsDataFromFile() (*StatsData, error) {
	statsData := &StatsData{
		Days:        make(map[string]DailyStats),
		LastUpdated: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if _, err := os.Stat(s.dataFilePath); err == nil {
		data, err := os.ReadFile(s.dataFilePath)
		if err != nil {
			return statsData, fmt.Errorf("读取统计数据文件失败: %v", err)
		}

		if err := json.Unmarshal(data, statsData); err != nil {
			return statsData, fmt.Errorf("解析统计数据文件失败: %v", err)
		}
	}

	return statsData, nil
}

// saveStatsDataToFile 将统计数据保存到文件
func (s *Summary) saveStatsDataToFile(statsData *StatsData) error {
	// 确保数据目录存在
	dataDir := filepath.Dir(s.dataFilePath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %v", err)
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(statsData)
	if err != nil {
		return fmt.Errorf("序列化统计数据失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(s.dataFilePath, jsonData, 0644); err != nil {
		return fmt.Errorf("保存统计数据失败: %v", err)
	}

	return nil
}

// createInitialStatsData 从nginx日志创建初始统计数据
func (s *Summary) createInitialStatsData() (*StatsData, error) {
	logrus.Info("未找到统计数据文件，将从nginx日志获取历史数据")

	// 初始化统计数据结构
	statsData := &StatsData{
		Days:        make(map[string]DailyStats),
		LastUpdated: time.Now(),
	}

	// 获取历史数据的时间范围 - 默认获取最近30天的数据
	now := time.Now()
	for day := now.AddDate(0, 0, -30); !day.After(now); day = day.AddDate(0, 0, 1) {
		// 创建当天的24小时数据结构
		date := day.Format("2006-01-02")
		dailyStats := DailyStats{
			Date:   date,
			Hourly: make([]HourlyStats, 24),
			Total:  StatsPoint{},
		}
		for hour := range 24 { // 处理每个小时的数据
			stats, _ := s.GetHourlyStatsWithSource(day, hour, "nginx")
			dailyStats.Hourly[hour] = HourlyStats{
				Hour:  hour,
				Stats: stats,
			}
		}
		dailyStats.Total = calculateDailyTotal(dailyStats.Hourly)

		// 将当天数据添加到统计数据集
		statsData.Days[date] = dailyStats
	}

	// 保存统计数据到文件
	if err := s.saveStatsDataToFile(statsData); err != nil {
		return nil, err
	}

	logrus.Info("成功从nginx日志创建统计数据文件")
	return statsData, nil
}

// calculateDailyTotal 计算日汇总数据
func calculateDailyTotal(hourlyStats []HourlyStats) StatsPoint {
	total := StatsPoint{}
	for _, hourStat := range hourlyStats {
		total.PV += hourStat.Stats.PV
		total.UV += hourStat.Stats.UV
		total.Traffic += hourStat.Stats.Traffic
	}
	return total
}

// createOrGetDailyStats 获取或创建某一天的统计数据
func createOrGetDailyStats(date string, existingData map[string]DailyStats) DailyStats {
	dailyStats, exists := existingData[date]
	if !exists {
		// 创建新的24小时数据
		hourlyStats := make([]HourlyStats, 24)
		for i := range 24 {
			hourlyStats[i] = HourlyStats{
				Hour:  i,
				Stats: StatsPoint{},
			}
		}

		dailyStats = DailyStats{
			Date:   date,
			Hourly: hourlyStats,
		}
	}
	return dailyStats
}

// GetHourlyStats 查询某小时的统计数据
func (s *Summary) GetHourlyStats(dayTime time.Time) (StatsPoint, error) {
	return s.GetHourlyStatsWithSource(dayTime, dayTime.Hour(), "recent")
}

// GetHourlyStatsWithSource 查询某小时的统计数据
func (s *Summary) GetHourlyStatsWithSource(
	dayTime time.Time, hour int, source string) (StatsPoint, error) {
	startOfHour := time.Date(
		dayTime.Year(), dayTime.Month(), dayTime.Day(),
		hour, 0, 0, 0, dayTime.Location())

	tr := &TimeRange{
		Start: startOfHour,
		End:   startOfHour.Add(time.Hour),
	}

	return s.StatsByTimeRange(tr, source)
}

// StatsByTimeRange 根据给定的时间范围查询统计数据
// source参数指定数据源："recent"表示从recent_logs表获取，"nginx"表示从nginx_logs表获取
func (s *Summary) StatsByTimeRange(tr *TimeRange, source string) (StatsPoint, error) {
	// 开始事务以确保查询一致性
	tx, err := s.repo.db.Begin()
	if err != nil {
		return StatsPoint{}, fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback() // 如果提交成功，这行不会有影响

	// 构建查询条件
	var whereClauses []string
	var queryParams []interface{}

	if tr != nil {
		if !tr.Start.IsZero() {
			whereClauses = append(whereClauses, "timestamp >= ?")
			queryParams = append(queryParams, tr.Start.Format("2006-01-02 15:04:05"))
		}
		if !tr.End.IsZero() {
			whereClauses = append(whereClauses, "timestamp < ?")
			queryParams = append(queryParams, tr.End.Format("2006-01-02 15:04:05"))
		}
	}

	// 构建 WHERE 子句
	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 确定数据源表
	tableName := "recent_logs" // 默认表
	if source == "nginx" {
		tableName = "nginx_logs"
	}

	// 构建完整的查询
	query := fmt.Sprintf(`
    SELECT 
        COUNT(CASE WHEN pageview_flag = 1 THEN 1 ELSE NULL END) as pv,
        COUNT(DISTINCT CASE WHEN pageview_flag = 1 THEN ip ELSE NULL END) as uv,
        COALESCE(SUM(bytes_sent), 0) as traffic
    FROM %s
    %s`, tableName, whereClause)

	// 执行查询
	var stats StatsPoint
	row := tx.QueryRow(query, queryParams...)
	if err := row.Scan(&stats.PV, &stats.UV, &stats.Traffic); err != nil {
		return StatsPoint{}, fmt.Errorf("查询统计数据失败: %v", err)
	}

	// 提交事务
	if err = tx.Commit(); err != nil {
		return StatsPoint{}, fmt.Errorf("提交事务失败: %v", err)
	}

	return stats, nil
}
