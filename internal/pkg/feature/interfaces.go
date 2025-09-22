package feature

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// FeatureFlag 功能开关类型
type FeatureFlag string

// 预定义的功能开关 - 支持分层结构
const (
	// 用户功能域
	FeatureUserManagement FeatureFlag = "user.management"
	FeatureUserProfile    FeatureFlag = "user.profile"
	FeatureUserSecurity   FeatureFlag = "user.security"

	// 系统功能域
	FeatureSystemConfig     FeatureFlag = "system.config"
	FeatureSystemMonitoring FeatureFlag = "system.monitoring"

	// 报表功能域
	FeatureReportBasic    FeatureFlag = "report.basic"
	FeatureReportAdvanced FeatureFlag = "report.advanced"

	// 向后兼容 - 原有功能开关
	FeatureCaptcha        FeatureFlag = "captcha"
	FeatureTOTP           FeatureFlag = "totp"
	FeaturePasswordPolicy FeatureFlag = "password_policy"
	FeatureAccountLockout FeatureFlag = "account_lockout"
	FeatureRateLimit      FeatureFlag = "rate_limit"
	FeatureAPIVersioning  FeatureFlag = "api_versioning"
	FeatureSwaggerUI      FeatureFlag = "swagger_ui"
	FeatureDataEncryption FeatureFlag = "data_encryption"
	FeatureAuditLog       FeatureFlag = "audit_log"
	FeatureSensitiveData  FeatureFlag = "sensitive_data"
	FeaturePluginSystem   FeatureFlag = "plugin_system"
	FeaturePluginSandbox  FeatureFlag = "plugin_sandbox"
	FeatureMetrics        FeatureFlag = "metrics"
	FeatureTracing        FeatureFlag = "tracing"
	FeatureHealthCheck    FeatureFlag = "health_check"
)

// 功能域定义
const (
	DomainUser   = "user"
	DomainSystem = "system"
	DomainReport = "report"
)

// FeatureStrategy 功能开关策略类型
type FeatureStrategy string

const (
	// 简单的开关策略
	StrategySimple FeatureStrategy = "simple"
	// 基于用户百分比的策略
	StrategyPercentage FeatureStrategy = "percentage"
	// 基于用户属性的策略
	StrategyUser FeatureStrategy = "user"
	// 基于时间的策略
	StrategyTime FeatureStrategy = "time"
	// 基于环境的策略
	StrategyEnvironment FeatureStrategy = "environment"
)

// FeatureCapabilities 功能集合的能力定义
type FeatureCapabilities struct {
	Permissions []string `yaml:"permissions" json:"permissions"`
	AuditLogs   []string `yaml:"audit_logs" json:"audit_logs"`
	APIs        []string `yaml:"apis" json:"apis"`
	Routes      []string `yaml:"routes" json:"routes"`
	Configs     []string `yaml:"configs" json:"configs"`
}

