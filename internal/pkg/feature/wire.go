package feature

import (
	"context"
	"kratos-boilerplate/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet 功能开关系统依赖注入配置
var ProviderSet = wire.NewSet(
	NewFeatureConfig,
	NewFeatureRepository,
	NewCompositeStrategy,
	NewToggleManager,
	NewServiceDecorator,
	NewConditionalDecorator,
	NewGradualRolloutDecorator,
	NewCircuitBreakerDecorator,
	wire.Bind(new(FeatureToggle), new(*ToggleManager)),
	wire.Bind(new(HierarchicalFeatureToggle), new(*ToggleManager)),
	wire.Bind(new(StrategyEvaluator), new(*CompositeStrategy)),
)

// FeatureConfig 功能开关配置
type FeatureConfig struct {
	Enabled            bool             `json:"enabled"`
	ConfigFile         string           `json:"config_file"`
	ConfigFormat       string           `json:"config_format"`
	WatchConfig        bool             `json:"watch_config"`
	DefaultEnvironment string           `json:"default_environment"`
	Repository         RepositoryConfig `json:"repository"`
}

// RepositoryConfig 存储库配置
type RepositoryConfig struct {
	Type       string `json:"type"`
	ConfigPath string `json:"config_path"`
	Format     string `json:"format"`
}

// NewFeatureConfig 创建功能开关配置
func NewFeatureConfig(c *conf.Bootstrap) *FeatureConfig {
	if c.Features == nil {
		// 返回默认配置
		return &FeatureConfig{
			Enabled:            true,
			ConfigFile:         "./configs/features.yaml",
			ConfigFormat:       "yaml",
			WatchConfig:        true,
			DefaultEnvironment: "production",
			Repository: RepositoryConfig{
				Type:       "file",
				ConfigPath: "./configs/features.yaml",
				Format:     "yaml",
			},
		}
	}

	config := &FeatureConfig{
		Enabled:            c.Features.Enabled,
		ConfigFile:         c.Features.ConfigFile,
		ConfigFormat:       c.Features.ConfigFormat,
		WatchConfig:        c.Features.WatchConfig,
		DefaultEnvironment: c.Features.DefaultEnvironment,
	}

	if c.Features.Repository != nil {
		config.Repository = RepositoryConfig{
			Type:       c.Features.Repository.Type,
			ConfigPath: c.Features.Repository.ConfigPath,
			Format:     c.Features.Repository.Format,
		}
	} else {
		config.Repository = RepositoryConfig{
			Type:       "file",
			ConfigPath: "./configs/features.yaml",
			Format:     "yaml",
		}
	}

	// 填充默认值
	if config.ConfigFile == "" {
		config.ConfigFile = "./configs/features.yaml"
	}
	if config.ConfigFormat == "" {
		config.ConfigFormat = "yaml"
	}
	if config.DefaultEnvironment == "" {
		config.DefaultEnvironment = "production"
	}
	if config.Repository.ConfigPath == "" {
		config.Repository.ConfigPath = config.ConfigFile
	}
	if config.Repository.Format == "" {
		config.Repository.Format = config.ConfigFormat
	}
	if config.Repository.Type == "" {
		config.Repository.Type = "file"
	}

	return config
}

// NewFeatureRepository 根据配置创建功能开关存储库
func NewFeatureRepository(config *FeatureConfig, logger log.Logger) FeatureRepository {
	switch config.Repository.Type {
	case "file":
		return NewFileRepository(config.Repository.ConfigPath, config.Repository.Format, logger)
	default:
		// 默认使用文件存储
		return NewFileRepository(config.Repository.ConfigPath, config.Repository.Format, logger)
	}
}

// InitializeFeatureSystem 初始化功能开关系统
func InitializeFeatureSystem(toggleManager *ToggleManager) error {
	return toggleManager.Initialize(context.Background())
}
