package metrics
package metrics

import (
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// MetricsCollector collects basic application metrics
type MetricsCollector struct {
	mu             sync.RWMutex
	requestCount   map[string]int64
	errorCount     map[string]int64
	responseTime   map[string]time.Duration
	startTime      time.Time
	logger         *log.Helper
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger log.Logger) *MetricsCollector {
	return &MetricsCollector{
		requestCount: make(map[string]int64),
		errorCount:   make(map[string]int64),
		responseTime: make(map[string]time.Duration),
		startTime:    time.Now(),
		logger:       log.NewHelper(log.With(logger, "component", "metrics")),
	}
}

// IncrementRequest increments request count for an endpoint
func (m *MetricsCollector) IncrementRequest(endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestCount[endpoint]++
}

// IncrementError increments error count for an endpoint
func (m *MetricsCollector) IncrementError(endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCount[endpoint]++
}

// RecordResponseTime records response time for an endpoint
func (m *MetricsCollector) RecordResponseTime(endpoint string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responseTime[endpoint] = duration
}

// GetMetrics returns current metrics
func (m *MetricsCollector) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Copy maps to avoid concurrent access issues
	requests := make(map[string]int64)
	errors := make(map[string]int64)
	responses := make(map[string]string)
	
	for k, v := range m.requestCount {
		requests[k] = v
	}
	
	for k, v := range m.errorCount {
		errors[k] = v
	}
	
	for k, v := range m.responseTime {
		responses[k] = v.String()
	}

	return map[string]interface{}{
		"uptime":        time.Since(m.startTime).String(),
		"requests":      requests,
		"errors":        errors,
		"response_time": responses,
		"timestamp":     time.Now().Format(time.RFC3339),
	}
}

// Reset resets all metrics
func (m *MetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.requestCount = make(map[string]int64)
	m.errorCount = make(map[string]int64)
	m.responseTime = make(map[string]time.Duration)
	m.startTime = time.Now()
	
	m.logger.Info("Metrics reset")
}