package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMetrics Prometheus指标收集器
type PrometheusMetrics struct {
	// HTTP请求指标
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestDuration  *prometheus.HistogramVec
	httpResponseSize     *prometheus.HistogramVec
	
	// 业务指标
	activeUsers          prometheus.Gauge
	databaseConnections  prometheus.Gauge
	cacheHits            prometheus.Counter
	cacheMisses          prometheus.Counter
	
	// 系统指标
	cpuUsage             prometheus.Gauge
	memoryUsage          prometheus.Gauge
	goroutines           prometheus.Gauge
	
	// 自定义业务指标
	businessMetrics      map[string]prometheus.Counter
	mu                   sync.Mutex
	
	logger               log.Logger
	registry             *prometheus.Registry
	server               *http.Server
}

// Config Prometheus配置
type Config struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Addr    string `yaml:"addr" json:"addr"`
	Path    string `yaml:"path" json:"path"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled: true,
		Addr:    ":9090",
		Path:    "/metrics",
	}
}

// NewPrometheusMetrics 创建Prometheus指标收集器
func NewPrometheusMetrics(config *Config, logger log.Logger) (*PrometheusMetrics, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if !config.Enabled {
		return &PrometheusMetrics{
			logger: logger,
		}, nil
	}

	// 创建自定义注册表
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())

	// 创建指标
	httpRequestsTotal := promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration := promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	httpResponseSize := promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "endpoint"},
	)

	activeUsers := promauto.With(registry).NewGauge(
		prometheus.GaugeOpts{
			Name: "active_users",
			Help: "Number of active users",
		},
	)

	databaseConnections := promauto.With(registry).NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections",
			Help: "Number of database connections",
		},
	)

	cacheHits := promauto.With(registry).NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	cacheMisses := promauto.With(registry).NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	cpuUsage := promauto.With(registry).NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_usage_percent",
			Help: "CPU usage percentage",
		},
	)

	memoryUsage := promauto.With(registry).NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Memory usage in bytes",
		},
	)

	goroutines := promauto.With(registry).NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutines",
			Help: "Number of goroutines",
		},
	)

	// 创建Prometheus HTTP处理器
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})

	// 创建HTTP服务器
	mux := http.NewServeMux()
	mux.Handle(config.Path, handler)
	
	server := &http.Server{
		Addr:    config.Addr,
		Handler: mux,
	}

	metrics := &PrometheusMetrics{
		httpRequestsTotal:    httpRequestsTotal,
		httpRequestDuration:  httpRequestDuration,
		httpResponseSize:     httpResponseSize,
		activeUsers:          activeUsers,
		databaseConnections:  databaseConnections,
		cacheHits:            cacheHits,
		cacheMisses:          cacheMisses,
		cpuUsage:             cpuUsage,
		memoryUsage:          memoryUsage,
		goroutines:           goroutines,
		businessMetrics:      make(map[string]prometheus.Counter),
		logger:               logger,
		registry:             registry,
		server:               server,
	}

	return metrics, nil
}

// Start 启动Prometheus指标服务器
func (p *PrometheusMetrics) Start() error {
	if p.server == nil {
		return nil
	}

	go func() {
		log.NewHelper(p.logger).Infof("Starting Prometheus metrics server on %s", p.server.Addr)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.NewHelper(p.logger).Errorf("Prometheus metrics server error: %v", err)
		}
	}()

	return nil
}

// Stop 停止Prometheus指标服务器
func (p *PrometheusMetrics) Stop(ctx context.Context) error {
	if p.server == nil {
		return nil
	}

	log.NewHelper(p.logger).Info("Stopping Prometheus metrics server")
	return p.server.Shutdown(ctx)
}

// RecordHTTPRequest 记录HTTP请求
func (p *PrometheusMetrics) RecordHTTPRequest(method, endpoint string, status int, duration time.Duration, size int64) {
	if p.httpRequestsTotal == nil {
		return
	}

	p.httpRequestsTotal.WithLabelValues(method, endpoint, fmt.Sprintf("%d", status)).Inc()
	p.httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	p.httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(size))
}

// SetActiveUsers 设置活跃用户数
func (p *PrometheusMetrics) SetActiveUsers(count int) {
	if p.activeUsers == nil {
		return
	}
	p.activeUsers.Set(float64(count))
}

// SetDatabaseConnections 设置数据库连接数
func (p *PrometheusMetrics) SetDatabaseConnections(count int) {
	if p.databaseConnections == nil {
		return
	}
	p.databaseConnections.Set(float64(count))
}

// IncrementCacheHit 增加缓存命中次数
func (p *PrometheusMetrics) IncrementCacheHit() {
	if p.cacheHits == nil {
		return
	}
	p.cacheHits.Inc()
}

// IncrementCacheMiss 增加缓存未命中次数
func (p *PrometheusMetrics) IncrementCacheMiss() {
	if p.cacheMisses == nil {
		return
	}
	p.cacheMisses.Inc()
}

// SetCPUUsage 设置CPU使用率
func (p *PrometheusMetrics) SetCPUUsage(usage float64) {
	if p.cpuUsage == nil {
		return
	}
	p.cpuUsage.Set(usage)
}

// SetMemoryUsage 设置内存使用量
func (p *PrometheusMetrics) SetMemoryUsage(bytes uint64) {
	if p.memoryUsage == nil {
		return
	}
	p.memoryUsage.Set(float64(bytes))
}

// SetGoroutines 设置goroutine数量
func (p *PrometheusMetrics) SetGoroutines(count int) {
	if p.goroutines == nil {
		return
	}
	p.goroutines.Set(float64(count))
}

// IncrementBusinessMetric 增加业务指标
func (p *PrometheusMetrics) IncrementBusinessMetric(name string) {
	if p.registry == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查指标是否已存在
	if counter, exists := p.businessMetrics[name]; exists {
		counter.Inc()
		return
	}

	// 创建新的指标
	counter := promauto.With(p.registry).NewCounter(
		prometheus.CounterOpts{
			Name: name,
			Help: fmt.Sprintf("Business metric: %s", name),
		},
	)

	counter.Inc()
	p.businessMetrics[name] = counter
}

// GetHandler 获取Prometheus HTTP处理器
func (p *PrometheusMetrics) GetHandler() http.Handler {
	if p.registry == nil {
		return http.NotFoundHandler()
	}
	
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}