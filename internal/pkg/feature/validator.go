package feature

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ValidationLevel 验证级别
type ValidationLevel string

const (
	ValidationLevelError   ValidationLevel = "error"   // 错误级别（必须修复）
	ValidationLevelWarning ValidationLevel = "warning" // 警告级别（建议修复）
	ValidationLevelInfo    ValidationLevel = "info"    // 信息级别（提示）
)

// ValidationResult 验证结果
type ValidationResult struct {
	Valid       bool              `json:"valid"`
	Errors      []ValidationIssue `json:"errors"`
	Warnings    []ValidationIssue `json:"warnings"`
	Infos       []ValidationIssue `json:"infos"`
	Summary     ValidationSummary `json:"summary"`
	ValidatedAt time.Time         `json:"validated_at"`
}

// ValidationIssue 验证问题
type ValidationIssue struct {
	Level      ValidationLevel `json:"level"`
	Code       string          `json:"code"`
	Message    string          `json:"message"`
	Feature    FeatureFlag     `json:"feature,omitempty"`
	Field      string          `json:"field,omitempty"`
	Value      interface{}     `json:"value,omitempty"`
	Suggestion string          `json:"suggestion,omitempty"`
	Reference  string          `json:"reference,omitempty"`
}

// ValidationSummary 验证摘要
type ValidationSummary struct {
	TotalFeatures   int `json:"total_features"`
	ValidFeatures   int `json:"valid_features"`
	InvalidFeatures int `json:"invalid_features"`
	ErrorCount      int `json:"error_count"`
	WarningCount    int `json:"warning_count"`
	InfoCount       int `json:"info_count"`
}

// FeatureConfiguration 功能配置（用于验证的完整配置结构）
type FeatureConfiguration struct {
	APIVersion string                `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                `yaml:"kind" json:"kind"`
	Metadata   ConfigurationMetadata `yaml:"metadata" json:"metadata"`
	Spec       ConfigurationSpec     `yaml:"spec" json:"spec"`
}

// ConfigurationMetadata 配置元数据
type ConfigurationMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Description string            `yaml:"description" json:"description"`
	CreatedAt   time.Time         `yaml:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time         `yaml:"updatedAt" json:"updatedAt"`
	Labels      map[string]string `yaml:"labels" json:"labels"`
	Annotations map[string]string `yaml:"annotations" json:"annotations"`
}

// ConfigurationSpec 配置规范
type ConfigurationSpec struct {
	Features   map[FeatureFlag]*ToggleConfig `yaml:"features" json:"features"`
	Defaults   *DefaultSettings              `yaml:"defaults" json:"defaults"`
	Validation *ValidationSettings           `yaml:"validation" json:"validation"`
}

// DefaultSettings 默认设置
type DefaultSettings struct {
	Strategy    FeatureStrategy        `yaml:"strategy" json:"strategy"`
	Enabled     bool                   `yaml:"enabled" json:"enabled"`
	Rules       map[string]interface{} `yaml:"rules" json:"rules"`
	Description string                 `yaml:"description" json:"description"`
}

// ValidationSettings 验证设置
type ValidationSettings struct {
	Strict                 bool     `yaml:"strict" json:"strict"`
	RequiredFields         []string `yaml:"requiredFields" json:"requiredFields"`
	AllowedStrategies      []string `yaml:"allowedStrategies" json:"allowedStrategies"`
	AllowedEnvironments    []string `yaml:"allowedEnvironments" json:"allowedEnvironments"`
	FeatureNamePattern     string   `yaml:"featureNamePattern" json:"featureNamePattern"`
	MaxFeatures            int      `yaml:"maxFeatures" json:"maxFeatures"`
	MaxRulesPerFeature     int      `yaml:"maxRulesPerFeature" json:"maxRulesPerFeature"`
	MaxCapabilitiesPerType int      `yaml:"maxCapabilitiesPerType" json:"maxCapabilitiesPerType"`
}

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	// ValidateConfig 验证配置
	ValidateConfig(config *FeatureConfiguration) *ValidationResult

	// ValidateToggle 验证单个功能开关
	ValidateToggle(feature FeatureFlag, config *ToggleConfig) *ValidationResult

	// ValidateConsistency 验证配置一致性
	ValidateConsistency(config *FeatureConfiguration) *ValidationResult

	// ValidateDependencies 验证依赖关系
	ValidateDependencies(config *FeatureConfiguration) *ValidationResult

	// ValidateSchema 验证模式
	ValidateSchema(config *FeatureConfiguration) *ValidationResult

	// ValidateBusinessRules 验证业务规则
	ValidateBusinessRules(config *FeatureConfiguration) *ValidationResult
}

