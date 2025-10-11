package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"kratos-boilerplate/internal/pkg/metrics"
)

// Monitor 系统监控器
type Monitor struct {
	config     *Config
	logger     log.Logger
	metrics    *metrics.PrometheusMetrics
	healthChan chan HealthStatus
	alertChan  chan Alert
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// Config 监控配置
type Config struct {
	// 检查间隔
	CheckInterval time.Duration `yaml:"check_interval" json:"check_interval"`
	
	// 资源阈值
	ResourceThresholds struct {
		CPUUsage    float64 `yaml:"cpu_usage" json:"cpu_usage"`
		MemoryUsage float64 `yaml:"memory_usage" json:"memory_usage"`
		DiskUsage   float64 `yaml:"disk_usage" json:"disk_usage"`
	} `yaml:"resource_thresholds" json:"resource_thresholds"`
	
	// 告警配置
	Alerting struct {
		Enabled   bool     `yaml:"enabled" json:"enabled"`
		Receivers []string `yaml:"receivers" json:"receivers"`
	} `yaml:"alerting" json:"alerting"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		CheckInterval: 30 * time.Second,
		ResourceThresholds: struct {
			CPUUsage    float64 `yaml:"cpu_usage" json:"cpu_usage"`
			MemoryUsage float64 `yaml:"memory_usage" json:"memory_usage"`
			DiskUsage   float64 `yaml:"disk_usage" json:"disk_usage"`
		}{
			CPUUsage:    80.0,
			MemoryUsage: 80.0,
			DiskUsage:   85.0,
		},
		Alerting: struct {
			Enabled   bool     `yaml:"enabled" json:"enabled"`
			Receivers []string `yaml:"receivers" json:"receivers"`
		}{
			Enabled:   true,
			Receivers: []string{"admin@example.com"},
		},
	}
}

// HealthStatus 健康状态
type HealthStatus struct {
	Timestamp    time.Time         `json:"timestamp"`
	Status       string            `json:"status"`
	Resources    ResourceUsage     `json:"resources"`
	CustomChecks map[string]string `json:"custom_checks"`
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Goroutines  int     `json:"goroutines"`
}

// Alert 告警
type Alert struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Resource  string    `json:"resource"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
}

// NewMonitor 创建监控器
func NewMonitor(config *Config, metrics *metrics.PrometheusMetrics, logger log.Logger) *Monitor {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	monitor := &Monitor{
		config:     config,
		logger:     logger,
		metrics:    metrics,
		healthChan: make(chan HealthStatus, 10),
		alertChan:  make(chan Alert, 10),
		cancel:     cancel,
	}

	return monitor
}

// Start 启动监控
func (m *Monitor) Start() {
	m.wg.Add(1)
	go m.run()

	// 启动告警处理协程
	if m.config.Alerting.Enabled {
		m.wg.Add(1)
		go m.handleAlerts()
	}

	log.NewHelper(m.logger).Info("Monitoring started")
}

// Stop 停止监控
func (m *Monitor) Stop() {
	m.cancel()
	m.wg.Wait()
	close(m.healthChan)
	close(m.alertChan)
	log.NewHelper(m.logger).Info("Monitoring stopped")
}

// run 运行监控循环
func (m *Monitor) run() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.cancel:
			return
		case <-ticker.C:
			m.checkHealth()
		}
	}
}

// checkHealth 检查健康状态
func (m *Monitor) checkHealth() {
	// 收集资源使用情况
	usage := m.collectResourceUsage()

	// 检查资源阈值并生成告警
	m.checkResourceThresholds(usage)

	// 创建健康状态
	health := HealthStatus{
		Timestamp: time.Now(),
		Status:    "healthy",
		Resources: usage,
		CustomChecks: make(map[string]string),
	}

	// 发送健康状态到通道
	select {
	case m.healthChan <- health:
	default:
		log.NewHelper(m.logger).Warn("Health channel is full, dropping health status")
	}

	// 更新Prometheus指标
	if m.metrics != nil {
		m.metrics.SetCPUUsage(usage.CPUUsage)
		m.metrics.SetMemoryUsage(uint64(usage.MemoryUsage))
		m.metrics.SetGoroutines(usage.Goroutines)
	}
}

