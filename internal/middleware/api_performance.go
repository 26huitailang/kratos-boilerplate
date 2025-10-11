package middleware

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"golang.org/x/time/rate"
)

// APIPerformanceConfig API性能配置
type APIPerformanceConfig struct {
	// 限流配置
	RateLimit struct {
		Enabled        bool    `yaml:"enabled" json:"enabled"`
		RequestsPerSec float64 `yaml:"requests_per_sec" json:"requests_per_sec"` // 每秒请求数
		Burst          int     `yaml:"burst" json:"burst"`                       // 突发请求数
	} `yaml:"rate_limit" json:"rate_limit"`

	// 熔断器配置
	CircuitBreaker struct {
		Enabled          bool          `yaml:"enabled" json:"enabled"`
		ErrorThreshold   float64       `yaml:"error_threshold" json:"error_threshold"`     // 错误率阈值 (0.0-1.0)
		RequestThreshold int64         `yaml:"request_threshold" json:"request_threshold"` // 触发熔断的最小请求数
		Timeout          time.Duration `yaml:"timeout" json:"timeout"`                     // 熔断器超时时间
	} `yaml:"circuit_breaker" json:"circuit_breaker"`

	// 超时配置
	Timeout struct {
		Enabled  bool          `yaml:"enabled" json:"enabled"`
		Duration time.Duration `yaml:"duration" json:"duration"` // 请求超时时间
		Deadline time.Duration `yaml:"deadline" json:"deadline"` // 请求截止时间
		Buffer   time.Duration `yaml:"buffer" json:"buffer"`     // 缓冲时间
	} `yaml:"timeout" json:"timeout"`

	// 背压控制配置
	Backpressure struct {
		Enabled       bool          `yaml:"enabled" json:"enabled"`
		MaxQueueSize  int           `yaml:"max_queue_size" json:"max_queue_size"`   // 最大队列大小
		MaxWaitTime   time.Duration `yaml:"max_wait_time" json:"max_wait_time"`     // 最大等待时间
		RejectOnQueue bool          `yaml:"reject_on_queue" json:"reject_on_queue"` // 队列满时是否拒绝请求
	} `yaml:"backpressure" json:"backpressure"`
}

// DefaultAPIPerformanceConfig 默认API性能配置
func DefaultAPIPerformanceConfig() *APIPerformanceConfig {
	config := &APIPerformanceConfig{}

	// 限流默认配置
	config.RateLimit.Enabled = true
	config.RateLimit.RequestsPerSec = 100.0 // 每秒100个请求
	config.RateLimit.Burst = 200            // 突发200个请求

	// 熔断器默认配置
	config.CircuitBreaker.Enabled = true
	config.CircuitBreaker.ErrorThreshold = 0.5  // 50%错误率
	config.CircuitBreaker.RequestThreshold = 10 // 至少10个请求
	config.CircuitBreaker.Timeout = 30 * time.Second

	// 超时默认配置
	config.Timeout.Enabled = true
	config.Timeout.Duration = 30 * time.Second // 30秒超时
	config.Timeout.Deadline = 60 * time.Second // 60秒截止时间
	config.Timeout.Buffer = 5 * time.Second    // 5秒缓冲

	// 背压控制默认配置
	config.Backpressure.Enabled = true
	config.Backpressure.MaxQueueSize = 1000
	config.Backpressure.MaxWaitTime = 10 * time.Second
	config.Backpressure.RejectOnQueue = true

	return config
}

// APIMiddleware API性能中间件
type APIMiddleware struct {
	config         *APIPerformanceConfig
	logger         log.Logger
	rateLimiter    *rate.Limiter
	circuitBreaker *CircuitBreaker
	backpressure   *BackpressureController
}

// NewAPIMiddleware 创建API性能中间件
func NewAPIMiddleware(config *APIPerformanceConfig, logger log.Logger) *APIMiddleware {
	if config == nil {
		config = DefaultAPIPerformanceConfig()
	}

	mw := &APIMiddleware{
		config:       config,
		logger:       logger,
		backpressure: NewBackpressureController(config.Backpressure.MaxQueueSize, config.Backpressure.MaxWaitTime),
	}

	// 初始化限流器
	if config.RateLimit.Enabled {
		mw.rateLimiter = rate.NewLimiter(rate.Limit(config.RateLimit.RequestsPerSec), config.RateLimit.Burst)
	}

	// 初始化熔断器
	if config.CircuitBreaker.Enabled {
		mw.circuitBreaker = NewCircuitBreaker(
			config.CircuitBreaker.ErrorThreshold,
			config.CircuitBreaker.RequestThreshold,
			config.CircuitBreaker.Timeout,
		)
	}

	return mw
}

