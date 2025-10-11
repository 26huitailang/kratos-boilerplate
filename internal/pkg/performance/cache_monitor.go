package performance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

// CacheStats 缓存统计信息
type CacheStats struct {
	// 基本统计
	Hits   map[string]*atomic.Int64 `json:"hits"`   // 按层级统计命中次数
	Misses map[string]*atomic.Int64 `json:"misses"` // 按层级统计未命中次数
	Errors map[string]*atomic.Int64 `json:"errors"` // 按操作类型统计错误次数
	Ops    map[string]*atomic.Int64 `json:"ops"`    // 按操作类型统计调用次数

	// 性能统计
	Latencies map[string]*LatencyStats `json:"latencies"` // 按操作类型统计延迟
	Memory    *MemoryStats             `json:"memory"`    // 内存使用统计

	// Redis统计
	RedisInfo *RedisInfo `json:"redis_info"` // Redis服务器信息

	mu sync.RWMutex
}

// LatencyStats 延迟统计
type LatencyStats struct {
	Count int64         `json:"count"`
	Total time.Duration `json:"total"`
	Min   time.Duration `json:"min"`
	Max   time.Duration `json:"max"`
	mu    sync.RWMutex
}

// MemoryStats 内存统计
type MemoryStats struct {
	L1Size  *atomic.Int64 `json:"l1_size"`  // L1缓存大小
	L1Limit *atomic.Int64 `json:"l1_limit"` // L1缓存限制
	L2Size  *atomic.Int64 `json:"l2_size"`  // L2缓存大小
}

// RedisInfo Redis信息
type RedisInfo struct {
	UsedMemory       int64  `json:"used_memory"`
	ConnectedClients int64  `json:"connected_clients"`
	TotalCommands    int64  `json:"total_commands"`
	Uptime           int64  `json:"uptime"`
	Version          string `json:"version"`
}

// NewCacheStats 创建缓存统计
func NewCacheStats() *CacheStats {
	stats := &CacheStats{
		Hits:      make(map[string]*atomic.Int64),
		Misses:    make(map[string]*atomic.Int64),
		Errors:    make(map[string]*atomic.Int64),
		Ops:       make(map[string]*atomic.Int64),
		Latencies: make(map[string]*LatencyStats),
		Memory: &MemoryStats{
			L1Size:  &atomic.Int64{},
			L1Limit: &atomic.Int64{},
			L2Size:  &atomic.Int64{},
		},
	}

	// 初始化层级统计
	for _, level := range []string{"L1", "L2"} {
		stats.Hits[level] = &atomic.Int64{}
		stats.Misses[level] = &atomic.Int64{}
	}

	// 初始化操作类型统计
	for _, op := range []string{"get", "set", "delete", "serialize", "compress"} {
		stats.Errors[op] = &atomic.Int64{}
		stats.Ops[op] = &atomic.Int64{}
		stats.Latencies[op] = &LatencyStats{}
	}

	return stats
}

// RecordHit 记录命中
func (cs *CacheStats) RecordHit(level string) {
	if hit, exists := cs.Hits[level]; exists {
		hit.Add(1)
	}
}

// RecordMiss 记录未命中
func (cs *CacheStats) RecordMiss(level string) {
	if miss, exists := cs.Misses[level]; exists {
		miss.Add(1)
	}
}

// RecordError 记录错误
func (cs *CacheStats) RecordError(operation string) {
	if err, exists := cs.Errors[operation]; exists {
		err.Add(1)
	}
}

// RecordOperation 记录操作
func (cs *CacheStats) RecordOperation(operation string, duration time.Duration) {
	if op, exists := cs.Ops[operation]; exists {
		op.Add(1)
	}

	if latency, exists := cs.Latencies[operation]; exists {
		latency.Update(duration)
	}
}

// UpdateMemoryStats 更新内存统计
func (cs *CacheStats) UpdateMemoryStats(l1Size, l1Limit, l2Size int64) {
	cs.Memory.L1Size.Store(l1Size)
	cs.Memory.L1Limit.Store(l1Limit)
	cs.Memory.L2Size.Store(l2Size)
}

// UpdateRedisInfo 更新Redis信息
func (cs *CacheStats) UpdateRedisInfo(info *RedisInfo) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.RedisInfo = info
}

