package integration

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kratos-boilerplate/internal/pkg/cache"
	"kratos-boilerplate/internal/pkg/config"
	"kratos-boilerplate/internal/pkg/errors"
	"kratos-boilerplate/internal/pkg/health"
	pkglog "kratos-boilerplate/internal/pkg/log"
	"kratos-boilerplate/internal/pkg/middleware"
	"kratos-boilerplate/internal/pkg/tracing"
	"kratos-boilerplate/internal/pkg/utils"
)

// TestBaseModulesIntegration 测试基础模块集成
func TestBaseModulesIntegration(t *testing.T) {
	// 测试配置管理模块
	t.Run("ConfigModule", func(t *testing.T) {
		logger := log.NewStdLogger(nil)
		manager := config.NewManager("", "", logger)

		// 测试基本配置操作
		assert.NotNil(t, manager)

		// 测试默认值
		assert.Equal(t, "", manager.GetString("nonexistent"))
		assert.Equal(t, "default", manager.GetString("nonexistent", "default"))
		assert.Equal(t, 42, manager.GetInt("nonexistent", 42))
		assert.Equal(t, true, manager.GetBool("nonexistent", true))
		assert.Equal(t, time.Hour, manager.GetDuration("nonexistent", time.Hour))
	})

	// 测试日志模块
	t.Run("LogModule", func(t *testing.T) {
		config := pkglog.DefaultConfig()
		config.Output = "stdout" // 确保输出到标准输出进行测试

		logger, err := pkglog.NewLogger(config)
		require.NoError(t, err)
		require.NotNil(t, logger)

		// 测试基本日志功能
		logger.Info("test message", pkglog.String("key", "value"))
		logger.Warn("test warning", pkglog.Int("code", 123))
		logger.Error("test error", pkglog.Error(errors.InternalError("test")))

		// 测试格式化日志
		logger.Infof("formatted message: %s", "test")

		// 测试带字段的日志器
		fieldsLogger := logger.WithFields(pkglog.String("component", "test"))
		fieldsLogger.Info("message with fields")

		// 测试关闭
		err = logger.Close()
		assert.NoError(t, err)
	})

	// 测试错误处理模块
	t.Run("ErrorModule", func(t *testing.T) {
		// 测试基本错误创建
		err := errors.NewError(errors.ErrCodeInvalidParams, "test error")
		assert.Equal(t, errors.ErrCodeInvalidParams, err.Code)
		assert.Equal(t, "test error", err.Message)
		assert.Equal(t, "[20000] test error", err.Error())

		// 测试错误包装
		originalErr := assert.AnError
		wrappedErr := errors.WrapError(originalErr, errors.ErrCodeInternalError, "wrapped")
		assert.Equal(t, originalErr, wrappedErr.Cause)

		// 测试预定义错误
		invalidParamsErr := errors.InvalidParams("invalid parameter")
		assert.Equal(t, errors.ErrCodeInvalidParams, invalidParamsErr.Code)

		unauthorizedErr := errors.Unauthorized("unauthorized access")
		assert.Equal(t, errors.ErrCodeUnauthorized, unauthorizedErr.Code)

		// 测试错误收集器
		collector := errors.NewErrorCollector()
		assert.False(t, collector.HasErrors())

		collector.Add(err)
		collector.Add(originalErr)
		assert.True(t, collector.HasErrors())
		assert.Len(t, collector.Errors(), 2)
		assert.Equal(t, err, collector.First())
	})

	// 测试工具模块
	t.Run("UtilsModule", func(t *testing.T) {
		// 测试ID生成
		gen := utils.NewIDGenerator(1)

		uuid := gen.GenerateUUID()
		assert.NotEmpty(t, uuid)
		assert.Len(t, uuid, 36)

		snowflake := gen.GenerateSnowflake()
		assert.Greater(t, snowflake, int64(0))

		shortID := gen.GenerateShortID()
		assert.NotEmpty(t, shortID)
		assert.Len(t, shortID, 8)

		customID := gen.GenerateCustomID("test_", 16)
		assert.NotEmpty(t, customID)
		assert.Len(t, customID, 16)
		assert.True(t, len(customID) >= 5 && customID[:5] == "test_")

		// 测试时间工具
		timeUtil := utils.NewTimeUtils()
		now := time.Now()
		iso8601 := timeUtil.FormatISO8601(now)
		assert.NotEmpty(t, iso8601)

		parsed, err := timeUtil.ParseISO8601(iso8601)
		assert.NoError(t, err)
		assert.Equal(t, now.Unix(), parsed.Unix())

		// 测试验证工具
		validator := utils.NewValidationUtils()
		assert.True(t, validator.IsEmail("test@example.com"))
		assert.False(t, validator.IsEmail("invalid-email"))
		assert.True(t, validator.IsPhone("13812345678"))
		assert.False(t, validator.IsPhone("invalid-phone"))

		// 测试字符串工具
		stringUtil := utils.NewStringUtils()
		assert.True(t, stringUtil.IsEmpty(""))
		assert.False(t, stringUtil.IsEmpty("hello"))
		assert.Equal(t, "te**@example.com", stringUtil.MaskEmail("test@example.com"))
		assert.Equal(t, "138****5678", stringUtil.MaskPhone("13812345678"))
	})
}

