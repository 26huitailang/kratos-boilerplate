package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// Status 健康状态枚举
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
	StatusUnknown   Status = "unknown"
)

// CheckResult 健康检查结果
type CheckResult struct {
	Name        string                 `json:"name"`
	Status      Status                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration"`
	Error       string                 `json:"error,omitempty"`
}

// OverallHealth 整体健康状态
type OverallHealth struct {
	Status    Status                  `json:"status"`
	Message   string                  `json:"message,omitempty"`
	Timestamp time.Time               `json:"timestamp"`
	Duration  time.Duration           `json:"duration"`
	Checks    map[string]CheckResult  `json:"checks"`
	Details   map[string]interface{}  `json:"details,omitempty"`
}

// Checker 健康检查器接口
type Checker interface {
	Check(ctx context.Context) CheckResult
	Name() string
}

// CheckerFunc 检查器函数类型
type CheckerFunc func(ctx context.Context) CheckResult

// Check 实现Checker接口
func (f CheckerFunc) Check(ctx context.Context) CheckResult {
	return f(ctx)
}

// Name 返回检查器名称
func (f CheckerFunc) Name() string {
	return "anonymous"
}

// NamedChecker 带名称的检查器
type NamedChecker struct {
	name    string
	checker CheckerFunc
}

// NewNamedChecker 创建带名称的检查器
func NewNamedChecker(name string, checker CheckerFunc) *NamedChecker {
	return &NamedChecker{
		name:    name,
		checker: checker,
	}
}

// Check 执行检查
func (nc *NamedChecker) Check(ctx context.Context) CheckResult {
	return nc.checker(ctx)
}

// Name 返回检查器名称
func (nc *NamedChecker) Name() string {
	return nc.name
}

// HealthChecker 健康检查管理器
type HealthChecker struct {
	checkers []Checker
	timeout  time.Duration
	logger   log.Logger
	mu       sync.RWMutex
}

// NewHealthChecker 创建健康检查管理器
func NewHealthChecker(timeout time.Duration, logger log.Logger) *HealthChecker {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	
	return &HealthChecker{
		checkers: make([]Checker, 0),
		timeout:  timeout,
		logger:   logger,
	}
}

// AddChecker 添加检查器
func (hc *HealthChecker) AddChecker(checker Checker) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checkers = append(hc.checkers, checker)
}

// RemoveChecker 移除检查器
func (hc *HealthChecker) RemoveChecker(name string) bool {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	for i, checker := range hc.checkers {
		if checker.Name() == name {
			hc.checkers = append(hc.checkers[:i], hc.checkers[i+1:]...)
			return true
		}
	}
	return false
}

// Check 执行所有健康检查
func (hc *HealthChecker) Check(ctx context.Context) *OverallHealth {
	start := time.Now()
	
	// 设置超时上下文
	checkCtx, cancel := context.WithTimeout(ctx, hc.timeout)
	defer cancel()
	
	hc.mu.RLock()
	checkers := make([]Checker, len(hc.checkers))
	copy(checkers, hc.checkers)
	hc.mu.RUnlock()
	
	// 并发执行所有检查
	results := make(chan CheckResult, len(checkers))
	var wg sync.WaitGroup
	
	for _, checker := range checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			result := hc.runSingleCheck(checkCtx, c)
			results <- result
		}(checker)
	}
	
	// 等待所有检查完成
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// 收集结果
	checks := make(map[string]CheckResult)
	for result := range results {
		checks[result.Name] = result
	}
	
	// 计算整体状态
	overallStatus := hc.calculateOverallStatus(checks)
	
	return &OverallHealth{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Duration:  time.Since(start),
		Checks:    checks,
	}
}

// runSingleCheck 执行单个检查
func (hc *HealthChecker) runSingleCheck(ctx context.Context, checker Checker) CheckResult {
	start := time.Now()
	
	defer func() {
		if r := recover(); r != nil {
			hc.logger.Log(log.LevelError, "msg", "health check panic", "checker", checker.Name(), "panic", r)
		}
	}()
	
	result := checker.Check(ctx)
	result.Name = checker.Name()
	result.Timestamp = time.Now()
	result.Duration = time.Since(start)
	
	return result
}

