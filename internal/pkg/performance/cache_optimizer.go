package performance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

// CacheConfig 缓存配置
type CacheConfig struct {
	// Redis配置
	Redis struct {
		// 主节点配置
		Master struct {
			Addr     string `yaml:"addr" json:"addr"`
			Password string `yaml:"password" json:"password"`
			DB       int    `yaml:"db" json:"db"`
		} `yaml:"master" json:"master"`

		// 从节点配置（用于读写分离）
		Slaves []struct {
			Addr     string `yaml:"addr" json:"addr"`
			Password string `yaml:"password" json:"password"`
			DB       int    `yaml:"db" json:"db"`
			Weight   int    `yaml:"weight" json:"weight"`
		} `yaml:"slaves" json:"slaves"`

		// 连接池配置
		Pool struct {
			PoolSize        int           `yaml:"pool_size" json:"pool_size"`
			MinIdleConns    int           `yaml:"min_idle_conns" json:"min_idle_conns"`
			MaxRetries      int           `yaml:"max_retries" json:"max_retries"`
			DialTimeout     time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
			ReadTimeout     time.Duration `yaml:"read_timeout" json:"read_timeout"`
			WriteTimeout    time.Duration `yaml:"write_timeout" json:"write_timeout"`
			ConnMaxIdleTime time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
		} `yaml:"pool" json:"pool"`
	} `yaml:"redis" json:"redis"`

	// 缓存策略配置
	Strategy struct {
		// 默认TTL配置
		DefaultTTL map[string]time.Duration `yaml:"default_ttl" json:"default_ttl"`

		// 多级缓存配置
		Levels []struct {
			Name      string        `yaml:"name" json:"name"`
			Type      string        `yaml:"type" json:"type"` // memory, redis
			MaxSize   int           `yaml:"max_size" json:"max_size"`
			TTL       time.Duration `yaml:"ttl" json:"ttl"`
			Algorithm string        `yaml:"algorithm" json:"algorithm"` // LRU, LFU, FIFO
		} `yaml:"levels" json:"levels"`

		// 缓存预热配置
		Warmup struct {
			Enabled   bool          `yaml:"enabled" json:"enabled"`
			Keys      []string      `yaml:"keys" json:"keys"`
			Interval  time.Duration `yaml:"interval" json:"interval"`
			BatchSize int           `yaml:"batch_size" json:"batch_size"`
		} `yaml:"warmup" json:"warmup"`

		// 缓存压缩配置
		Compression struct {
			Enabled   bool   `yaml:"enabled" json:"enabled"`
			Algorithm string `yaml:"algorithm" json:"algorithm"` // gzip, snappy
			MinSize   int    `yaml:"min_size" json:"min_size"`
			Level     int    `yaml:"level" json:"level"`
		} `yaml:"compression" json:"compression"`
	} `yaml:"strategy" json:"strategy"`

	// 监控配置
	Monitoring struct {
		Enabled       bool          `yaml:"enabled" json:"enabled"`
		MetricsPrefix string        `yaml:"metrics_prefix" json:"metrics_prefix"`
		Interval      time.Duration `yaml:"interval" json:"interval"`
		Thresholds    struct {
			HitRatio    float64       `yaml:"hit_ratio" json:"hit_ratio"`
			Latency     time.Duration `yaml:"latency" json:"latency"`
			ErrorRate   float64       `yaml:"error_rate" json:"error_rate"`
			MemoryUsage float64       `yaml:"memory_usage" json:"memory_usage"`
		} `yaml:"thresholds" json:"thresholds"`
	} `yaml:"monitoring" json:"monitoring"`
}

