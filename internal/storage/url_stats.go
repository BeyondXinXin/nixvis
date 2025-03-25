package storage

import (
	"fmt"
	"time"
)

// URLStats 存储URL访问统计结果
type URLStats struct {
	URL     string `json:"url"`
	PV      int    `json:"pv"`
	UV      int    `json:"uv"`
	Traffic int64  `json:"traffic"`
}

type URLStatsManager struct {
	repo *Repository
}

// NewURLStatsManager 创建一个新的 URLStatsManager 实例
func NewURLStatsManager(userRepoPtr *Repository) *URLStatsManager {
	return &URLStatsManager{
		repo: userRepoPtr,
	}
}

// GetTopURLs 获取指定时间范围内访问量前N的URL统计
// timeRange参数可以是"today", "week", "last7days", "month", "last30days"
func (s *URLStatsManager) GetTopURLs(websiteID, timeRange string, limit int) ([]URLStats, error) {
	// 计算时间范围
	startTime, endTime, err := s.calculateTimeRange(timeRange)
	if err != nil {
		return nil, err
	}

	// 构建查询
	query := fmt.Sprintf(`
        SELECT 
            path AS url, 
            COUNT(*) AS pv,
            COUNT(DISTINCT ip) AS uv,
            SUM(bytes_sent) AS traffic
        FROM "%s_nginx_logs"
        WHERE timestamp BETWEEN ? AND ?
        AND pageview_flag = 1  -- 只统计有效页面浏览
        GROUP BY path
        ORDER BY uv DESC
        LIMIT ?
    `, websiteID)

	// 执行查询
	rows, err := s.repo.db.Query(query,
		startTime.Format("2006-01-02 15:04:05"),
		endTime.Format("2006-01-02 15:04:05"),
		limit)
	if err != nil {
		return nil, fmt.Errorf("查询URL统计失败: %v", err)
	}
	defer rows.Close()

	// 处理结果
	var results []URLStats
	for rows.Next() {
		var stat URLStats
		if err := rows.Scan(&stat.URL, &stat.PV, &stat.UV, &stat.Traffic); err != nil {
			return nil, fmt.Errorf("解析URL统计结果失败: %v", err)
		}
		results = append(results, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历URL统计结果失败: %v", err)
	}

	return results, nil
}

// calculateTimeRange 根据时间范围字符串计算开始和结束时间
func (s *URLStatsManager) calculateTimeRange(timeRange string) (time.Time, time.Time, error) {
	now := time.Now()
	endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	var startTime time.Time

	switch timeRange {
	case "today":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		// 周一作为一周的开始
		weekday := int(now.Weekday())
		if weekday == 0 { // Sunday
			weekday = 7
		}
		startTime = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
	case "last7days":
		startTime = time.Date(now.Year(), now.Month(), now.Day()-6, 0, 0, 0, 0, now.Location())
	case "month":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case "last30days":
		startTime = time.Date(now.Year(), now.Month(), now.Day()-29, 0, 0, 0, 0, now.Location())
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("不支持的时间范围: %s", timeRange)
	}

	return startTime, endTime, nil
}

// UpdateURLStats 更新 URL 统计信息 - 因为使用直接查询，此方法保留但为空实现
func (s *URLStatsManager) UpdateURLStats() {
	// 此方法保留但不实现
	// 因为我们采用直接查询方案，不需要预计算
}