// ToggleConfig 功能开关配置
type ToggleConfig struct {
	Enabled      bool                   `yaml:"enabled" json:"enabled"`
	Strategy     FeatureStrategy        `yaml:"strategy" json:"strategy"`
	Rules        map[string]interface{} `yaml:"rules" json:"rules"`
	Capabilities *FeatureCapabilities   `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Description  string                 `yaml:"description" json:"description"`
	Tags         []string               `yaml:"tags" json:"tags"`
	CreatedAt    time.Time              `yaml:"created_at" json:"created_at"`
	UpdatedAt    time.Time              `yaml:"updated_at" json:"updated_at"`
}

// EvaluationContext 评估上下文
type EvaluationContext struct {
	UserID      string            `json:"user_id"`
	UserType    string            `json:"user_type"`
	Environment string            `json:"environment"`
	Version     string            `json:"version"`
	Attributes  map[string]string `json:"attributes"`
}

// FeatureChecker 基础功能检查接口
type FeatureChecker interface {
	// IsEnabled 检查功能是否启用
	IsEnabled(ctx context.Context, flag FeatureFlag) bool
	// IsEnabledWithContext 基于上下文检查功能是否启用
	IsEnabledWithContext(ctx context.Context, flag FeatureFlag, evalCtx *EvaluationContext) bool
}

// HierarchicalChecker 分层功能检查接口
type HierarchicalChecker interface {
	// IsFeatureEnabled 检查分层功能是否启用
	IsFeatureEnabled(featurePath string) bool
	// IsDomainEnabled 检查功能域是否有任何启用的功能
	IsDomainEnabled(domain string) bool
	// GetDomainFeatures 获取域下的所有功能
	GetDomainFeatures(domain string) []string
	// GetFeatureTree 获取功能树结构
	GetFeatureTree() map[string][]string
}

// CapabilityChecker 能力检查接口
type CapabilityChecker interface {
	// HasPermission 检查功能集合中是否包含指定权限
	HasPermission(featurePath, permission string) bool
	// HasAPI 检查功能集合中是否包含指定API
	HasAPI(featurePath, api string) bool
	// HasRoute 检查功能集合中是否包含指定路由
	HasRoute(featurePath, route string) bool
	// HasConfig 检查功能集合中是否包含指定配置
	HasConfig(featurePath, config string) bool
	// HasAuditLog 检查功能集合中是否包含指定审计日志
	HasAuditLog(featurePath, logType string) bool
	// GetFeatureCapabilities 获取功能集合的能力
	GetFeatureCapabilities(featurePath string) *FeatureCapabilities
}

// BatchChecker 批量检查接口
type BatchChecker interface {
	// IsAnyFeatureEnabled 检查是否有任何功能启用
	IsAnyFeatureEnabled(featurePaths ...string) bool
	// AreAllFeaturesEnabled 检查是否所有功能都启用
	AreAllFeaturesEnabled(featurePaths ...string) bool
}

// FeatureManager 功能管理接口
type FeatureManager interface {
	// GetToggleConfig 获取功能开关配置
	GetToggleConfig(flag FeatureFlag) (*ToggleConfig, error)
	// UpdateToggle 更新功能开关配置
	UpdateToggle(flag FeatureFlag, config *ToggleConfig) error
	// ListToggles 列出所有功能开关
	ListToggles() map[FeatureFlag]*ToggleConfig
	// EnableFeature 启用功能
	EnableFeature(flag FeatureFlag) error
	// DisableFeature 禁用功能
	DisableFeature(flag FeatureFlag) error
	// DeleteToggle 删除功能开关
	DeleteToggle(flag FeatureFlag) error
}

// EventManager 事件管理接口
type EventManager interface {
	// Subscribe 订阅功能开关变更事件
	Subscribe(callback ToggleChangeCallback) error
	// Unsubscribe 取消订阅功能开关变更事件
	Unsubscribe(callback ToggleChangeCallback) error
}

// ExportManager 导出管理接口
type ExportManager interface {
	// ExportCSV 导出CSV格式的功能集合信息
	ExportCSV() ([]byte, error)
}

// HierarchicalFeatureToggle 分层功能开关管理接口（组合接口，保持向后兼容）
type HierarchicalFeatureToggle interface {
	FeatureChecker
	HierarchicalChecker
	CapabilityChecker
	BatchChecker
	FeatureManager
	EventManager
	ExportManager
}

// FeatureToggle 功能开关管理接口 (向后兼容)
type FeatureToggle interface {
	// IsEnabled 检查功能是否启用
	IsEnabled(ctx context.Context, flag FeatureFlag) bool

	// IsEnabledWithContext 基于上下文检查功能是否启用
	IsEnabledWithContext(ctx context.Context, flag FeatureFlag, evalCtx *EvaluationContext) bool

	// GetToggleConfig 获取功能开关配置
	GetToggleConfig(flag FeatureFlag) (*ToggleConfig, error)

	// UpdateToggle 更新功能开关配置
	UpdateToggle(flag FeatureFlag, config *ToggleConfig) error

	// ListToggles 列出所有功能开关
	ListToggles() map[FeatureFlag]*ToggleConfig

	// EnableFeature 启用功能
	EnableFeature(flag FeatureFlag) error

	// DisableFeature 禁用功能
	DisableFeature(flag FeatureFlag) error

	// DeleteToggle 删除功能开关
	DeleteToggle(flag FeatureFlag) error

	// Subscribe 订阅功能开关变更事件
	Subscribe(callback ToggleChangeCallback) error

	// Unsubscribe 取消订阅功能开关变更事件
	Unsubscribe(callback ToggleChangeCallback) error
}

// ToggleChangeCallback 功能开关变更回调函数
type ToggleChangeCallback func(flag FeatureFlag, oldConfig, newConfig *ToggleConfig)

// FeatureRepository 功能开关持久化接口
type FeatureRepository interface {
	// LoadToggles 加载所有功能开关配置
	LoadToggles(ctx context.Context) (map[FeatureFlag]*ToggleConfig, error)

	// SaveToggle 保存单个功能开关配置
	SaveToggle(ctx context.Context, flag FeatureFlag, config *ToggleConfig) error

	// DeleteToggle 删除功能开关配置
	DeleteToggle(ctx context.Context, flag FeatureFlag) error

	// Watch 监听配置变更
	Watch(ctx context.Context, callback func(map[FeatureFlag]*ToggleConfig)) error
}

// FeatureAware 支持功能开关的接口
type FeatureAware interface {
	// RequiredFeatures 返回需要的功能开关列表
	RequiredFeatures() []FeatureFlag

	// OnFeatureChanged 当功能开关状态变更时调用
	OnFeatureChanged(flag FeatureFlag, enabled bool)
}

// FeatureDecorator 功能装饰器接口
type FeatureDecorator interface {
	// Decorate 装饰服务方法，可以根据功能开关状态进行处理
	Decorate(ctx context.Context, flag FeatureFlag, next func(context.Context) error) error
}

// FeatureMiddleware 功能中间件接口
type FeatureMiddleware interface {
	// CheckFeature 检查功能开关并执行相应处理
	CheckFeature(flag FeatureFlag) func(next func(context.Context) error) func(context.Context) error
}

// StrategyEvaluator 策略评估器接口
type StrategyEvaluator interface {
	// Evaluate 评估策略是否满足条件
	Evaluate(ctx context.Context, config *ToggleConfig, evalCtx *EvaluationContext) bool

	// SupportsStrategy 是否支持指定策略
	SupportsStrategy(strategy FeatureStrategy) bool
}

// ParseFeaturePath 解析功能路径
func ParseFeaturePath(featurePath string) (domain, module string) {
	parts := strings.Split(featurePath, ".")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

// BuildFeaturePath 构建功能路径
func BuildFeaturePath(domain, module string) string {
	return fmt.Sprintf("%s.%s", domain, module)
}

// IsHierarchicalFeature 检查是否为分层功能
func IsHierarchicalFeature(featurePath string) bool {
	return strings.Contains(featurePath, ".")
}
