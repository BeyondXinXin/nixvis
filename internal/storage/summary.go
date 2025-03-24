package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beyondxinxin/nixvis/internal/util"
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
	Days map[string]DailyStats `json:"days"` // 键为日期(2006-01-02)
}

// TimeRange 表示统计的时间范围
type TimeRange struct {
	Start time.Time
	End   time.Time
}

type Summary struct {
	repo        *Repository
	recentDate  map[string][]HourlyStats // 各网站最新日志的统计数据
	dataFileDir string                   // 统计数据文件所在目录
}

// NewSummary 创建一个新的 Summary 实例
func NewSummary(userRepoPtr *Repository) *Summary {
	return &Summary{
		repo:        userRepoPtr,
		recentDate:  make(map[string][]HourlyStats),
		dataFileDir: "./data",
	}
}

// UpdateStats 更新所有网站的统计数据并保存到文件
func (s *Summary) UpdateStats() error {
	websiteIDs := util.GetAllWebsiteIDs()
	now := time.Now()
	prevHour := now.Add(-1 * time.Hour)

	for _, id := range websiteIDs {
		website, _ := util.GetWebsiteByID(id)
		statsData, err := s.loadStatsDataFromFile(id)

		// 如果文件不存在，创建初始统计数据
		if os.IsNotExist(err) || statsData == nil || len(statsData.Days) == 0 {
			s.createInitialStatsDataForWebsite(id)
			statsData, err = s.loadStatsDataFromFile(id)
			if err != nil {
				logrus.Warnf("%s (%s) 创建网站的统计数据失败: %v", website.Name, id, err)
			}
		}

		//更新上个小时的数据
		prevDayStr := prevHour.Format("2006-01-02")
		prevDayStats := createOrGetDailyStats(prevDayStr, statsData.Days)
		prevDayStats.Hourly[prevHour.Hour()] = s.getHourlyStatsForWebsite(id, prevHour)
		prevDayStats.Total = calculateDailyTotal(prevDayStats.Hourly)
		statsData.Days[prevDayStr] = prevDayStats

		// 更新今天的数据
		todayStr := now.Format("2006-01-02")
		todayStats := createOrGetDailyStats(todayStr, statsData.Days)
		todayStats.Hourly[now.Hour()] = s.getHourlyStatsForWebsite(id, now)
		todayStats.Total = calculateDailyTotal(todayStats.Hourly)
		statsData.Days[todayStr] = todayStats

		if err := s.saveStatsDataToFile(id, statsData); err != nil {
			logrus.Warnf("保存网站 %s (%s) 的统计数据失败: %v", website.Name, id, err)
		}
	}

	return nil
}

// 获取特定网站的统计数据文件路径
func (s *Summary) getStatsFilePath(websiteID string) string {
	return filepath.Join(s.dataFileDir, fmt.Sprintf("%s_stats_data.json", websiteID))
}

// GetStatsDataForWebsite 获取指定网站的统计数据
func (s *Summary) GetStatsDataForWebsite(websiteID string) (*StatsData, error) {
	statsData, err := s.loadStatsDataFromFile(websiteID)
	if err != nil {
		if os.IsNotExist(err) { // 如果文件不存在，返回空的统计数据结构
			return &StatsData{
				Days: make(map[string]DailyStats),
			}, nil
		}
		return nil, fmt.Errorf("加载网站 %s 的统计数据失败: %v", websiteID, err)
	}

	return statsData, nil
}

// loadStatsDataFromFile 从文件加载特定网站的统计数据
func (s *Summary) loadStatsDataFromFile(websiteID string) (*StatsData, error) {
	statsData := &StatsData{
		Days: make(map[string]DailyStats),
	}

	filePath := s.getStatsFilePath(websiteID)
	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return statsData, fmt.Errorf("读取统计数据文件失败: %v", err)
		}

		if err := json.Unmarshal(data, statsData); err != nil {
			return statsData, fmt.Errorf("解析统计数据文件失败: %v", err)
		}
	} else if os.IsNotExist(err) {
		return statsData, err
	} else {
		return statsData, fmt.Errorf("检查统计数据文件失败: %v", err)
	}

	return statsData, nil
}

// createInitialStatsDataForWebsite 从nginx日志创建特定网站的初始统计数据
func (s *Summary) createInitialStatsDataForWebsite(websiteID string) (*StatsData, error) {
	website, ok := util.GetWebsiteByID(websiteID)
	if !ok {
		return nil, fmt.Errorf("找不到ID为 %s 的网站配置", websiteID)
	}

	logrus.Infof("未找到网站 %s (%s) 的统计数据文件，将从nginx日志获取历史数据",
		website.Name, websiteID)

	// 初始化统计数据结构
	statsData := &StatsData{
		Days: make(map[string]DailyStats),
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

		for hour := 0; hour < 24; hour++ { // 处理每个小时的数据
			hourTime := time.Date(day.Year(), day.Month(), day.Day(), hour, 0, 0, 0, day.Location())
			dailyStats.Hourly[hour] = s.getHourlyStatsForWebsite(websiteID, hourTime)
		}
		dailyStats.Total = calculateDailyTotal(dailyStats.Hourly)

		// 将当天数据添加到统计数据集
		statsData.Days[date] = dailyStats
	}

	// 保存统计数据到文件
	if err := s.saveStatsDataToFile(websiteID, statsData); err != nil {
		return nil, err
	}

	logrus.Infof("成功从nginx日志创建网站 %s (%s) 的统计数据文件",
		website.Name, websiteID)
	return statsData, nil
}

// saveStatsDataToFile 将特定网站的统计数据保存到文件
func (s *Summary) saveStatsDataToFile(websiteID string, statsData *StatsData) error {
	// 确保数据目录存在
	if err := os.MkdirAll(s.dataFileDir, 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %v", err)
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(statsData)
	if err != nil {
		return fmt.Errorf("序列化统计数据失败: %v", err)
	}

	// 写入文件
	filePath := s.getStatsFilePath(websiteID)
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("保存统计数据文件失败: %v", err)
	}

	return nil
}

// getHourlyStatsForWebsite 查询指定网站某小时的统计数据，指定数据源
func (s *Summary) getHourlyStatsForWebsite(websiteID string, dayTime time.Time) HourlyStats {

	startOfHour := time.Date(
		dayTime.Year(), dayTime.Month(), dayTime.Day(),
		dayTime.Hour(), 0, 0, 0, dayTime.Location())

	tr := &TimeRange{
		Start: startOfHour,
		End:   startOfHour.Add(time.Hour),
	}

	currentStats, _ := s.statsByTimeRangeForWebsite(websiteID, tr)

	hourlyStats := HourlyStats{
		Hour:  dayTime.Hour(),
		Stats: currentStats,
	}

	return hourlyStats
}

// statsByTimeRangeForWebsite 根据给定的时间范围查询特定网站的统计数据
func (s *Summary) statsByTimeRangeForWebsite(websiteID string, tr *TimeRange) (StatsPoint, error) {
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
	tableName := fmt.Sprintf("%s_nginx_logs", websiteID)

	// 构建完整的查询
	query := fmt.Sprintf(`
    SELECT 
        COUNT(CASE WHEN pageview_flag = 1 THEN 1 ELSE NULL END) as pv,
        COUNT(DISTINCT CASE WHEN pageview_flag = 1 THEN ip ELSE NULL END) as uv,
        COALESCE(SUM(bytes_sent), 0) as traffic
    FROM "%s"
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
