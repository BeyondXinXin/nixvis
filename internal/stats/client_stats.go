package stats

import (
	"fmt"
	"math"

	"github.com/beyondxinxin/nixvis/internal/storage"
	"github.com/beyondxinxin/nixvis/internal/util"
)

type ClientStats struct {
	Key       []string `json:"key"`        // 统计项的键
	PV        []int    `json:"pv"`         // 页面浏览量
	UV        []int    `json:"uv"`         // 独立访客数
	PVPercent []int    `json:"pv_percent"` // PV 百分比
	UVPercent []int    `json:"uv_percent"` // UV 百分比
}

func (s ClientStats) GetType() string {
	return "client"
}

type ClientStatsManager struct {
	repo      *storage.Repository
	statsType string
}

func NewURLStatsManager(userRepoPtr *storage.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "url",
	}
}

func NewrefererStatsManager(userRepoPtr *storage.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "referer",
	}
}

func NewBrowserStatsManager(userRepoPtr *storage.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "user_browser",
	}
}

func NewOsStatsManager(userRepoPtr *storage.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "user_os",
	}
}

func NewDeviceStatsManager(userRepoPtr *storage.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "user_device",
	}
}

func NewLocationStatsManager(userRepoPtr *storage.Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "location",
	}
}

// 实现 StatsManager 接口
func (s *ClientStatsManager) Query(query StatsQuery) (StatsResult, error) {
	result := ClientStats{
		Key:       make([]string, 0),
		PV:        make([]int, 0),
		UV:        make([]int, 0),
		PVPercent: make([]int, 0),
		UVPercent: make([]int, 0),
	}

	statsType := s.statsType
	if s.statsType == "location" {
		statsType = query.ExtraParam["locationType"].(string) + "_location"
	}
	limit, _ := query.ExtraParam["limit"].(int)
	timeRange := query.ExtraParam["timeRange"].(string)
	startTime, endTime, err := util.TimePeriod(timeRange)
	if err != nil {
		return result, err
	}

	// 构建、执行查询
	dbQueryStr := fmt.Sprintf(`
        SELECT 
            %[1]s AS url, 
            COUNT(*) AS pv,
            COUNT(DISTINCT ip) AS uv
        FROM "%[2]s_nginx_logs" INDEXED BY idx_%[2]s_pv_ts_ip
        WHERE pageview_flag = 1 AND timestamp >= ? AND timestamp < ?
        GROUP BY %[1]s
        ORDER BY uv DESC
        LIMIT ?`,
		statsType, query.WebsiteID)

	rows, err := s.repo.GetDB().Query(dbQueryStr, startTime.Unix(), endTime.Unix(), limit)
	if err != nil {
		return result, fmt.Errorf("查询URL统计失败: %v", err)
	}
	defer rows.Close()

	totalPV := 0
	totalUV := 0

	for rows.Next() {
		var url string
		var pv, uv int
		if err := rows.Scan(&url, &pv, &uv); err != nil {
			return result, fmt.Errorf("解析URL统计结果失败: %v", err)
		}
		result.Key = append(result.Key, url)
		result.PV = append(result.PV, pv)
		result.UV = append(result.UV, uv)
		totalPV += pv
		totalUV += uv
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("遍历URL统计结果失败: %v", err)
	}

	if totalPV > 0 && totalUV > 0 {
		for i := range result.PV {
			result.PVPercent = append(
				result.PVPercent, int(
					math.Round(float64(result.PV[i])/float64(totalPV)*100)))
			result.UVPercent = append(
				result.UVPercent, int(
					math.Round(float64(result.UV[i])/float64(totalUV)*100)))
		}
	}

	return result, nil

}
