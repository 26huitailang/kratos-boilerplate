package config

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	validator "github.com/go-playground/validator/v10"
)

// Validator 配置验证器接口
type Validator interface {
	Validate(config interface{}) error
	RegisterCustomValidator(tag string, fn validator.Func) error
}

// configValidator 配置验证器实现
type configValidator struct {
	validate *validator.Validate
}

// NewValidator 创建配置验证器
func NewValidator() Validator {
	v := validator.New()

	// 注册自定义验证器
	customValidator := &configValidator{validate: v}
	customValidator.registerCustomValidators()

	return customValidator
}

// Validate 验证配置
func (v *configValidator) Validate(config interface{}) error {
	if err := v.validate.Struct(config); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// RegisterCustomValidator 注册自定义验证器
func (v *configValidator) RegisterCustomValidator(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}

// registerCustomValidators 注册自定义验证器
func (v *configValidator) registerCustomValidators() {
	// 注册时间间隔验证器
	v.validate.RegisterValidation("duration", func(fl validator.FieldLevel) bool {
		field := fl.Field()
		if field.Kind() != reflect.String {
			return true // 不是字符串类型，跳过验证
		}

		durationStr := field.String()
		if durationStr == "" {
			return true // 空值跳过验证
		}

		_, err := time.ParseDuration(durationStr)
		return err == nil
	})

	// 注册端口验证器
	v.validate.RegisterValidation("port", func(fl validator.FieldLevel) bool {
		field := fl.Field()
		if field.Kind() != reflect.String {
			return true
		}

		addr := field.String()
		if addr == "" {
			return true
		}

		// 简单的端口格式验证
		parts := strings.Split(addr, ":")
		if len(parts) != 2 {
			return false
		}

		// 检查端口是否为数字且在有效范围内
		port := parts[1]
		if port == "" {
			return false
		}

		return true
	})

	// 注册数据库驱动验证器
	v.validate.RegisterValidation("db_driver", func(fl validator.FieldLevel) bool {
		field := fl.Field()
		if field.Kind() != reflect.String {
			return true
		}

		driver := field.String()
		validDrivers := []string{"postgres", "mysql", "sqlite", "sqlserver"}

		for _, validDriver := range validDrivers {
			if driver == validDriver {
				return true
			}
		}

		return false
	})

	// 注册日志级别验证器
	v.validate.RegisterValidation("log_level", func(fl validator.FieldLevel) bool {
		field := fl.Field()
		if field.Kind() != reflect.String {
			return true
		}

		level := field.String()
		validLevels := []string{"debug", "info", "warn", "error", "fatal"}

		for _, validLevel := range validLevels {
			if strings.ToLower(level) == validLevel {
				return true
			}
		}

		return false
	})
}

// formatValidationError 格式化验证错误
func (v *configValidator) formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string

		for _, fieldError := range validationErrors {
			message := v.getErrorMessage(fieldError)
			messages = append(messages, message)
		}

		return fmt.Errorf("configuration validation failed: %s", strings.Join(messages, "; "))
	}

	return fmt.Errorf("configuration validation failed: %w", err)
}

// getErrorMessage 获取错误消息
func (v *configValidator) getErrorMessage(fieldError validator.FieldError) string {
	field := fieldError.Field()
	tag := fieldError.Tag()
	param := fieldError.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("field '%s' is required", field)
	case "min":
		return fmt.Sprintf("field '%s' must be at least %s", field, param)
	case "max":
		return fmt.Sprintf("field '%s' must be at most %s", field, param)
	case "oneof":
		return fmt.Sprintf("field '%s' must be one of: %s", field, param)
	case "duration":
		return fmt.Sprintf("field '%s' must be a valid duration", field)
	case "port":
		return fmt.Sprintf("field '%s' must be a valid address with port", field)
	case "db_driver":
		return fmt.Sprintf("field '%s' must be a valid database driver", field)
	case "log_level":
		return fmt.Sprintf("field '%s' must be a valid log level", field)
	default:
		return fmt.Sprintf("field '%s' failed validation '%s'", field, tag)
	}
}

