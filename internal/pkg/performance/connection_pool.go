package performance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// ConnectionPoolConfig 连接池优化配置
type ConnectionPoolConfig struct {
	// 主库配置
	Master struct {
		MaxOpenConns    int           `yaml:"max_open_conns" json:"max_open_conns"`         // 最大开放连接数
		MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`         // 最大空闲连接数
		ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`   // 连接最大生命周期
		ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"` // 连接最大空闲时间
	} `yaml:"master" json:"master"`

	// 从库配置
	Slaves []struct {
		Name            string        `yaml:"name" json:"name"`
		MaxOpenConns    int           `yaml:"max_open_conns" json:"max_open_conns"`
		MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`
		ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
		ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`
		Weight          int           `yaml:"weight" json:"weight"` // 负载均衡权重
	} `yaml:"slaves" json:"slaves"`

	// 监控配置
	Monitoring struct {
		Enabled         bool          `yaml:"enabled" json:"enabled"`
		CheckInterval   time.Duration `yaml:"check_interval" json:"check_interval"`
		AlertThresholds struct {
			MaxOpenConnsUsage  float64       `yaml:"max_open_conns_usage" json:"max_open_conns_usage"` // 最大连接使用率告警阈值
			MaxIdleConnsUsage  float64       `yaml:"max_idle_conns_usage" json:"max_idle_conns_usage"` // 最大空闲连接使用率告警阈值
			AvgConnLifetime    time.Duration `yaml:"avg_conn_lifetime" json:"avg_conn_lifetime"`       // 平均连接生命周期告警阈值
			SlowQueryThreshold time.Duration `yaml:"slow_query_threshold" json:"slow_query_threshold"` // 慢查询告警阈值
		} `yaml:"alert_thresholds" json:"alert_thresholds"`
	} `yaml:"monitoring" json:"monitoring"`
}

// DefaultConnectionPoolConfig 返回默认的连接池配置
func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	config := &ConnectionPoolConfig{}

	// 主库默认配置 - 针对写操作优化
	config.Master.MaxOpenConns = 50                  // 生产环境建议根据CPU核心数调整
	config.Master.MaxIdleConns = 10                  // 保持足够的空闲连接
	config.Master.ConnMaxLifetime = 2 * time.Hour    // 避免长时间连接
	config.Master.ConnMaxIdleTime = 30 * time.Minute // 及时释放空闲连接

	// 从库默认配置 - 针对读操作优化
	config.Slaves = []struct {
		Name            string        `yaml:"name" json:"name"`
		MaxOpenConns    int           `yaml:"max_open_conns" json:"max_open_conns"`
		MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`
		ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
		ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`
		Weight          int           `yaml:"weight" json:"weight"`
	}{
		{
			Name:            "slave1",
			MaxOpenConns:    30,               // 读操作通常需要更多并发
			MaxIdleConns:    15,               // 保持更多空闲连接用于读查询
			ConnMaxLifetime: 4 * time.Hour,    // 读连接可以保持更久
			ConnMaxIdleTime: 45 * time.Minute, // 读连接空闲时间可以更长
			Weight:          100,
		},
	}

	// 监控默认配置
	config.Monitoring.Enabled = true
	config.Monitoring.CheckInterval = 30 * time.Second
	config.Monitoring.AlertThresholds.MaxOpenConnsUsage = 0.8 // 80%
	config.Monitoring.AlertThresholds.MaxIdleConnsUsage = 0.9 // 90%
	config.Monitoring.AlertThresholds.AvgConnLifetime = time.Hour
	config.Monitoring.AlertThresholds.SlowQueryThreshold = 500 * time.Millisecond

	return config
}

// ConnectionPoolOptimizer 连接池优化器
type ConnectionPoolOptimizer struct {
	config   *ConnectionPoolConfig
	logger   log.Logger
	masterDB *gorm.DB
	slavesDB []*gorm.DB
	monitor  *ConnectionPoolMonitor
	mu       sync.RWMutex
}

// NewConnectionPoolOptimizer 创建连接池优化器
func NewConnectionPoolOptimizer(config *ConnectionPoolConfig, logger log.Logger) *ConnectionPoolOptimizer {
	if config == nil {
		config = DefaultConnectionPoolConfig()
	}

	optimizer := &ConnectionPoolOptimizer{
		config:  config,
		logger:  logger,
		monitor: NewConnectionPoolMonitor(config, logger),
	}

	return optimizer
}

