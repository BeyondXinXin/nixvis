package stats

import (
	"fmt"
	"strings"
	"time"

	"github.com/beyondxinxin/nixvis/internal/storage"
)

// LogEntry 表示单条日志信息
type LogEntry struct {
	ID               int    `json:"id"`
	IP               string `json:"ip"`
	Timestamp        int64  `json:"timestamp"`
	Time             string `json:"time"` // 格式化后的时间字符串
	Method           string `json:"method"`
	URL              string `json:"url"`
	StatusCode       int    `json:"status_code"`
	BytesSent        int    `json:"bytes_sent"`
	Referer          string `json:"referer"`
	UserBrowser      string `json:"user_browser"`
	UserOS           string `json:"user_os"`
	UserDevice       string `json:"user_device"`
	DomesticLocation string `json:"domestic_location"`
	GlobalLocation   string `json:"global_location"`
	PageviewFlag     bool   `json:"pageview_flag"`
}

// LogsStats 日志查询结果
type LogsStats struct {
	Logs       []LogEntry `json:"logs"`
	Pagination struct {
		Total    int `json:"total"`
		Page     int `json:"page"`
		PageSize int `json:"pageSize"`
		Pages    int `json:"pages"`
	} `json:"pagination"`
}

// GetType 实现 StatsResult 接口
func (s LogsStats) GetType() string {
	return "logs"
}

// LogsStatsManager 实现日志查询功能
type LogsStatsManager struct {
	repo *storage.Repository
}

// NewLogsStatsManager 创建日志查询管理器
func NewLogsStatsManager(userRepoPtr *storage.Repository) *LogsStatsManager {
	return &LogsStatsManager{
		repo: userRepoPtr,
	}
}

// Query 实现 StatsManager 接口
func (m *LogsStatsManager) Query(query StatsQuery) (StatsResult, error) {
	result := LogsStats{}

	// 从查询参数中获取分页和排序信息
	page := 1
	pageSize := 100
	sortField := "timestamp"
	sortOrder := "desc"
	var filter string

	if pageVal, ok := query.ExtraParam["page"].(int); ok && pageVal > 0 {
		page = pageVal
	}

	if pageSizeVal, ok := query.ExtraParam["pageSize"].(int); ok && pageSizeVal > 0 {
		pageSize = pageSizeVal
		if pageSize > 1000 {
			pageSize = 1000 // 设置上限以防过大查询
		}
	}

	if field, ok := query.ExtraParam["sortField"].(string); ok && field != "" {
		// 验证字段名有效性，防止SQL注入
		validFields := map[string]bool{
			"timestamp": true, "ip": true, "url": true,
			"status_code": true, "bytes_sent": true,
		}
		if validFields[field] {
			sortField = field
		}
	}

	if order, ok := query.ExtraParam["sortOrder"].(string); ok {
		if order == "asc" || order == "desc" {
			sortOrder = order
		}
	}

	if filterVal, ok := query.ExtraParam["filter"].(string); ok {
		filter = filterVal
	}

	// 计算分页
	offset := (page - 1) * pageSize
	tableName := fmt.Sprintf("%s_nginx_logs", query.WebsiteID)

	// 构建查询语句
	var queryBuilder strings.Builder
	queryBuilder.WriteString(fmt.Sprintf(`
        SELECT 
            id, ip, timestamp, method, url, status_code, 
            bytes_sent, referer, user_browser, user_os, user_device, 
            domestic_location, global_location, pageview_flag
        FROM "%s"`, tableName))

	// 添加过滤条件
	var args []interface{}
	if filter != "" {
		queryBuilder.WriteString(" WHERE url LIKE ? OR ip LIKE ? OR referer LIKE ? OR domestic_location LIKE ?")
		filterArg := "%" + filter + "%"
		args = append(args, filterArg, filterArg, filterArg, filterArg)
	}

	// 添加排序
	queryBuilder.WriteString(fmt.Sprintf(" ORDER BY %s %s", sortField, sortOrder))

	// 添加分页
	queryBuilder.WriteString(" LIMIT ? OFFSET ?")
	args = append(args, pageSize, offset)

	// 执行查询
	rows, err := m.repo.GetDB().Query(queryBuilder.String(), args...)
	if err != nil {
		return result, fmt.Errorf("查询日志失败: %v", err)
	}
	defer rows.Close()

	// 处理结果
	logs := make([]LogEntry, 0)
	for rows.Next() {
		var log LogEntry
		var pageviewFlag int

		err := rows.Scan(&log.ID, &log.IP, &log.Timestamp, &log.Method, &log.URL, &log.StatusCode,
			&log.BytesSent, &log.Referer, &log.UserBrowser, &log.UserOS, &log.UserDevice,
			&log.DomesticLocation, &log.GlobalLocation, &pageviewFlag)

		if err != nil {
			return result, fmt.Errorf("解析日志行失败: %v", err)
		}

		// 处理时间
		log.Time = time.Unix(log.Timestamp, 0).Format("2006-01-02 15:04:05")

		// 处理pageview_flag (SQLite中存储为0/1)
		log.PageviewFlag = pageviewFlag == 1

		logs = append(logs, log)
	}

	// 查询总记录数
	var countQuery strings.Builder
	countQuery.WriteString(fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, tableName))

	var countArgs []interface{}
	if filter != "" {
		countQuery.WriteString(" WHERE url LIKE ? OR ip LIKE ? OR referer LIKE ? OR domestic_location LIKE ?")
		filterArg := "%" + filter + "%"
		countArgs = append(countArgs, filterArg, filterArg, filterArg, filterArg)
	}

	var total int
	err = m.repo.GetDB().QueryRow(countQuery.String(), countArgs...).Scan(&total)
	if err != nil {
		return result, fmt.Errorf("获取日志总数失败: %v", err)
	}

	// 设置返回结果
	result.Logs = logs
	result.Pagination.Total = total
	result.Pagination.Page = page
	result.Pagination.PageSize = pageSize
	result.Pagination.Pages = (total + pageSize - 1) / pageSize

	return result, nil
}