// DefaultCacheConfig 返回默认缓存配置
func DefaultCacheConfig() *CacheConfig {
	config := &CacheConfig{}

	// Redis默认配置
	config.Redis.Master.Addr = "127.0.0.1:6379"
	config.Redis.Master.DB = 0
	config.Redis.Pool.PoolSize = 100
	config.Redis.Pool.MinIdleConns = 10
	config.Redis.Pool.MaxRetries = 3
	config.Redis.Pool.DialTimeout = 5 * time.Second
	config.Redis.Pool.ReadTimeout = 3 * time.Second
	config.Redis.Pool.WriteTimeout = 3 * time.Second
	config.Redis.Pool.ConnMaxIdleTime = 5 * time.Minute

	// 默认TTL配置
	config.Strategy.DefaultTTL = map[string]time.Duration{
		"user":    30 * time.Minute,
		"session": 24 * time.Hour,
		"config":  6 * time.Hour,
		"temp":    5 * time.Minute,
	}

	// 多级缓存配置
	config.Strategy.Levels = []struct {
		Name      string        `yaml:"name" json:"name"`
		Type      string        `yaml:"type" json:"type"`
		MaxSize   int           `yaml:"max_size" json:"max_size"`
		TTL       time.Duration `yaml:"ttl" json:"ttl"`
		Algorithm string        `yaml:"algorithm" json:"algorithm"`
	}{
		{
			Name:      "L1",
			Type:      "memory",
			MaxSize:   1000,
			TTL:       5 * time.Minute,
			Algorithm: "LRU",
		},
		{
			Name:      "L2",
			Type:      "redis",
			MaxSize:   100000,
			TTL:       30 * time.Minute,
			Algorithm: "LRU",
		},
	}

	// 缓存预热配置
	config.Strategy.Warmup.Enabled = true
	config.Strategy.Warmup.Keys = []string{"config:*", "user:active:*"}
	config.Strategy.Warmup.Interval = 1 * time.Hour
	config.Strategy.Warmup.BatchSize = 100

	// 缓存压缩配置
	config.Strategy.Compression.Enabled = true
	config.Strategy.Compression.Algorithm = "gzip"
	config.Strategy.Compression.MinSize = 1024 // 1KB
	config.Strategy.Compression.Level = 6

	// 监控配置
	config.Monitoring.Enabled = true
	config.Monitoring.MetricsPrefix = "cache"
	config.Monitoring.Interval = 30 * time.Second
	config.Monitoring.Thresholds.HitRatio = 0.8 // 80%
	config.Monitoring.Thresholds.Latency = 10 * time.Millisecond
	config.Monitoring.Thresholds.ErrorRate = 0.01  // 1%
	config.Monitoring.Thresholds.MemoryUsage = 0.8 // 80%

	return config
}

// CacheOptimizer 缓存优化器
type CacheOptimizer struct {
	config       *CacheConfig
	logger       log.Logger
	masterClient redis.UniversalClient
	slaveClients []redis.UniversalClient
	l1Cache      *MemoryCache
	monitor      *CacheMonitor
	mu           sync.RWMutex
	stats        *CacheStats
}

// NewCacheOptimizer 创建缓存优化器
func NewCacheOptimizer(config *CacheConfig, logger log.Logger) *CacheOptimizer {
	if config == nil {
		config = DefaultCacheConfig()
	}

	optimizer := &CacheOptimizer{
		config:  config,
		logger:  logger,
		stats:   NewCacheStats(),
		monitor: NewCacheMonitor(config, logger),
	}

	// 初始化内存缓存
	if len(config.Strategy.Levels) > 0 && config.Strategy.Levels[0].Type == "memory" {
		l1Config := config.Strategy.Levels[0]
		optimizer.l1Cache = NewMemoryCache(l1Config.MaxSize, l1Config.TTL, l1Config.Algorithm)
	}

	return optimizer
}