// calculateOverallStatus 计算整体状态
func (hc *HealthChecker) calculateOverallStatus(checks map[string]CheckResult) Status {
	if len(checks) == 0 {
		return StatusUnknown
	}
	
	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0
	
	for _, result := range checks {
		switch result.Status {
		case StatusHealthy:
			healthyCount++
		case StatusDegraded:
			degradedCount++
		case StatusUnhealthy:
			unhealthyCount++
		}
	}
	
	// 如果有任何不健康的服务，整体状态为不健康
	if unhealthyCount > 0 {
		return StatusUnhealthy
	}
	
	// 如果有降级的服务，整体状态为降级
	if degradedCount > 0 {
		return StatusDegraded
	}
	
	// 如果所有服务都健康，整体状态为健康
	if healthyCount > 0 {
		return StatusHealthy
	}
	
	return StatusUnknown
}

// GetHTTPHandler 获取HTTP处理器
func (hc *HealthChecker) GetHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		health := hc.Check(ctx)
		
		// 设置HTTP状态码
		var statusCode int
		switch health.Status {
		case StatusHealthy:
			statusCode = http.StatusOK
		case StatusDegraded:
			statusCode = http.StatusOK // 降级状态仍返回200
		case StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		default:
			statusCode = http.StatusServiceUnavailable
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		
		if err := json.NewEncoder(w).Encode(health); err != nil {
			hc.logger.Log(log.LevelError, "msg", "failed to encode health response", "error", err)
		}
	}
}

// PrebuiltCheckers 预构建的检查器

// DatabaseChecker 数据库检查器
type DatabaseChecker struct {
	name string
	ping func(ctx context.Context) error
}

// NewDatabaseChecker 创建数据库检查器
func NewDatabaseChecker(name string, ping func(ctx context.Context) error) *DatabaseChecker {
	return &DatabaseChecker{
		name: name,
		ping: ping,
	}
}

