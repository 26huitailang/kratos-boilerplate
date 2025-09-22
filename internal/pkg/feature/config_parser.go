package feature

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// YAMLConfigurationParser YAML配置解析器
type YAMLConfigurationParser struct{}

// NewYAMLConfigurationParser 创建YAML配置解析器
func NewYAMLConfigurationParser() *YAMLConfigurationParser {
	return &YAMLConfigurationParser{}
}

// Parse 解析YAML配置
func (ycp *YAMLConfigurationParser) Parse(data []byte) (*StandardizedConfiguration, error) {
	var config StandardizedConfiguration
	
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	
	// 设置默认值
	if config.APIVersion == "" {
		config.APIVersion = "v1"
	}
	
	if config.Kind == "" {
		config.Kind = "FeatureConfiguration"
	}
	
	return &config, nil
}

// Serialize 序列化配置为YAML
func (ycp *YAMLConfigurationParser) Serialize(config *StandardizedConfiguration) ([]byte, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}
	
	return data, nil
}

// Validate 验证YAML格式
func (ycp *YAMLConfigurationParser) Validate(data []byte) error {
	var temp interface{}
	if err := yaml.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("invalid YAML format: %w", err)
	}
	
	return nil
}

// GetFormat 获取格式类型
func (ycp *YAMLConfigurationParser) GetFormat() string {
	return "yaml"
}

// DefaultConfigurationMigrator 默认配置迁移器
type DefaultConfigurationMigrator struct {
	supportedVersions []string
}

// NewDefaultConfigurationMigrator 创建默认配置迁移器
func NewDefaultConfigurationMigrator() *DefaultConfigurationMigrator {
	return &DefaultConfigurationMigrator{
		supportedVersions: []string{"v1"},
	}
}

// Migrate 迁移配置到新版本
func (dcm *DefaultConfigurationMigrator) Migrate(oldConfig interface{}, targetVersion string) (*StandardizedConfiguration, error) {
	switch targetVersion {
	case "v1":
		return dcm.migrateToV1(oldConfig)
	default:
		return nil, fmt.Errorf("unsupported target version: %s", targetVersion)
	}
}

// GetSupportedVersions 获取支持的版本列表
func (dcm *DefaultConfigurationMigrator) GetSupportedVersions() []string {
	return dcm.supportedVersions
}

// IsBackwardCompatible 检查向后兼容性
func (dcm *DefaultConfigurationMigrator) IsBackwardCompatible(fromVersion, toVersion string) bool {
	// 简单实现：只支持v1
	return fromVersion == "v1" && toVersion == "v1"
}

// migrateToV1 迁移到v1版本
func (dcm *DefaultConfigurationMigrator) migrateToV1(oldConfig interface{}) (*StandardizedConfiguration, error) {
	// 如果已经是StandardizedConfiguration类型，直接返回
	if config, ok := oldConfig.(*StandardizedConfiguration); ok {
		config.APIVersion = "v1"
		return config, nil
	}
	
	// 如果是旧的FeatureConfiguration类型，进行转换
	if oldFeatureConfig, ok := oldConfig.(*FeatureConfiguration); ok {
		return dcm.convertFromOldFormat(oldFeatureConfig)
	}
	
	return nil, fmt.Errorf("unsupported configuration type for migration")
}

// convertFromOldFormat 从旧格式转换
func (dcm *DefaultConfigurationMigrator) convertFromOldFormat(oldConfig *FeatureConfiguration) (*StandardizedConfiguration, error) {
	newConfig := &StandardizedConfiguration{
		APIVersion: "v1",
		Kind:       "FeatureConfiguration",
		Metadata: StandardizedMetadata{
			Name:        oldConfig.Metadata.Name,
			Version:     oldConfig.Metadata.Version,
			Description: oldConfig.Metadata.Description,
			CreatedAt:   oldConfig.Metadata.CreatedAt,
			UpdatedAt:   oldConfig.Metadata.UpdatedAt,
			Labels:      oldConfig.Metadata.Labels,
			Annotations: oldConfig.Metadata.Annotations,
			Tags:        []string{},
		},
		Spec: StandardizedConfigurationSpec{
			Features: make(map[FeatureFlag]*FeatureSpec),
		},
	}
	
	// 转换功能配置
	for feature, toggleConfig := range oldConfig.Spec.Features {
		featureSpec := &FeatureSpec{
			Enabled:      toggleConfig.Enabled,
			Strategy:     toggleConfig.Strategy,
			Rules:        toggleConfig.Rules,
			Capabilities: toggleConfig.Capabilities,
			Dependencies: []string{},
			Metadata: &FeatureMetadata{
				Description: toggleConfig.Description,
				Tags:        toggleConfig.Tags,
				CreatedAt:   toggleConfig.CreatedAt,
				UpdatedAt:   toggleConfig.UpdatedAt,
				Labels:      make(map[string]string),
			},
		}
		
		newConfig.Spec.Features[feature] = featureSpec
	}
	
	// 转换默认设置
	if oldConfig.Spec.Defaults != nil {
		newConfig.Spec.Defaults = &DefaultFeatureSpec{
			Strategy:    oldConfig.Spec.Defaults.Strategy,
			Enabled:     oldConfig.Spec.Defaults.Enabled,
			Rules:       oldConfig.Spec.Defaults.Rules,
			Description: oldConfig.Spec.Defaults.Description,
			Tags:        []string{},
		}
	}
	
	// 转换验证设置
	if oldConfig.Spec.Validation != nil {
		newConfig.Spec.Validation = &ValidationRules{
			Strict:                 oldConfig.Spec.Validation.Strict,
			RequiredFields:         oldConfig.Spec.Validation.RequiredFields,
			AllowedStrategies:      oldConfig.Spec.Validation.AllowedStrategies,
			AllowedEnvironments:    oldConfig.Spec.Validation.AllowedEnvironments,
			FeatureNamePattern:     oldConfig.Spec.Validation.FeatureNamePattern,
			MaxFeatures:            oldConfig.Spec.Validation.MaxFeatures,
			MaxRulesPerFeature:     oldConfig.Spec.Validation.MaxRulesPerFeature,
			MaxCapabilitiesPerType: oldConfig.Spec.Validation.MaxCapabilitiesPerType,
			RequireOwnership:       false,
			RequireDocumentation:   false,
		}
	}
	
	return newConfig, nil
}