// Clone 克隆统计信息
func (cs *CacheStats) Clone() *CacheStats {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	clone := &CacheStats{
		Hits:      make(map[string]*atomic.Int64),
		Misses:    make(map[string]*atomic.Int64),
		Errors:    make(map[string]*atomic.Int64),
		Ops:       make(map[string]*atomic.Int64),
		Latencies: make(map[string]*LatencyStats),
		Memory: &MemoryStats{
			L1Size:  &atomic.Int64{},
			L1Limit: &atomic.Int64{},
			L2Size:  &atomic.Int64{},
		},
	}

	// 复制命中统计
	for level, hit := range cs.Hits {
		newHit := &atomic.Int64{}
		newHit.Store(hit.Load())
		clone.Hits[level] = newHit
	}

	// 复制未命中统计
	for level, miss := range cs.Misses {
		newMiss := &atomic.Int64{}
		newMiss.Store(miss.Load())
		clone.Misses[level] = newMiss
	}

	// 复制错误统计
	for op, err := range cs.Errors {
		newErr := &atomic.Int64{}
		newErr.Store(err.Load())
		clone.Errors[op] = newErr
	}

	// 复制操作统计
	for op, opCount := range cs.Ops {
		newOp := &atomic.Int64{}
		newOp.Store(opCount.Load())
		clone.Ops[op] = newOp
	}

	// 复制延迟统计
	for op, latency := range cs.Latencies {
		clone.Latencies[op] = latency.Clone()
	}

	// 复制内存统计
	clone.Memory.L1Size.Store(cs.Memory.L1Size.Load())
	clone.Memory.L1Limit.Store(cs.Memory.L1Limit.Load())
	clone.Memory.L2Size.Store(cs.Memory.L2Size.Load())

	// 复制Redis信息
	if cs.RedisInfo != nil {
		clone.RedisInfo = &RedisInfo{
			UsedMemory:       cs.RedisInfo.UsedMemory,
			ConnectedClients: cs.RedisInfo.ConnectedClients,
			TotalCommands:    cs.RedisInfo.TotalCommands,
			Uptime:           cs.RedisInfo.Uptime,
			Version:          cs.RedisInfo.Version,
		}
	}

	return clone
}

// Update 更新延迟统计
func (ls *LatencyStats) Update(duration time.Duration) {
	ls.mu.Lock()
	defer ls.mu.Unlock()

	ls.Count++
	ls.Total += duration

	if ls.Min == 0 || duration < ls.Min {
		ls.Min = duration
	}

	if duration > ls.Max {
		ls.Max = duration
	}
}

// Clone 克隆延迟统计
func (ls *LatencyStats) Clone() *LatencyStats {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	return &LatencyStats{
		Count: ls.Count,
		Total: ls.Total,
		Min:   ls.Min,
		Max:   ls.Max,
	}
}

// GetAverage 获取平均延迟
func (ls *LatencyStats) GetAverage() time.Duration {
	ls.mu.RLock()
	defer ls.mu.RUnlock()

	if ls.Count == 0 {
		return 0
	}
	return ls.Total / time.Duration(ls.Count)
}

// CacheMonitor 缓存监控器
type CacheMonitor struct {
	config *CacheConfig
	logger log.Logger
}

// NewCacheMonitor 创建缓存监控器
func NewCacheMonitor(config *CacheConfig, logger log.Logger) *CacheMonitor {
	return &CacheMonitor{
		config: config,
		logger: logger,
	}
}

// Start 启动监控
func (cm *CacheMonitor) Start(ctx context.Context, masterClient redis.UniversalClient, slaveClients []redis.UniversalClient, stats *CacheStats) error {
	if !cm.config.Monitoring.Enabled {
		return nil
	}

	ticker := time.NewTicker(cm.config.Monitoring.Interval)
	defer ticker.Stop()

	helper := log.NewHelper(cm.logger)
	helper.Info("Cache monitoring started")

	go func() {
		for {
			select {
			case <-ctx.Done():
				helper.Info("Cache monitoring stopped")
				return
			case <-ticker.C:
				cm.collectMetrics(masterClient, slaveClients, stats)
				cm.checkThresholds(stats)
			}
		}
	}()

	return nil
}