// TestServiceModulesIntegration 测试服务模块集成
func TestServiceModulesIntegration(t *testing.T) {
	logger := log.NewStdLogger(nil)

	// 测试缓存模块（使用内存实现进行测试）
	t.Run("CacheModule", func(t *testing.T) {
		// 注意：这里需要有运行的Redis实例，或者使用内存缓存进行测试
		// 为了测试，我们使用默认配置但跳过实际连接
		config := cache.DefaultConfig()

		// 这里应该有Redis连接，测试时可能需要mock
		_, err := cache.NewCache(config, logger)
		if err != nil {
			t.Skipf("Skipping cache test due to connection error: %v", err)
			return
		}

		// 如果连接成功，可以进行基本操作测试
		// 这里添加实际的缓存操作测试
	})

	// 测试健康检查模块
	t.Run("HealthModule", func(t *testing.T) {
		healthChecker := health.NewHealthChecker(5*time.Second, logger)
		assert.NotNil(t, healthChecker)

		// 添加一个简单的检查器
		checker := health.NewNamedChecker("test", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{
				Name:      "test",
				Status:    health.StatusHealthy,
				Message:   "test is healthy",
				Timestamp: time.Now(),
				Duration:  time.Millisecond,
			}
		})

		healthChecker.AddChecker(checker)

		// 执行健康检查
		ctx := context.Background()
		result := healthChecker.Check(ctx)

		assert.NotNil(t, result)
		assert.Equal(t, health.StatusHealthy, result.Status)
		assert.Contains(t, result.Checks, "test")
		assert.Equal(t, health.StatusHealthy, result.Checks["test"].Status)
	})

	// 测试中间件模块
	t.Run("MiddlewareModule", func(t *testing.T) {
		// 测试中间件链
		chain := middleware.NewMiddlewareChain()
		assert.NotNil(t, chain)

		// 添加中间件
		chain.Use(middleware.Recovery(logger))
		chain.Use(middleware.RequestID())
		chain.Use(middleware.Logging(logger))

		middlewares := chain.Build()
		assert.Len(t, middlewares, 3)

		// 测试默认中间件
		defaultMiddlewares := middleware.DefaultMiddlewares(logger)
		assert.NotEmpty(t, defaultMiddlewares)

		// 测试令牌桶限流器
		limiter := middleware.NewTokenBucket(10, 20)
		assert.NotNil(t, limiter)

		// 初始应该允许请求
		assert.True(t, limiter.Allow())
	})

	// 测试链路追踪模块
	t.Run("TracingModule", func(t *testing.T) {
		config := tracing.DefaultConfig()
		config.Enabled = false // 禁用以避免需要Jaeger连接

		provider, err := tracing.NewTracingProvider(config, logger)
		require.NoError(t, err)
		require.NotNil(t, provider)

		tracer := provider.GetTracer()
		assert.NotNil(t, tracer)

		// 测试创建Span
		ctx := context.Background()
		ctx, span := tracer.Start(ctx, "test-span")
		assert.NotNil(t, span)
		span.End()

		// 测试追踪助手
		helper := tracing.NewTracingHelper(tracer, logger)
		assert.NotNil(t, helper)

		err = helper.WithSpan(ctx, "test-operation", func(ctx context.Context) error {
			// 模拟一些操作
			time.Sleep(time.Millisecond)
			return nil
		})
		assert.NoError(t, err)
	})
}

