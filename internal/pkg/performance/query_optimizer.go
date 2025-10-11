package performance

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// QueryOptimizer 查询优化器
type QueryOptimizer struct {
	db     *sql.DB
	logger log.Logger
	stats  *QueryStats
	config *QueryOptimizerConfig
}

// QueryOptimizerConfig 查询优化器配置
type QueryOptimizerConfig struct {
	// 慢查询阈值
	SlowQueryThreshold time.Duration `yaml:"slow_query_threshold" json:"slow_query_threshold"`
	// 是否启用查询分析
	EnableAnalysis bool `yaml:"enable_analysis" json:"enable_analysis"`
	// 是否启用查询缓存
	EnableQueryCache bool `yaml:"enable_query_cache" json:"enable_query_cache"`
	// 查询缓存大小
	QueryCacheSize int `yaml:"query_cache_size" json:"query_cache_size"`
	// 是否启用索引建议
	EnableIndexSuggestion bool `yaml:"enable_index_suggestion" json:"enable_index_suggestion"`
}

// DefaultQueryOptimizerConfig 默认查询优化器配置
func DefaultQueryOptimizerConfig() *QueryOptimizerConfig {
	return &QueryOptimizerConfig{
		SlowQueryThreshold:    500 * time.Millisecond, // 500毫秒
		EnableAnalysis:        true,
		EnableQueryCache:      true,
		QueryCacheSize:        1000,
		EnableIndexSuggestion: true,
	}
}

// QueryStats 查询统计信息
type QueryStats struct {
	TotalQueries     int64         `json:"total_queries"`
	SlowQueries      int64         `json:"slow_queries"`
	AverageDuration  time.Duration `json:"average_duration"`
	TotalDuration    time.Duration `json:"total_duration"`
	CacheHits        int64         `json:"cache_hits"`
	CacheMisses      int64         `json:"cache_misses"`
	IndexSuggestions []string      `json:"index_suggestions"`
	mu               sync.RWMutex
}

// NewQueryOptimizer 创建查询优化器
func NewQueryOptimizer(db *sql.DB, config *QueryOptimizerConfig, logger log.Logger) *QueryOptimizer {
	if config == nil {
		config = DefaultQueryOptimizerConfig()
	}

	return &QueryOptimizer{
		db:     db,
		logger: logger,
		stats:  &QueryStats{},
		config: config,
	}
}

// ExecuteQuery 执行查询并进行优化分析
func (qo *QueryOptimizer) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	qo.stats.mu.Lock()
	qo.stats.TotalQueries++
	qo.stats.mu.Unlock()

	// 检查查询缓存
	if qo.config.EnableQueryCache {
		// 简化的缓存检查逻辑
		// 实际项目中应该使用更完善的缓存机制
	}

	// 执行查询
	rows, err := qo.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	// 记录查询统计
	qo.stats.mu.Lock()
	qo.stats.TotalDuration += duration
	if qo.stats.TotalQueries > 0 {
		qo.stats.AverageDuration = qo.stats.TotalDuration / time.Duration(qo.stats.TotalQueries)
	}

	// 检查慢查询
	if duration > qo.config.SlowQueryThreshold {
		qo.stats.SlowQueries++
		log.NewHelper(qo.logger).Warnf("Slow query detected: %s (duration: %v)", query, duration)

		// 如果启用索引建议，生成索引建议
		if qo.config.EnableIndexSuggestion {
			suggestion := qo.generateIndexSuggestion(query)
			if suggestion != "" {
				qo.stats.IndexSuggestions = append(qo.stats.IndexSuggestions, suggestion)
				log.NewHelper(qo.logger).Infof("Index suggestion: %s", suggestion)
			}
		}
	}
	qo.stats.mu.Unlock()

	return rows, err
}

// ExecuteQueryRow 执行单行查询并进行优化分析
func (qo *QueryOptimizer) ExecuteQueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	qo.stats.mu.Lock()
	qo.stats.TotalQueries++
	qo.stats.mu.Unlock()

	// 执行查询
	row := qo.db.QueryRowContext(ctx, query, args...)
	duration := time.Since(start)

	// 记录查询统计
	qo.stats.mu.Lock()
	qo.stats.TotalDuration += duration
	if qo.stats.TotalQueries > 0 {
		qo.stats.AverageDuration = qo.stats.TotalDuration / time.Duration(qo.stats.TotalQueries)
	}

	// 检查慢查询
	if duration > qo.config.SlowQueryThreshold {
		qo.stats.SlowQueries++
		log.NewHelper(qo.logger).Warnf("Slow query detected: %s (duration: %v)", query, duration)

		// 如果启用索引建议，生成索引建议
		if qo.config.EnableIndexSuggestion {
			suggestion := qo.generateIndexSuggestion(query)
			if suggestion != "" {
				qo.stats.IndexSuggestions = append(qo.stats.IndexSuggestions, suggestion)
				log.NewHelper(qo.logger).Infof("Index suggestion: %s", suggestion)
			}
		}
	}
	qo.stats.mu.Unlock()

	return row
}

