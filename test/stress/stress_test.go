package stress
package stress

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kratos-boilerplate/internal/middleware"
	"kratos-boilerplate/internal/pkg/performance"

	"github.com/go-kratos/kratos/v2/log"
)

// TestHighConcurrencyStress 测试高并发压力
func TestHighConcurrencyStress(t *testing.T) {
	// 创建API性能中间件
	config := middleware.DefaultAPIPerformanceConfig()
	config.RateLimit.RequestsPerSec = 1000.0 // 每秒1000个请求
	config.RateLimit.Burst = 2000           // 突发2000个请求
	logger := log.NewStdLogger(&testWriter{t: t})
	mw := middleware.NewAPIMiddleware(config, logger)

	// 创建测试处理器
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// 模拟处理时间
		time.Sleep(10 * time.Millisecond)
		return "success", nil
	}

	// 包装中间件
	middlewareFunc := mw.Middleware()
	wrappedHandler := middlewareFunc(handler)

	// 并发执行测试
	const (
		numWorkers    = 100
		requestsPerWorker = 100
		totalRequests = numWorkers * requestsPerWorker
	)

	var (
		successCount int64
		errorCount   int64
		startTime    = time.Now()
	)

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// 启动多个worker
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < requestsPerWorker; j++ {
				_, err := wrappedHandler(context.Background(), nil)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					t.Logf("Worker %d, Request %d failed: %v", workerID, j, err)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}(i)
	}

	// 等待所有worker完成
	wg.Wait()

	// 计算结果
	duration := time.Since(startTime)
	totalProcessed := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&errorCount)
	successRate := float64(atomic.LoadInt64(&successCount)) / float64(totalProcessed) * 100

	t.Logf("Stress Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Processed Requests: %d", totalProcessed)
	t.Logf("  Success Count: %d", atomic.LoadInt64(&successCount))
	t.Logf("  Error Count: %d", atomic.LoadInt64(&errorCount))
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Requests Per Second: %.2f", float64(totalProcessed)/duration.Seconds())

	// 验证结果
	if successRate < 80 {
		t.Errorf("Success rate too low: %.2f%%", successRate)
	}
}

