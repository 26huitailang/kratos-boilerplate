package feature

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// AuditEventType 审计事件类型
type AuditEventType string

const (
	// 配置变更事件
	EventFeatureCreated  AuditEventType = "feature_created"
	EventFeatureUpdated  AuditEventType = "feature_updated"
	EventFeatureDeleted  AuditEventType = "feature_deleted"
	EventFeatureEnabled  AuditEventType = "feature_enabled"
	EventFeatureDisabled AuditEventType = "feature_disabled"

	// 功能访问事件
	EventFeatureAccessed   AuditEventType = "feature_accessed"
	EventFeatureEvaluated  AuditEventType = "feature_evaluated"
	EventPermissionChecked AuditEventType = "permission_checked"
	EventAPIAccessed       AuditEventType = "api_accessed"
	EventRouteAccessed     AuditEventType = "route_accessed"
	EventConfigAccessed    AuditEventType = "config_accessed"

	// 系统事件
	EventSystemStartup  AuditEventType = "system_startup"
	EventSystemShutdown AuditEventType = "system_shutdown"
	EventConfigReloaded AuditEventType = "config_reloaded"
	EventErrorOccurred  AuditEventType = "error_occurred"
)

// AuditLevel 审计级别
type AuditLevel string

const (
	AuditLevelDebug AuditLevel = "debug"
	AuditLevelInfo  AuditLevel = "info"
	AuditLevelWarn  AuditLevel = "warn"
	AuditLevelError AuditLevel = "error"
)

// AuditLog 审计日志结构
type AuditLog struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   AuditEventType         `json:"event_type"`
	Level       AuditLevel             `json:"level"`
	Feature     FeatureFlag            `json:"feature,omitempty"`
	Operation   string                 `json:"operation"`
	UserID      string                 `json:"user_id,omitempty"`
	UserType    string                 `json:"user_type,omitempty"`
	Source      string                 `json:"source"`
	ClientIP    string                 `json:"client_ip,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	OldValue    interface{}            `json:"old_value,omitempty"`
	NewValue    interface{}            `json:"new_value,omitempty"`
	Result      bool                   `json:"result,omitempty"`
	EvalContext *EvaluationContext     `json:"eval_context,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AuditFilter 审计日志过滤器
type AuditFilter struct {
	StartTime  *time.Time       `json:"start_time,omitempty"`
	EndTime    *time.Time       `json:"end_time,omitempty"`
	EventTypes []AuditEventType `json:"event_types,omitempty"`
	Features   []FeatureFlag    `json:"features,omitempty"`
	UserID     string           `json:"user_id,omitempty"`
	Source     string           `json:"source,omitempty"`
	Level      AuditLevel       `json:"level,omitempty"`
	Operation  string           `json:"operation,omitempty"`
	Limit      int              `json:"limit,omitempty"`
	Offset     int              `json:"offset,omitempty"`
}

// AuditLogger 审计日志记录器接口
type AuditLogger interface {
	// LogFeatureChange 记录功能配置变更
	LogFeatureChange(ctx context.Context, eventType AuditEventType, feature FeatureFlag, oldConfig, newConfig *ToggleConfig) error

	// LogFeatureAccess 记录功能访问
	LogFeatureAccess(ctx context.Context, feature FeatureFlag, operation string, result bool, evalCtx *EvaluationContext, duration time.Duration) error

	// LogPermissionCheck 记录权限检查
	LogPermissionCheck(ctx context.Context, feature FeatureFlag, permission string, result bool, evalCtx *EvaluationContext) error

	// LogError 记录错误事件
	LogError(ctx context.Context, operation string, err error, metadata map[string]interface{}) error

	// LogSystemEvent 记录系统事件
	LogSystemEvent(ctx context.Context, eventType AuditEventType, message string, metadata map[string]interface{}) error

	// GetAuditLogs 查询审计日志
	GetAuditLogs(ctx context.Context, filter AuditFilter) ([]AuditLog, error)
}

// AuditLogRepository 审计日志存储接口
type AuditLogRepository interface {
	// Save 保存审计日志
	Save(ctx context.Context, log *AuditLog) error

	// Query 查询审计日志
	Query(ctx context.Context, filter AuditFilter) ([]AuditLog, error)

	// Delete 删除过期的审计日志
	Delete(ctx context.Context, beforeTime time.Time) (int64, error)

	// Count 统计审计日志数量
	Count(ctx context.Context, filter AuditFilter) (int64, error)
}

// DefaultAuditLogger 默认审计日志记录器
type DefaultAuditLogger struct {
	repository       AuditLogRepository
	contextExtractor EvaluationContextExtractor
	logger           *log.Helper
	mu               sync.RWMutex
	enabled          bool
	excludeEvents    map[AuditEventType]bool
	maskSensitive    bool
}

