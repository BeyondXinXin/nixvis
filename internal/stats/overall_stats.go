package stats

import (
	"fmt"
	"time"

	"github.com/beyondxinxin/nixvis/internal/storage"
	"github.com/beyondxinxin/nixvis/internal/util"
)

type OverallStats struct {
	PV      int   `json:"pv"`      // 页面浏览量
	UV      int   `json:"uv"`      // 独立访客数
	Traffic int64 `json:"traffic"` // 流量（字节）
}

// OverallStats 实现 StatsResult 接口
func (s OverallStats) GetType() string {
	return "overall"
}

type OverallStatsManager struct {
	repo *storage.Repository
}

// NewOverallStatsManager 创建一个新的 OverallStatsManager 实例
func NewOverallStatsManager(userRepoPtr *storage.Repository) *OverallStatsManager {
	return &OverallStatsManager{
		repo: userRepoPtr,
	}
}

// 实现 StatsManager 接口
func (s *OverallStatsManager) Query(query StatsQuery) (StatsResult, error) {

	result := OverallStats{
		PV:      0,
		UV:      0,
		Traffic: 0,
	}

	timeRange := query.ExtraParam["timeRange"].(string)
	startTime, endTime, err := util.TimePeriod(timeRange)
	if err != nil {
		return result, err
	}

	err = s.statsByTimeRangeForWebsite(query.WebsiteID, startTime, endTime, &result)
	if err != nil {
		return result, fmt.Errorf("获取总体统计失败: %v", err)
	}

	return result, nil
}

// StatsByTimePoints 直接使用 db.Query() 方法查询数据库获取指定时间点的统计数据
func (s *OverallStatsManager) statsByTimeRangeForWebsite(
	websiteID string, startTime, endTime time.Time, overall *OverallStats) error {

	// 初始化结果
	overall.PV = 0
	overall.UV = 0
	overall.Traffic = 0

	tableName := fmt.Sprintf("%s_nginx_logs", websiteID)

	// 为更精确的统计，直接在数据库中进行全范围的唯一IP计数
	countQuery := fmt.Sprintf(`
        SELECT 
            COUNT(*) as pv,
            COUNT(DISTINCT ip) as uv,
            COALESCE(SUM(bytes_sent), 0) as traffic
        FROM "%s" INDEXED BY idx_%s_pv_ts_ip
        WHERE pageview_flag = 1 AND timestamp >= ? AND timestamp < ?`,
		tableName, websiteID)

	// 执行全范围查询
	row := s.repo.GetDB().QueryRow(countQuery, startTime.Unix(), endTime.Unix())

	if err := row.Scan(&overall.PV, &overall.UV, &overall.Traffic); err != nil {
		return fmt.Errorf("查询总体统计数据失败: %v", err)
	}

	return nil
}