// Middleware 返回中间件函数
func (mw *APIMiddleware) Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			// 获取请求信息
			helper := log.NewHelper(mw.logger)

			// 1. 背压控制
			if mw.config.Backpressure.Enabled {
				if err := mw.backpressure.Acquire(ctx); err != nil {
					helper.Warnf("Request rejected due to backpressure: %v", err)
					return nil, fmt.Errorf("service temporarily unavailable due to high load")
				}
				defer mw.backpressure.Release()
			}

			// 2. 限流控制
			if mw.config.RateLimit.Enabled && mw.rateLimiter != nil {
				if !mw.rateLimiter.Allow() {
					helper.Warn("Request rate limited")
					return nil, fmt.Errorf("request rate limit exceeded")
				}
			}

			// 3. 熔断器检查
			if mw.config.CircuitBreaker.Enabled && mw.circuitBreaker != nil {
				if mw.circuitBreaker.IsOpen() {
					helper.Warn("Circuit breaker is open")
					return nil, fmt.Errorf("service temporarily unavailable")
				}
			}

			// 4. 超时控制
			var cancel context.CancelFunc
			if mw.config.Timeout.Enabled {
				// 创建带超时的上下文
				ctx, cancel = context.WithTimeout(ctx, mw.config.Timeout.Duration)
				defer cancel()
			}

			// 5. 执行请求
			start := time.Now()
			reply, err := handler(ctx, req)
			duration := time.Since(start)

			// 6. 熔断器状态更新
			if mw.config.CircuitBreaker.Enabled && mw.circuitBreaker != nil {
				if err != nil {
					mw.circuitBreaker.RecordFailure()
				} else {
					mw.circuitBreaker.RecordSuccess()
				}
			}

			// 7. 记录性能指标
			helper.Debugf("Request completed in %v", duration)

			return reply, err
		}
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	errorThreshold   float64
	requestThreshold int64
	timeout          time.Duration

	state        int32 // 0: closed, 1: open, 2: half-open
	lastFailure  time.Time
	successCount *atomic.Int64
	failureCount *atomic.Int64
	requestCount *atomic.Int64
	mutex        sync.RWMutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(errorThreshold float64, requestThreshold int64, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		errorThreshold:   errorThreshold,
		requestThreshold: requestThreshold,
		timeout:          timeout,
		successCount:     &atomic.Int64{},
		failureCount:     &atomic.Int64{},
		requestCount:     &atomic.Int64{},
	}
}

// IsOpen 检查熔断器是否打开
func (cb *CircuitBreaker) IsOpen() bool {
	state := atomic.LoadInt32(&cb.state)

	// 如果是半开状态，允许一个请求通过
	if state == 2 {
		return false
	}

	// 如果是打开状态，检查是否应该转为半开
	if state == 1 {
		cb.mutex.RLock()
		timeSinceFailure := time.Since(cb.lastFailure)
		cb.mutex.RUnlock()

		if timeSinceFailure >= cb.timeout {
			// 尝试转为半开状态
			if atomic.CompareAndSwapInt32(&cb.state, 1, 2) {
				// 重置计数器
				cb.successCount.Store(0)
				cb.failureCount.Store(0)
				cb.requestCount.Store(0)
			}
			return false
		}
		return true
	}

	// 检查是否应该打开熔断器
	requestCount := cb.requestCount.Load()
	if requestCount >= cb.requestThreshold {
		errorRate := float64(cb.failureCount.Load()) / float64(requestCount)
		if errorRate >= cb.errorThreshold {
			atomic.StoreInt32(&cb.state, 1)
			cb.mutex.Lock()
			cb.lastFailure = time.Now()
			cb.mutex.Unlock()
			return true
		}
	}

	return false
}

// RecordSuccess 记录成功请求
func (cb *CircuitBreaker) RecordSuccess() {
	cb.requestCount.Add(1)
	cb.successCount.Add(1)

	// 如果是半开状态且成功，转为关闭状态
	if atomic.LoadInt32(&cb.state) == 2 {
		atomic.StoreInt32(&cb.state, 0)
		// 重置计数器
		cb.successCount.Store(0)
		cb.failureCount.Store(0)
		cb.requestCount.Store(0)
	}
}

// RecordFailure 记录失败请求
func (cb *CircuitBreaker) RecordFailure() {
	cb.requestCount.Add(1)
	cb.failureCount.Add(1)

	// 如果是半开状态且失败，转为打开状态
	state := atomic.LoadInt32(&cb.state)
	if state == 2 {
		atomic.StoreInt32(&cb.state, 1)
		cb.mutex.Lock()
		cb.lastFailure = time.Now()
		cb.mutex.Unlock()
	}
}

// BackpressureController 背压控制器
type BackpressureController struct {
	maxQueueSize int
	maxWaitTime  time.Duration
	currentSize  *atomic.Int64
	mutex        sync.Mutex
	cond         *sync.Cond
}

// NewBackpressureController 创建背压控制器
func NewBackpressureController(maxQueueSize int, maxWaitTime time.Duration) *BackpressureController {
	bc := &BackpressureController{
		maxQueueSize: maxQueueSize,
		maxWaitTime:  maxWaitTime,
		currentSize:  &atomic.Int64{},
	}
	bc.cond = sync.NewCond(&bc.mutex)
	return bc
}

// Acquire 获取资源
func (bc *BackpressureController) Acquire(ctx context.Context) error {
	// 检查队列是否已满
	if bc.currentSize.Load() >= int64(bc.maxQueueSize) {
		return fmt.Errorf("request queue is full")
	}

	// 增加当前队列大小
	bc.currentSize.Add(1)
	return nil
}

// Release 释放资源
func (bc *BackpressureController) Release() {
	bc.currentSize.Add(-1)
	bc.cond.Broadcast()
}

// GetQueueSize 获取当前队列大小
func (bc *BackpressureController) GetQueueSize() int64 {
	return bc.currentSize.Load()
}