// DefaultConfigValidator 默认配置验证器
type DefaultConfigValidator struct {
	settings         *ValidationSettings
	strategyRegistry map[FeatureStrategy]bool
	customValidators []CustomValidator
}

// CustomValidator 自定义验证器接口
type CustomValidator interface {
	// Validate 执行自定义验证
	Validate(config *FeatureConfiguration) []ValidationIssue

	// GetName 获取验证器名称
	GetName() string
}

// NewDefaultConfigValidator 创建默认配置验证器
func NewDefaultConfigValidator(settings *ValidationSettings) *DefaultConfigValidator {
	validator := &DefaultConfigValidator{
		settings: settings,
		strategyRegistry: map[FeatureStrategy]bool{
			StrategySimple:      true,
			StrategyPercentage:  true,
			StrategyUser:        true,
			StrategyTime:        true,
			StrategyEnvironment: true,
		},
		customValidators: make([]CustomValidator, 0),
	}

	if validator.settings == nil {
		validator.settings = &ValidationSettings{
			Strict:                 false,
			RequiredFields:         []string{"enabled", "strategy"},
			AllowedStrategies:      []string{"simple", "percentage", "user", "time", "environment"},
			AllowedEnvironments:    []string{"development", "staging", "production"},
			FeatureNamePattern:     `^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*$`,
			MaxFeatures:            1000,
			MaxRulesPerFeature:     10,
			MaxCapabilitiesPerType: 100,
		}
	}

	return validator
}

// ValidateConfig 验证配置
func (dcv *DefaultConfigValidator) ValidateConfig(config *FeatureConfiguration) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationIssue, 0),
		Warnings:    make([]ValidationIssue, 0),
		Infos:       make([]ValidationIssue, 0),
		ValidatedAt: time.Now(),
	}

	// 验证模式
	schemaResult := dcv.ValidateSchema(config)
	result.mergeResult(schemaResult)

	// 验证业务规则
	businessResult := dcv.ValidateBusinessRules(config)
	result.mergeResult(businessResult)

	// 验证一致性
	consistencyResult := dcv.ValidateConsistency(config)
	result.mergeResult(consistencyResult)

	// 验证依赖关系
	dependencyResult := dcv.ValidateDependencies(config)
	result.mergeResult(dependencyResult)

	// 执行自定义验证
	for _, customValidator := range dcv.customValidators {
		issues := customValidator.Validate(config)
		for _, issue := range issues {
			result.addIssue(issue)
		}
	}

	// 生成摘要
	result.generateSummary(config)

	return result
}

// ValidateToggle 验证单个功能开关
func (dcv *DefaultConfigValidator) ValidateToggle(feature FeatureFlag, config *ToggleConfig) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationIssue, 0),
		Warnings:    make([]ValidationIssue, 0),
		Infos:       make([]ValidationIssue, 0),
		ValidatedAt: time.Now(),
	}

	// 验证功能名称
	if err := dcv.validateFeatureName(feature); err != nil {
		result.addIssue(ValidationIssue{
			Level:   ValidationLevelError,
			Code:    "INVALID_FEATURE_NAME",
			Message: err.Error(),
			Feature: feature,
			Field:   "name",
			Value:   feature,
		})
	}

	// 验证必需字段
	dcv.validateRequiredFields(feature, config, result)

	// 验证策略
	dcv.validateStrategy(feature, config, result)

	// 验证规则
	dcv.validateRules(feature, config, result)

	// 验证能力
	dcv.validateCapabilities(feature, config, result)

	// 验证时间戳
	dcv.validateTimestamps(feature, config, result)

	return result
}

