package storage

import (
	"database/sql"
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
	db Database
}

func NewRepository() (*Repository, error) {
	cfg := util.ReadConfig()

	db, err := NewDatabase(cfg.PostgreSQL)
	if err != nil {
		return nil, err
	}

	return &Repository{
		db: db,
	}, nil
}

// 初始化数据库
func (r *Repository) Init() error {
	return r.db.Init()
}

// 关闭数据库连接
func (r *Repository) Close() error {
	logrus.Info("关闭数据库")
	return r.db.Close()
}

// 获取数据库连接（返回底层 *sql.DB，不委托给 Database 接口）
func (r *Repository) GetDB() *sql.DB {
	return r.db.GetDB().(*sql.DB)
}

// DBType 返回数据库类型
func (r *Repository) DBType() string {
	return r.db.DBType()
}

// 为特定网站批量插入日志记录
func (r *Repository) BatchInsertLogsForWebsite(websiteID string, logs []NginxLogRecord) error {
	return r.db.BatchInsertLogsForWebsite(websiteID, logs)
}

// CleanOldLogs 清理45天前的日志数据
func (r *Repository) CleanOldLogs() error {
	return r.db.CleanOldLogs()
}
