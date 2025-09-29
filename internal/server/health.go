package server

import (
	"context"
	"kratos-boilerplate/internal/conf"
	"kratos-boilerplate/internal/data"
	"kratos-boilerplate/internal/pkg/health"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// NewHealthChecker creates a new health checker with default checks
func NewHealthChecker(data *data.Data, c *conf.Bootstrap, logger log.Logger) *health.HealthChecker {
	hc := health.NewHealthChecker(30*time.Second, logger)

	// Add basic liveness check
	livenessChecker := health.NewNamedChecker("liveness", func(ctx context.Context) health.CheckResult {
		return health.CheckResult{
			Name:      "liveness",
			Status:    health.StatusHealthy,
			Message:   "Service is alive",
			Timestamp: time.Now(),
			Duration:  time.Millisecond,
		}
	})
	hc.AddChecker(livenessChecker)

	// Add memory check
	memoryChecker := health.NewMemoryChecker("memory", 85.0) // Alert at 85% memory usage
	hc.AddChecker(memoryChecker)

	// Add data layer health checks if data is available
	if data != nil {
		// Database health check
		dbChecker := health.NewDatabaseChecker("database", func(ctx context.Context) error {
			return data.Ping(ctx)
		})
		hc.AddChecker(dbChecker)

		// Redis health check
		redisChecker := health.NewCacheChecker("redis", func(ctx context.Context) error {
			return data.PingRedis(ctx)
		})
		hc.AddChecker(redisChecker)
	} else {
		// Add simple status checks for missing data layer
		mockDbChecker := health.NewNamedChecker("database", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{
				Name:      "database",
				Status:    health.StatusDegraded,
				Message:   "Database configuration not available",
				Timestamp: time.Now(),
				Duration:  time.Microsecond,
			}
		})
		hc.AddChecker(mockDbChecker)

		mockRedisChecker := health.NewNamedChecker("redis", func(ctx context.Context) health.CheckResult {
			return health.CheckResult{
				Name:      "redis",
				Status:    health.StatusDegraded,
				Message:   "Redis configuration not available",
				Timestamp: time.Now(),
				Duration:  time.Microsecond,
			}
		})
		hc.AddChecker(mockRedisChecker)
	}

	log.NewHelper(logger).Info("Health checker initialized with basic checks")
	return hc
}
