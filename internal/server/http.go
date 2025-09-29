package server

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	authv1 "kratos-boilerplate/api/auth/v1"
	v1 "kratos-boilerplate/api/helloworld/v1"
	"kratos-boilerplate/internal/conf"
	"kratos-boilerplate/internal/pkg/health"
	"kratos-boilerplate/internal/pkg/security"
	"kratos-boilerplate/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, greeter *service.GreeterService, auth *service.AuthService, healthChecker *health.HealthChecker, logger log.Logger) *kratosHttp.Server {
	// Security configuration
	securityConfig := security.DefaultSecurityConfig()

	var opts = []kratosHttp.ServerOption{
		kratosHttp.Middleware(
			recovery.Recovery(),
			// Security middleware will be added as filters
			// 暂时注释掉操作日志中间件，等实现了 repo 后再启用
			// middleware.OperationLogMiddleware(repo),
		),
		// Add security filters
		kratosHttp.Filter(
			security.SecurityHeadersFilter(securityConfig),
			security.CORSFilter(securityConfig),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, kratosHttp.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, kratosHttp.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, kratosHttp.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := kratosHttp.NewServer(opts...)

	// Register API handlers
	v1.RegisterGreeterHTTPServer(srv, greeter)
	authv1.RegisterAuthHTTPServer(srv, auth)

	// Register health check endpoints
	if healthChecker != nil {
		srv.HandleFunc("/health", healthChecker.GetHTTPHandler())
		srv.HandleFunc("/health/live", healthChecker.GetHTTPHandler())  // Kubernetes liveness probe
		srv.HandleFunc("/health/ready", healthChecker.GetHTTPHandler()) // Kubernetes readiness probe
	}

	// Register basic metrics endpoint
	srv.HandleFunc("/metrics", createMetricsHandler(logger))

	// Register Swagger UI with security
	RegisterSecureSwaggerUI(srv, nil, logger)

	return srv
}

// createMetricsHandler creates a basic metrics endpoint
func createMetricsHandler(logger log.Logger) http.HandlerFunc {
	var startTime = time.Now()

	return func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		metrics := map[string]interface{}{
			"uptime":     time.Since(startTime).String(),
			"goroutines": runtime.NumGoroutine(),
			"memory": map[string]interface{}{
				"allocated_mb":       float64(m.Alloc) / 1024 / 1024,
				"total_allocated_mb": float64(m.TotalAlloc) / 1024 / 1024,
				"system_mb":          float64(m.Sys) / 1024 / 1024,
				"gc_count":           m.NumGC,
			},
			"timestamp": time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metrics); err != nil {
			log.NewHelper(logger).Errorf("Failed to encode metrics: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}
