package storage

import (
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

var (
	dataSourceName = "./data/my_data.db"
)

type NginxLogRecord struct {
	ID        int64     `json:"id"`
	IP        string    `json:"ip"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	BytesSent int       `json:"bytes_sent"`
	Referer   string    `json:"referer"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
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

// 插入日志记录
func (r *Repository) InsertLog(log NginxLogRecord) error {
	// 开始事务
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 插入到 nginx_logs 表
	_, err = tx.Exec(`
		 INSERT INTO nginx_logs (ip, user_id, timestamp, method, path, status_code, bytes_sent, referer, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	 `, log.IP, log.UserID, log.Timestamp, log.Method, log.Path, log.Status, log.BytesSent, log.Referer, log.UserAgent)
	if err != nil {
		return err
	}

	// 插入到 recent_logs 表
	now := time.Now()
	currentHourStart := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	if log.Timestamp.After(currentHourStart) || log.Timestamp.Equal(currentHourStart) {
		_, err = tx.Exec(`
		 INSERT INTO recent_logs (ip, user_id, timestamp, method, path, status_code, bytes_sent, referer, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 `, log.IP, log.UserID, log.Timestamp, log.Method, log.Path, log.Status, log.BytesSent, log.Referer, log.UserAgent)
		if err != nil {
			return err
		}
	}

	// 提交事务
	return tx.Commit()
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
	currentHourStart := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	// 执行清理操作
	result, err := r.db.Exec(`
	 DELETE FROM recent_logs 
	 WHERE timestamp < ? `, currentHourStart)

	if err != nil {
		return err
	}

	// 记录清理结果
	rowsAffected, _ := result.RowsAffected()

	// 更新清理标记
	r.lastCleanupHour = currentHour

	// 记录一下清理的记录数
	if rowsAffected > 0 {
		logrus.Infof("已清理 %d 条过期日志记录", rowsAffected)
	}

	return nil
}

// 批量插入日志记录
func (r *Repository) BatchInsertLogs(logs []NginxLogRecord) error {
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
	stmtNginx, err := tx.Prepare(`
        INSERT INTO nginx_logs (ip, user_id, timestamp, method, path, status_code, bytes_sent, referer, user_agent)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmtNginx.Close()

	stmtRecent, err := tx.Prepare(`
        INSERT INTO recent_logs (ip, user_id, timestamp, method, path, status_code, bytes_sent, referer, user_agent)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmtRecent.Close()

	// 获取当前小时开始时间点
	now := time.Now()
	currentHourStart := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())

	// 执行批量插入
	for _, log := range logs {
		// 原始日志表
		_, err = stmtNginx.Exec(
			log.IP, log.UserID, log.Timestamp, log.Method, log.Path,
			log.Status, log.BytesSent, log.Referer, log.UserAgent,
		)
		if err != nil {
			return err
		}

		// 最近日志表
		if log.Timestamp.After(currentHourStart) || log.Timestamp.Equal(currentHourStart) {
			_, err = stmtRecent.Exec(
				log.IP, log.UserID, log.Timestamp, log.Method, log.Path,
				log.Status, log.BytesSent, log.Referer, log.UserAgent,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// 创建表
func (r *Repository) createTables() error {
	// 实现实际的表创建逻辑
	_, err := r.db.Exec(`
		-- 原始日志表
		CREATE TABLE IF NOT EXISTS nginx_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL,
			user_id TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			bytes_sent INTEGER NOT NULL,
			referer TEXT NOT NULL,
			user_agent TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		
		-- 最新1小时数据表
		CREATE TABLE IF NOT EXISTS recent_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL,
			user_id TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			bytes_sent INTEGER NOT NULL,
			referer TEXT NOT NULL,
			user_agent TEXT NOT NULL
		);
	
		-- 创建索引
		CREATE INDEX IF NOT EXISTS idx_nginx_logs_timestamp ON nginx_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_nginx_logs_path ON nginx_logs(path);
		CREATE INDEX IF NOT EXISTS idx_nginx_logs_ip ON nginx_logs(ip);
		`)

	return err
}
