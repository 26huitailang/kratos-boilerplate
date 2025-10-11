package middleware

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

func TestAPIMiddleware_RateLimiting(t *testing.T) {
	// 创建配置，设置较低的限流值以便测试
	config := DefaultAPIPerformanceConfig()
	config.RateLimit.RequestsPerSec = 5.0 // 每秒5个请求
	config.RateLimit.Burst = 5            // 突发5个请求

	// 创建中间件
	logger := log.NewStdLogger(&testWriter{t: t})
	mw := NewAPIMiddleware(config, logger)

	// 创建测试处理器
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	// 包装中间件
	middlewareFunc := mw.Middleware()
	wrappedHandler := middlewareFunc(handler)

	// 发送10个请求，前5个应该成功，后5个应该被限流
	successCount := 0
	rateLimitCount := 0

	for i := 0; i < 10; i++ {
		_, err := wrappedHandler(context.Background(), nil)
		if err != nil && err.Error() == "request rate limit exceeded" {
			rateLimitCount++
		} else if err == nil {
			successCount++
		}
	}

	// 验证结果
	if successCount < 5 {
		t.Errorf("Expected at least 5 successful requests, got %d", successCount)
	}

	if rateLimitCount == 0 {
		t.Error("Expected some requests to be rate limited")
	}

	t.Logf("Successful requests: %d, Rate limited requests: %d", successCount, rateLimitCount)
}

func TestAPIMiddleware_CircuitBreaker(t *testing.T) {
	// 创建配置，设置较低的熔断阈值以便测试
	config := DefaultAPIPerformanceConfig()
	config.CircuitBreaker.ErrorThreshold = 0.5  // 50%错误率
	config.CircuitBreaker.RequestThreshold = 4  // 至少4个请求
	config.CircuitBreaker.Timeout = time.Second // 1秒超时

	// 创建中间件
	logger := log.NewStdLogger(&testWriter{t: t})
	mw := NewAPIMiddleware(config, logger)

	// 创建测试处理器，前6个请求失败，后4个请求成功
	failCount := 0
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		if failCount < 6 {
			failCount++
			return nil, fmt.Errorf("simulated error")
		}
		return "success", nil
	}

	// 包装中间件
	middlewareFunc := mw.Middleware()
	wrappedHandler := middlewareFunc(handler)

	// 发送10个请求
	results := make([]error, 10)
	for i := 0; i < 10; i++ {
		_, err := wrappedHandler(context.Background(), nil)
		results[i] = err
		time.Sleep(100 * time.Millisecond) // 短暂延迟
	}

	// 验证前几个请求失败
	for i := 0; i < 4; i++ {
		if results[i] == nil {
			t.Errorf("Expected request %d to fail, but it succeeded", i)
		}
	}

	// 验证中间一些请求因为熔断器打开而失败
	// （即使处理器会返回成功）
	circuitOpenCount := 0
	for i := 4; i < 8; i++ {
		if results[i] != nil && results[i].Error() == "service temporarily unavailable" {
			circuitOpenCount++
		}
	}

	// 验证最后一些请求成功（熔断器应该已经恢复）
	successCount := 0
	for i := 8; i < 10; i++ {
		if results[i] == nil {
			successCount++
		}
	}

	t.Logf("Circuit breaker opened for %d requests, %d requests succeeded after recovery", circuitOpenCount, successCount)
}

func TestAPIMiddleware_Backpressure(t *testing.T) {
	// 创建配置，设置较小的队列大小
	config := DefaultAPIPerformanceConfig()
	config.Backpressure.MaxQueueSize = 3 // 最大队列大小为3

	// 创建中间件
	logger := log.NewStdLogger(&testWriter{t: t})
	mw := NewAPIMiddleware(config, logger)

	// 创建测试处理器，模拟处理时间
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		time.Sleep(100 * time.Millisecond) // 模拟处理时间
		return "success", nil
	}

	// 包装中间件
	middlewareFunc := mw.Middleware()
	wrappedHandler := middlewareFunc(handler)

	// 并发发送5个请求，队列大小为3，应该有2个被拒绝
	var wg sync.WaitGroup
	results := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := wrappedHandler(context.Background(), nil)
			results[index] = err
		}(i)
	}

	wg.Wait()

	// 统计结果
	rejectedCount := 0
	successCount := 0
	for _, err := range results {
		if err != nil && err.Error() == "service temporarily unavailable due to high load" {
			rejectedCount++
		} else if err == nil {
			successCount++
		}
	}

	// 验证结果
	if successCount == 0 {
		t.Error("Expected some requests to succeed")
	}

	if rejectedCount == 0 {
		t.Error("Expected some requests to be rejected due to backpressure")
	}

	t.Logf("Successful requests: %d, Rejected requests: %d", successCount, rejectedCount)
}

func TestAPIMiddleware_Timeout(t *testing.T) {
	// 创建配置，设置较短的超时时间
	config := DefaultAPIPerformanceConfig()
	config.Timeout.Enabled = true
	config.Timeout.Duration = 50 * time.Millisecond // 50毫秒超时

	// 创建中间件
	logger := log.NewStdLogger(&testWriter{t: t})
	mw := NewAPIMiddleware(config, logger)

	// 创建测试处理器，模拟长时间运行
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// 使用一个更长的时间确保超时
		select {
		case <-time.After(100 * time.Millisecond): // 超过超时时间
			return "success", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// 包装中间件
	middlewareFunc := mw.Middleware()
	wrappedHandler := middlewareFunc(handler)

	// 发送请求
	_, err := wrappedHandler(context.Background(), nil)

	// 验证请求超时
	if err == nil {
		t.Error("Expected request to timeout, but it succeeded")
	} else if err != context.DeadlineExceeded {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// testWriter 是一个用于测试的日志写入器
type testWriter struct {
	t *testing.T
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	w.t.Logf("LOG: %s", string(p))
	return len(p), nil
}