// AuditLoggerConfig 审计记录器配置
type AuditLoggerConfig struct {
	Enabled       bool             `yaml:"enabled" json:"enabled"`
	ExcludeEvents []AuditEventType `yaml:"exclude_events" json:"exclude_events"`
	MaskSensitive bool             `yaml:"mask_sensitive" json:"mask_sensitive"`
	Retention     time.Duration    `yaml:"retention" json:"retention"`
}

// NewDefaultAuditLogger 创建默认审计日志记录器
func NewDefaultAuditLogger(
	repository AuditLogRepository,
	contextExtractor EvaluationContextExtractor,
	logger log.Logger,
	config *AuditLoggerConfig,
) *DefaultAuditLogger {
	al := &DefaultAuditLogger{
		repository:       repository,
		contextExtractor: contextExtractor,
		logger:           log.NewHelper(logger),
		enabled:          true,
		excludeEvents:    make(map[AuditEventType]bool),
		maskSensitive:    true,
	}

	if config != nil {
		al.enabled = config.Enabled
		al.maskSensitive = config.MaskSensitive

		for _, eventType := range config.ExcludeEvents {
			al.excludeEvents[eventType] = true
		}
	}

	return al
}

// LogFeatureChange 记录功能配置变更
func (al *DefaultAuditLogger) LogFeatureChange(ctx context.Context, eventType AuditEventType, feature FeatureFlag, oldConfig, newConfig *ToggleConfig) error {
	if !al.isEnabled() || al.isExcluded(eventType) {
		return nil
	}

	auditLog := al.createBaseLog(ctx, eventType, AuditLevelInfo)
	auditLog.Feature = feature
	auditLog.Operation = "config_change"

	if oldConfig != nil {
		if al.maskSensitive {
			auditLog.OldValue = al.maskConfig(oldConfig)
		} else {
			auditLog.OldValue = oldConfig
		}
	}

	if newConfig != nil {
		if al.maskSensitive {
			auditLog.NewValue = al.maskConfig(newConfig)
		} else {
			auditLog.NewValue = newConfig
		}
	}

	return al.repository.Save(ctx, auditLog)
}

// LogFeatureAccess 记录功能访问
func (al *DefaultAuditLogger) LogFeatureAccess(ctx context.Context, feature FeatureFlag, operation string, result bool, evalCtx *EvaluationContext, duration time.Duration) error {
	if !al.isEnabled() || al.isExcluded(EventFeatureAccessed) {
		return nil
	}

	auditLog := al.createBaseLog(ctx, EventFeatureAccessed, AuditLevelDebug)
	auditLog.Feature = feature
	auditLog.Operation = operation
	auditLog.Result = result
	auditLog.Duration = duration

	if evalCtx != nil {
		if al.maskSensitive {
			auditLog.EvalContext = al.maskEvaluationContext(evalCtx)
		} else {
			auditLog.EvalContext = evalCtx
		}
	}

	return al.repository.Save(ctx, auditLog)
}

// LogPermissionCheck 记录权限检查
func (al *DefaultAuditLogger) LogPermissionCheck(ctx context.Context, feature FeatureFlag, permission string, result bool, evalCtx *EvaluationContext) error {
	if !al.isEnabled() || al.isExcluded(EventPermissionChecked) {
		return nil
	}

	auditLog := al.createBaseLog(ctx, EventPermissionChecked, AuditLevelInfo)
	auditLog.Feature = feature
	auditLog.Operation = "check_permission"
	auditLog.Result = result

	if auditLog.Metadata == nil {
		auditLog.Metadata = make(map[string]interface{})
	}
	auditLog.Metadata["permission"] = permission

	if evalCtx != nil {
		if al.maskSensitive {
			auditLog.EvalContext = al.maskEvaluationContext(evalCtx)
		} else {
			auditLog.EvalContext = evalCtx
		}
	}

	return al.repository.Save(ctx, auditLog)
}

// LogError 记录错误事件
func (al *DefaultAuditLogger) LogError(ctx context.Context, operation string, err error, metadata map[string]interface{}) error {
	if !al.isEnabled() || al.isExcluded(EventErrorOccurred) {
		return nil
	}

	auditLog := al.createBaseLog(ctx, EventErrorOccurred, AuditLevelError)
	auditLog.Operation = operation
	auditLog.Error = err.Error()
	auditLog.Metadata = metadata

	return al.repository.Save(ctx, auditLog)
}