// TestModulesInteraction 测试模块间交互
func TestModulesInteraction(t *testing.T) {
	logger := log.NewStdLogger(nil)

	t.Run("LoggingWithTracing", func(t *testing.T) {
		// 测试日志与链路追踪的集成
		tracingConfig := tracing.DefaultConfig()
		tracingConfig.Enabled = false

		provider, err := tracing.NewTracingProvider(tracingConfig, logger)
		require.NoError(t, err)

		tracer := provider.GetTracer()
		ctx := context.Background()
		ctx, span := tracer.Start(ctx, "test-logging")
		defer span.End()

		// 从上下文获取追踪信息
		traceID := tracing.TraceIDFromContext(ctx)
		spanID := tracing.SpanIDFromContext(ctx)

		// 在没有启用追踪的情况下，这些应该为空
		// 但函数应该正常工作
		assert.NotPanics(t, func() {
			_ = traceID
			_ = spanID
		})
	})

	t.Run("ErrorsWithLogging", func(t *testing.T) {
		// 测试错误处理与日志的集成
		logConfig := pkglog.DefaultConfig()
		testLogger, err := pkglog.NewLogger(logConfig)
		require.NoError(t, err)

		// 创建错误并记录
		testErr := errors.InternalError("test internal error")
		testLogger.Error("operation failed", pkglog.Error(testErr))

		// 测试错误收集器
		collector := errors.NewErrorCollector()
		collector.Add(testErr)
		collector.Add(errors.InvalidParams("invalid input"))

		assert.True(t, collector.HasErrors())
		multiError := collector.ToMultiError()
		assert.NotNil(t, multiError)
		assert.Equal(t, 2, multiError.Count)
	})

	t.Run("HealthCheckWithDependencies", func(t *testing.T) {
		// 测试健康检查与其他模块的集成
		healthChecker := health.NewHealthChecker(5*time.Second, logger)

		// 添加内存检查器
		memoryChecker := health.NewMemoryChecker("memory", 80.0)
		healthChecker.AddChecker(memoryChecker)

		// 添加HTTP检查器（可能会失败，但不应该崩溃）
		httpChecker := health.NewHTTPChecker("external_service", "http://httpbin.org/status/200", 5*time.Second)
		healthChecker.AddChecker(httpChecker)

		ctx := context.Background()
		result := healthChecker.Check(ctx)

		assert.NotNil(t, result)
		assert.Contains(t, result.Checks, "memory")
		assert.Contains(t, result.Checks, "external_service")

		// 内存检查应该成功
		assert.Equal(t, health.StatusHealthy, result.Checks["memory"].Status)
	})
}

// BenchmarkModules 基础模块性能基准测试
func BenchmarkModules(b *testing.B) {
	logger := log.NewStdLogger(nil)

	b.Run("IDGeneration", func(b *testing.B) {
		gen := utils.NewIDGenerator(1)
		b.ResetTimer()

		b.Run("UUID", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				gen.GenerateUUID()
			}
		})

		b.Run("Snowflake", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				gen.GenerateSnowflake()
			}
		})

		b.Run("ShortID", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				gen.GenerateShortID()
			}
		})
	})

	b.Run("ErrorCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			errors.NewError(errors.ErrCodeInvalidParams, "test error")
		}
	})

	b.Run("StringMasking", func(b *testing.B) {
		stringUtil := utils.NewStringUtils()
		b.ResetTimer()

		b.Run("Email", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				stringUtil.MaskEmail("test@example.com")
			}
		})

		b.Run("Phone", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				stringUtil.MaskPhone("13812345678")
			}
		})
	})

	b.Run("Logging", func(b *testing.B) {
		config := pkglog.DefaultConfig()
		testLogger, err := pkglog.NewLogger(config)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testLogger.Info("benchmark message", pkglog.String("key", "value"))
		}

		testLogger.Close()
	})

	b.Run("HealthCheck", func(b *testing.B) {
		healthChecker := health.NewHealthChecker(5*time.Second, logger)

		// 添加简单检查器
		checker := health.NewNamedChecker("test", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{
				Name:      "test",
				Status:    health.StatusHealthy,
				Message:   "healthy",
				Timestamp: time.Now(),
				Duration:  time.Microsecond,
			}
		})
		healthChecker.AddChecker(checker)

		ctx := context.Background()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			healthChecker.Check(ctx)
		}
	})
}
