package feature

import (
	"fmt"
	"time"
)

// ConfigurationVersion 配置版本信息
type ConfigurationVersion struct {
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`
	Kind       string `yaml:"kind" json:"kind"`
}

// StandardizedMetadata 标准化元数据
type StandardizedMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	CreatedAt   time.Time         `yaml:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time         `yaml:"updatedAt" json:"updatedAt"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	Owner       string            `yaml:"owner,omitempty" json:"owner,omitempty"`
	Tags        []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// FeatureSpec 功能规范（增强版）
type FeatureSpec struct {
	Enabled      bool                   `yaml:"enabled" json:"enabled"`
	Strategy     FeatureStrategy        `yaml:"strategy" json:"strategy"`
	Rules        map[string]interface{} `yaml:"rules,omitempty" json:"rules,omitempty"`
	Capabilities *FeatureCapabilities   `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Metadata     *FeatureMetadata       `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Dependencies []string               `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Deprecation  *DeprecationInfo       `yaml:"deprecation,omitempty" json:"deprecation,omitempty"`
}

// FeatureMetadata 功能元数据
type FeatureMetadata struct {
	Description   string            `yaml:"description,omitempty" json:"description,omitempty"`
	Tags          []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Owner         string            `yaml:"owner,omitempty" json:"owner,omitempty"`
	Reviewers     []string          `yaml:"reviewers,omitempty" json:"reviewers,omitempty"`
	Documentation string            `yaml:"documentation,omitempty" json:"documentation,omitempty"`
	ContactInfo   string            `yaml:"contactInfo,omitempty" json:"contactInfo,omitempty"`
	Priority      string            `yaml:"priority,omitempty" json:"priority,omitempty"`
	Environments  []string          `yaml:"environments,omitempty" json:"environments,omitempty"`
	CreatedAt     time.Time         `yaml:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt     time.Time         `yaml:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// DeprecationInfo 废弃信息
type DeprecationInfo struct {
	Deprecated     bool       `yaml:"deprecated" json:"deprecated"`
	DeprecatedAt   *time.Time `yaml:"deprecatedAt,omitempty" json:"deprecatedAt,omitempty"`
	RemovalDate    *time.Time `yaml:"removalDate,omitempty" json:"removalDate,omitempty"`
	Reason         string     `yaml:"reason,omitempty" json:"reason,omitempty"`
	Replacement    string     `yaml:"replacement,omitempty" json:"replacement,omitempty"`
	MigrationGuide string     `yaml:"migrationGuide,omitempty" json:"migrationGuide,omitempty"`
}

// StandardizedConfiguration 标准化的功能配置
type StandardizedConfiguration struct {
	APIVersion string                           `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                           `yaml:"kind" json:"kind"`
	Metadata   StandardizedMetadata             `yaml:"metadata" json:"metadata"`
	Spec       StandardizedConfigurationSpec    `yaml:"spec" json:"spec"`
}

// StandardizedConfigurationSpec 标准化配置规范
type StandardizedConfigurationSpec struct {
	Features   map[FeatureFlag]*FeatureSpec `yaml:"features" json:"features"`
	Defaults   *DefaultFeatureSpec          `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	Validation *ValidationRules             `yaml:"validation,omitempty" json:"validation,omitempty"`
	Policies   *ConfigurationPolicies       `yaml:"policies,omitempty" json:"policies,omitempty"`
}

// DefaultFeatureSpec 默认功能规范
type DefaultFeatureSpec struct {
	Strategy    FeatureStrategy        `yaml:"strategy" json:"strategy"`
	Enabled     bool                   `yaml:"enabled" json:"enabled"`
	Rules       map[string]interface{} `yaml:"rules,omitempty" json:"rules,omitempty"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Owner       string                 `yaml:"owner,omitempty" json:"owner,omitempty"`
	Tags        []string               `yaml:"tags,omitempty" json:"tags,omitempty"`
	TTL         time.Duration          `yaml:"ttl,omitempty" json:"ttl,omitempty"`
}

// ValidationRules 验证规则
type ValidationRules struct {
	Strict                bool     `yaml:"strict" json:"strict"`
	RequiredFields        []string `yaml:"requiredFields,omitempty" json:"requiredFields,omitempty"`
	AllowedStrategies     []string `yaml:"allowedStrategies,omitempty" json:"allowedStrategies,omitempty"`
	AllowedEnvironments   []string `yaml:"allowedEnvironments,omitempty" json:"allowedEnvironments,omitempty"`
	FeatureNamePattern    string   `yaml:"featureNamePattern,omitempty" json:"featureNamePattern,omitempty"`
	MaxFeatures           int      `yaml:"maxFeatures,omitempty" json:"maxFeatures,omitempty"`
	MaxRulesPerFeature    int      `yaml:"maxRulesPerFeature,omitempty" json:"maxRulesPerFeature,omitempty"`
	MaxCapabilitiesPerType int     `yaml:"maxCapabilitiesPerType,omitempty" json:"maxCapabilitiesPerType,omitempty"`
	RequireOwnership      bool     `yaml:"requireOwnership" json:"requireOwnership"`
	RequireDocumentation  bool     `yaml:"requireDocumentation" json:"requireDocumentation"`
}

// ConfigurationPolicies 配置策略
type ConfigurationPolicies struct {
	AutoCleanup       *AutoCleanupPolicy       `yaml:"autoCleanup,omitempty" json:"autoCleanup,omitempty"`
	DeprecationPolicy *DeprecationPolicy       `yaml:"deprecationPolicy,omitempty" json:"deprecationPolicy,omitempty"`
	AccessControl     *AccessControlPolicy     `yaml:"accessControl,omitempty" json:"accessControl,omitempty"`
	ChangeManagement  *ChangeManagementPolicy  `yaml:"changeManagement,omitempty" json:"changeManagement,omitempty"`
}

// AutoCleanupPolicy 自动清理策略
type AutoCleanupPolicy struct {
	Enabled              bool          `yaml:"enabled" json:"enabled"`
	RetentionPeriod      time.Duration `yaml:"retentionPeriod" json:"retentionPeriod"`
	CleanupUnusedFeatures bool          `yaml:"cleanupUnusedFeatures" json:"cleanupUnusedFeatures"`
	CleanupDeprecated    bool          `yaml:"cleanupDeprecated" json:"cleanupDeprecated"`
}

// DeprecationPolicy 废弃策略
type DeprecationPolicy struct {
	MinDeprecationPeriod time.Duration `yaml:"minDeprecationPeriod" json:"minDeprecationPeriod"`
	RequireReason        bool          `yaml:"requireReason" json:"requireReason"`
	RequireReplacement   bool          `yaml:"requireReplacement" json:"requireReplacement"`
	NotificationPeriod   time.Duration `yaml:"notificationPeriod" json:"notificationPeriod"`
}

// AccessControlPolicy 访问控制策略
type AccessControlPolicy struct {
	RequireApproval       bool     `yaml:"requireApproval" json:"requireApproval"`
	ApprovalRequired      []string `yaml:"approvalRequired,omitempty" json:"approvalRequired,omitempty"`
	ReadOnlyUsers         []string `yaml:"readOnlyUsers,omitempty" json:"readOnlyUsers,omitempty"`
	AdminUsers            []string `yaml:"adminUsers,omitempty" json:"adminUsers,omitempty"`
	EnvironmentRestrictions map[string][]string `yaml:"environmentRestrictions,omitempty" json:"environmentRestrictions,omitempty"`
}

// ChangeManagementPolicy 变更管理策略
type ChangeManagementPolicy struct {
	RequireChangeLog     bool          `yaml:"requireChangeLog" json:"requireChangeLog"`
	RequireTestPlan      bool          `yaml:"requireTestPlan" json:"requireTestPlan"`
	RollbackTimeout      time.Duration `yaml:"rollbackTimeout" json:"rollbackTimeout"`
	GradualRollout       bool          `yaml:"gradualRollout" json:"gradualRollout"`
	CanaryPercentage     float64       `yaml:"canaryPercentage" json:"canaryPercentage"`
}

// ConfigurationParser 配置解析器接口
type ConfigurationParser interface {
	// Parse 解析配置
	Parse(data []byte) (*StandardizedConfiguration, error)
	
	// Serialize 序列化配置
	Serialize(config *StandardizedConfiguration) ([]byte, error)
	
	// Validate 验证配置格式
	Validate(data []byte) error
	
	// GetFormat 获取格式类型
	GetFormat() string
}

// ConfigurationMigrator 配置迁移器
type ConfigurationMigrator interface {
	// Migrate 迁移配置到新版本
	Migrate(oldConfig interface{}, targetVersion string) (*StandardizedConfiguration, error)
	
	// GetSupportedVersions 获取支持的版本列表
	GetSupportedVersions() []string
	
	// IsBackwardCompatible 检查向后兼容性
	IsBackwardCompatible(fromVersion, toVersion string) bool
}

// ConfigurationManager 配置管理器
type ConfigurationManager struct {
	parser     ConfigurationParser
	migrator   ConfigurationMigrator
	validator  ConfigValidator
	currentVersion string
}

// NewConfigurationManager 创建配置管理器
func NewConfigurationManager(
	parser ConfigurationParser,
	migrator ConfigurationMigrator,
	validator ConfigValidator,
) *ConfigurationManager {
	return &ConfigurationManager{
		parser:         parser,
		migrator:       migrator,
		validator:      validator,
		currentVersion: "v1",
	}
}

// LoadConfiguration 加载配置
func (cm *ConfigurationManager) LoadConfiguration(data []byte) (*StandardizedConfiguration, error) {
	// 解析配置
	config, err := cm.parser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}
	
	// 检查版本兼容性
	if config.APIVersion != cm.currentVersion {
		if cm.migrator.IsBackwardCompatible(config.APIVersion, cm.currentVersion) {
			// 自动迁移
			migratedConfig, err := cm.migrator.Migrate(config, cm.currentVersion)
			if err != nil {
				return nil, fmt.Errorf("failed to migrate configuration from %s to %s: %w", 
					config.APIVersion, cm.currentVersion, err)
			}
			config = migratedConfig
		} else {
			return nil, fmt.Errorf("unsupported configuration version: %s, current version: %s", 
				config.APIVersion, cm.currentVersion)
		}
	}
	
	// 验证配置
	validationResult := cm.validator.ValidateConfig(&FeatureConfiguration{
		APIVersion: config.APIVersion,
		Kind:       config.Kind,
		Metadata: ConfigurationMetadata{
			Name:        config.Metadata.Name,
			Version:     config.Metadata.Version,
			Description: config.Metadata.Description,
			CreatedAt:   config.Metadata.CreatedAt,
			UpdatedAt:   config.Metadata.UpdatedAt,
			Labels:      config.Metadata.Labels,
			Annotations: config.Metadata.Annotations,
		},
		Spec: ConfigurationSpec{
			Features: cm.convertToToggleConfigs(config.Spec.Features),
			Defaults: cm.convertToDefaultSettings(config.Spec.Defaults),
			Validation: cm.convertToValidationSettings(config.Spec.Validation),
		},
	})
	
	if !validationResult.Valid {
		return nil, fmt.Errorf("configuration validation failed: %d errors, %d warnings", 
			len(validationResult.Errors), len(validationResult.Warnings))
	}
	
	return config, nil
}

// SaveConfiguration 保存配置
func (cm *ConfigurationManager) SaveConfiguration(config *StandardizedConfiguration) ([]byte, error) {
	// 更新时间戳
	config.Metadata.UpdatedAt = time.Now()
	
	// 设置当前版本
	config.APIVersion = cm.currentVersion
	
	// 序列化配置
	data, err := cm.parser.Serialize(config)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize configuration: %w", err)
	}
	
	return data, nil
}

// CreateDefaultConfiguration 创建默认配置
func (cm *ConfigurationManager) CreateDefaultConfiguration(name string) *StandardizedConfiguration {
	now := time.Now()
	
	return &StandardizedConfiguration{
		APIVersion: cm.currentVersion,
		Kind:       "FeatureConfiguration",
		Metadata: StandardizedMetadata{
			Name:        name,
			Version:     "1.0.0",
			Description: "Feature configuration for " + name,
			CreatedAt:   now,
			UpdatedAt:   now,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Tags:        []string{},
		},
		Spec: StandardizedConfigurationSpec{
			Features: make(map[FeatureFlag]*FeatureSpec),
			Defaults: &DefaultFeatureSpec{
				Strategy:    StrategySimple,
				Enabled:     false,
				Rules:       make(map[string]interface{}),
				Description: "Default feature configuration",
				TTL:         24 * time.Hour,
			},
			Validation: &ValidationRules{
				Strict:                 false,
				RequiredFields:         []string{"enabled", "strategy"},
				AllowedStrategies:      []string{"simple", "percentage", "user", "time", "environment"},
				AllowedEnvironments:    []string{"development", "staging", "production"},
				FeatureNamePattern:     `^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*$`,
				MaxFeatures:            1000,
				MaxRulesPerFeature:     10,
				MaxCapabilitiesPerType: 100,
				RequireOwnership:       false,
				RequireDocumentation:   false,
			},
			Policies: &ConfigurationPolicies{
				AutoCleanup: &AutoCleanupPolicy{
					Enabled:               false,
					RetentionPeriod:       30 * 24 * time.Hour,
					CleanupUnusedFeatures: false,
					CleanupDeprecated:     false,
				},
				DeprecationPolicy: &DeprecationPolicy{
					MinDeprecationPeriod: 30 * 24 * time.Hour,
					RequireReason:        true,
					RequireReplacement:   false,
					NotificationPeriod:   7 * 24 * time.Hour,
				},
				AccessControl: &AccessControlPolicy{
					RequireApproval:         false,
					ApprovalRequired:        []string{},
					ReadOnlyUsers:           []string{},
					AdminUsers:              []string{},
					EnvironmentRestrictions: make(map[string][]string),
				},
				ChangeManagement: &ChangeManagementPolicy{
					RequireChangeLog: false,
					RequireTestPlan:  false,
					RollbackTimeout:  5 * time.Minute,
					GradualRollout:   false,
					CanaryPercentage: 10.0,
				},
			},
		},
	}
}

// 转换函数

func (cm *ConfigurationManager) convertToToggleConfigs(specs map[FeatureFlag]*FeatureSpec) map[FeatureFlag]*ToggleConfig {
	configs := make(map[FeatureFlag]*ToggleConfig)
	
	for feature, spec := range specs {
		config := &ToggleConfig{
			Enabled:      spec.Enabled,
			Strategy:     spec.Strategy,
			Rules:        spec.Rules,
			Capabilities: spec.Capabilities,
			Description:  "",
			Tags:         []string{},
		}
		
		if spec.Metadata != nil {
			config.Description = spec.Metadata.Description
			config.Tags = spec.Metadata.Tags
			config.CreatedAt = spec.Metadata.CreatedAt
			config.UpdatedAt = spec.Metadata.UpdatedAt
		}
		
		configs[feature] = config
	}
	
	return configs
}

func (cm *ConfigurationManager) convertToDefaultSettings(spec *DefaultFeatureSpec) *DefaultSettings {
	if spec == nil {
		return nil
	}
	
	return &DefaultSettings{
		Strategy:    spec.Strategy,
		Enabled:     spec.Enabled,
		Rules:       spec.Rules,
		Description: spec.Description,
	}
}

func (cm *ConfigurationManager) convertToValidationSettings(rules *ValidationRules) *ValidationSettings {
	if rules == nil {
		return nil
	}
	
	return &ValidationSettings{
		Strict:                 rules.Strict,
		RequiredFields:         rules.RequiredFields,
		AllowedStrategies:      rules.AllowedStrategies,
		AllowedEnvironments:    rules.AllowedEnvironments,
		FeatureNamePattern:     rules.FeatureNamePattern,
		MaxFeatures:            rules.MaxFeatures,
		MaxRulesPerFeature:     rules.MaxRulesPerFeature,
		MaxCapabilitiesPerType: rules.MaxCapabilitiesPerType,
	}
}