// ValidateSchema 验证模式
func (dcv *DefaultConfigValidator) ValidateSchema(config *FeatureConfiguration) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationIssue, 0),
		Warnings:    make([]ValidationIssue, 0),
		Infos:       make([]ValidationIssue, 0),
		ValidatedAt: time.Now(),
	}

	// 验证API版本
	if config.APIVersion == "" {
		result.addIssue(ValidationIssue{
			Level:   ValidationLevelError,
			Code:    "MISSING_API_VERSION",
			Message: "API version is required",
			Field:   "apiVersion",
		})
	} else if config.APIVersion != "v1" {
		result.addIssue(ValidationIssue{
			Level:   ValidationLevelWarning,
			Code:    "UNSUPPORTED_API_VERSION",
			Message: fmt.Sprintf("API version %s may not be supported", config.APIVersion),
			Field:   "apiVersion",
			Value:   config.APIVersion,
		})
	}

	// 验证Kind
	if config.Kind == "" {
		result.addIssue(ValidationIssue{
			Level:   ValidationLevelError,
			Code:    "MISSING_KIND",
			Message: "Kind is required",
			Field:   "kind",
		})
	} else if config.Kind != "FeatureConfiguration" {
		result.addIssue(ValidationIssue{
			Level:   ValidationLevelError,
			Code:    "INVALID_KIND",
			Message: fmt.Sprintf("Invalid kind: %s, expected: FeatureConfiguration", config.Kind),
			Field:   "kind",
			Value:   config.Kind,
		})
	}

	// 验证元数据
	dcv.validateMetadata(&config.Metadata, result)

	return result
}

// ValidateBusinessRules 验证业务规则
func (dcv *DefaultConfigValidator) ValidateBusinessRules(config *FeatureConfiguration) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationIssue, 0),
		Warnings:    make([]ValidationIssue, 0),
		Infos:       make([]ValidationIssue, 0),
		ValidatedAt: time.Now(),
	}

	if config.Spec.Features == nil {
		result.addIssue(ValidationIssue{
			Level:   ValidationLevelError,
			Code:    "NO_FEATURES_DEFINED",
			Message: "No features defined in configuration",
			Field:   "spec.features",
		})
		return result
	}

	// 验证功能数量限制
	if dcv.settings.MaxFeatures > 0 && len(config.Spec.Features) > dcv.settings.MaxFeatures {
		result.addIssue(ValidationIssue{
			Level: ValidationLevelError,
			Code:  "TOO_MANY_FEATURES",
			Message: fmt.Sprintf("Too many features defined: %d, maximum allowed: %d",
				len(config.Spec.Features), dcv.settings.MaxFeatures),
			Field: "spec.features",
			Value: len(config.Spec.Features),
		})
	}

	// 验证每个功能
	for feature, toggleConfig := range config.Spec.Features {
		toggleResult := dcv.ValidateToggle(feature, toggleConfig)
		result.mergeResult(toggleResult)
	}

	return result
}

// ValidateConsistency 验证配置一致性
func (dcv *DefaultConfigValidator) ValidateConsistency(config *FeatureConfiguration) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationIssue, 0),
		Warnings:    make([]ValidationIssue, 0),
		Infos:       make([]ValidationIssue, 0),
		ValidatedAt: time.Now(),
	}

	// 验证功能名称唯一性
	dcv.validateFeatureNameUniqueness(config, result)

	// 验证层次结构一致性
	dcv.validateHierarchyConsistency(config, result)

	// 验证标签一致性
	dcv.validateTagConsistency(config, result)

	return result
}

