package benchmark

import (
	"context"
	"fmt"
	"testing"
	"time"

	"kratos-boilerplate/internal/middleware"
	"kratos-boilerplate/internal/pkg/performance"

	"github.com/go-kratos/kratos/v2/log"
)

// BenchmarkAPIPerformanceMiddleware 测试API性能中间件的性能
func BenchmarkAPIPerformanceMiddleware(b *testing.B) {
	// 创建API性能中间件，禁用限流以避免测试失败
	config := middleware.DefaultAPIPerformanceConfig()
	config.RateLimit.Enabled = false      // 禁用限流
	config.CircuitBreaker.Enabled = false // 禁用熔断器
	config.Backpressure.Enabled = false   // 禁用背压控制
	logger := log.NewStdLogger(&testWriter{b: b})
	mw := middleware.NewAPIMiddleware(config, logger)

	// 创建测试处理器
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// 模拟简单的处理逻辑
		return "success", nil
	}

	// 包装中间件
	middlewareFunc := mw.Middleware()
	wrappedHandler := middlewareFunc(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := wrappedHandler(context.Background(), nil)
		if err != nil {
			b.Fatalf("Handler failed: %v", err)
		}
	}
}

// BenchmarkCachePerformance 测试缓存性能
func BenchmarkCachePerformance(b *testing.B) {
	// 创建内存缓存
	cache := performance.NewMemoryCache(1000, 5*time.Minute, "LRU")

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i)
			value := fmt.Sprintf("value_%d", i)
			cache.Set(key, value)
		}
	})

	b.Run("Get", func(b *testing.B) {
		// 预先填充缓存
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("key_%d", i)
			value := fmt.Sprintf("value_%d", i)
			cache.Set(key, value)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i%1000)
			_, found := cache.Get(key)
			if !found {
				b.Fatalf("Key not found: %s", key)
			}
		}
	})
}

// BenchmarkQueryOptimizer 测试查询优化器性能
func BenchmarkQueryOptimizer(b *testing.B) {
	// 创建查询优化器配置
	config := performance.DefaultQueryOptimizerConfig()
	logger := log.NewStdLogger(&testWriter{b: b})

	// 由于我们没有实际的数据库连接，这里只测试优化器的开销
	qo := performance.NewQueryOptimizer(nil, config, logger)

	b.Run("GetStats", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// 测试获取统计信息的性能
			_ = qo.GetStats()
		}
	})
}

// BenchmarkMemoryCache 测试内存缓存性能
func BenchmarkMemoryCache(b *testing.B) {
	// 创建内存缓存
	cache := performance.NewMemoryCache(10000, 5*time.Minute, "LRU")

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i)
			value := fmt.Sprintf("value_%d", i)
			cache.Set(key, value)
		}
	})

	b.Run("Get", func(b *testing.B) {
		// 预先填充缓存
		for i := 0; i < 10000; i++ {
			key := fmt.Sprintf("key_%d", i)
			value := fmt.Sprintf("value_%d", i)
			cache.Set(key, value)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key_%d", i%10000)
			_, found := cache.Get(key)
			if !found {
				b.Fatalf("Key not found: %s", key)
			}
		}
	})
}

// testWriter 简单的日志写入器实现
type testWriter struct {
	b *testing.B
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	// 空实现，不记录日志以避免影响基准测试
	return len(p), nil
}

// BenchmarkCircuitBreaker 测试熔断器性能
func BenchmarkCircuitBreaker(b *testing.B) {
	// 创建熔断器
	cb := middleware.NewCircuitBreaker(0.5, 100, 30*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 记录成功和失败
		if i%2 == 0 {
			cb.RecordSuccess()
		} else {
			cb.RecordFailure()
		}

		// 检查熔断器状态
		_ = cb.IsOpen()
	}
}
