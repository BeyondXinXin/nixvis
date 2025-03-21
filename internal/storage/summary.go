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

type Summary struct {
	repo         *Repository
	currentHour  *HourlyStats // 当前小时所有指标的实时数据
	dataFilePath string
	mutex        sync.RWMutex
}

// TimeRange 表示统计的时间范围
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// 创建一个新的 Summary 实例
func NewSummary(userRepoPtr *Repository) *Summary {
	s := &Summary{
		repo:         userRepoPtr,
		dataFilePath: filepath.Join("./data", "stats_data.json"),
	}
	return s
}

// UpdateStats 更新当前小时的统计数据
func (s *Summary) UpdateStats() error {
	now := time.Now()

	// 使用新的查询函数获取当前小时的统计
	stats, err := s.GetHourlyStats(now)
	if err != nil {
		return err
	}

	// 更新当前小时统计
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.currentHour = &HourlyStats{
		Hour:  now.Hour(),
		Stats: stats,
	}

	logrus.Infof(
		"当前小时统计数据: PV=%d, UV=%d, Traffic=%d",
		stats.PV, stats.UV, stats.Traffic)

	return nil
}

// GetStatsData
// 获取统计数据、返回完整的统计数据集、在必要时保存到文件
func (s *Summary) GetStatsData() (*StatsData, error) {

	// 获取统计数据
	statsData := &StatsData{
		Days:        make(map[string]DailyStats),
		LastUpdated: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if _, err := os.Stat(s.dataFilePath); err == nil {
		data, err := os.ReadFile(s.dataFilePath)
		if err == nil {
			if err := json.Unmarshal(data, statsData); err != nil {
				return statsData, fmt.Errorf("解析统计数据文件失败: %v", err)
			}
		}
	}

	// 合并数据
	s.mutex.RLock()
	currentHourData := s.currentHour
	s.mutex.RUnlock()

	if currentHourData == nil {
		return statsData, fmt.Errorf("当前小时数据未初始化")
	}

	// 获取今天的日期
	now := time.Now()
	today := now.Format("2006-01-02")

	// 获取或创建今天的数据
	dailyStats, dailyStatsExists := statsData.Days[today]
	if !dailyStatsExists {
		// 创建新的24小时数据
		hourlyStats := make([]HourlyStats, 24)
		for i := range 24 {
			hourlyStats[i] = HourlyStats{
				Hour:  i,
				Stats: StatsPoint{},
			}
		}

		dailyStats = DailyStats{
			Date:   today,
			Hourly: hourlyStats,
		}
	}

	// 更新当前小时
	dailyStats.Hourly[currentHourData.Hour] = *currentHourData

	// 计算日合计
	total := StatsPoint{}
	for _, hourStat := range dailyStats.Hourly {
		total.PV += hourStat.Stats.PV
		total.UV += hourStat.Stats.UV
		total.Traffic += hourStat.Stats.Traffic
	}
	dailyStats.Total = total

	// 更新数据集
	statsData.Days[today] = dailyStats

	// 在必要时保存到文件
	if now.After(statsData.LastUpdated) {
		nextHour := now.Add(time.Hour).Truncate(time.Hour)
		statsData.LastUpdated = nextHour
		// 确保数据目录存在
		dataDir := filepath.Dir(s.dataFilePath)
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return statsData, fmt.Errorf("创建数据目录失败: %v", err)
		}

		// 序列化为JSON
		jsonData, err := json.Marshal(statsData)
		if err != nil {
			return statsData, fmt.Errorf("序列化统计数据失败: %v", err)
		}

		// 写入文件
		if err := os.WriteFile(s.dataFilePath, jsonData, 0644); err != nil {
			return statsData, fmt.Errorf("保存统计数据失败: %v", err)
		}
	}

	return statsData, nil
}

// GetHourlyStats 查询某小时的统计数据
func (s *Summary) GetHourlyStats(datetime time.Time) (StatsPoint, error) {
	// 设置小时开始和结束时间
	startOfHour := time.Date(datetime.Year(), datetime.Month(), datetime.Day(), datetime.Hour(), 0, 0, 0, datetime.Location())
	endOfHour := startOfHour.Add(time.Hour)

	return s.StatsByTimeRange(&TimeRange{
		Start: startOfHour,
		End:   endOfHour,
	})
}

// StatsByTimeRange 根据给定的时间范围查询统计数据
func (s *Summary) StatsByTimeRange(tr *TimeRange) (StatsPoint, error) {
	// 开始事务以确保查询一致性
	tx, err := s.repo.db.Begin()
	if err != nil {
		return StatsPoint{}, fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback() // 如果提交成功，这行不会有影响

	var whereClauses []string
	var queryParams []interface{}

	// 根据是否提供了时间范围构建查询条件
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

	// 构建完整的查询
	query := fmt.Sprintf(`
    SELECT 
        COUNT(CASE WHEN pageview_flag = 1 THEN 1 ELSE NULL END) as pv,
        COUNT(DISTINCT CASE WHEN pageview_flag = 1 THEN ip ELSE NULL END) as uv,
        COALESCE(SUM(bytes_sent), 0) as traffic
    FROM recent_logs
    %s`, whereClause)

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