// ValidateDependencies 验证依赖关系
func (dcv *DefaultConfigValidator) ValidateDependencies(config *FeatureConfiguration) *ValidationResult {
	result := &ValidationResult{
		Valid:       true,
		Errors:      make([]ValidationIssue, 0),
		Warnings:    make([]ValidationIssue, 0),
		Infos:       make([]ValidationIssue, 0),
		ValidatedAt: time.Now(),
	}

	// 验证能力依赖
	dcv.validateCapabilityDependencies(config, result)

	// 验证规则依赖
	dcv.validateRuleDependencies(config, result)

	return result
}

// validateFeatureName 验证功能名称
func (dcv *DefaultConfigValidator) validateFeatureName(feature FeatureFlag) error {
	featureName := string(feature)

	// 检查是否为空
	if featureName == "" {
		return fmt.Errorf("feature name cannot be empty")
	}

	// 检查长度
	if len(featureName) > 100 {
		return fmt.Errorf("feature name too long: %d characters, maximum allowed: 100", len(featureName))
	}

	// 检查模式
	if dcv.settings.FeatureNamePattern != "" {
		matched, err := regexp.MatchString(dcv.settings.FeatureNamePattern, featureName)
		if err != nil {
			return fmt.Errorf("invalid feature name pattern: %v", err)
		}
		if !matched {
			return fmt.Errorf("feature name %s does not match required pattern: %s",
				featureName, dcv.settings.FeatureNamePattern)
		}
	}

	return nil
}

// validateRequiredFields 验证必需字段
func (dcv *DefaultConfigValidator) validateRequiredFields(feature FeatureFlag, config *ToggleConfig, result *ValidationResult) {
	for _, field := range dcv.settings.RequiredFields {
		switch field {
		case "enabled":
			// enabled 字段是布尔值，总是有值
		case "strategy":
			if config.Strategy == "" {
				result.addIssue(ValidationIssue{
					Level:   ValidationLevelError,
					Code:    "MISSING_REQUIRED_FIELD",
					Message: fmt.Sprintf("Required field %s is missing", field),
					Feature: feature,
					Field:   field,
				})
			}
		case "description":
			if config.Description == "" {
				result.addIssue(ValidationIssue{
					Level:   ValidationLevelWarning,
					Code:    "MISSING_DESCRIPTION",
					Message: "Feature description is recommended",
					Feature: feature,
					Field:   field,
				})
			}
		}
	}
}

// validateStrategy 验证策略
func (dcv *DefaultConfigValidator) validateStrategy(feature FeatureFlag, config *ToggleConfig, result *ValidationResult) {
	// 检查策略是否支持
	if !dcv.strategyRegistry[config.Strategy] {
		result.addIssue(ValidationIssue{
			Level:   ValidationLevelError,
			Code:    "UNSUPPORTED_STRATEGY",
			Message: fmt.Sprintf("Strategy %s is not supported", config.Strategy),
			Feature: feature,
			Field:   "strategy",
			Value:   config.Strategy,
		})
		return
	}

	// 检查策略是否在允许列表中
	if len(dcv.settings.AllowedStrategies) > 0 {
		allowed := false
		for _, allowedStrategy := range dcv.settings.AllowedStrategies {
			if string(config.Strategy) == allowedStrategy {
				allowed = true
				break
			}
		}
		if !allowed {
			result.addIssue(ValidationIssue{
				Level:   ValidationLevelError,
				Code:    "STRATEGY_NOT_ALLOWED",
				Message: fmt.Sprintf("Strategy %s is not allowed in this environment", config.Strategy),
				Feature: feature,
				Field:   "strategy",
				Value:   config.Strategy,
			})
		}
	}
}