// OptimizeMasterPool 优化主库连接池
func (cpo *ConnectionPoolOptimizer) OptimizeMasterPool(masterDB *gorm.DB) error {
	cpo.mu.Lock()
	defer cpo.mu.Unlock()

	sqlDB, err := masterDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 应用主库连接池配置
	sqlDB.SetMaxOpenConns(cpo.config.Master.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cpo.config.Master.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cpo.config.Master.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cpo.config.Master.ConnMaxIdleTime)

	cpo.masterDB = masterDB

	log.NewHelper(cpo.logger).Infof("Master database connection pool optimized: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
		cpo.config.Master.MaxOpenConns,
		cpo.config.Master.MaxIdleConns,
		cpo.config.Master.ConnMaxLifetime,
		cpo.config.Master.ConnMaxIdleTime)

	return nil
}

// OptimizeSlavesPools 优化从库连接池
func (cpo *ConnectionPoolOptimizer) OptimizeSlavesPools(slavesDB []*gorm.DB) error {
	cpo.mu.Lock()
	defer cpo.mu.Unlock()

	if len(slavesDB) != len(cpo.config.Slaves) {
		return fmt.Errorf("slaves database count mismatch: expected %d, got %d", len(cpo.config.Slaves), len(slavesDB))
	}

	for i, slaveDB := range slavesDB {
		sqlDB, err := slaveDB.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying sql.DB for slave %d: %w", i, err)
		}

		slaveConfig := cpo.config.Slaves[i]

		// 应用从库连接池配置
		sqlDB.SetMaxOpenConns(slaveConfig.MaxOpenConns)
		sqlDB.SetMaxIdleConns(slaveConfig.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(slaveConfig.ConnMaxLifetime)
		sqlDB.SetConnMaxIdleTime(slaveConfig.ConnMaxIdleTime)

		log.NewHelper(cpo.logger).Infof("Slave database %s connection pool optimized: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
			slaveConfig.Name,
			slaveConfig.MaxOpenConns,
			slaveConfig.MaxIdleConns,
			slaveConfig.ConnMaxLifetime,
			slaveConfig.ConnMaxIdleTime)
	}

	cpo.slavesDB = slavesDB
	return nil
}

// StartMonitoring 启动连接池监控
func (cpo *ConnectionPoolOptimizer) StartMonitoring(ctx context.Context) error {
	if !cpo.config.Monitoring.Enabled {
		return nil
	}

	return cpo.monitor.Start(ctx, cpo.masterDB, cpo.slavesDB)
}

// GetStats 获取连接池统计信息
func (cpo *ConnectionPoolOptimizer) GetStats() (*ConnectionPoolStats, error) {
	cpo.mu.RLock()
	defer cpo.mu.RUnlock()

	stats := &ConnectionPoolStats{
		Timestamp: time.Now(),
	}

	// 获取主库统计信息
	if cpo.masterDB != nil {
		sqlDB, err := cpo.masterDB.DB()
		if err == nil {
			dbStats := sqlDB.Stats()
			stats.Master = &DBPoolStats{
				MaxOpenConnections: dbStats.MaxOpenConnections,
				OpenConnections:    dbStats.OpenConnections,
				InUse:              dbStats.InUse,
				Idle:               dbStats.Idle,
				WaitCount:          dbStats.WaitCount,
				WaitDuration:       dbStats.WaitDuration,
				MaxIdleClosed:      dbStats.MaxIdleClosed,
				MaxLifetimeClosed:  dbStats.MaxLifetimeClosed,
			}
		}
	}

	// 获取从库统计信息
	for i, slaveDB := range cpo.slavesDB {
		if slaveDB != nil {
			sqlDB, err := slaveDB.DB()
			if err == nil {
				dbStats := sqlDB.Stats()
				slaveStats := &DBPoolStats{
					Name:               cpo.config.Slaves[i].Name,
					MaxOpenConnections: dbStats.MaxOpenConnections,
					OpenConnections:    dbStats.OpenConnections,
					InUse:              dbStats.InUse,
					Idle:               dbStats.Idle,
					WaitCount:          dbStats.WaitCount,
					WaitDuration:       dbStats.WaitDuration,
					MaxIdleClosed:      dbStats.MaxIdleClosed,
					MaxLifetimeClosed:  dbStats.MaxLifetimeClosed,
				}
				stats.Slaves = append(stats.Slaves, slaveStats)
			}
		}
	}

	return stats, nil
}