// collectResourceUsage 收集资源使用情况
func (m *Monitor) collectResourceUsage() ResourceUsage {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	// 计算内存使用百分比 (简化实现)
	memUsage := float64(ms.Alloc) / float64(ms.Sys) * 100

	// 获取goroutine数量
	goroutines := runtime.NumGoroutine()

	return ResourceUsage{
		CPUUsage:    0.0, // 简化实现，实际应该收集CPU使用率
		MemoryUsage: memUsage,
		DiskUsage:   0.0, // 简化实现，实际应该收集磁盘使用率
		Goroutines:  goroutines,
	}
}

// checkResourceThresholds 检查资源阈值
func (m *Monitor) checkResourceThresholds(usage ResourceUsage) {
	// 检查CPU使用率
	if usage.CPUUsage > m.config.ResourceThresholds.CPUUsage {
		alert := Alert{
			Timestamp: time.Now(),
			Level:     "warning",
			Message:   "CPU usage is high",
			Resource:  "CPU",
			Value:     usage.CPUUsage,
			Threshold: m.config.ResourceThresholds.CPUUsage,
		}
		m.sendAlert(alert)
	}

	// 检查内存使用率
	if usage.MemoryUsage > m.config.ResourceThresholds.MemoryUsage {
		alert := Alert{
			Timestamp: time.Now(),
			Level:     "warning",
			Message:   "Memory usage is high",
			Resource:  "Memory",
			Value:     usage.MemoryUsage,
			Threshold: m.config.ResourceThresholds.MemoryUsage,
		}
		m.sendAlert(alert)
	}

	// 检查磁盘使用率
	if usage.DiskUsage > m.config.ResourceThresholds.DiskUsage {
		alert := Alert{
			Timestamp: time.Now(),
			Level:     "warning",
			Message:   "Disk usage is high",
			Resource:  "Disk",
			Value:     usage.DiskUsage,
			Threshold: m.config.ResourceThresholds.DiskUsage,
		}
		m.sendAlert(alert)
	}
}

// sendAlert 发送告警
func (m *Monitor) sendAlert(alert Alert) {
	if !m.config.Alerting.Enabled {
		return
	}

	select {
	case m.alertChan <- alert:
	default:
		log.NewHelper(m.logger).Warn("Alert channel is full, dropping alert")
	}
}

// handleAlerts 处理告警
func (m *Monitor) handleAlerts() {
	defer m.wg.Done()

	for {
		select {
		case <-m.cancel:
			return
		case alert := <-m.alertChan:
			m.processAlert(alert)
		}
	}
}

// processAlert 处理告警
func (m *Monitor) processAlert(alert Alert) {
	// 记录告警日志
	log.NewHelper(m.logger).Warnf("Alert: %s - %s (Value: %.2f, Threshold: %.2f)", 
		alert.Resource, alert.Message, alert.Value, alert.Threshold)

	// 这里应该实现实际的告警通知逻辑
	// 例如发送邮件、短信、调用Webhook等
	// 为简化起见，我们只记录日志
}

// AddCustomCheck 添加自定义检查
func (m *Monitor) AddCustomCheck(name, status string) {
	// 这里应该实现添加自定义检查的逻辑
	// 为简化起见，我们只记录日志
	log.NewHelper(m.logger).Infof("Custom check added: %s - %s", name, status)
}

// GetHealthStatus 获取健康状态通道
func (m *Monitor) GetHealthStatus() <-chan HealthStatus {
	return m.healthChan
}

// GetAlerts 获取告警通道
func (m *Monitor) GetAlerts() <-chan Alert {
	return m.alertChan
}