// validateRules 验证规则
func (dcv *DefaultConfigValidator) validateRules(feature FeatureFlag, config *ToggleConfig, result *ValidationResult) {
	if config.Rules == nil {
		return
	}

	// 检查规则数量
	if dcv.settings.MaxRulesPerFeature > 0 && len(config.Rules) > dcv.settings.MaxRulesPerFeature {
		result.addIssue(ValidationIssue{
			Level: ValidationLevelError,
			Code:  "TOO_MANY_RULES",
			Message: fmt.Sprintf("Too many rules for feature %s: %d, maximum allowed: %d",
				feature, len(config.Rules), dcv.settings.MaxRulesPerFeature),
			Feature: feature,
			Field:   "rules",
			Value:   len(config.Rules),
		})
	}

	// 验证环境规则
	if environments, ok := config.Rules["environments"].([]interface{}); ok {
		dcv.validateEnvironmentRules(feature, environments, result)
	}

	// 验证百分比规则
	if percentage, ok := config.Rules["percentage"].(float64); ok {
		if percentage < 0 || percentage > 100 {
			result.addIssue(ValidationIssue{
				Level:   ValidationLevelError,
				Code:    "INVALID_PERCENTAGE",
				Message: fmt.Sprintf("Invalid percentage value: %f, must be between 0 and 100", percentage),
				Feature: feature,
				Field:   "rules.percentage",
				Value:   percentage,
			})
		}
	}
}

// validateCapabilities 验证能力
func (dcv *DefaultConfigValidator) validateCapabilities(feature FeatureFlag, config *ToggleConfig, result *ValidationResult) {
	if config.Capabilities == nil {
		return
	}

	// 验证权限数量
	if dcv.settings.MaxCapabilitiesPerType > 0 {
		if len(config.Capabilities.Permissions) > dcv.settings.MaxCapabilitiesPerType {
			result.addIssue(ValidationIssue{
				Level: ValidationLevelWarning,
				Code:  "TOO_MANY_PERMISSIONS",
				Message: fmt.Sprintf("Too many permissions for feature %s: %d, recommended maximum: %d",
					feature, len(config.Capabilities.Permissions), dcv.settings.MaxCapabilitiesPerType),
				Feature: feature,
				Field:   "capabilities.permissions",
				Value:   len(config.Capabilities.Permissions),
			})
		}
	}

	// 验证API格式
	for _, api := range config.Capabilities.APIs {
		if !dcv.isValidAPIFormat(api) {
			result.addIssue(ValidationIssue{
				Level:   ValidationLevelWarning,
				Code:    "INVALID_API_FORMAT",
				Message: fmt.Sprintf("API %s does not follow recommended format (e.g., 'GET /api/v1/resource')", api),
				Feature: feature,
				Field:   "capabilities.apis",
				Value:   api,
			})
		}
	}
}

// validateTimestamps 验证时间戳
func (dcv *DefaultConfigValidator) validateTimestamps(feature FeatureFlag, config *ToggleConfig, result *ValidationResult) {
	if !config.CreatedAt.IsZero() && !config.UpdatedAt.IsZero() {
		if config.UpdatedAt.Before(config.CreatedAt) {
			result.addIssue(ValidationIssue{
				Level:   ValidationLevelError,
				Code:    "INVALID_TIMESTAMPS",
				Message: "Updated time cannot be before created time",
				Feature: feature,
				Field:   "timestamps",
			})
		}
	}
}

// 其他验证方法的实现...

// mergeResult 合并验证结果
func (vr *ValidationResult) mergeResult(other *ValidationResult) {
	vr.Errors = append(vr.Errors, other.Errors...)
	vr.Warnings = append(vr.Warnings, other.Warnings...)
	vr.Infos = append(vr.Infos, other.Infos...)

	if len(vr.Errors) > 0 {
		vr.Valid = false
	}
}

// addIssue 添加验证问题
func (vr *ValidationResult) addIssue(issue ValidationIssue) {
	switch issue.Level {
	case ValidationLevelError:
		vr.Errors = append(vr.Errors, issue)
		vr.Valid = false
	case ValidationLevelWarning:
		vr.Warnings = append(vr.Warnings, issue)
	case ValidationLevelInfo:
		vr.Infos = append(vr.Infos, issue)
	}
}

