package feature

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// ToggleManager 功能开关管理器实现
type ToggleManager struct {
	mu                sync.RWMutex
	toggles           map[FeatureFlag]*ToggleConfig
	repository        FeatureRepository
	strategyEvaluator StrategyEvaluator
	callbacks         []ToggleChangeCallback
	logger            *log.Helper
	defaultContext    *EvaluationContext
}

// NewToggleManager 创建新的功能开关管理器
func NewToggleManager(
	repository FeatureRepository,
	strategyEvaluator StrategyEvaluator,
	logger log.Logger,
) *ToggleManager {
	return &ToggleManager{
		toggles:           make(map[FeatureFlag]*ToggleConfig),
		repository:        repository,
		strategyEvaluator: strategyEvaluator,
		callbacks:         make([]ToggleChangeCallback, 0),
		logger:            log.NewHelper(logger),
		defaultContext: &EvaluationContext{
			Environment: "production",
			Version:     "1.0.0",
			Attributes:  make(map[string]string),
		},
	}
}

// Initialize 初始化功能开关管理器
func (tm *ToggleManager) Initialize(ctx context.Context) error {
	// 从存储加载配置
	toggles, err := tm.repository.LoadToggles(ctx)
	if err != nil {
		tm.logger.Errorf("Failed to load toggles: %v", err)
		return err
	}

	tm.mu.Lock()
	tm.toggles = toggles
	tm.mu.Unlock()

	// 启动配置监听
	go tm.watchConfigChanges(ctx)

	tm.logger.Infof("Feature toggle manager initialized with %d toggles", len(toggles))
	return nil
}

// IsEnabled 检查功能是否启用
func (tm *ToggleManager) IsEnabled(ctx context.Context, flag FeatureFlag) bool {
	return tm.IsEnabledWithContext(ctx, flag, tm.defaultContext)
}

// IsEnabledWithContext 基于上下文检查功能是否启用
func (tm *ToggleManager) IsEnabledWithContext(ctx context.Context, flag FeatureFlag, evalCtx *EvaluationContext) bool {
	tm.mu.RLock()
	config, exists := tm.toggles[flag]
	tm.mu.RUnlock()

	if !exists {
		// 默认启用未配置的功能
		tm.logger.Warnf("Feature toggle %s not found, defaulting to enabled", flag)
		return true
	}

	if evalCtx == nil {
		evalCtx = tm.defaultContext
	}

	result := tm.strategyEvaluator.Evaluate(ctx, config, evalCtx)

	tm.logger.Debugf("Feature toggle %s evaluation result: %v", flag, result)
	return result
}

// GetToggleConfig 获取功能开关配置
func (tm *ToggleManager) GetToggleConfig(flag FeatureFlag) (*ToggleConfig, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	config, exists := tm.toggles[flag]
	if !exists {
		return nil, fmt.Errorf("feature toggle %s not found", flag)
	}

	// 返回配置的副本
	configCopy := *config
	return &configCopy, nil
}

// UpdateToggle 更新功能开关配置
func (tm *ToggleManager) UpdateToggle(flag FeatureFlag, config *ToggleConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	tm.mu.Lock()
	oldConfig := tm.toggles[flag]

	// 更新时间戳
	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	tm.toggles[flag] = config
	tm.mu.Unlock()

	// 保存到存储
	if err := tm.repository.SaveToggle(context.Background(), flag, config); err != nil {
		tm.logger.Errorf("Failed to save toggle %s: %v", flag, err)
		// 回滚内存中的变更
		tm.mu.Lock()
		if oldConfig != nil {
			tm.toggles[flag] = oldConfig
		} else {
			delete(tm.toggles, flag)
		}
		tm.mu.Unlock()
		return err
	}

	// 通知变更
	tm.notifyChange(flag, oldConfig, config)

	tm.logger.Infof("Feature toggle %s updated", flag)
	return nil
}

// ListToggles 列出所有功能开关
func (tm *ToggleManager) ListToggles() map[FeatureFlag]*ToggleConfig {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make(map[FeatureFlag]*ToggleConfig)
	for flag, config := range tm.toggles {
		// 返回配置的副本
		configCopy := *config
		result[flag] = &configCopy
	}

	return result
}