// ExecuteExec 执行更新/插入/删除操作并进行优化分析
func (qo *QueryOptimizer) ExecuteExec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	qo.stats.mu.Lock()
	qo.stats.TotalQueries++
	qo.stats.mu.Unlock()

	// 执行操作
	result, err := qo.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	// 记录查询统计
	qo.stats.mu.Lock()
	qo.stats.TotalDuration += duration
	if qo.stats.TotalQueries > 0 {
		qo.stats.AverageDuration = qo.stats.TotalDuration / time.Duration(qo.stats.TotalQueries)
	}

	// 检查慢查询
	if duration > qo.config.SlowQueryThreshold {
		qo.stats.SlowQueries++
		log.NewHelper(qo.logger).Warnf("Slow query detected: %s (duration: %v)", query, duration)
	}
	qo.stats.mu.Unlock()

	return result, err
}

// GetStats 获取查询统计信息
func (qo *QueryOptimizer) GetStats() *QueryStats {
	qo.stats.mu.RLock()
	defer qo.stats.mu.RUnlock()

	// 返回统计信息的副本
	stats := &QueryStats{
		TotalQueries:    qo.stats.TotalQueries,
		SlowQueries:     qo.stats.SlowQueries,
		AverageDuration: qo.stats.AverageDuration,
		TotalDuration:   qo.stats.TotalDuration,
		CacheHits:       qo.stats.CacheHits,
		CacheMisses:     qo.stats.CacheMisses,
	}

	// 复制索引建议
	stats.IndexSuggestions = make([]string, len(qo.stats.IndexSuggestions))
	copy(stats.IndexSuggestions, qo.stats.IndexSuggestions)

	return stats
}

// generateIndexSuggestion 生成索引建议
func (qo *QueryOptimizer) generateIndexSuggestion(query string) string {
	// 简化的索引建议生成逻辑
	// 实际项目中应该使用更复杂的分析算法

	query = strings.ToLower(query)

	// 检查WHERE子句中的条件
	if strings.Contains(query, "where") {
		// 提取WHERE子句
		whereStart := strings.Index(query, "where")
		whereEnd := strings.Index(query[whereStart:], "order by")
		if whereEnd == -1 {
			whereEnd = strings.Index(query[whereStart:], "group by")
		}
		if whereEnd == -1 {
			whereEnd = len(query[whereStart:])
		} else {
			whereEnd += whereStart
		}

		whereClause := query[whereStart:whereEnd]

		// 查找常用的比较操作符
		fields := []string{}
		if strings.Contains(whereClause, "=") {
			// 简单相等条件
			// 实际项目中应该解析具体的字段名
			fields = append(fields, "column_name")
		}

		if strings.Contains(whereClause, "like") {
			// LIKE条件，建议使用前缀索引
			fields = append(fields, "column_name")
		}

		if len(fields) > 0 {
			return fmt.Sprintf("Consider adding index on table for columns: %v", fields)
		}
	}

	// 检查ORDER BY子句
	if strings.Contains(query, "order by") {
		return "Consider adding index for ORDER BY columns"
	}

	// 检查JOIN操作
	if strings.Contains(query, "join") {
		return "Consider adding index for JOIN columns"
	}

	return ""
}

// ResetStats 重置统计信息
func (qo *QueryOptimizer) ResetStats() {
	qo.stats.mu.Lock()
	defer qo.stats.mu.Unlock()

	qo.stats.TotalQueries = 0
	qo.stats.SlowQueries = 0
	qo.stats.AverageDuration = 0
	qo.stats.TotalDuration = 0
	qo.stats.CacheHits = 0
	qo.stats.CacheMisses = 0
	qo.stats.IndexSuggestions = []string{}
}

// Close 关闭查询优化器
func (qo *QueryOptimizer) Close() error {
	// 清理资源
	qo.ResetStats()
	return nil
}
