package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/beyondxinxin/nixvis/internal/util"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

// Database 接口定义数据库操作
type Database interface {
	Init() error
	Close() error
	GetDB() interface{}
	DBType() string
	BatchInsertLogsForWebsite(websiteID string, logs []NginxLogRecord) error
	CleanOldLogs() error
}

// SQLiteDatabase SQLite 数据库实现
type SQLiteDatabase struct {
	db *sql.DB
}

func NewSQLiteDatabase(dataSourceName string) (*SQLiteDatabase, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec(`
        PRAGMA journal_mode=WAL;
        PRAGMA synchronous=NORMAL;
        PRAGMA cache_size=32768;
        PRAGMA temp_store=MEMORY;`); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteDatabase{
		db: db,
	}, nil
}

func (s *SQLiteDatabase) Init() error {
	logrus.Info("SQLite 数据库初始化")
	return s.createTables()
}

func (s *SQLiteDatabase) createTables() error {
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
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteDatabase) Close() error {
	logrus.Info("关闭 SQLite 数据库")
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteDatabase) GetDB() interface{} {
	return s.db
}

func (s *SQLiteDatabase) DBType() string {
	return "sqlite"
}

func (s *SQLiteDatabase) BatchInsertLogsForWebsite(websiteID string, logs []NginxLogRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

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

	for _, log := range logs {
		_, err = stmtNginx.Exec(
			log.IP, log.PageviewFlag, log.Timestamp.Unix(), log.Method, log.Url,
			log.Status, log.BytesSent, log.Referer, log.UserBrowser, log.UserOs, log.UserDevice,
			log.DomesticLocation, log.GlobalLocation,
		)
		if err != nil {
			return err
		}
	}

	return err
}

func (s *SQLiteDatabase) CleanOldLogs() error {
	cutoffTime := time.Now().AddDate(0, 0, -45).Unix()

	deletedCount := 0

	rows, err := s.db.Query(`
        SELECT name FROM sqlite_master 
        WHERE type='table' AND name LIKE '%_nginx_logs'
    `)
	if err != nil {
		return fmt.Errorf("查询表名失败: %v", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			logrus.WithError(err).Error("扫描表名失败")
			continue
		}
		tableNames = append(tableNames, tableName)
	}

	for _, tableName := range tableNames {
		result, err := s.db.Exec(
			fmt.Sprintf(`DELETE FROM "%s" WHERE timestamp < ?`, tableName), cutoffTime,
		)
		if err != nil {
			logrus.WithError(err).Errorf("清理表 %s 的旧日志失败", tableName)
			continue
		}

		count, _ := result.RowsAffected()
		deletedCount += int(count)
	}

	if deletedCount > 0 {
		logrus.Infof("删除了 %d 条45天前的日志记录", deletedCount)
		if _, err := s.db.Exec("VACUUM"); err != nil {
			logrus.WithError(err).Error("数据库压缩失败")
		}
	}
	return nil
}

// PostgreSQLDatabase PostgreSQL 数据库实现
type PostgreSQLDatabase struct {
	db  *sql.DB
	cfg *util.PostgreSQLConfig
}

func NewPostgreSQLDatabase(cfg *util.PostgreSQLConfig) (*PostgreSQLDatabase, error) {
	if cfg == nil {
		return nil, fmt.Errorf("PostgreSQL 配置不能为空")
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgreSQLDatabase{
		db:  db,
		cfg: cfg,
	}, nil
}

func (p *PostgreSQLDatabase) Init() error {
	logrus.Info("PostgreSQL 数据库初始化")
	return p.createTables()
}

func (p *PostgreSQLDatabase) createTables() error {
	common := `id SERIAL PRIMARY KEY,
	ip TEXT NOT NULL,
	pageview_flag INTEGER NOT NULL DEFAULT 0,
	timestamp BIGINT NOT NULL,
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
		if _, err := p.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (p *PostgreSQLDatabase) Close() error {
	logrus.Info("关闭 PostgreSQL 数据库")
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *PostgreSQLDatabase) GetDB() interface{} {
	return p.db
}

func (p *PostgreSQLDatabase) DBType() string {
	return "postgresql"
}

func (p *PostgreSQLDatabase) BatchInsertLogsForWebsite(websiteID string, logs []NginxLogRecord) error {
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	nginxTable := fmt.Sprintf("%s_nginx_logs", websiteID)

	stmt, err := tx.Prepare(pq.CopyIn(nginxTable,
		"ip", "pageview_flag", "timestamp", "method", "url",
		"status_code", "bytes_sent", "referer",
		"user_browser", "user_os", "user_device", "domestic_location", "global_location"))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, log := range logs {
		_, err = stmt.Exec(
			log.IP, log.PageviewFlag, log.Timestamp.Unix(), log.Method, log.Url,
			log.Status, log.BytesSent, log.Referer, log.UserBrowser, log.UserOs, log.UserDevice,
			log.DomesticLocation, log.GlobalLocation)
		if err != nil {
			return err
		}
	}

	_, err = stmt.Exec()
	return err
}

func (p *PostgreSQLDatabase) CleanOldLogs() error {
	cutoffTime := time.Now().AddDate(0, 0, -45).Unix()

	deletedCount := 0

	rows, err := p.db.Query(`
        SELECT tablename 
        FROM pg_tables 
        WHERE schemaname = 'public' AND tablename LIKE '%_nginx_logs'
    `)
	if err != nil {
		return fmt.Errorf("查询表名失败: %v", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			logrus.WithError(err).Error("扫描表名失败")
			continue
		}
		tableNames = append(tableNames, tableName)
	}

	for _, tableName := range tableNames {
		result, err := p.db.Exec(
			fmt.Sprintf(`DELETE FROM "%s" WHERE timestamp < $1`, tableName), cutoffTime,
		)
		if err != nil {
			logrus.WithError(err).Errorf("清理表 %s 的旧日志失败", tableName)
			continue
		}

		count, _ := result.RowsAffected()
		deletedCount += int(count)
	}

	if deletedCount > 0 {
		logrus.Infof("删除了 %d 条45天前的日志记录", deletedCount)
		if _, err := p.db.Exec("VACUUM"); err != nil {
			logrus.WithError(err).Error("数据库压缩失败")
		}
	}

	return nil
}
