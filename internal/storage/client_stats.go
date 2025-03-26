package storage

import (
	"fmt"

	"github.com/beyondxinxin/nixvis/internal/util"
)

type ClientStats struct {
	URL        []string    `json:"url"`
	URLOverall []StatPoint `json:"url_overall"`
}

func (s ClientStats) GetType() string {
	return "client"
}

type ClientStatsManager struct {
	repo      *Repository
	statsType string
}

func NewURLStatsManager(userRepoPtr *Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "url",
	}
}

func NewrefererStatsManager(userRepoPtr *Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "referer",
	}
}

func NewBrowserStatsManager(userRepoPtr *Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "user_browser",
	}
}

func NewOsStatsManager(userRepoPtr *Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "user_os",
	}
}

func NewDeviceStatsManager(userRepoPtr *Repository) *ClientStatsManager {
	return &ClientStatsManager{
		repo:      userRepoPtr,
		statsType: "user_device",
	}
}

// 实现 StatsManager 接口
func (s *ClientStatsManager) Query(query StatsQuery) (StatsResult, error) {
	result := ClientStats{
		URL:        make([]string, 0),
		URLOverall: make([]StatPoint, 0),
	}

	limit := 10 // 默认值
	if query.ExtraParam != nil {
		if l, ok := query.ExtraParam["limit"].(int); ok {
			limit = l
		}
	}

	startTime, endTime, err := util.TimePeriod(query.TimeRange)
	if err != nil {
		return result, err
	}

	// 构建、执行查询
	dbQueryStr := fmt.Sprintf(`
        SELECT 
            %s AS url, 
            COUNT(*) AS pv,
            COUNT(DISTINCT ip) AS uv,
            COALESCE(SUM(bytes_sent), 0) AS traffic
        FROM "%s_nginx_logs" INDEXED BY idx_%s_pv_ts_ip
        WHERE pageview_flag = 1 AND timestamp >= ? AND timestamp < ?
        GROUP BY %s
        ORDER BY uv DESC
        LIMIT ?`,
		s.statsType, query.WebsiteID, query.WebsiteID, s.statsType)

	rows, err := s.repo.db.Query(dbQueryStr, startTime.Unix(), endTime.Unix(), limit)
	if err != nil {
		return result, fmt.Errorf("查询URL统计失败: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		// 解析每一行数据
		var url string
		var urlStats StatPoint
		if err := rows.Scan(&url, &urlStats.PV, &urlStats.UV, &urlStats.Traffic); err != nil {
			return result, fmt.Errorf("解析URL统计结果失败: %v", err)
		}
		result.URL = append(result.URL, url)
		result.URLOverall = append(result.URLOverall, urlStats)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("遍历URL统计结果失败: %v", err)
	}

	return result, nil

}