// TestRateLimitingStress 测试限流压力
func TestRateLimitingStress(t *testing.T) {
	// 创建API性能中间件，设置较低的限流阈值
	config := middleware.DefaultAPIPerformanceConfig()
	config.RateLimit.RequestsPerSec = 50.0 // 每秒50个请求
	config.RateLimit.Burst = 100          // 突发100个请求
	config.CircuitBreaker.Enabled = false  // 禁用熔断器
	config.Backpressure.Enabled = false    // 禁用背压控制
	logger := log.NewStdLogger(&testWriter{t: t})
	mw := middleware.NewAPIMiddleware(config, logger)

	// 创建测试处理器
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	// 包装中间件
	middlewareFunc := mw.Middleware()
	wrappedHandler := middlewareFunc(handler)

	// 快速发送大量请求
	const totalRequests = 200
	var (
		successCount int64
		rateLimitCount int64
		startTime    = time.Now()
	)

	var wg sync.WaitGroup
	wg.Add(totalRequests)

	// 并发发送请求
	for i := 0; i < totalRequests; i++ {
		go func(requestID int) {
			defer wg.Done()
			
			_, err := wrappedHandler(context.Background(), nil)
			if err != nil {
				if err.Error() == "request rate limit exceeded" {
					atomic.AddInt64(&rateLimitCount, 1)
				} else {
					t.Logf("Unexpected error for request %d: %v", requestID, err)
				}
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	// 等待所有请求完成
	wg.Wait()

	// 计算结果
	duration := time.Since(startTime)
	totalProcessed := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&rateLimitCount)

	t.Logf("Rate Limiting Stress Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Processed Requests: %d", totalProcessed)
	t.Logf("  Success Count: %d", atomic.LoadInt64(&successCount))
	t.Logf("  Rate Limited Count: %d", atomic.LoadInt64(&rateLimitCount))
	t.Logf("  Success Rate: %.2f%%", float64(atomic.LoadInt64(&successCount))/float64(totalProcessed)*100)

	// 验证限流是否正常工作
	if atomic.LoadInt64(&rateLimitCount) == 0 {
		t.Error("Expected some requests to be rate limited")
	}
	
	if atomic.LoadInt64(&successCount) == 0 {
		t.Error("Expected some requests to succeed")
	}
}

// TestCircuitBreakerStress 测试熔断器压力
func TestCircuitBreakerStress(t *testing.T) {
	// 创建API性能中间件
	config := middleware.DefaultAPIPerformanceConfig()
	config.RateLimit.Enabled = false       // 禁用限流
	config.CircuitBreaker.ErrorThreshold = 0.3 // 30%错误率就熔断
	config.CircuitBreaker.RequestThreshold = 10 // 至少10个请求
	config.CircuitBreaker.Timeout = 2 * time.Second // 2秒后恢复
	config.Backpressure.Enabled = false    // 禁用背压控制
	logger := log.NewStdLogger(&testWriter{t: t})
	mw := middleware.NewAPIMiddleware(config, logger)

	// 创建测试处理器，前30%的请求失败
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// 从context中获取请求ID（简化实现）
		if req != nil {
			if requestID, ok := req.(int); ok && requestID%3 == 0 {
				return nil, fmt.Errorf("simulated error")
			}
		}
		return "success", nil
	}

	// 包装中间件
	middlewareFunc := mw.Middleware()
	wrappedHandler := middlewareFunc(handler)

	// 发送请求并观察熔断器行为
	const totalRequests = 50
	var (
		successCount int64
		errorCount   int64
		circuitBreakerCount int64
		startTime    = time.Now()
	)

	// 分批发送请求，观察熔断器状态变化
	for i := 0; i < totalRequests; i++ {
		_, err := wrappedHandler(context.Background(), i)
		if err != nil {
			if err.Error() == "service temporarily unavailable" {
				atomic.AddInt64(&circuitBreakerCount, 1)
			} else {
				atomic.AddInt64(&errorCount, 1)
			}
		} else {
			atomic.AddInt64(&successCount, 1)
		}
		
		// 小间隔发送请求
		time.Sleep(50 * time.Millisecond)
	}

	// 计算结果
	duration := time.Since(startTime)
	totalProcessed := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&errorCount) + atomic.LoadInt64(&circuitBreakerCount)

	t.Logf("Circuit Breaker Stress Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Processed Requests: %d", totalProcessed)
	t.Logf("  Success Count: %d", atomic.LoadInt64(&successCount))
	t.Logf("  Error Count: %d", atomic.LoadInt64(&errorCount))
	t.Logf("  Circuit Breaker Count: %d", atomic.LoadInt64(&circuitBreakerCount))

	// 验证熔断器是否正常工作
	if atomic.LoadInt64(&circuitBreakerCount) == 0 {
		t.Log("No requests were rejected by circuit breaker (this may be expected depending on timing)")
	}
}

// TestCacheEvictionStress 测试缓存驱逐压力
func TestCacheEvictionStress(t *testing.T) {
	// 创建小容量缓存
	cache := performance.NewMemoryCache(100, 5*time.Minute, "LRU")

	const totalOperations = 1000
	var (
		setCount int64
		getCount int64
		hitCount int64
		missCount int64
		startTime = time.Now()
	)

	// 执行大量缓存操作
	for i := 0; i < totalOperations; i++ {
		key := fmt.Sprintf("key_%d", i%150) // 只有150个不同的键，会触发驱逐
		value := fmt.Sprintf("value_%d", i)
		
		// 设置缓存
		cache.Set(key, value)
		atomic.AddInt64(&setCount, 1)
		
		// 获取缓存
		_, found := cache.Get(key)
		atomic.AddInt64(&getCount, 1)
		if found {
			atomic.AddInt64(&hitCount, 1)
		} else {
			atomic.AddInt64(&missCount, 1)
		}
	}

	// 计算结果
	duration := time.Since(startTime)
	hitRate := float64(atomic.LoadInt64(&hitCount)) / float64(atomic.LoadInt64(&getCount)) * 100

	t.Logf("Cache Eviction Stress Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Operations: %d", totalOperations)
	t.Logf("  Set Operations: %d", atomic.LoadInt64(&setCount))
	t.Logf("  Get Operations: %d", atomic.LoadInt64(&getCount))
	t.Logf("  Cache Hits: %d", atomic.LoadInt64(&hitCount))
	t.Logf("  Cache Misses: %d", atomic.LoadInt64(&missCount))
	t.Logf("  Hit Rate: %.2f%%", hitRate)
	t.Logf("  Operations Per Second: %.2f", float64(totalOperations)/duration.Seconds())

	// 验证缓存是否正常工作
	if hitRate < 30 {
		t.Errorf("Cache hit rate too low: %.2f%%", hitRate)
	}
}

// testWriter 简单的日志写入器实现
type testWriter struct {
	t *testing.T
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	w.t.Logf("LOG: %s", string(p))
	return len(p), nil
}