package sensitive

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
)

// StructuredLogger 结构化日志器，支持自动脱敏
type StructuredLogger struct {
	helper     *log.Helper
	logger     log.Logger // 保存原始logger用于WithFields
	anonymizer Anonymizer
	rules      map[string]AnonymizeRule
	config     *StructuredLogConfig
}

// StructuredLogConfig 结构化日志配置
type StructuredLogConfig struct {
	Enabled           bool                     `json:"enabled"`            // 是否启用脱敏
	AutoDetect        bool                     `json:"auto_detect"`        // 是否自动检测敏感信息
	CustomRules       map[string]AnonymizeRule `json:"custom_rules"`       // 自定义脱敏规则
	SensitiveKeys     []string                 `json:"sensitive_keys"`     // 敏感字段名列表
	MaxValueLength    int                      `json:"max_value_length"`   // 最大值长度
	TruncateThreshold int                      `json:"truncate_threshold"` // 截断阈值
}

// DefaultStructuredLogConfig 默认配置
func DefaultStructuredLogConfig() *StructuredLogConfig {
	return &StructuredLogConfig{
		Enabled:           true,
		AutoDetect:        true,
		CustomRules:       GetDefaultRules(),
		SensitiveKeys:     []string{"password", "token", "secret", "key", "auth", "credential"},
		MaxValueLength:    1000,
		TruncateThreshold: 500,
	}
}

// NewStructuredLogger 创建结构化日志器
func NewStructuredLogger(helper *log.Helper, config *StructuredLogConfig) *StructuredLogger {
	if config == nil {
		config = DefaultStructuredLogConfig()
	}
	
	return &StructuredLogger{
		helper:     helper,
		logger:     nil, // 从helper创建时无法获取原始logger
		anonymizer: NewAnonymizer(),
		rules:      config.CustomRules,
		config:     config,
	}
}

// NewStructuredLoggerFromLogger 从原始logger创建结构化日志器
func NewStructuredLoggerFromLogger(logger log.Logger, config *StructuredLogConfig) *StructuredLogger {
	if config == nil {
		config = DefaultStructuredLogConfig()
	}
	
	helper := log.NewHelper(logger)
	return &StructuredLogger{
		helper:     helper,
		logger:     logger,
		anonymizer: NewAnonymizer(),
		rules:      config.CustomRules,
		config:     config,
	}
}

// Infow 信息级别结构化日志
func (s *StructuredLogger) Infow(msg string, keysAndValues ...interface{}) {
	if s.helper == nil {
		return
	}
	
	sanitized := s.sanitizeKeyValues(keysAndValues...)
	// 构建完整的参数列表：msg + sanitized key-value pairs
	args := make([]interface{}, 0, len(sanitized)+1)
	args = append(args, msg)
	args = append(args, sanitized...)
	s.helper.Info(args...)
}

// Debugw 调试级别结构化日志
func (s *StructuredLogger) Debugw(msg string, keysAndValues ...interface{}) {
	if s.helper == nil {
		return
	}
	
	sanitized := s.sanitizeKeyValues(keysAndValues...)
	args := make([]interface{}, 0, len(sanitized)+1)
	args = append(args, msg)
	args = append(args, sanitized...)
	s.helper.Debug(args...)
}

// Errorw 错误级别结构化日志
func (s *StructuredLogger) Errorw(msg string, keysAndValues ...interface{}) {
	if s.helper == nil {
		return
	}
	
	sanitized := s.sanitizeKeyValues(keysAndValues...)
	args := make([]interface{}, 0, len(sanitized)+1)
	args = append(args, msg)
	args = append(args, sanitized...)
	s.helper.Error(args...)
}

// Warnw 警告级别结构化日志
func (s *StructuredLogger) Warnw(msg string, keysAndValues ...interface{}) {
	if s.helper == nil {
		return
	}
	
	sanitized := s.sanitizeKeyValues(keysAndValues...)
	args := make([]interface{}, 0, len(sanitized)+1)
	args = append(args, msg)
	args = append(args, sanitized...)
	s.helper.Warn(args...)
}