// generateSummary 生成摘要
func (vr *ValidationResult) generateSummary(config *FeatureConfiguration) {
	vr.Summary = ValidationSummary{
		TotalFeatures: len(config.Spec.Features),
		ErrorCount:    len(vr.Errors),
		WarningCount:  len(vr.Warnings),
		InfoCount:     len(vr.Infos),
	}

	if vr.Valid {
		vr.Summary.ValidFeatures = vr.Summary.TotalFeatures
		vr.Summary.InvalidFeatures = 0
	} else {
		// 计算无效功能数量（简化计算）
		invalidFeatures := make(map[FeatureFlag]bool)
		for _, err := range vr.Errors {
			if err.Feature != "" {
				invalidFeatures[err.Feature] = true
			}
		}
		vr.Summary.InvalidFeatures = len(invalidFeatures)
		vr.Summary.ValidFeatures = vr.Summary.TotalFeatures - vr.Summary.InvalidFeatures
	}
}

// 其他辅助方法的实现...

func (dcv *DefaultConfigValidator) validateMetadata(metadata *ConfigurationMetadata, result *ValidationResult) {
	if metadata.Name == "" {
		result.addIssue(ValidationIssue{
			Level:   ValidationLevelWarning,
			Code:    "MISSING_METADATA_NAME",
			Message: "Configuration name is recommended",
			Field:   "metadata.name",
		})
	}
}

func (dcv *DefaultConfigValidator) validateFeatureNameUniqueness(config *FeatureConfiguration, result *ValidationResult) {
	// 功能名称在map中已经保证唯一性，这里可以添加其他唯一性检查
}

func (dcv *DefaultConfigValidator) validateHierarchyConsistency(config *FeatureConfiguration, result *ValidationResult) {
	// 验证分层功能的一致性
	domains := make(map[string][]FeatureFlag)
	for feature := range config.Spec.Features {
		if IsHierarchicalFeature(string(feature)) {
			domain, _ := ParseFeaturePath(string(feature))
			domains[domain] = append(domains[domain], feature)
		}
	}
}

func (dcv *DefaultConfigValidator) validateTagConsistency(config *FeatureConfiguration, result *ValidationResult) {
	// 验证标签的一致性
}

func (dcv *DefaultConfigValidator) validateCapabilityDependencies(config *FeatureConfiguration, result *ValidationResult) {
	// 验证能力依赖关系
}

func (dcv *DefaultConfigValidator) validateRuleDependencies(config *FeatureConfiguration, result *ValidationResult) {
	// 验证规则依赖关系
}

func (dcv *DefaultConfigValidator) validateEnvironmentRules(feature FeatureFlag, environments []interface{}, result *ValidationResult) {
	for _, env := range environments {
		if envStr, ok := env.(string); ok {
			if len(dcv.settings.AllowedEnvironments) > 0 {
				allowed := false
				for _, allowedEnv := range dcv.settings.AllowedEnvironments {
					if envStr == allowedEnv {
						allowed = true
						break
					}
				}
				if !allowed {
					result.addIssue(ValidationIssue{
						Level:   ValidationLevelError,
						Code:    "INVALID_ENVIRONMENT",
						Message: fmt.Sprintf("Environment %s is not allowed", envStr),
						Feature: feature,
						Field:   "rules.environments",
						Value:   envStr,
					})
				}
			}
		}
	}
}

func (dcv *DefaultConfigValidator) isValidAPIFormat(api string) bool {
	// 简单的API格式验证：METHOD /path
	parts := strings.SplitN(api, " ", 2)
	if len(parts) != 2 {
		return false
	}

	method := parts[0]
	path := parts[1]

	// 验证HTTP方法
	validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	methodValid := false
	for _, validMethod := range validMethods {
		if method == validMethod {
			methodValid = true
			break
		}
	}

	// 验证路径格式
	pathValid := strings.HasPrefix(path, "/")

	return methodValid && pathValid
}

// AddCustomValidator 添加自定义验证器
func (dcv *DefaultConfigValidator) AddCustomValidator(validator CustomValidator) {
	dcv.customValidators = append(dcv.customValidators, validator)
}

// RemoveCustomValidator 移除自定义验证器
func (dcv *DefaultConfigValidator) RemoveCustomValidator(name string) {
	for i, validator := range dcv.customValidators {
		if validator.GetName() == name {
			dcv.customValidators = append(dcv.customValidators[:i], dcv.customValidators[i+1:]...)
			break
		}
	}
}