// Initialize 初始化缓存优化器
func (co *CacheOptimizer) Initialize() error {
	// 初始化主Redis客户端
	masterOptions := &redis.Options{
		Addr:            co.config.Redis.Master.Addr,
		Password:        co.config.Redis.Master.Password,
		DB:              co.config.Redis.Master.DB,
		PoolSize:        co.config.Redis.Pool.PoolSize,
		MinIdleConns:    co.config.Redis.Pool.MinIdleConns,
		MaxRetries:      co.config.Redis.Pool.MaxRetries,
		DialTimeout:     co.config.Redis.Pool.DialTimeout,
		ReadTimeout:     co.config.Redis.Pool.ReadTimeout,
		WriteTimeout:    co.config.Redis.Pool.WriteTimeout,
		ConnMaxIdleTime: co.config.Redis.Pool.ConnMaxIdleTime,
	}

	co.masterClient = redis.NewClient(masterOptions)

	// 测试主连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := co.masterClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to master redis: %w", err)
	}

	// 初始化从Redis客户端
	for _, slaveConfig := range co.config.Redis.Slaves {
		slaveOptions := &redis.Options{
			Addr:            slaveConfig.Addr,
			Password:        slaveConfig.Password,
			DB:              slaveConfig.DB,
			PoolSize:        co.config.Redis.Pool.PoolSize,
			MinIdleConns:    co.config.Redis.Pool.MinIdleConns,
			MaxRetries:      co.config.Redis.Pool.MaxRetries,
			DialTimeout:     co.config.Redis.Pool.DialTimeout,
			ReadTimeout:     co.config.Redis.Pool.ReadTimeout,
			WriteTimeout:    co.config.Redis.Pool.WriteTimeout,
			ConnMaxIdleTime: co.config.Redis.Pool.ConnMaxIdleTime,
		}

		slaveClient := redis.NewClient(slaveOptions)

		// 测试从连接
		if err := slaveClient.Ping(ctx).Err(); err != nil {
			log.NewHelper(co.logger).Warnf("Failed to connect to slave redis %s: %v", slaveConfig.Addr, err)
			continue
		}

		co.slaveClients = append(co.slaveClients, slaveClient)
	}

	log.NewHelper(co.logger).Infof("Cache optimizer initialized with %d slave(s)", len(co.slaveClients))
	return nil
}

// Get 多级缓存获取
func (co *CacheOptimizer) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	defer func() {
		co.stats.RecordOperation("get", time.Since(start))
	}()

	// L1缓存查找
	if co.l1Cache != nil {
		if value, found := co.l1Cache.Get(key); found {
			co.stats.RecordHit("L1")
			return value, nil
		}
		co.stats.RecordMiss("L1")
	}

	// L2缓存（Redis）查找
	client := co.getReadClient()
	value, err := client.Get(ctx, co.buildKey(key)).Result()

	if err == redis.Nil {
		co.stats.RecordMiss("L2")
		return "", fmt.Errorf("key not found: %s", key)
	}

	if err != nil {
		co.stats.RecordError("get")
		return "", fmt.Errorf("redis get error: %w", err)
	}

	co.stats.RecordHit("L2")

	// 回填L1缓存
	if co.l1Cache != nil {
		co.l1Cache.Set(key, value)
	}

	return value, nil
}