// ConnectionPoolStats 连接池统计信息
type ConnectionPoolStats struct {
	Timestamp time.Time      `json:"timestamp"`
	Master    *DBPoolStats   `json:"master,omitempty"`
	Slaves    []*DBPoolStats `json:"slaves,omitempty"`
}

// DBPoolStats 数据库连接池统计信息
type DBPoolStats struct {
	Name               string        `json:"name,omitempty"`
	MaxOpenConnections int           `json:"max_open_connections"`
	OpenConnections    int           `json:"open_connections"`
	InUse              int           `json:"in_use"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"wait_count"`
	WaitDuration       time.Duration `json:"wait_duration"`
	MaxIdleClosed      int64         `json:"max_idle_closed"`
	MaxLifetimeClosed  int64         `json:"max_lifetime_closed"`
}

// ConnectionPoolMonitor 连接池监控器
type ConnectionPoolMonitor struct {
	config *ConnectionPoolConfig
	logger log.Logger
}

// NewConnectionPoolMonitor 创建连接池监控器
func NewConnectionPoolMonitor(config *ConnectionPoolConfig, logger log.Logger) *ConnectionPoolMonitor {
	return &ConnectionPoolMonitor{
		config: config,
		logger: logger,
	}
}

// Start 启动监控
func (cpm *ConnectionPoolMonitor) Start(ctx context.Context, masterDB *gorm.DB, slavesDB []*gorm.DB) error {
	if !cpm.config.Monitoring.Enabled {
		return nil
	}

	ticker := time.NewTicker(cpm.config.Monitoring.CheckInterval)
	defer ticker.Stop()

	helper := log.NewHelper(cpm.logger)
	helper.Info("Connection pool monitoring started")

	go func() {
		for {
			select {
			case <-ctx.Done():
				helper.Info("Connection pool monitoring stopped")
				return
			case <-ticker.C:
				cpm.checkConnectionPools(masterDB, slavesDB)
			}
		}
	}()

	return nil
}

// checkConnectionPools 检查连接池状态
func (cpm *ConnectionPoolMonitor) checkConnectionPools(masterDB *gorm.DB, slavesDB []*gorm.DB) {
	helper := log.NewHelper(cpm.logger)

	// 检查主库
	if masterDB != nil {
		if err := cpm.checkSinglePool("master", masterDB, &cpm.config.Master.MaxOpenConns, &cpm.config.Master.MaxIdleConns); err != nil {
			helper.Errorf("Master database pool check failed: %v", err)
		}
	}

	// 检查从库
	for i, slaveDB := range slavesDB {
		if slaveDB != nil && i < len(cpm.config.Slaves) {
			slaveConfig := &cpm.config.Slaves[i]
			if err := cpm.checkSinglePool(slaveConfig.Name, slaveDB, &slaveConfig.MaxOpenConns, &slaveConfig.MaxIdleConns); err != nil {
				helper.Errorf("Slave database %s pool check failed: %v", slaveConfig.Name, err)
			}
		}
	}
}

// checkSinglePool 检查单个连接池
func (cpm *ConnectionPoolMonitor) checkSinglePool(name string, db *gorm.DB, maxOpen, maxIdle *int) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	stats := sqlDB.Stats()
	helper := log.NewHelper(cpm.logger)

	// 检查连接使用率
	openUsage := float64(stats.OpenConnections) / float64(*maxOpen)
	if openUsage > cpm.config.Monitoring.AlertThresholds.MaxOpenConnsUsage {
		helper.Warnf("High open connection usage for %s: %.2f%% (%d/%d)",
			name, openUsage*100, stats.OpenConnections, *maxOpen)
	}

	idleUsage := float64(stats.Idle) / float64(*maxIdle)
	if idleUsage > cpm.config.Monitoring.AlertThresholds.MaxIdleConnsUsage {
		helper.Warnf("High idle connection usage for %s: %.2f%% (%d/%d)",
			name, idleUsage*100, stats.Idle, *maxIdle)
	}

	// 检查等待时间
	if stats.WaitCount > 0 && stats.WaitDuration > cpm.config.Monitoring.AlertThresholds.SlowQueryThreshold {
		helper.Warnf("High connection wait time for %s: %v (count: %d)",
			name, stats.WaitDuration, stats.WaitCount)
	}

	// 记录基本统计信息
	helper.Debugf("Connection pool stats for %s: Open=%d, InUse=%d, Idle=%d, Wait=%d, WaitDuration=%v",
		name, stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount, stats.WaitDuration)

	return nil
}
