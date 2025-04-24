package stats

import (
	"fmt"
	"strings"
	"time"

	"github.com/beyondxinxin/nixvis/internal/storage"
	"github.com/beyondxinxin/nixvis/internal/util"
)

type StatPoint struct {
	PV int `json:"pv"` // 页面浏览量
	UV int `json:"uv"` // 独立访客数
}

type TimeSeriesStats struct {
	Labels    []string `json:"labels"`
	Visitors  []int    `json:"visitors"`
	Pageviews []int    `json:"pageviews"`
	PvMinusUv []int    `json:"pvMinusUv"` // PV - UV
}

// TimeSeriesStats 实现 StatsResult 接口
func (s TimeSeriesStats) GetType() string {
	return "timeseries"
}

type TimeSeriesStatsManager struct {
	repo *storage.Repository
}

// NewTimeSeriesStatsManager 创建一个新的 TimeSeriesStatsManager 实例
func NewTimeSeriesStatsManager(userRepoPtr *storage.Repository) *TimeSeriesStatsManager {
	return &TimeSeriesStatsManager{
		repo: userRepoPtr,
	}
}

// 实现 StatsManager 接口
func (s *TimeSeriesStatsManager) Query(query StatsQuery) (StatsResult, error) {
	timeRange := query.ExtraParam["timeRange"].(string)
	viewType := query.ExtraParam["viewType"].(string)
	timePoints, labels := util.TimePointsAndLabels(timeRange, viewType)
	result := TimeSeriesStats{
		Labels:    labels,
		Visitors:  make([]int, len(timePoints)),
		Pageviews: make([]int, len(timePoints)),
		PvMinusUv: make([]int, len(timePoints)),
	}

	statPoints, err := s.statsByTimePointsForWebsite(query.WebsiteID, timePoints)
	if err != nil {
		return result, fmt.Errorf("获取图表数据失败: %v", err)
	}
	for i, point := range statPoints {
		result.Pageviews[i] = point.PV
		result.Visitors[i] = point.UV
		result.PvMinusUv[i] = point.PV - point.UV
	}

	return result, nil
}

// statsByTimePointsForWebsite 根据多个时间点批量查询统计数据
func (s *TimeSeriesStatsManager) statsByTimePointsForWebsite(
	websiteID string, timePoints []time.Time) ([]StatPoint, error) {

	timePointsSize := len(timePoints)
	timeOffset := timePoints[1].Sub(timePoints[0])
	results := make([]StatPoint, timePointsSize)

	tx, err := s.repo.GetDB().Begin()
	if err != nil {
		return nil, fmt.Errorf("开始事务失败: %v", err)
	}
	defer tx.Rollback()

	tableName := fmt.Sprintf("%s_nginx_logs", websiteID)

	// 关键优化点1: 合并多个查询为一个批量查询
	args := make([]any, 0, timePointsSize*2)

	for i := range timePointsSize {
		startTime := timePoints[i]
		endTime := startTime.Add(timeOffset)
		args = append(args, startTime.Unix(), endTime.Unix())
	}

	// 关键优化点3: 构建一次性批量查询SQL
	batchQuery := fmt.Sprintf(`
        WITH time_ranges(range_index, start_time, end_time) AS (
            VALUES %s
        )
        SELECT 
            tr.range_index,
            COUNT(l.pageview_flag) as pv,
            COUNT(DISTINCT l.ip) as uv
        FROM time_ranges tr
        LEFT JOIN "%s" l INDEXED BY idx_%s_pv_ts_ip
            ON l.pageview_flag = 1 AND l.timestamp >= tr.start_time AND l.timestamp < tr.end_time
        GROUP BY tr.range_index
        ORDER BY tr.range_index`,
		formatRangeValues(timePointsSize), tableName, websiteID)

	rows, err := tx.Query(batchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("执行批量查询失败: %v", err)
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		var rangeIdx int
		if err := rows.Scan(&rangeIdx, &results[i].PV, &results[i].UV); err != nil {
			return nil, fmt.Errorf("读取查询结果失败: %v", err)
		}
		i++
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集时发生错误: %v", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %v", err)
	}

	return results, nil
}

// formatRangeValues 生成SQL中的值列表 (0, ?, ?), (1, ?, ?), ...
func formatRangeValues(count int) string {
	values := make([]string, count)
	for i := 0; i < count; i++ {
		values[i] = fmt.Sprintf("(%d, ?, ?)", i)
	}
	return strings.Join(values, ", ")
}
