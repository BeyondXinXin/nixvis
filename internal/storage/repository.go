package storage

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

var (
	dataSourceName = filepath.Join(util.DataDir, "nixvis.db")
)

type NginxLogRecord struct {
	ID               int64     `json:"id"`
	IP               string    `json:"ip"`
	PageviewFlag     int       `json:"pageview_flag"`
	Timestamp        time.Time `json:"timestamp"`
	Method           string    `json:"method"`
	Url              string    `json:"url"`
	Status           int       `json:"status"`
	BytesSent        int       `json:"bytes_sent"`
	Referer          string    `json:"referer"`
	UserBrowser      string    `json:"user_browser"`
	UserOs           string    `json:"user_os"`
	UserDevice       string    `json:"user_device"`
	DomesticLocation string    `json:"domestic_location"`
	GlobalLocation   string    `json:"global_location"`
}

type Repository struct {
	db *sql.DB
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
        PRAGMA cache_size=32768;
        PRAGMA temp_store=MEMORY;`); err != nil {
		db.Close()
		return nil, err
	}

	return &Repository{
		db: db,
	}, nil
}

// 初始化数据库
func (r *Repository) Init() error {
	return r.createTables()
}

// 关闭数据库连接
func (r *Repository) Close() error {
	logrus.Info("关闭数据库")
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// 获取数据库连接
func (r *Repository) GetDB() *sql.DB {
	return r.db
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

	stmtNginx, err := tx.Prepare(fmt.Sprintf(`
        INSERT INTO "%s" (
        ip, pageview_flag, timestamp, method, url, 
        status_code, bytes_sent, referer, 
        user_browser, user_os, user_device, domestic_location, global_location)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, nginxTable))
	if err != nil {
		return err
	}
	defer stmtNginx.Close()

	// 执行批量插入
	for _, log := range logs {
		// 原始日志表
		_, err = stmtNginx.Exec(
			log.IP, log.PageviewFlag, log.Timestamp.Unix(), log.Method, log.Url,
			log.Status, log.BytesSent, log.Referer, log.UserBrowser, log.UserOs, log.UserDevice,
			log.DomesticLocation, log.GlobalLocation,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) createTables() error {
	common := `id INTEGER PRIMARY KEY AUTOINCREMENT,
	ip TEXT NOT NULL,
	pageview_flag INTEGER NOT NULL DEFAULT 0,
	timestamp INTEGER NOT NULL,
	method TEXT NOT NULL,
	url TEXT NOT NULL,
	status_code INTEGER NOT NULL,
	bytes_sent INTEGER NOT NULL,
	referer TEXT NOT NULL,
	user_browser TEXT NOT NULL,
	user_os TEXT NOT NULL,
	user_device TEXT NOT NULL,
	domestic_location TEXT NOT NULL,
	global_location TEXT NOT NULL`
	for _, id := range util.GetAllWebsiteIDs() {
		q := fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS "%[1]s_nginx_logs" (%[2]s);
             
             -- 单列索引
             CREATE INDEX IF NOT EXISTS idx_%[1]s_timestamp ON "%[1]s_nginx_logs"(timestamp);
             CREATE INDEX IF NOT EXISTS idx_%[1]s_url ON "%[1]s_nginx_logs"(url);
             CREATE INDEX IF NOT EXISTS idx_%[1]s_ip ON "%[1]s_nginx_logs"(ip);
             CREATE INDEX IF NOT EXISTS idx_%[1]s_referer ON "%[1]s_nginx_logs"(referer);
             CREATE INDEX IF NOT EXISTS idx_%[1]s_user_browser ON "%[1]s_nginx_logs"(user_browser);
             CREATE INDEX IF NOT EXISTS idx_%[1]s_user_os ON "%[1]s_nginx_logs"(user_os);
             CREATE INDEX IF NOT EXISTS idx_%[1]s_user_device ON "%[1]s_nginx_logs"(user_device);
             CREATE INDEX IF NOT EXISTS idx_%[1]s_domestic_location ON "%[1]s_nginx_logs"(domestic_location);
             CREATE INDEX IF NOT EXISTS idx_%[1]s_global_location ON "%[1]s_nginx_logs"(global_location);
             
             -- 复合索引
             CREATE INDEX IF NOT EXISTS idx_%[1]s_pv_ts_ip ON "%[1]s_nginx_logs" (pageview_flag, timestamp, ip);`,
			id, common,
		)
		if _, err := r.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