// Check 执行数据库检查
func (dc *DatabaseChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	
	if err := dc.ping(ctx); err != nil {
		return CheckResult{
			Name:      dc.name,
			Status:    StatusUnhealthy,
			Message:   "Database connection failed",
			Error:     err.Error(),
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
	
	return CheckResult{
		Name:      dc.name,
		Status:    StatusHealthy,
		Message:   "Database connection successful",
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}
}

// Name 返回检查器名称
func (dc *DatabaseChecker) Name() string {
	return dc.name
}

// CacheChecker 缓存检查器
type CacheChecker struct {
	name string
	ping func(ctx context.Context) error
}

// NewCacheChecker 创建缓存检查器
func NewCacheChecker(name string, ping func(ctx context.Context) error) *CacheChecker {
	return &CacheChecker{
		name: name,
		ping: ping,
	}
}

// Check 执行缓存检查
func (cc *CacheChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	
	if err := cc.ping(ctx); err != nil {
		return CheckResult{
			Name:      cc.name,
			Status:    StatusUnhealthy,
			Message:   "Cache connection failed",
			Error:     err.Error(),
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
	
	return CheckResult{
		Name:      cc.name,
		Status:    StatusHealthy,
		Message:   "Cache connection successful",
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}
}

// Name 返回检查器名称
func (cc *CacheChecker) Name() string {
	return cc.name
}

// HTTPChecker HTTP服务检查器
type HTTPChecker struct {
	name   string
	url    string
	client *http.Client
}

// NewHTTPChecker 创建HTTP检查器
func NewHTTPChecker(name, url string, timeout time.Duration) *HTTPChecker {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	
	return &HTTPChecker{
		name: name,
		url:  url,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Check 执行HTTP检查
func (hc *HTTPChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	
	req, err := http.NewRequestWithContext(ctx, "GET", hc.url, nil)
	if err != nil {
		return CheckResult{
			Name:      hc.name,
			Status:    StatusUnhealthy,
			Message:   "Failed to create HTTP request",
			Error:     err.Error(),
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
	
	resp, err := hc.client.Do(req)
	if err != nil {
		return CheckResult{
			Name:      hc.name,
			Status:    StatusUnhealthy,
			Message:   "HTTP request failed",
			Error:     err.Error(),
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return CheckResult{
			Name:      hc.name,
			Status:    StatusHealthy,
			Message:   "HTTP service is healthy",
			Details:   map[string]interface{}{"status_code": resp.StatusCode},
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
	
	return CheckResult{
		Name:      hc.name,
		Status:    StatusUnhealthy,
		Message:   "HTTP service returned error status",
		Details:   map[string]interface{}{"status_code": resp.StatusCode},
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}
}

// Name 返回检查器名称
func (hc *HTTPChecker) Name() string {
	return hc.name
}

// MemoryChecker 内存使用检查器
type MemoryChecker struct {
	name      string
	threshold float64 // 内存使用阈值（百分比）
}

// NewMemoryChecker 创建内存检查器
func NewMemoryChecker(name string, threshold float64) *MemoryChecker {
	if threshold <= 0 {
		threshold = 80.0 // 默认80%阈值
	}
	
	return &MemoryChecker{
		name:      name,
		threshold: threshold,
	}
}

// Check 执行内存检查
func (mc *MemoryChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	
	// 这里应该实现实际的内存使用率检查
	// 简化实现，实际项目中应该使用 runtime.MemStats 或系统调用
	memUsage := 65.5 // 模拟内存使用率
	
	status := StatusHealthy
	message := "Memory usage is normal"
	
	if memUsage > mc.threshold {
		if memUsage > mc.threshold*1.2 { // 超过阈值20%认为不健康
			status = StatusUnhealthy
			message = "Memory usage is critically high"
		} else {
			status = StatusDegraded
			message = "Memory usage is high"
		}
	}
	
	return CheckResult{
		Name:    mc.name,
		Status:  status,
		Message: message,
		Details: map[string]interface{}{
			"usage_percent": memUsage,
			"threshold":     mc.threshold,
		},
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}
}

// Name 返回检查器名称
func (mc *MemoryChecker) Name() string {
	return mc.name
}

// ReadinessChecker 就绪检查器
type ReadinessChecker struct {
	healthChecker *HealthChecker
	required      []string // 必需的检查项
}

// NewReadinessChecker 创建就绪检查器
func NewReadinessChecker(healthChecker *HealthChecker, required []string) *ReadinessChecker {
	return &ReadinessChecker{
		healthChecker: healthChecker,
		required:      required,
	}
}

// Check 执行就绪检查
func (rc *ReadinessChecker) Check(ctx context.Context) *OverallHealth {
	health := rc.healthChecker.Check(ctx)
	
	// 检查必需的服务是否都健康
	for _, required := range rc.required {
		if check, exists := health.Checks[required]; exists {
			if check.Status == StatusUnhealthy {
				health.Status = StatusUnhealthy
				health.Message = fmt.Sprintf("Required service %s is unhealthy", required)
				break
			}
		} else {
			health.Status = StatusUnhealthy
			health.Message = fmt.Sprintf("Required service %s not found", required)
			break
		}
	}
	
	return health
}

// LivenessChecker 存活检查器
type LivenessChecker struct {
	healthChecker *HealthChecker
}

// NewLivenessChecker 创建存活检查器
func NewLivenessChecker(healthChecker *HealthChecker) *LivenessChecker {
	return &LivenessChecker{
		healthChecker: healthChecker,
	}
}

// Check 执行存活检查（简化版，只检查服务是否响应）
func (lc *LivenessChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	
	// 存活检查通常只检查服务基本功能
	return CheckResult{
		Name:      "liveness",
		Status:    StatusHealthy,
		Message:   "Service is alive",
		Timestamp: time.Now(),
		Duration:  time.Since(start),
	}
}

// Name 返回检查器名称
func (lc *LivenessChecker) Name() string {
	return "liveness"
}