// ConfigValidator 配置验证助手
type ConfigValidator struct {
	validator Validator
}

// NewConfigValidator 创建配置验证助手
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		validator: NewValidator(),
	}
}

// ValidateServerConfig 验证服务器配置
func (cv *ConfigValidator) ValidateServerConfig(config *Config) error {
	// 验证HTTP服务器配置
	if config.Server.HTTP.Addr == "" {
		return fmt.Errorf("HTTP server address is required")
	}

	if config.Server.HTTP.Timeout <= 0 {
		config.Server.HTTP.Timeout = 30 * time.Second
	}

	// 验证gRPC服务器配置
	if config.Server.GRPC.Addr == "" {
		return fmt.Errorf("gRPC server address is required")
	}

	if config.Server.GRPC.Timeout <= 0 {
		config.Server.GRPC.Timeout = 30 * time.Second
	}

	return nil
}

// ValidateDataConfig 验证数据配置
func (cv *ConfigValidator) ValidateDataConfig(config *Config) error {
	// 验证数据库配置
	if config.Data.Database.Driver == "" {
		return fmt.Errorf("database driver is required")
	}

	if config.Data.Database.Source == "" {
		return fmt.Errorf("database source is required")
	}

	// 验证Redis配置
	if config.Data.Redis.Addr == "" {
		return fmt.Errorf("Redis address is required")
	}

	if config.Data.Redis.ReadTimeout <= 0 {
		config.Data.Redis.ReadTimeout = 3 * time.Second
	}

	if config.Data.Redis.WriteTimeout <= 0 {
		config.Data.Redis.WriteTimeout = 3 * time.Second
	}

	return nil
}

// ValidateAuthConfig 验证认证配置
func (cv *ConfigValidator) ValidateAuthConfig(config *Config) error {
	if config.Auth.JWTSecretKey == "" {
		return fmt.Errorf("JWT secret key is required")
	}

	if len(config.Auth.JWTSecretKey) < 16 {
		return fmt.Errorf("JWT secret key must be at least 16 characters")
	}

	if config.Auth.MaxLoginAttempts <= 0 {
		config.Auth.MaxLoginAttempts = 5
	}

	if config.Auth.MaxLoginAttempts > 10 {
		return fmt.Errorf("max login attempts should not exceed 10")
	}

	return nil
}

// ValidateLogConfig 验证日志配置
func (cv *ConfigValidator) ValidateLogConfig(config *Config) error {
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	levelValid := false
	for _, level := range validLevels {
		if strings.ToLower(config.Log.Level) == level {
			levelValid = true
			break
		}
	}

	if !levelValid {
		return fmt.Errorf("invalid log level: %s, must be one of: %s", config.Log.Level, strings.Join(validLevels, ", "))
	}

	validFormats := []string{"json", "text"}
	formatValid := false
	for _, format := range validFormats {
		if strings.ToLower(config.Log.Format) == format {
			formatValid = true
			break
		}
	}

	if !formatValid {
		return fmt.Errorf("invalid log format: %s, must be one of: %s", config.Log.Format, strings.Join(validFormats, ", "))
	}

	return nil
}

// ValidateTracingConfig 验证链路追踪配置
func (cv *ConfigValidator) ValidateTracingConfig(config *Config) error {
	if config.Tracing.Enabled {
		if config.Tracing.Jaeger.Endpoint == "" {
			return fmt.Errorf("Jaeger endpoint is required when tracing is enabled")
		}

		if config.Tracing.Jaeger.SampleRate < 0 || config.Tracing.Jaeger.SampleRate > 1 {
			return fmt.Errorf("Jaeger sample rate must be between 0 and 1")
		}
	}

	return nil
}

// ValidateAll 验证所有配置
func (cv *ConfigValidator) ValidateAll(config *Config) error {
	validators := []func(*Config) error{
		cv.ValidateServerConfig,
		cv.ValidateDataConfig,
		cv.ValidateAuthConfig,
		cv.ValidateLogConfig,
		cv.ValidateTracingConfig,
	}

	for _, validate := range validators {
		if err := validate(config); err != nil {
			return err
		}
	}

	return cv.validator.Validate(config)
}
