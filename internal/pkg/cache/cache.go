package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
)

// 错误定义
var (
	ErrKeyNotFound = errors.New("key not found")
	ErrLockFailed  = errors.New("failed to acquire lock")
)

// Cache 缓存接口
type Cache interface {
	// 基础操作
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	
	// 分布式锁
	Lock(ctx context.Context, key string, expiration time.Duration) (bool, error)
	Unlock(ctx context.Context, key, value string) error
	
	// 管理操作
	Ping(ctx context.Context) error
	Close() error
}

// Config 缓存配置
type Config struct {
	Addrs    []string      `yaml:"addrs"`
	Password string        `yaml:"password"`
	DB       int           `yaml:"db"`
	PoolSize int           `yaml:"pool_size"`
	Timeout  time.Duration `yaml:"timeout"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Addrs:    []string{"127.0.0.1:6379"},
		DB:       0,
		PoolSize: 10,
		Timeout:  5 * time.Second,
	}
}

// redisCache Redis缓存实现
type redisCache struct {
	client redis.UniversalClient
	config *Config
	logger log.Logger
}

// NewCache 创建缓存实例
func NewCache(config *Config, logger log.Logger) (Cache, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	opts := &redis.UniversalOptions{
		Addrs:    config.Addrs,
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	}
	
	client := redis.NewUniversalClient(opts)
	
	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}
	
	return &redisCache{
		client: client,
		config: config,
		logger: logger,
	}, nil
}

// Get 获取值
func (c *redisCache) Get(ctx context.Context, key string) (string, error) {
	result, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrKeyNotFound
		}
		return "", err
	}
	return result, nil
}

// Set 设置值
func (c *redisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	var serializedValue string
	
	switch v := value.(type) {
	case string:
		serializedValue = v
	case []byte:
		serializedValue = string(v)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to serialize value: %w", err)
		}
		serializedValue = string(data)
	}
	
	return c.client.Set(ctx, key, serializedValue, expiration).Err()
}

// Del 删除键
func (c *redisCache) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (c *redisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.client.Exists(ctx, keys...).Result()
}

// Lock 分布式锁
func (c *redisCache) Lock(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	result, err := c.client.SetNX(ctx, key, "locked", expiration).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

// Unlock 释放锁
func (c *redisCache) Unlock(ctx context.Context, key, value string) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`
	return c.client.Eval(ctx, script, []string{key}, value).Err()
}

// Ping 健康检查
func (c *redisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close 关闭连接
func (c *redisCache) Close() error {
	return c.client.Close()
}

// CacheManager 缓存管理器
type CacheManager struct {
	cache  Cache
	logger log.Logger
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(cache Cache, logger log.Logger) *CacheManager {
	return &CacheManager{
		cache:  cache,
		logger: logger,
	}
}

// GetWithCallback 带回调的获取
func (cm *CacheManager) GetWithCallback(ctx context.Context, key string, callback func() (interface{}, error), expiration time.Duration) (string, error) {
	// 先尝试从缓存获取
	value, err := cm.cache.Get(ctx, key)
	if err == nil {
		return value, nil
	}
	
	if err != ErrKeyNotFound {
		return "", err
	}
	
	// 缓存未命中，执行回调获取数据
	data, err := callback()
	if err != nil {
		return "", err
	}
	
	// 将数据存入缓存
	if err := cm.cache.Set(ctx, key, data, expiration); err != nil {
		cm.logger.Log(log.LevelWarn, "msg", "failed to set cache", "key", key, "error", err)
	}
	
	// 返回数据
	switch v := data.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		jsonData, err := json.Marshal(data)
		if err != nil {
			return "", fmt.Errorf("failed to marshal data: %w", err)
		}
		return string(jsonData), nil
	}
}

// DistributedLock 分布式锁
type DistributedLock struct {
	cache      Cache
	key        string
	value      string
	expiration time.Duration
}

// NewDistributedLock 创建分布式锁
func NewDistributedLock(cache Cache, key string, expiration time.Duration) *DistributedLock {
	return &DistributedLock{
		cache:      cache,
		key:        key,
		value:      fmt.Sprintf("lock-%d", time.Now().UnixNano()),
		expiration: expiration,
	}
}

// Acquire 获取锁
func (dl *DistributedLock) Acquire(ctx context.Context) error {
	acquired, err := dl.cache.Lock(ctx, dl.key, dl.expiration)
	if err != nil {
		return err
	}
	
	if !acquired {
		return ErrLockFailed
	}
	
	return nil
}

// Release 释放锁
func (dl *DistributedLock) Release(ctx context.Context) error {
	return dl.cache.Unlock(ctx, dl.key, dl.value)
}

// WithLock 带锁执行函数
func (dl *DistributedLock) WithLock(ctx context.Context, fn func() error) error {
	if err := dl.Acquire(ctx); err != nil {
		return err
	}
	defer dl.Release(ctx)
	
	return fn()
}