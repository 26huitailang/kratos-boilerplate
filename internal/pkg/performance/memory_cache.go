package performance

import (
	"container/list"
	"sync"
	"time"
)

// MemoryCache 内存缓存实现
type MemoryCache struct {
	maxSize   int
	ttl       time.Duration
	algorithm string
	cache     map[string]*cacheItem
	list      *list.List
	mutex     sync.RWMutex
}

// cacheItem 缓存项
type cacheItem struct {
	key       string
	value     string
	expiredAt time.Time
	element   *list.Element
}

// NewMemoryCache 创建内存缓存
func NewMemoryCache(maxSize int, ttl time.Duration, algorithm string) *MemoryCache {
	return &MemoryCache{
		maxSize:   maxSize,
		ttl:       ttl,
		algorithm: algorithm,
		cache:     make(map[string]*cacheItem),
		list:      list.New(),
	}
}

// Get 获取缓存值
func (mc *MemoryCache) Get(key string) (string, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	item, exists := mc.cache[key]
	if !exists {
		return "", false
	}

	// 检查是否过期
	if time.Now().After(item.expiredAt) {
		mc.remove(key)
		return "", false
	}

	// LRU算法：移动到列表前端
	if mc.algorithm == "LRU" && item.element != nil {
		mc.list.MoveToFront(item.element)
	}

	return item.value, true
}

// Set 设置缓存值
func (mc *MemoryCache) Set(key, value string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// 检查是否已存在
	if item, exists := mc.cache[key]; exists {
		// 更新值和过期时间
		item.value = value
		item.expiredAt = time.Now().Add(mc.ttl)

		// LRU算法：移动到列表前端
		if mc.algorithm == "LRU" && item.element != nil {
			mc.list.MoveToFront(item.element)
		}
		return
	}

	// 检查容量限制
	if len(mc.cache) >= mc.maxSize {
		mc.evict()
	}

	// 创建新项
	item := &cacheItem{
		key:       key,
		value:     value,
		expiredAt: time.Now().Add(mc.ttl),
	}

	// LRU算法：添加到列表前端
	if mc.algorithm == "LRU" {
		item.element = mc.list.PushFront(key)
	}

	mc.cache[key] = item
}

// Delete 删除缓存项
func (mc *MemoryCache) Delete(key string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.remove(key)
}

// remove 内部删除方法
func (mc *MemoryCache) remove(key string) {
	if item, exists := mc.cache[key]; exists {
		// 从列表中移除
		if item.element != nil {
			mc.list.Remove(item.element)
		}
		// 从map中删除
		delete(mc.cache, key)
	}
}

// evict 淘汰缓存项
func (mc *MemoryCache) evict() {
	switch mc.algorithm {
	case "LRU":
		// 移除最久未使用的项
		if element := mc.list.Back(); element != nil {
			key := element.Value.(string)
			mc.remove(key)
		}
	case "FIFO":
		// 移除最早加入的项
		if element := mc.list.Back(); element != nil {
			key := element.Value.(string)
			mc.remove(key)
		}
	case "LFU":
		// 简化实现：移除随机项
		for key := range mc.cache {
			mc.remove(key)
			break
		}
	}
}

// Size 获取缓存大小
func (mc *MemoryCache) Size() int {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return len(mc.cache)
}

// Clear 清空缓存
func (mc *MemoryCache) Clear() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.cache = make(map[string]*cacheItem)
	mc.list = list.New()
}