// Subscribe 订阅功能开关变更事件
func (tm *ToggleManager) Subscribe(callback ToggleChangeCallback) error {
	if callback == nil {
		return fmt.Errorf("callback cannot be nil")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.callbacks = append(tm.callbacks, callback)
	return nil
}

// Unsubscribe 取消订阅功能开关变更事件
func (tm *ToggleManager) Unsubscribe(callback ToggleChangeCallback) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for i, cb := range tm.callbacks {
		// 使用函数指针比较可能不可靠，这里简化处理
		// 在生产环境中应该使用更好的方式来标识回调函数
		if &cb == &callback {
			tm.callbacks = append(tm.callbacks[:i], tm.callbacks[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("callback not found")
}

// EnableFeature 启用功能
func (tm *ToggleManager) EnableFeature(flag FeatureFlag) error {
	config, err := tm.GetToggleConfig(flag)
	if err != nil {
		// 创建新的配置
		config = &ToggleConfig{
			Enabled:     true,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: fmt.Sprintf("Feature %s", flag),
			Tags:        []string{},
		}
	} else {
		config.Enabled = true
	}

	return tm.UpdateToggle(flag, config)
}

// DisableFeature 禁用功能
func (tm *ToggleManager) DisableFeature(flag FeatureFlag) error {
	config, err := tm.GetToggleConfig(flag)
	if err != nil {
		// 创建新的配置
		config = &ToggleConfig{
			Enabled:     false,
			Strategy:    StrategySimple,
			Rules:       make(map[string]interface{}),
			Description: fmt.Sprintf("Feature %s", flag),
			Tags:        []string{},
		}
	} else {
		config.Enabled = false
	}

	return tm.UpdateToggle(flag, config)
}

// DeleteToggle 删除功能开关
func (tm *ToggleManager) DeleteToggle(flag FeatureFlag) error {
	tm.mu.Lock()
	oldConfig := tm.toggles[flag]
	delete(tm.toggles, flag)
	tm.mu.Unlock()

	// 从存储删除
	if err := tm.repository.DeleteToggle(context.Background(), flag); err != nil {
		tm.logger.Errorf("Failed to delete toggle %s: %v", flag, err)
		// 回滚内存中的变更
		if oldConfig != nil {
			tm.mu.Lock()
			tm.toggles[flag] = oldConfig
			tm.mu.Unlock()
		}
		return err
	}

	// 通知变更
	tm.notifyChange(flag, oldConfig, nil)

	tm.logger.Infof("Feature toggle %s deleted", flag)
	return nil
}

// SetDefaultContext 设置默认评估上下文
func (tm *ToggleManager) SetDefaultContext(ctx *EvaluationContext) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if ctx == nil {
		ctx = &EvaluationContext{
			Environment: "production",
			Version:     "1.0.0",
			Attributes:  make(map[string]string),
		}
	}

	tm.defaultContext = ctx
}

// GetStats 获取功能开关统计信息
func (tm *ToggleManager) GetStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_toggles":  len(tm.toggles),
		"enabled_count":  0,
		"disabled_count": 0,
		"strategies":     make(map[FeatureStrategy]int),
	}

	for _, config := range tm.toggles {
		if config.Enabled {
			stats["enabled_count"] = stats["enabled_count"].(int) + 1
		} else {
			stats["disabled_count"] = stats["disabled_count"].(int) + 1
		}

		strategies := stats["strategies"].(map[FeatureStrategy]int)
		strategies[config.Strategy]++
	}

	return stats
}

// 分层功能相关方法

// IsFeatureEnabled 检查分层feature是否启用
func (tm *ToggleManager) IsFeatureEnabled(featurePath string) bool {
	return tm.IsEnabled(context.Background(), FeatureFlag(featurePath))
}

// IsDomainEnabled 检查功能域是否有任何启用的功能
func (tm *ToggleManager) IsDomainEnabled(domain string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	for flag := range tm.toggles {
		featurePath := string(flag)
		if IsHierarchicalFeature(featurePath) {
			featureDomain, _ := ParseFeaturePath(featurePath)
			if featureDomain == domain {
				if tm.IsEnabled(context.Background(), flag) {
					return true
				}
			}
		}
	}
	return false
}

// GetDomainFeatures 获取域下的所有功能
func (tm *ToggleManager) GetDomainFeatures(domain string) []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var features []string
	for flag := range tm.toggles {
		featurePath := string(flag)
		if IsHierarchicalFeature(featurePath) {
			featureDomain, _ := ParseFeaturePath(featurePath)
			if featureDomain == domain {
				features = append(features, featurePath)
			}
		}
	}
	return features
}