// LogSystemEvent 记录系统事件
func (al *DefaultAuditLogger) LogSystemEvent(ctx context.Context, eventType AuditEventType, message string, metadata map[string]interface{}) error {
	if !al.isEnabled() || al.isExcluded(eventType) {
		return nil
	}

	auditLog := al.createBaseLog(ctx, eventType, AuditLevelInfo)
	auditLog.Operation = "system_event"
	auditLog.Metadata = metadata

	if auditLog.Metadata == nil {
		auditLog.Metadata = make(map[string]interface{})
	}
	auditLog.Metadata["message"] = message

	return al.repository.Save(ctx, auditLog)
}

// GetAuditLogs 查询审计日志
func (al *DefaultAuditLogger) GetAuditLogs(ctx context.Context, filter AuditFilter) ([]AuditLog, error) {
	return al.repository.Query(ctx, filter)
}

// createBaseLog 创建基础审计日志
func (al *DefaultAuditLogger) createBaseLog(ctx context.Context, eventType AuditEventType, level AuditLevel) *AuditLog {
	auditLog := &AuditLog{
		ID:        al.generateID(),
		Timestamp: time.Now(),
		EventType: eventType,
		Level:     level,
		Source:    "feature_toggle_system",
		Metadata:  make(map[string]interface{}),
	}

	// 从上下文中提取信息
	if al.contextExtractor != nil {
		evalCtx := al.contextExtractor.Extract(ctx)
		if evalCtx != nil {
			auditLog.UserID = evalCtx.UserID
			auditLog.UserType = evalCtx.UserType
		}
	}

	// 从上下文中提取HTTP相关信息
	if clientIP, ok := ctx.Value("client_ip").(string); ok {
		auditLog.ClientIP = clientIP
	}

	if userAgent, ok := ctx.Value("user_agent").(string); ok {
		auditLog.UserAgent = userAgent
	}

	if sessionID, ok := ctx.Value("session_id").(string); ok {
		auditLog.SessionID = sessionID
	}

	if requestID, ok := ctx.Value("request_id").(string); ok {
		auditLog.RequestID = requestID
	}

	return auditLog
}

// isEnabled 检查是否启用审计
func (al *DefaultAuditLogger) isEnabled() bool {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.enabled
}

// isExcluded 检查事件类型是否被排除
func (al *DefaultAuditLogger) isExcluded(eventType AuditEventType) bool {
	al.mu.RLock()
	defer al.mu.RUnlock()
	return al.excludeEvents[eventType]
}

// SetEnabled 设置启用状态
func (al *DefaultAuditLogger) SetEnabled(enabled bool) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.enabled = enabled
}

// SetExcludeEvent 设置排除事件类型
func (al *DefaultAuditLogger) SetExcludeEvent(eventType AuditEventType, exclude bool) {
	al.mu.Lock()
	defer al.mu.Unlock()
	al.excludeEvents[eventType] = exclude
}

