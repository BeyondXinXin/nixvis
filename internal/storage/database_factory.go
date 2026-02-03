package storage

import (
	"fmt"
	"log"

	"github.com/beyondxinxin/nixvis/internal/util"
)

// NewDatabase 根据配置创建数据库实例
func NewDatabase(cfg *util.Config) (Database, error) {
	switch cfg.DBType {
	case "sqlite":
		log.Println("连接 SQLite 数据库")
		dataSourceName := fmt.Sprintf("%s/nixvis.db", util.DataDir)
		return NewSQLiteDatabase(dataSourceName)
	case "postgresql":
		log.Println("连接 PostgreSQL 数据库")
		return NewPostgreSQLDatabase(cfg.PostgreSQL)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", cfg.DBType)
	}
}
