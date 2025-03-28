package stats

import (
	"sync"
	"time"
)

// CacheItem 缓存项，包含数据和时间戳
type CacheItem struct {
	Data      interface{}
	Timestamp time.Time
}

// StatsCache 通用缓存组件
type StatsCache struct {
	items map[string]CacheItem
	mutex sync.RWMutex
}

// NewStatsCache 创建一个新的统计缓存
func NewStatsCache() *StatsCache {
	return &StatsCache{
		items: make(map[string]CacheItem),
	}
}

// Get 从缓存中获取数据，带过期检查
func (c *StatsCache) Get(key string, expiry time.Duration) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// 检查是否过期
	if time.Since(item.Timestamp) > expiry {
		return nil, false
	}

	return item.Data, true
}

// Set 添加数据到缓存
func (c *StatsCache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items[key] = CacheItem{
		Data:      value,
		Timestamp: time.Now(),
	}
}
