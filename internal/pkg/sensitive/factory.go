package sensitive

import (
	"github.com/go-kratos/kratos/v2/log"
)

// LoggerFactory 日志器工厂
type LoggerFactory struct {
	defaultConfig *StructuredLogConfig
}

// NewLoggerFactory 创建日志器工厂
func NewLoggerFactory(config *StructuredLogConfig) *LoggerFactory {
	if config == nil {
		config = DefaultStructuredLogConfig()
	}
	
	return &LoggerFactory{
		defaultConfig: config,
	}
}

// CreateStructuredLogger 创建结构化日志器
func (f *LoggerFactory) CreateStructuredLogger(logger log.Logger) StructuredLoggerInterface {
	return NewStructuredLoggerFromLogger(logger, f.defaultConfig)
}

// CreateStructuredLoggerWithConfig 使用自定义配置创建结构化日志器
func (f *LoggerFactory) CreateStructuredLoggerWithConfig(logger log.Logger, config *StructuredLogConfig) StructuredLoggerInterface {
	return NewStructuredLoggerFromLogger(logger, config)
}

// CreateStructuredLoggerFromHelper 从Helper创建结构化日志器
func (f *LoggerFactory) CreateStructuredLoggerFromHelper(helper *log.Helper) StructuredLoggerInterface {
	return NewStructuredLogger(helper, f.defaultConfig)
}

// 全局工厂实例
var defaultFactory *LoggerFactory

// InitDefaultFactory 初始化默认工厂
func InitDefaultFactory(config *StructuredLogConfig) {
	defaultFactory = NewLoggerFactory(config)
}

// GetDefaultFactory 获取默认工厂
func GetDefaultFactory() *LoggerFactory {
	if defaultFactory == nil {
		defaultFactory = NewLoggerFactory(nil)
	}
	return defaultFactory
}

// 便捷函数

// NewStructuredLoggerQuick 快速创建结构化日志器
func NewStructuredLoggerQuick(logger log.Logger) StructuredLoggerInterface {
	return GetDefaultFactory().CreateStructuredLogger(logger)
}

// NewStructuredLoggerWithSanitization 创建带脱敏功能的结构化日志器
func NewStructuredLoggerWithSanitization(logger log.Logger, sensitiveKeys []string) StructuredLoggerInterface {
	config := DefaultStructuredLogConfig()
	config.SensitiveKeys = append(config.SensitiveKeys, sensitiveKeys...)
	return NewStructuredLoggerFromLogger(logger, config)
}

// NewStructuredLoggerWithCustomRules 创建带自定义规则的结构化日志器
func NewStructuredLoggerWithCustomRules(logger log.Logger, customRules map[string]AnonymizeRule) StructuredLoggerInterface {
	config := DefaultStructuredLogConfig()
	for key, rule := range customRules {
		config.CustomRules[key] = rule
	}
	return NewStructuredLoggerFromLogger(logger, config)
}

// NewStructuredLoggerDisabled 创建禁用脱敏的结构化日志器
func NewStructuredLoggerDisabled(logger log.Logger) StructuredLoggerInterface {
	config := DefaultStructuredLogConfig()
	config.Enabled = false
	return NewStructuredLoggerFromLogger(logger, config)
}

// WrapExistingHelper 包装现有的log.Helper
func WrapExistingHelper(helper *log.Helper) StructuredLoggerInterface {
	return GetDefaultFactory().CreateStructuredLoggerFromHelper(helper)
}

// WrapExistingHelperWithConfig 使用自定义配置包装现有的log.Helper
func WrapExistingHelperWithConfig(helper *log.Helper, config *StructuredLogConfig) StructuredLoggerInterface {
	return NewStructuredLogger(helper, config)
}

// CreateProductionLogger 创建生产环境日志器
func CreateProductionLogger(logger log.Logger) StructuredLoggerInterface {
	config := &StructuredLogConfig{
		Enabled:           true,
		AutoDetect:        true,
		CustomRules:       GetDefaultRules(),
		SensitiveKeys:     []string{"password", "token", "secret", "key", "auth", "credential", "authorization", "session", "cookie"},
		MaxValueLength:    500,
		TruncateThreshold: 300,
	}
	return NewStructuredLoggerFromLogger(logger, config)
}

// CreateDevelopmentLogger 创建开发环境日志器
func CreateDevelopmentLogger(logger log.Logger) StructuredLoggerInterface {
	config := &StructuredLogConfig{
		Enabled:           false, // 开发环境可以禁用脱敏
		AutoDetect:        false,
		CustomRules:       GetDefaultRules(),
		SensitiveKeys:     []string{},
		MaxValueLength:    2000,
		TruncateThreshold: 1000,
	}
	return NewStructuredLoggerFromLogger(logger, config)
}

// CreateTestLogger 创建测试环境日志器
func CreateTestLogger(logger log.Logger) StructuredLoggerInterface {
	config := &StructuredLogConfig{
		Enabled:           true,
		AutoDetect:        true,
		CustomRules:       GetDefaultRules(),
		SensitiveKeys:     []string{"password", "token", "secret"},
		MaxValueLength:    1000,
		TruncateThreshold: 500,
	}
	return NewStructuredLoggerFromLogger(logger, config)
}

// MigrateFromHelper 从现有Helper迁移到结构化日志器
// 这个函数可以帮助现有代码逐步迁移
func MigrateFromHelper(helper *log.Helper, enableSanitization bool) StructuredLoggerInterface {
	var config *StructuredLogConfig
	if enableSanitization {
		config = DefaultStructuredLogConfig()
	} else {
		config = DefaultStructuredLogConfig()
		config.Enabled = false
	}
	return NewStructuredLogger(helper, config)
}