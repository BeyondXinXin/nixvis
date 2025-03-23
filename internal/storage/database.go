package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

var (
	dataSourceName = "./data/my_data.db"
)

type NginxLogRecord struct {
	ID           int64     `json:"id"`
	IP           string    `json:"ip"`
	PageviewFlag int       `json:"pageview_flag"`
	Timestamp    time.Time `json:"timestamp"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	Status       int       `json:"status"`
	BytesSent    int       `json:"bytes_sent"`
	Referer      string    `json:"referer"`
	UserAgent    string    `json:"user_agent"`
}

type Repository struct {
	db              *sql.DB
	lastCleanupHour int // 上次清理的小时
}

func NewRepository() (*Repository, error) {
	// 打开数据库
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}
	// 链接数据库
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	// 性能优化设置
	if _, err := db.Exec(`
        PRAGMA journal_mode=WAL;
        PRAGMA synchronous=NORMAL;
        PRAGMA cache_size=5000;
        PRAGMA temp_store=MEMORY;`); err != nil {
		db.Close()
		return nil, err
	}

	return &Repository{
		db:              db,
		lastCleanupHour: -1,
	}, nil
}

// 初始化数据库
func (r *Repository) Init() error {
	return r.createTables()
}

// 关闭数据库连接
func (r *Repository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// 清理过期数据
func (r *Repository) CleanupOldData() error {
	// 检查当前小时是否已清理
	now := time.Now()
	currentHour := now.Hour()

	// 如果当前小时已清理，则跳过
	if currentHour == r.lastCleanupHour {
		return nil
	}

	// 获取当前小时开始时间
	currentHourStart := time.Date(
		now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	// 获取所有网站ID并清理它们的recent_logs表
	totalRowsAffected := int64(0)
	for _, websiteID := range util.GetAllWebsiteIDs() {
		recentTable := fmt.Sprintf("%s_recent_logs", websiteID)

		// 执行清理操作
		result, err := r.db.Exec(fmt.Sprintf(`
         DELETE FROM "%s" 
         WHERE timestamp < ?`, recentTable), currentHourStart)

		if err != nil {
			return fmt.Errorf("清理 %s 表时出错: %w", recentTable, err)
		}

		// 累计清理结果
		rowsAffected, _ := result.RowsAffected()
		totalRowsAffected += rowsAffected
	}

	// 更新清理标记
	r.lastCleanupHour = currentHour

	// 记录一下清理的记录数
	if totalRowsAffected > 0 {
		logrus.Infof("已清理 %d 条过期日志记录", totalRowsAffected)
	}

	return nil
}

// 为特定网站批量插入日志记录
func (r *Repository) BatchInsertLogsForWebsite(websiteID string, logs []NginxLogRecord) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 准备批量插入语句
	nginxTable := fmt.Sprintf("%s_nginx_logs", websiteID)
	recentTable := fmt.Sprintf("%s_recent_logs", websiteID)

	stmtNginx, err := tx.Prepare(fmt.Sprintf(`
        INSERT INTO "%s" (
        ip, pageview_flag, timestamp, method, path, 
        status_code, bytes_sent, referer, user_agent)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, nginxTable))
	if err != nil {
		return err
	}
	defer stmtNginx.Close()

	stmtRecent, err := tx.Prepare(fmt.Sprintf(`
        INSERT INTO "%s" (
        ip, pageview_flag, timestamp, method, path, 
        status_code, bytes_sent, referer, user_agent)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, recentTable))
	if err != nil {
		return err
	}
	defer stmtRecent.Close()

	// 获取当前小时开始时间点
	now := time.Now()
	currentHourStart := time.Date(
		now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	// 执行批量插入
	for _, log := range logs {
		// 原始日志表
		_, err = stmtNginx.Exec(
			log.IP, log.PageviewFlag, log.Timestamp, log.Method, log.Path,
			log.Status, log.BytesSent, log.Referer, log.UserAgent,
		)
		if err != nil {
			return err
		}

		// 最近日志表
		if log.Timestamp.After(currentHourStart) ||
			log.Timestamp.Equal(currentHourStart) {
			_, err = stmtRecent.Exec(
				log.IP, log.PageviewFlag, log.Timestamp, log.Method, log.Path,
				log.Status, log.BytesSent, log.Referer, log.UserAgent,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *Repository) createTables() error {
	common := `id INTEGER PRIMARY KEY AUTOINCREMENT,
	ip TEXT NOT NULL,
	pageview_flag INTEGER NOT NULL DEFAULT 0,
	timestamp DATETIME NOT NULL,
	method TEXT NOT NULL,
	path TEXT NOT NULL,
	status_code INTEGER NOT NULL,
	bytes_sent INTEGER NOT NULL,
	referer TEXT NOT NULL,
	user_agent TEXT NOT NULL`
	for _, id := range util.GetAllWebsiteIDs() {
		q := fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%s_nginx_logs" (%s);
			 CREATE TABLE IF NOT EXISTS "%s_recent_logs" (%s);
			 CREATE INDEX IF NOT EXISTS idx_%s_nginx_logs_timestamp ON "%s_nginx_logs"(timestamp);
			 CREATE INDEX IF NOT EXISTS idx_%s_nginx_logs_path ON "%s_nginx_logs"(path);
			 CREATE INDEX IF NOT EXISTS idx_%s_nginx_logs_ip ON "%s_nginx_logs"(ip);`,
			id, common, id, common, id, id, id, id, id, id,
		)
		if _, err := r.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