// Fatalw 致命错误级别结构化日志
func (s *StructuredLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	if s.helper == nil {
		return
	}
	
	sanitized := s.sanitizeKeyValues(keysAndValues...)
	args := make([]interface{}, 0, len(sanitized)+1)
	args = append(args, msg)
	args = append(args, sanitized...)
	s.helper.Fatal(args...)
}

// WithContext 添加上下文
func (s *StructuredLogger) WithContext(ctx context.Context) StructuredLoggerInterface {
	return &StructuredLogger{
		helper:     s.helper.WithContext(ctx),
		logger:     s.logger,
		anonymizer: s.anonymizer,
		rules:      s.rules,
		config:     s.config,
	}
}

// WithFields 添加字段
func (s *StructuredLogger) WithFields(keysAndValues ...interface{}) StructuredLoggerInterface {
	sanitized := s.sanitizeKeyValues(keysAndValues...)
	
	// 如果有原始logger，使用它创建新的logger
	if s.logger != nil {
		newLogger := log.With(s.logger, sanitized...)
		newHelper := log.NewHelper(newLogger)
		return &StructuredLogger{
			helper:     newHelper,
			logger:     newLogger,
			anonymizer: s.anonymizer,
			rules:      s.rules,
			config:     s.config,
		}
	}
	
	// 如果没有原始logger，尝试从helper获取logger并创建新的
	// 这是一个fallback方案，可能不完全支持所有功能
	return s
}

// 兼容性方法 - 保持与现有代码的兼容性

// Info 信息级别日志
func (s *StructuredLogger) Info(args ...interface{}) {
	if s.helper != nil {
		s.helper.Info(args...)
	}
}

// Infof 格式化信息级别日志
func (s *StructuredLogger) Infof(template string, args ...interface{}) {
	if s.helper != nil {
		s.helper.Infof(template, args...)
	}
}

// Debug 调试级别日志
func (s *StructuredLogger) Debug(args ...interface{}) {
	if s.helper != nil {
		s.helper.Debug(args...)
	}
}

// Debugf 格式化调试级别日志
func (s *StructuredLogger) Debugf(template string, args ...interface{}) {
	if s.helper != nil {
		s.helper.Debugf(template, args...)
	}
}

// Error 错误级别日志
func (s *StructuredLogger) Error(args ...interface{}) {
	if s.helper != nil {
		s.helper.Error(args...)
	}
}

// Errorf 格式化错误级别日志
func (s *StructuredLogger) Errorf(template string, args ...interface{}) {
	if s.helper != nil {
		s.helper.Errorf(template, args...)
	}
}

// Warn 警告级别日志
func (s *StructuredLogger) Warn(args ...interface{}) {
	if s.helper != nil {
		s.helper.Warn(args...)
	}
}

// Warnf 格式化警告级别日志
func (s *StructuredLogger) Warnf(template string, args ...interface{}) {
	if s.helper != nil {
		s.helper.Warnf(template, args...)
	}
}

// Fatal 致命错误级别日志
func (s *StructuredLogger) Fatal(args ...interface{}) {
	if s.helper != nil {
		s.helper.Fatal(args...)
	}
}

// Fatalf 格式化致命错误级别日志
func (s *StructuredLogger) Fatalf(template string, args ...interface{}) {
	if s.helper != nil {
		s.helper.Fatalf(template, args...)
	}
}

