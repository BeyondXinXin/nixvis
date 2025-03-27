package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// StatPoint 通用的统计点数据结构
type StatPoint struct {
	PV      int   `json:"pv"`      // 页面浏览量
	UV      int   `json:"uv"`      // 独立访客数
	Traffic int64 `json:"traffic"` // 流量（字节）
}

// StatsResult 统计结果的基础接口
type StatsResult interface {
	GetType() string
}

// StatsQuery 统计查询的通用参数
type StatsQuery struct {
	WebsiteID  string
	TimeRange  string
	ViewType   string
	ExtraParam map[string]interface{}
}

// StatsManager 统计管理器接口
type StatsManager interface {
	Query(query StatsQuery) (StatsResult, error)
}

// StatsFactory 统计工厂，管理所有统计管理器
type StatsFactory struct {
	repo     *Repository
	managers map[string]StatsManager
	cache    *StatsCache
	mu       sync.RWMutex
}

// NewStatsFactory 创建新的统计工厂
func NewStatsFactory(repo *Repository) *StatsFactory {
	factory := &StatsFactory{
		repo:     repo,
		managers: make(map[string]StatsManager),
		cache:    NewStatsCache(),
	}

	// 注册默认的统计管理器
	factory.registerDefaultManagers()

	return factory
}

// registerDefaultManagers 注册默认的统计管理器
func (f *StatsFactory) registerDefaultManagers() {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 注册各种统计管理器
	f.managers["timeseries"] = NewTimeSeriesStatsManager(f.repo)
	f.managers["overall"] = NewOverallStatsManager(f.repo)

	f.managers["url"] = NewURLStatsManager(f.repo)
	f.managers["referer"] = NewrefererStatsManager(f.repo)

	f.managers["browser"] = NewBrowserStatsManager(f.repo)
	f.managers["os"] = NewOsStatsManager(f.repo)
	f.managers["device"] = NewDeviceStatsManager(f.repo)

	f.managers["location"] = NewLocationStatsManager(f.repo)
}

// GetManager 获取指定类型的统计管理器
func (f *StatsFactory) GetManager(managerType string) (StatsManager, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	manager, exists := f.managers[managerType]
	return manager, exists
}

// QueryStats 通过指定类型的管理器查询统计数据
func (f *StatsFactory) QueryStats(managerType string, query StatsQuery) (StatsResult, error) {
	// 构建缓存键
	cacheKey := f.buildCacheKey(managerType, query)

	// 尝试从缓存获取
	if cachedResult, ok := f.cache.Get(cacheKey, 5*time.Minute); ok {
		return cachedResult.(StatsResult), nil
	}

	// 获取对应的管理器
	manager, exists := f.GetManager(managerType)
	if !exists {
		return nil, fmt.Errorf("未找到统计管理器: %s", managerType)
	}

	// 执行查询
	result, err := manager.Query(query)
	if err != nil {
		return nil, err
	}

	// 缓存结果
	f.cache.Set(cacheKey, result)

	return result, nil
}

// buildCacheKey 构建缓存键
func (f *StatsFactory) buildCacheKey(managerType string, query StatsQuery) string {
	key := fmt.Sprintf("%s-%s-%s-%s", managerType, query.WebsiteID, query.TimeRange, query.ViewType)

	if query.ExtraParam != nil {
		if limit, ok := query.ExtraParam["limit"].(int); ok {
			key = fmt.Sprintf("%s-limit:%d", key, limit)
		}

		if locationType, ok := query.ExtraParam["locationType"].(string); ok {
			key = fmt.Sprintf("%s-locationType:%s", key, locationType)
		}
	}
	logrus.Info(key)

	return key
}
