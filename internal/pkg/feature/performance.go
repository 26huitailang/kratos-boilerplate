package feature

import (
	"context"
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// CacheEntry 缓存条目
type CacheEntry struct {
	Value      interface{} `json:"value"`
	Timestamp  time.Time   `json:"timestamp"`
	TTL        time.Duration `json:"ttl"`
	AccessCount int64      `json:"access_count"`
	LastAccess time.Time   `json:"last_access"`
}

// IsExpired 检查是否过期
func (ce *CacheEntry) IsExpired() bool {
	if ce.TTL <= 0 {
		return false // 永不过期
	}
	return time.Since(ce.Timestamp) > ce.TTL
}

// Touch 更新访问时间
func (ce *CacheEntry) Touch() {
	ce.AccessCount++
	ce.LastAccess = time.Now()
}

// EvaluationCache 评估结果缓存
type EvaluationCache struct {
	mu           sync.RWMutex
	cache        map[string]*CacheEntry
	maxSize      int
	defaultTTL   time.Duration
	cleanupTicker *time.Ticker
	logger       *log.Helper
	enabled      bool
	
	// 统计信息
	hitCount  int64
	missCount int64
	evictions int64
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled    bool          `yaml:"enabled" json:"enabled"`
	MaxSize    int           `yaml:"max_size" json:"max_size"`
	DefaultTTL time.Duration `yaml:"default_ttl" json:"default_ttl"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
}

// NewEvaluationCache 创建评估缓存
func NewEvaluationCache(config *CacheConfig, logger log.Logger) *EvaluationCache {
	cache := &EvaluationCache{
		cache:      make(map[string]*CacheEntry),
		maxSize:    1000,
		defaultTTL: 5 * time.Minute,
		logger:     log.NewHelper(logger),
		enabled:    true,
	}
	
	if config != nil {
		cache.enabled = config.Enabled
		if config.MaxSize > 0 {
			cache.maxSize = config.MaxSize
		}
		if config.DefaultTTL > 0 {
			cache.defaultTTL = config.DefaultTTL
		}
		
		cleanupInterval := config.CleanupInterval
		if cleanupInterval <= 0 {
			cleanupInterval = time.Minute
		}
		
		// 启动定期清理
		cache.cleanupTicker = time.NewTicker(cleanupInterval)
		go cache.cleanup()
	}
	
	return cache
}

// Get 获取缓存值
func (ec *EvaluationCache) Get(key string) (interface{}, bool) {
	if !ec.enabled {
		return nil, false
	}
	
	ec.mu.RLock()
	entry, exists := ec.cache[key]
	ec.mu.RUnlock()
	
	if !exists {
		ec.mu.Lock()
		ec.missCount++
		ec.mu.Unlock()
		return nil, false
	}
	
	if entry.IsExpired() {
		ec.mu.Lock()
		delete(ec.cache, key)
		ec.missCount++
		ec.mu.Unlock()
		return nil, false
	}
	
	ec.mu.Lock()
	entry.Touch()
	ec.hitCount++
	value := entry.Value
	ec.mu.Unlock()
	
	return value, true
}

// Set 设置缓存值
func (ec *EvaluationCache) Set(key string, value interface{}, ttl time.Duration) {
	if !ec.enabled {
		return
	}
	
	if ttl <= 0 {
		ttl = ec.defaultTTL
	}
	
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	// 检查缓存大小限制
	if len(ec.cache) >= ec.maxSize {
		ec.evictLRU()
	}
	
	ec.cache[key] = &CacheEntry{
		Value:       value,
		Timestamp:   time.Now(),
		TTL:         ttl,
		AccessCount: 1,
		LastAccess:  time.Now(),
	}
}

// Delete 删除缓存值
func (ec *EvaluationCache) Delete(key string) {
	if !ec.enabled {
		return
	}
	
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	delete(ec.cache, key)
}

// Clear 清空缓存
func (ec *EvaluationCache) Clear() {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	ec.cache = make(map[string]*CacheEntry)
	ec.hitCount = 0
	ec.missCount = 0
	ec.evictions = 0
}

// GetStats 获取缓存统计
func (ec *EvaluationCache) GetStats() map[string]interface{} {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	totalRequests := ec.hitCount + ec.missCount
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(ec.hitCount) / float64(totalRequests)
	}
	
	return map[string]interface{}{
		"enabled":       ec.enabled,
		"size":          len(ec.cache),
		"max_size":      ec.maxSize,
		"hit_count":     ec.hitCount,
		"miss_count":    ec.missCount,
		"hit_rate":      hitRate,
		"evictions":     ec.evictions,
		"total_requests": totalRequests,
	}
}

// SetEnabled 设置启用状态
func (ec *EvaluationCache) SetEnabled(enabled bool) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	ec.enabled = enabled
	if !enabled {
		ec.cache = make(map[string]*CacheEntry)
	}
}

// evictLRU 淘汰最少使用的条目
func (ec *EvaluationCache) evictLRU() {
	if len(ec.cache) == 0 {
		return
	}
	
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range ec.cache {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}
	
	if oldestKey != "" {
		delete(ec.cache, oldestKey)
		ec.evictions++
	}
}

// cleanup 定期清理过期条目
func (ec *EvaluationCache) cleanup() {
	for range ec.cleanupTicker.C {
		ec.mu.Lock()
		for key, entry := range ec.cache {
			if entry.IsExpired() {
				delete(ec.cache, key)
			}
		}
		ec.mu.Unlock()
	}
}

// Stop 停止缓存
func (ec *EvaluationCache) Stop() {
	if ec.cleanupTicker != nil {
		ec.cleanupTicker.Stop()
	}
}

// generateCacheKey 生成缓存键
func GenerateCacheKey(feature FeatureFlag, evalCtx *EvaluationContext) string {
	if evalCtx == nil {
		return string(feature)
	}
	
	// 创建包含上下文信息的键
	contextStr := fmt.Sprintf("%s_%s_%s_%s", 
		evalCtx.UserID, evalCtx.UserType, evalCtx.Environment, evalCtx.Version)
	
	// 添加属性信息
	for k, v := range evalCtx.Attributes {
		contextStr += fmt.Sprintf("_%s_%s", k, v)
	}
	
	// 使用MD5生成短键
	hash := md5.Sum([]byte(fmt.Sprintf("%s_%s", feature, contextStr)))
	return fmt.Sprintf("%x", hash)
}

// ObjectPool 对象池
type ObjectPool[T any] struct {
	pool    sync.Pool
	factory func() T
	reset   func(T)
}

// NewObjectPool 创建对象池
func NewObjectPool[T any](factory func() T, reset func(T)) *ObjectPool[T] {
	return &ObjectPool[T]{
		factory: factory,
		reset:   reset,
		pool: sync.Pool{
			New: func() interface{} {
				return factory()
			},
		},
	}
}

// Get 从池中获取对象
func (op *ObjectPool[T]) Get() T {
	return op.pool.Get().(T)
}

// Put 将对象放回池中
func (op *ObjectPool[T]) Put(obj T) {
	if op.reset != nil {
		op.reset(obj)
	}
	op.pool.Put(obj)
}

// 预定义的对象池

// ToggleConfigPool 配置对象池
var ToggleConfigPool = NewObjectPool(
	func() *ToggleConfig {
		return &ToggleConfig{
			Rules: make(map[string]interface{}),
			Tags:  make([]string, 0),
		}
	},
	func(config *ToggleConfig) {
		config.Enabled = false
		config.Strategy = ""
		config.Description = ""
		config.Capabilities = nil
		config.CreatedAt = time.Time{}
		config.UpdatedAt = time.Time{}
		
		// 清空但保留底层数组
		for k := range config.Rules {
			delete(config.Rules, k)
		}
		config.Tags = config.Tags[:0]
	},
)

// EvaluationContextPool 评估上下文对象池
var EvaluationContextPool = NewObjectPool(
	func() *EvaluationContext {
		return &EvaluationContext{
			Attributes: make(map[string]string),
		}
	},
	func(ctx *EvaluationContext) {
		ctx.UserID = ""
		ctx.UserType = ""
		ctx.Environment = ""
		ctx.Version = ""
		
		// 清空但保留底层map
		for k := range ctx.Attributes {
			delete(ctx.Attributes, k)
		}
	},
)

// ValidationResultPool 验证结果对象池
var ValidationResultPool = NewObjectPool(
	func() *ValidationResult {
		return &ValidationResult{
			Errors:   make([]ValidationIssue, 0),
			Warnings: make([]ValidationIssue, 0),
			Infos:    make([]ValidationIssue, 0),
		}
	},
	func(result *ValidationResult) {
		result.Valid = true
		result.Errors = result.Errors[:0]
		result.Warnings = result.Warnings[:0]
		result.Infos = result.Infos[:0]
		result.Summary = ValidationSummary{}
		result.ValidatedAt = time.Time{}
	},
)

// CachedToggleManager 带缓存的功能开关管理器
type CachedToggleManager struct {
	*ToggleManager
	cache            *EvaluationCache
	configPool       *ObjectPool[*ToggleConfig]
	contextPool      *ObjectPool[*EvaluationContext]
	enableCacheWrite bool
}

// CachedManagerConfig 缓存管理器配置
type CachedManagerConfig struct {
	CacheConfig      *CacheConfig `yaml:"cache" json:"cache"`
	EnableCacheWrite bool         `yaml:"enable_cache_write" json:"enable_cache_write"`
}

// NewCachedToggleManager 创建带缓存的功能开关管理器
func NewCachedToggleManager(
	manager *ToggleManager,
	config *CachedManagerConfig,
	logger log.Logger,
) *CachedToggleManager {
	var cacheConfig *CacheConfig
	enableCacheWrite := true
	
	if config != nil {
		cacheConfig = config.CacheConfig
		enableCacheWrite = config.EnableCacheWrite
	}
	
	return &CachedToggleManager{
		ToggleManager:    manager,
		cache:            NewEvaluationCache(cacheConfig, logger),
		configPool:       ToggleConfigPool,
		contextPool:      EvaluationContextPool,
		enableCacheWrite: enableCacheWrite,
	}
}

// IsEnabledWithContext 带缓存的上下文检查
func (ctm *CachedToggleManager) IsEnabledWithContext(ctx context.Context, flag FeatureFlag, evalCtx *EvaluationContext) bool {
	// 生成缓存键
	cacheKey := GenerateCacheKey(flag, evalCtx)
	
	// 尝试从缓存获取
	if cached, found := ctm.cache.Get(cacheKey); found {
		if result, ok := cached.(bool); ok {
			return result
		}
	}
	
	// 从原管理器获取结果
	result := ctm.ToggleManager.IsEnabledWithContext(ctx, flag, evalCtx)
	
	// 写入缓存
	if ctm.enableCacheWrite {
		ctm.cache.Set(cacheKey, result, 0) // 使用默认TTL
	}
	
	return result
}

// GetToggleConfig 带对象池的配置获取
func (ctm *CachedToggleManager) GetToggleConfig(flag FeatureFlag) (*ToggleConfig, error) {
	// 从原管理器获取配置
	config, err := ctm.ToggleManager.GetToggleConfig(flag)
	if err != nil {
		return nil, err
	}
	
	// 从对象池获取新配置对象
	pooledConfig := ctm.configPool.Get()
	
	// 复制配置数据
	*pooledConfig = *config
	
	// 深复制需要的字段
	if config.Rules != nil {
		pooledConfig.Rules = make(map[string]interface{})
		for k, v := range config.Rules {
			pooledConfig.Rules[k] = v
		}
	}
	
	if config.Tags != nil {
		pooledConfig.Tags = make([]string, len(config.Tags))
		copy(pooledConfig.Tags, config.Tags)
	}
	
	if config.Capabilities != nil {
		pooledConfig.Capabilities = &FeatureCapabilities{
			Permissions: make([]string, len(config.Capabilities.Permissions)),
			AuditLogs:   make([]string, len(config.Capabilities.AuditLogs)),
			APIs:        make([]string, len(config.Capabilities.APIs)),
			Routes:      make([]string, len(config.Capabilities.Routes)),
			Configs:     make([]string, len(config.Capabilities.Configs)),
		}
		copy(pooledConfig.Capabilities.Permissions, config.Capabilities.Permissions)
		copy(pooledConfig.Capabilities.AuditLogs, config.Capabilities.AuditLogs)
		copy(pooledConfig.Capabilities.APIs, config.Capabilities.APIs)
		copy(pooledConfig.Capabilities.Routes, config.Capabilities.Routes)
		copy(pooledConfig.Capabilities.Configs, config.Capabilities.Configs)
	}
	
	return pooledConfig, nil
}

// ReturnToggleConfig 将配置对象返回到池中
func (ctm *CachedToggleManager) ReturnToggleConfig(config *ToggleConfig) {
	if config != nil {
		ctm.configPool.Put(config)
	}
}

// UpdateToggle 更新配置并清理相关缓存
func (ctm *CachedToggleManager) UpdateToggle(flag FeatureFlag, config *ToggleConfig) error {
	err := ctm.ToggleManager.UpdateToggle(flag, config)
	if err != nil {
		return err
	}
	
	// 清理相关缓存条目
	ctm.invalidateFeatureCache(flag)
	
	return nil
}

// DeleteToggle 删除配置并清理相关缓存
func (ctm *CachedToggleManager) DeleteToggle(flag FeatureFlag) error {
	err := ctm.ToggleManager.DeleteToggle(flag)
	if err != nil {
		return err
	}
	
	// 清理相关缓存条目
	ctm.invalidateFeatureCache(flag)
	
	return nil
}

// invalidateFeatureCache 清理功能相关的缓存
func (ctm *CachedToggleManager) invalidateFeatureCache(flag FeatureFlag) {
	// 简单实现：清空所有缓存
	// 在生产环境中，可以实现更精确的缓存失效策略
	ctm.cache.Clear()
}

// GetCacheStats 获取缓存统计
func (ctm *CachedToggleManager) GetCacheStats() map[string]interface{} {
	return ctm.cache.GetStats()
}

// SetCacheEnabled 设置缓存启用状态
func (ctm *CachedToggleManager) SetCacheEnabled(enabled bool) {
	ctm.cache.SetEnabled(enabled)
}

// Stop 停止缓存管理器
func (ctm *CachedToggleManager) Stop() {
	ctm.cache.Stop()
}

// PerformanceMonitor 性能监控器（轻量版本，仅用于基础性能跟踪）
type PerformanceMonitor struct {
	mu             sync.RWMutex
	evaluationTimes map[FeatureFlag]time.Duration
	lastUpdate     time.Time
	logger         *log.Helper
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor(logger log.Logger) *PerformanceMonitor {
	return &PerformanceMonitor{
		evaluationTimes: make(map[FeatureFlag]time.Duration),
		lastUpdate:     time.Now(),
		logger:         log.NewHelper(logger),
	}
}

// RecordEvaluation 记录评估性能（简化版本）
func (pm *PerformanceMonitor) RecordEvaluation(feature FeatureFlag, duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	// 简单记录最后一次评估时间
	pm.evaluationTimes[feature] = duration
	pm.lastUpdate = time.Now()
}

// GetLastEvaluationTime 获取最后一次评估时间
func (pm *PerformanceMonitor) GetLastEvaluationTime(feature FeatureFlag) time.Duration {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	return pm.evaluationTimes[feature]
}

// Clear 清理性能数据
func (pm *PerformanceMonitor) Clear() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	pm.evaluationTimes = make(map[FeatureFlag]time.Duration)
	pm.lastUpdate = time.Now()
}