// sanitizeKeyValues 脱敏键值对
func (s *StructuredLogger) sanitizeKeyValues(keysAndValues ...interface{}) []interface{} {
	if !s.config.Enabled || len(keysAndValues) == 0 {
		return keysAndValues
	}
	
	result := make([]interface{}, len(keysAndValues))
	copy(result, keysAndValues)
	
	// 处理键值对（key, value, key, value...）
	for i := 0; i < len(result)-1; i += 2 {
		key, ok := result[i].(string)
		if !ok {
			continue
		}
		
		value := result[i+1]
		
		// 检查是否为敏感字段
		if s.isSensitiveKey(key) {
			result[i+1] = s.sanitizeValue(key, value)
		} else if s.config.AutoDetect {
			// 自动检测敏感信息
			result[i+1] = s.autoDetectAndSanitize(value)
		}
		
		// 检查值长度，必要时截断
		result[i+1] = s.truncateIfNeeded(result[i+1])
	}
	
	return result
}

// isSensitiveKey 检查是否为敏感字段
func (s *StructuredLogger) isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	
	// 检查预定义的敏感字段
	for _, sensitiveKey := range s.config.SensitiveKeys {
		if strings.Contains(key, strings.ToLower(sensitiveKey)) {
			return true
		}
	}
	
	// 检查是否有对应的脱敏规则
	_, exists := s.rules[key]
	return exists
}

// sanitizeValue 脱敏值
func (s *StructuredLogger) sanitizeValue(key string, value interface{}) interface{} {
	if value == nil {
		return value
	}
	
	// 转换为字符串进行脱敏
	strValue := fmt.Sprintf("%v", value)
	
	// 获取对应的脱敏规则
	key = strings.ToLower(key)
	if rule, exists := s.rules[key]; exists {
		return s.anonymizer.AnonymizeString(strValue, rule)
	}
	
	// 使用默认脱敏策略
	return s.defaultSanitize(key, strValue)
}

// autoDetectAndSanitize 自动检测并脱敏
func (s *StructuredLogger) autoDetectAndSanitize(value interface{}) interface{} {
	if value == nil {
		return value
	}
	
	strValue := fmt.Sprintf("%v", value)
	
	// 使用敏感信息检测器
	detector := NewSensitiveDetector()
	detected := detector.DetectAll(strValue)
	
	result := strValue
	for fieldType, matches := range detected {
		if rule, exists := s.rules[fieldType]; exists {
			for _, match := range matches {
				anonymized := s.anonymizer.AnonymizeString(match, rule)
				result = strings.ReplaceAll(result, match, anonymized)
			}
		}
	}
	
	return result
}

// defaultSanitize 默认脱敏策略
func (s *StructuredLogger) defaultSanitize(key, value string) string {
	// 对于包含敏感关键词的字段，使用通用脱敏
	for _, sensitiveKey := range s.config.SensitiveKeys {
		if strings.Contains(key, strings.ToLower(sensitiveKey)) {
			if len(value) <= 4 {
				return "[REDACTED]"
			}
			return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
		}
	}
	
	return value
}

// truncateIfNeeded 必要时截断值
func (s *StructuredLogger) truncateIfNeeded(value interface{}) interface{} {
	if s.config.MaxValueLength <= 0 {
		return value
	}
	
	strValue := fmt.Sprintf("%v", value)
	if len(strValue) > s.config.MaxValueLength {
		if s.config.TruncateThreshold > 0 && len(strValue) > s.config.TruncateThreshold {
			return strValue[:s.config.TruncateThreshold] + "...[truncated]"
		}
		return strValue[:s.config.MaxValueLength] + "..."
	}
	
	return value
}

// UpdateRules 更新脱敏规则
func (s *StructuredLogger) UpdateRules(rules map[string]AnonymizeRule) {
	if s.rules == nil {
		s.rules = make(map[string]AnonymizeRule)
	}
	
	for key, rule := range rules {
		s.rules[key] = rule
	}
}

// AddSensitiveKey 添加敏感字段
func (s *StructuredLogger) AddSensitiveKey(key string) {
	s.config.SensitiveKeys = append(s.config.SensitiveKeys, key)
}

// GetHelper 获取底层的log.Helper
func (s *StructuredLogger) GetHelper() *log.Helper {
	return s.helper
}