// GetFeatureTree 获取功能树结构
func (tm *ToggleManager) GetFeatureTree() map[string][]string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tree := make(map[string][]string)
	for flag := range tm.toggles {
		featurePath := string(flag)
		if IsHierarchicalFeature(featurePath) {
			domain, _ := ParseFeaturePath(featurePath)
			tree[domain] = append(tree[domain], featurePath)
		}
	}
	return tree
}

// 功能集合能力检查方法

// HasPermission 检查功能集合中是否包含指定权限
func (tm *ToggleManager) HasPermission(featurePath, permission string) bool {
	if !tm.IsFeatureEnabled(featurePath) {
		return false
	}

	tm.mu.RLock()
	config, exists := tm.toggles[FeatureFlag(featurePath)]
	tm.mu.RUnlock()

	if !exists || config.Capabilities == nil {
		return false
	}

	for _, p := range config.Capabilities.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAPI 检查功能集合中是否包含指定API
func (tm *ToggleManager) HasAPI(featurePath, api string) bool {
	if !tm.IsFeatureEnabled(featurePath) {
		return false
	}

	tm.mu.RLock()
	config, exists := tm.toggles[FeatureFlag(featurePath)]
	tm.mu.RUnlock()

	if !exists || config.Capabilities == nil {
		return false
	}

	for _, a := range config.Capabilities.APIs {
		if a == api {
			return true
		}
	}
	return false
}

// HasRoute 检查功能集合中是否包含指定路由
func (tm *ToggleManager) HasRoute(featurePath, route string) bool {
	if !tm.IsFeatureEnabled(featurePath) {
		return false
	}

	tm.mu.RLock()
	config, exists := tm.toggles[FeatureFlag(featurePath)]
	tm.mu.RUnlock()

	if !exists || config.Capabilities == nil {
		return false
	}

	for _, r := range config.Capabilities.Routes {
		if r == route {
			return true
		}
	}
	return false
}

// HasConfig 检查功能集合中是否包含指定配置
func (tm *ToggleManager) HasConfig(featurePath, config string) bool {
	if !tm.IsFeatureEnabled(featurePath) {
		return false
	}

	tm.mu.RLock()
	toggleConfig, exists := tm.toggles[FeatureFlag(featurePath)]
	tm.mu.RUnlock()

	if !exists || toggleConfig.Capabilities == nil {
		return false
	}

	for _, c := range toggleConfig.Capabilities.Configs {
		if c == config {
			return true
		}
	}
	return false
}

// HasAuditLog 检查功能集合中是否包含指定审计日志
func (tm *ToggleManager) HasAuditLog(featurePath, logType string) bool {
	if !tm.IsFeatureEnabled(featurePath) {
		return false
	}

	tm.mu.RLock()
	config, exists := tm.toggles[FeatureFlag(featurePath)]
	tm.mu.RUnlock()

	if !exists || config.Capabilities == nil {
		return false
	}

	for _, l := range config.Capabilities.AuditLogs {
		if l == logType {
			return true
		}
	}
	return false
}

// 批量检查方法

// IsAnyFeatureEnabled 检查是否有任何功能启用
func (tm *ToggleManager) IsAnyFeatureEnabled(featurePaths ...string) bool {
	for _, path := range featurePaths {
		if tm.IsFeatureEnabled(path) {
			return true
		}
	}
	return false
}

// AreAllFeaturesEnabled 检查是否所有功能都启用
func (tm *ToggleManager) AreAllFeaturesEnabled(featurePaths ...string) bool {
	for _, path := range featurePaths {
		if !tm.IsFeatureEnabled(path) {
			return false
		}
	}
	return true
}

// GetFeatureCapabilities 获取功能集合的能力
func (tm *ToggleManager) GetFeatureCapabilities(featurePath string) *FeatureCapabilities {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	config, exists := tm.toggles[FeatureFlag(featurePath)]
	if !exists {
		return nil
	}

	return config.Capabilities
}

// ExportCSV 导出CSV格式的功能集合信息
func (tm *ToggleManager) ExportCSV() ([]byte, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// 写入表头
	header := []string{"Domain", "Feature", "Type", "Item", "Status", "Source"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	for flag, config := range tm.toggles {
		featurePath := string(flag)
		domain := ""
		if IsHierarchicalFeature(featurePath) {
			domain, _ = ParseFeaturePath(featurePath)
		}

		// 写入功能集合本身
		status := "disabled"
		if config.Enabled {
			status = "enabled"
		}
		row := []string{domain, featurePath, "feature", featurePath, status, "base"}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write feature row: %w", err)
		}

		// 写入功能集合的能力
		if config.Capabilities != nil {
			// 权限点
			for _, permission := range config.Capabilities.Permissions {
				row := []string{domain, featurePath, "permission", permission, status, "base"}
				if err := writer.Write(row); err != nil {
					return nil, fmt.Errorf("failed to write permission row: %w", err)
				}
			}

			// API接口
			for _, api := range config.Capabilities.APIs {
				row := []string{domain, featurePath, "api", api, status, "base"}
				if err := writer.Write(row); err != nil {
					return nil, fmt.Errorf("failed to write api row: %w", err)
				}
			}

			// 路由
			for _, route := range config.Capabilities.Routes {
				row := []string{domain, featurePath, "route", route, status, "base"}
				if err := writer.Write(row); err != nil {
					return nil, fmt.Errorf("failed to write route row: %w", err)
				}
			}

			// 配置点
			for _, configItem := range config.Capabilities.Configs {
				row := []string{domain, featurePath, "config", configItem, status, "base"}
				if err := writer.Write(row); err != nil {
					return nil, fmt.Errorf("failed to write config row: %w", err)
				}
			}

			// 审计日志
			for _, auditLog := range config.Capabilities.AuditLogs {
				row := []string{domain, featurePath, "audit_log", auditLog, status, "base"}
				if err := writer.Write(row); err != nil {
					return nil, fmt.Errorf("failed to write audit_log row: %w", err)
				}
			}
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	return buf.Bytes(), nil
}

// watchConfigChanges 监听配置变更
func (tm *ToggleManager) watchConfigChanges(ctx context.Context) {
	err := tm.repository.Watch(ctx, func(toggles map[FeatureFlag]*ToggleConfig) {
		tm.mu.Lock()
		oldToggles := tm.toggles
		tm.toggles = toggles
		tm.mu.Unlock()

		// 通知变更
		for flag, newConfig := range toggles {
			oldConfig := oldToggles[flag]
			if oldConfig == nil || !tm.configEqual(oldConfig, newConfig) {
				tm.notifyChange(flag, oldConfig, newConfig)
			}
		}

		// 检查已删除的配置
		for flag, oldConfig := range oldToggles {
			if _, exists := toggles[flag]; !exists {
				tm.notifyChange(flag, oldConfig, nil)
			}
		}

		tm.logger.Infof("Feature toggles reloaded, total: %d", len(toggles))
	})

	if err != nil {
		tm.logger.Errorf("Failed to watch config changes: %v", err)
	}
}

// notifyChange 通知配置变更
func (tm *ToggleManager) notifyChange(flag FeatureFlag, oldConfig, newConfig *ToggleConfig) {
	for _, callback := range tm.callbacks {
		go func(cb ToggleChangeCallback) {
			defer func() {
				if r := recover(); r != nil {
					tm.logger.Errorf("Panic in toggle change callback: %v", r)
				}
			}()
			cb(flag, oldConfig, newConfig)
		}(callback)
	}
}

// configEqual 比较两个配置是否相等
func (tm *ToggleManager) configEqual(config1, config2 *ToggleConfig) bool {
	if config1 == nil || config2 == nil {
		return config1 == config2
	}

	return config1.Enabled == config2.Enabled &&
		config1.Strategy == config2.Strategy &&
		config1.Description == config2.Description
}