// generateID 生成唯一ID
func (al *DefaultAuditLogger) generateID() string {
	// 简单的ID生成策略，生产环境可以使用UUID
	return fmt.Sprintf("audit_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

// maskConfig 脱敏配置信息
func (al *DefaultAuditLogger) maskConfig(config *ToggleConfig) *ToggleConfig {
	if config == nil {
		return nil
	}

	// 创建配置副本
	masked := *config

	// 脱敏规则信息
	if masked.Rules != nil {
		maskedRules := make(map[string]interface{})
		for k, v := range masked.Rules {
			// 脱敏敏感字段
			if al.isSensitiveField(k) {
				maskedRules[k] = "***"
			} else {
				maskedRules[k] = v
			}
		}
		masked.Rules = maskedRules
	}

	return &masked
}

// maskEvaluationContext 脱敏评估上下文
func (al *DefaultAuditLogger) maskEvaluationContext(evalCtx *EvaluationContext) *EvaluationContext {
	if evalCtx == nil {
		return nil
	}

	masked := *evalCtx

	// 脱敏用户ID（只显示前后几位）
	if masked.UserID != "" && len(masked.UserID) > 6 {
		masked.UserID = masked.UserID[:3] + "***" + masked.UserID[len(masked.UserID)-3:]
	}

	// 脱敏属性中的敏感信息
	if masked.Attributes != nil {
		maskedAttrs := make(map[string]string)
		for k, v := range masked.Attributes {
			if al.isSensitiveField(k) {
				maskedAttrs[k] = "***"
			} else {
				maskedAttrs[k] = v
			}
		}
		masked.Attributes = maskedAttrs
	}

	return &masked
}

// isSensitiveField 判断是否为敏感字段
func (al *DefaultAuditLogger) isSensitiveField(field string) bool {
	sensitiveFields := []string{
		"password", "token", "secret", "key", "credential",
		"phone", "email", "ssn", "id_card", "passport",
	}

	for _, sensitive := range sensitiveFields {
		if field == sensitive {
			return true
		}
	}

	return false
}

// MemoryAuditRepository 内存审计日志存储库（用于测试）
type MemoryAuditRepository struct {
	mu   sync.RWMutex
	logs []AuditLog
}

// NewMemoryAuditRepository 创建内存审计日志存储库
func NewMemoryAuditRepository() *MemoryAuditRepository {
	return &MemoryAuditRepository{
		logs: make([]AuditLog, 0),
	}
}

// Save 保存审计日志
func (mar *MemoryAuditRepository) Save(ctx context.Context, log *AuditLog) error {
	mar.mu.Lock()
	defer mar.mu.Unlock()

	mar.logs = append(mar.logs, *log)
	return nil
}

// Query 查询审计日志
func (mar *MemoryAuditRepository) Query(ctx context.Context, filter AuditFilter) ([]AuditLog, error) {
	mar.mu.RLock()
	defer mar.mu.RUnlock()

	var result []AuditLog

	for _, log := range mar.logs {
		if mar.matchesFilter(&log, &filter) {
			result = append(result, log)
		}
	}

	// 应用限制和偏移
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result, nil
}

// Delete 删除过期的审计日志
func (mar *MemoryAuditRepository) Delete(ctx context.Context, beforeTime time.Time) (int64, error) {
	mar.mu.Lock()
	defer mar.mu.Unlock()

	var newLogs []AuditLog
	deletedCount := int64(0)

	for _, log := range mar.logs {
		if log.Timestamp.After(beforeTime) {
			newLogs = append(newLogs, log)
		} else {
			deletedCount++
		}
	}

	mar.logs = newLogs
	return deletedCount, nil
}

// Count 统计审计日志数量
func (mar *MemoryAuditRepository) Count(ctx context.Context, filter AuditFilter) (int64, error) {
	mar.mu.RLock()
	defer mar.mu.RUnlock()

	count := int64(0)
	for _, log := range mar.logs {
		if mar.matchesFilter(&log, &filter) {
			count++
		}
	}

	return count, nil
}

// matchesFilter 检查日志是否匹配过滤器
func (mar *MemoryAuditRepository) matchesFilter(log *AuditLog, filter *AuditFilter) bool {
	// 时间过滤
	if filter.StartTime != nil && log.Timestamp.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && log.Timestamp.After(*filter.EndTime) {
		return false
	}

	// 事件类型过滤
	if len(filter.EventTypes) > 0 {
		found := false
		for _, eventType := range filter.EventTypes {
			if log.EventType == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 功能过滤
	if len(filter.Features) > 0 {
		found := false
		for _, feature := range filter.Features {
			if log.Feature == feature {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 用户过滤
	if filter.UserID != "" && log.UserID != filter.UserID {
		return false
	}

	// 来源过滤
	if filter.Source != "" && log.Source != filter.Source {
		return false
	}

	// 级别过滤
	if filter.Level != "" && log.Level != filter.Level {
		return false
	}

	// 操作过滤
	if filter.Operation != "" && log.Operation != filter.Operation {
		return false
	}

	return true
}

// NoOpAuditLogger 无操作审计记录器（禁用审计时使用）
type NoOpAuditLogger struct{}

// NewNoOpAuditLogger 创建无操作审计记录器
func NewNoOpAuditLogger() *NoOpAuditLogger {
	return &NoOpAuditLogger{}
}

// LogFeatureChange 空实现
func (nal *NoOpAuditLogger) LogFeatureChange(ctx context.Context, eventType AuditEventType, feature FeatureFlag, oldConfig, newConfig *ToggleConfig) error {
	return nil
}

// LogFeatureAccess 空实现
func (nal *NoOpAuditLogger) LogFeatureAccess(ctx context.Context, feature FeatureFlag, operation string, result bool, evalCtx *EvaluationContext, duration time.Duration) error {
	return nil
}

// LogPermissionCheck 空实现
func (nal *NoOpAuditLogger) LogPermissionCheck(ctx context.Context, feature FeatureFlag, permission string, result bool, evalCtx *EvaluationContext) error {
	return nil
}

// LogError 空实现
func (nal *NoOpAuditLogger) LogError(ctx context.Context, operation string, err error, metadata map[string]interface{}) error {
	return nil
}

// LogSystemEvent 空实现
func (nal *NoOpAuditLogger) LogSystemEvent(ctx context.Context, eventType AuditEventType, message string, metadata map[string]interface{}) error {
	return nil
}

// GetAuditLogs 空实现
func (nal *NoOpAuditLogger) GetAuditLogs(ctx context.Context, filter AuditFilter) ([]AuditLog, error) {
	return []AuditLog{}, nil
}