// Set 多级缓存设置
func (co *CacheOptimizer) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		co.stats.RecordOperation("set", time.Since(start))
	}()

	// 序列化值
	data, err := co.serialize(value)
	if err != nil {
		co.stats.RecordError("serialize")
		return fmt.Errorf("serialization error: %w", err)
	}

	// 压缩处理
	if co.config.Strategy.Compression.Enabled && len(data) > co.config.Strategy.Compression.MinSize {
		data, err = co.compress(data)
		if err != nil {
			log.NewHelper(co.logger).Warnf("Compression failed for key %s: %v", key, err)
		}
	}

	// 确定TTL
	if ttl == 0 {
		ttl = co.getDefaultTTL(key)
	}

	// 设置L1缓存
	if co.l1Cache != nil {
		co.l1Cache.Set(key, string(data))
	}

	// 设置L2缓存（Redis）
	client := co.getWriteClient()
	err = client.Set(ctx, co.buildKey(key), data, ttl).Err()
	if err != nil {
		co.stats.RecordError("set")
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (co *CacheOptimizer) Delete(ctx context.Context, keys ...string) error {
	start := time.Now()
	defer func() {
		co.stats.RecordOperation("delete", time.Since(start))
	}()

	// 删除L1缓存
	if co.l1Cache != nil {
		for _, key := range keys {
			co.l1Cache.Delete(key)
		}
	}

	// 删除L2缓存
	redisKeys := make([]string, len(keys))
	for i, key := range keys {
		redisKeys[i] = co.buildKey(key)
	}

	client := co.getWriteClient()
	err := client.Del(ctx, redisKeys...).Err()
	if err != nil {
		co.stats.RecordError("delete")
		return fmt.Errorf("redis delete error: %w", err)
	}

	return nil
}

// Warmup 缓存预热
func (co *CacheOptimizer) Warmup(ctx context.Context) error {
	if !co.config.Strategy.Warmup.Enabled {
		return nil
	}

	helper := log.NewHelper(co.logger)
	helper.Info("Starting cache warmup")

	client := co.getReadClient()

	for _, pattern := range co.config.Strategy.Warmup.Keys {
		keys, err := client.Keys(ctx, pattern).Result()
		if err != nil {
			helper.Errorf("Failed to get keys for pattern %s: %v", pattern, err)
			continue
		}

		// 批量预热
		for i := 0; i < len(keys); i += co.config.Strategy.Warmup.BatchSize {
			end := i + co.config.Strategy.Warmup.BatchSize
			if end > len(keys) {
				end = len(keys)
			}

			batch := keys[i:end]
			values, err := client.MGet(ctx, batch...).Result()
			if err != nil {
				helper.Errorf("Failed to warmup batch %v: %v", batch, err)
				continue
			}

			// 加载到L1缓存
			if co.l1Cache != nil {
				for j, value := range values {
					if value != nil {
						if strValue, ok := value.(string); ok {
							cleanKey := strings.TrimPrefix(batch[j], co.getKeyPrefix())
							co.l1Cache.Set(cleanKey, strValue)
						}
					}
				}
			}
		}

		helper.Infof("Warmed up %d keys for pattern %s", len(keys), pattern)
	}

	helper.Info("Cache warmup completed")
	return nil
}

// StartMonitoring 启动缓存监控
func (co *CacheOptimizer) StartMonitoring(ctx context.Context) error {
	if !co.config.Monitoring.Enabled {
		return nil
	}

	return co.monitor.Start(ctx, co.masterClient, co.slaveClients, co.stats)
}

// GetStats 获取缓存统计信息
func (co *CacheOptimizer) GetStats() *CacheStats {
	co.mu.RLock()
	defer co.mu.RUnlock()
	return co.stats.Clone()
}

// 辅助方法

// getReadClient 获取读客户端（优先使用从库）
func (co *CacheOptimizer) getReadClient() redis.UniversalClient {
	if len(co.slaveClients) > 0 {
		// 简单轮询策略
		index := int(time.Now().UnixNano()) % len(co.slaveClients)
		return co.slaveClients[index]
	}
	return co.masterClient
}

// getWriteClient 获取写客户端（始终使用主库）
func (co *CacheOptimizer) getWriteClient() redis.UniversalClient {
	return co.masterClient
}

// buildKey 构建缓存键
func (co *CacheOptimizer) buildKey(key string) string {
	return co.getKeyPrefix() + key
}

// getKeyPrefix 获取键前缀
func (co *CacheOptimizer) getKeyPrefix() string {
	return "app:cache:"
}

// getDefaultTTL 获取默认TTL
func (co *CacheOptimizer) getDefaultTTL(key string) time.Duration {
	for prefix, ttl := range co.config.Strategy.DefaultTTL {
		if strings.HasPrefix(key, prefix) {
			return ttl
		}
	}
	return 1 * time.Hour // 默认1小时
}

// serialize 序列化值
func (co *CacheOptimizer) serialize(value interface{}) ([]byte, error) {
	return json.Marshal(value)
}

// compress 压缩数据
func (co *CacheOptimizer) compress(data []byte) ([]byte, error) {
	// 这里实现压缩逻辑，简化示例
	return data, nil
}

// generateCacheKey 生成缓存键
func (co *CacheOptimizer) generateCacheKey(parts ...string) string {
	combined := strings.Join(parts, ":")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:8]) // 使用前8字节作为短哈希
}
