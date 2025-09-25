//go:build wireinject
// +build wireinject

package pkg

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	
	"kratos-boilerplate/internal/pkg/auth"
	"kratos-boilerplate/internal/pkg/cache"
	"kratos-boilerplate/internal/pkg/config"
	"kratos-boilerplate/internal/pkg/db"
	"kratos-boilerplate/internal/pkg/errors"
	"kratos-boilerplate/internal/pkg/health"
	pkglog "kratos-boilerplate/internal/pkg/log"
	"kratos-boilerplate/internal/pkg/middleware"
	"kratos-boilerplate/internal/pkg/tracing"
	"kratos-boilerplate/internal/pkg/utils"
)

// ProviderSet 基础模块提供者集合
var ProviderSet = wire.NewSet(
	ConfigProviderSet,
	LogProviderSet,
	ErrorProviderSet,
	UtilsProviderSet,
	DatabaseProviderSet,
	CacheProviderSet,
	HealthProviderSet,
	MiddlewareProviderSet,
	TracingProviderSet,
	AuthProviderSet,
)

// ConfigProviderSet 配置模块提供者
var ConfigProviderSet = wire.NewSet(
	config.DefaultConfig,
	config.NewManager,
	config.NewValidator,
	wire.Bind(new(config.ConfigManager), new(*config.Manager)),
)

// LogProviderSet 日志模块提供者
var LogProviderSet = wire.NewSet(
	pkglog.DefaultConfig,
	pkglog.NewLogger,
	wire.Bind(new(pkglog.Logger), new(*pkglog.zapLogger)),
)

// ErrorProviderSet 错误处理模块提供者
var ErrorProviderSet = wire.NewSet(
	NewDefaultErrorMiddleware,
	wire.Bind(new(errors.ErrorMiddleware), new(*errors.DefaultErrorMiddleware)),
)

// UtilsProviderSet 工具模块提供者
var UtilsProviderSet = wire.NewSet(
	utils.GetDefaultIDManager,
	utils.NewIDValidator,
	utils.NewTimeUtils,
	utils.NewValidationUtils,
	utils.NewStringUtils,
	utils.NewNetworkUtils,
)

// DatabaseProviderSet 数据库模块提供者
var DatabaseProviderSet = wire.NewSet(
	db.DefaultConfig,
	db.NewDatabase,
	wire.Bind(new(db.Database), new(*db.database)),
)

// CacheProviderSet 缓存模块提供者
var CacheProviderSet = wire.NewSet(
	cache.DefaultConfig,
	cache.NewCache,
	cache.NewCacheManager,
	wire.Bind(new(cache.Cache), new(*cache.redisCache)),
)

// HealthProviderSet 健康检查模块提供者
var HealthProviderSet = wire.NewSet(
	NewHealthChecker,
)

// MiddlewareProviderSet 中间件模块提供者
var MiddlewareProviderSet = wire.NewSet(
	middleware.NewMiddlewareChain,
	NewDefaultMiddlewares,
)

// TracingProviderSet 链路追踪模块提供者
var TracingProviderSet = wire.NewSet(
	tracing.DefaultConfig,
	tracing.NewTracingProvider,
)

// AuthProviderSet 认证模块提供者
var AuthProviderSet = wire.NewSet(
	auth.NewAuthManager,
	wire.Bind(new(auth.AuthManager), new(*auth.DefaultAuthManager)),
)

// Provider functions

// NewDefaultErrorMiddleware 创建默认错误中间件
func NewDefaultErrorMiddleware() *errors.DefaultErrorMiddleware {
	return &errors.DefaultErrorMiddleware{}
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(
	logger log.Logger,
	database db.Database,
	cache cache.Cache,
) *health.HealthChecker {
	hc := health.NewHealthChecker(30*time.Second, logger)
	
	// 添加数据库健康检查
	dbChecker := health.NewDatabaseChecker("database", func(ctx context.Context) error {
		return database.Health(ctx)
	})
	hc.AddChecker(dbChecker)
	
	// 添加缓存健康检查
	cacheChecker := health.NewCacheChecker("cache", func(ctx context.Context) error {
		return cache.Ping(ctx)
	})
	hc.AddChecker(cacheChecker)
	
	// 添加内存检查
	memoryChecker := health.NewMemoryChecker("memory", 80.0)
	hc.AddChecker(memoryChecker)
	
	return hc
}

// NewDefaultMiddlewares 创建默认中间件
func NewDefaultMiddlewares(
	logger log.Logger,
	tracingProvider *tracing.TracingProvider,
) []middleware.Middleware {
	return middleware.NewMiddlewareChain().
		Use(middleware.Recovery(logger)).
		Use(middleware.RequestID()).
		Use(tracingProvider.Middleware()).
		Use(middleware.Logging(logger)).
		Use(middleware.Timeout(30*time.Second)).
		Build()
}

// Infrastructure 基础设施结构
type Infrastructure struct {
	Config   config.ConfigManager
	Logger   pkglog.Logger
	Database db.Database
	Cache    cache.Cache
	Health   *health.HealthChecker
	Tracing  *tracing.TracingProvider
}

// NewInfrastructure 创建基础设施
func NewInfrastructure(
	config config.ConfigManager,
	logger pkglog.Logger,
	database db.Database,
	cache cache.Cache,
	health *health.HealthChecker,
	tracing *tracing.TracingProvider,
) *Infrastructure {
	return &Infrastructure{
		Config:   config,
		Logger:   logger,
		Database: database,
		Cache:    cache,
		Health:   health,
		Tracing:  tracing,
	}
}