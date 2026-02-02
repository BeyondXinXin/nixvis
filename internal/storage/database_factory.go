package storage

import (
	"fmt"
	"log"

	"github.com/beyondxinxin/nixvis/internal/util"
)

// NewDatabase 根据配置创建数据库实例
func NewDatabase(cfg *util.PostgreSQLConfig) (Database, error) {
	if cfg != nil {
		log.Println("尝试连接 PostgreSQL 数据库...")

		db, err := NewPostgreSQLDatabase(cfg)
		if err == nil {
			log.Printf("PostgreSQL 数据库连接成功: host=%s port=%d database=%s", cfg.Host, cfg.Port, cfg.Database)
			return db, nil
		}

		log.Printf("PostgreSQL 数据库连接失败: %v", err)
		log.Println("回退到 SQLite 数据库")
	}

	log.Println("使用 SQLite 数据库（默认）")
	dataSourceName := fmt.Sprintf("%s/nixvis.db", util.DataDir)
	return NewSQLiteDatabase(dataSourceName)
}