// collectMetrics 收集指标
func (cm *CacheMonitor) collectMetrics(masterClient redis.UniversalClient, slaveClients []redis.UniversalClient, stats *CacheStats) {
	helper := log.NewHelper(cm.logger)

	// 收集Redis信息
	if masterClient != nil {
		info, err := cm.getRedisInfo(masterClient)
		if err != nil {
			helper.Warnf("Failed to get Redis info: %v", err)
		} else {
			stats.UpdateRedisInfo(info)
		}
	}
}

// getRedisInfo 获取Redis信息
func (cm *CacheMonitor) getRedisInfo(client redis.UniversalClient) (*RedisInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info := &RedisInfo{}

	// 获取内存信息
	memoryInfo, err := client.Info(ctx, "memory").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}
	info.UsedMemory = cm.parseInfoValue(memoryInfo, "used_memory_rss")

	// 获取客户端信息
	clientInfo, err := client.Info(ctx, "clients").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get client info: %w", err)
	}
	info.ConnectedClients = cm.parseInfoValue(clientInfo, "connected_clients")

	// 获取统计信息
	statsInfo, err := client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get stats info: %w", err)
	}
	info.TotalCommands = cm.parseInfoValue(statsInfo, "total_commands_processed")

	// 获取服务器信息
	serverInfo, err := client.Info(ctx, "server").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}
	info.Uptime = cm.parseInfoValue(serverInfo, "uptime_in_seconds")

	// 解析版本
	versionLine := cm.findInfoLine(serverInfo, "redis_version:")
	if versionLine != "" {
		info.Version = versionLine
	}

	return info, nil
}

// parseInfoValue 解析INFO命令返回的值
func (cm *CacheMonitor) parseInfoValue(info, key string) int64 {
	line := cm.findInfoLine(info, key+":")
	if line == "" {
		return 0
	}

	var value int64
	fmt.Sscanf(line, "%d", &value)
	return value
}

// findInfoLine 查找INFO信息中的行
func (cm *CacheMonitor) findInfoLine(info, prefix string) string {
	lines := []byte(info)
	prefixBytes := []byte(prefix)

	for i := 0; i < len(lines); i++ {
		// 查找行开始
		if i == 0 || lines[i-1] == '\n' || lines[i-1] == '\r' {
			// 检查是否匹配前缀
			if i+len(prefixBytes) <= len(lines) && string(lines[i:i+len(prefixBytes)]) == prefix {
				// 找到值部分
				start := i + len(prefixBytes)
				end := start
				for end < len(lines) && lines[end] != '\n' && lines[end] != '\r' {
					end++
				}
				return string(lines[start:end])
			}
		}
	}
	return ""
}

// checkThresholds 检查阈值
func (cm *CacheMonitor) checkThresholds(stats *CacheStats) {
	helper := log.NewHelper(cm.logger)

	// 检查命中率
	for level := range stats.Hits {
		hitCount := stats.Hits[level].Load()
		missCount := stats.Misses[level].Load()
		total := hitCount + missCount

		if total > 0 {
			hitRatio := float64(hitCount) / float64(total)
			if hitRatio < cm.config.Monitoring.Thresholds.HitRatio {
				helper.Warnf("Low cache hit ratio for %s: %.2f%%", level, hitRatio*100)
			}
		}
	}

	// 检查延迟
	for op, latency := range stats.Latencies {
		avg := latency.GetAverage()
		if avg > cm.config.Monitoring.Thresholds.Latency {
			helper.Warnf("High cache latency for %s: %v", op, avg)
		}
	}

	// 检查错误率
	for op := range stats.Errors {
		errorCount := stats.Errors[op].Load()
		opCount := stats.Ops[op].Load()

		if opCount > 0 {
			errorRate := float64(errorCount) / float64(opCount)
			if errorRate > cm.config.Monitoring.Thresholds.ErrorRate {
				helper.Warnf("High error rate for %s: %.2f%%", op, errorRate*100)
			}
		}
	}

	// 检查内存使用
	if stats.RedisInfo != nil {
		// 这里可以添加Redis内存使用检查
	}
}
