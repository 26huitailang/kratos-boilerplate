package sensitive

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// LogSanitizeConfig 日志脱敏配置
type LogSanitizeConfig struct {
	Enabled         bool                     `json:"enabled"`          // 是否启用脱敏
	SanitizeRequest bool                     `json:"sanitize_request"`  // 是否脱敏请求
	SanitizeResponse bool                    `json:"sanitize_response"` // 是否脱敏响应
	SanitizeHeaders  bool                    `json:"sanitize_headers"`  // 是否脱敏请求头
	CustomRules      map[string]AnonymizeRule `json:"custom_rules"`      // 自定义脱敏规则
	ExcludePaths     []string                 `json:"exclude_paths"`     // 排除路径
	MaxLogLength     int                      `json:"max_log_length"`    // 最大日志长度
}

// DefaultLogSanitizeConfig 默认日志脱敏配置
func DefaultLogSanitizeConfig() *LogSanitizeConfig {
	return &LogSanitizeConfig{
		Enabled:         true,
		SanitizeRequest: true,
		SanitizeResponse: true,
		SanitizeHeaders:  true,
		CustomRules:      GetDefaultRules(),
		ExcludePaths:     []string{"/health", "/metrics", "/ping"},
		MaxLogLength:     10000,
	}
}

// LogSanitizeMiddleware 日志脱敏中间件
type LogSanitizeMiddleware struct {
	config    *LogSanitizeConfig
	sanitizer LogSanitizer
	logger    log.Logger
}

// NewLogSanitizeMiddleware 创建日志脱敏中间件
func NewLogSanitizeMiddleware(config *LogSanitizeConfig, logger log.Logger) *LogSanitizeMiddleware {
	if config == nil {
		config = DefaultLogSanitizeConfig()
	}
	
	return &LogSanitizeMiddleware{
		config:    config,
		sanitizer: NewLogSanitizer(),
		logger:    logger,
	}
}

// Middleware 返回Kratos中间件
func (m *LogSanitizeMiddleware) Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if !m.config.Enabled {
				return handler(ctx, req)
			}
			
			// 获取传输信息
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}
			
			// 检查是否在排除路径中
			if m.isExcludedPath(tr.Operation()) {
				return handler(ctx, req)
			}
			
			start := time.Now()
			
			// 记录请求日志
			m.logRequest(ctx, tr, req)
			
			// 执行处理器
			resp, err := handler(ctx, req)
			
			// 记录响应日志
			m.logResponse(ctx, tr, resp, err, time.Since(start))
			
			return resp, err
		}
	}
}

// logRequest 记录请求日志
func (m *LogSanitizeMiddleware) logRequest(ctx context.Context, tr transport.Transporter, req interface{}) {
	if !m.config.SanitizeRequest {
		return
	}
	
	logData := map[string]interface{}{
		"kind":      tr.Kind().String(),
		"operation": tr.Operation(),
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	// 添加请求数据
	if req != nil {
		logData["request"] = m.sanitizeData(req)
	}
	
	// 添加请求头信息
	if m.config.SanitizeHeaders {
		m.addHeadersToLog(tr, logData)
	}
	
	// 记录日志
	m.logWithSanitization("request", logData)
}

// logResponse 记录响应日志
func (m *LogSanitizeMiddleware) logResponse(ctx context.Context, tr transport.Transporter, resp interface{}, err error, duration time.Duration) {
	if !m.config.SanitizeResponse {
		return
	}
	
	logData := map[string]interface{}{
		"kind":      tr.Kind().String(),
		"operation": tr.Operation(),
		"duration":  duration.String(),
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	// 添加响应数据
	if resp != nil {
		logData["response"] = m.sanitizeData(resp)
	}
	
	// 添加错误信息
	if err != nil {
		logData["error"] = m.sanitizer.SanitizeLogMessage(err.Error())
	}
	
	// 记录日志
	m.logWithSanitization("response", logData)
}

// addHeadersToLog 添加请求头到日志
func (m *LogSanitizeMiddleware) addHeadersToLog(tr transport.Transporter, logData map[string]interface{}) {
	headers := make(map[string]interface{})
	
	switch t := tr.(type) {
	case *http.Transport:
		// HTTP请求头
		for key, values := range t.Request().Header {
			if m.isSensitiveHeader(key) {
				headers[key] = "[REDACTED]"
			} else {
				headers[key] = values
			}
		}
	case *grpc.Transport:
		// gRPC元数据
		if md := t.RequestHeader(); md != nil {
			// 使用Keys()方法获取所有键
			for _, key := range md.Keys() {
				values := md.Get(key)
				if m.isSensitiveHeader(key) {
					headers[key] = "[REDACTED]"
				} else {
					headers[key] = values
				}
			}
		}
	}
	
	if len(headers) > 0 {
		logData["headers"] = headers
	}
}

// sanitizeData 脱敏数据
func (m *LogSanitizeMiddleware) sanitizeData(data interface{}) interface{} {
	if data == nil {
		return nil
	}
	
	// 如果实现了MakeSensitive接口
	if sensitive, ok := data.(MakeSensitive); ok {
		return sensitive.Anonymize()
	}
	
	// 如果实现了LogSafeStringer接口
	if logSafe, ok := data.(LogSafeStringer); ok {
		return logSafe.LogSafeString()
	}
	
	// 使用通用脱敏
	anonymizer := NewAnonymizer()
	return anonymizer.AnonymizeValue(data, m.config.CustomRules)
}

// logWithSanitization 带脱敏的日志记录
func (m *LogSanitizeMiddleware) logWithSanitization(logType string, data map[string]interface{}) {
	// 序列化日志数据
	jsonData, err := json.Marshal(data)
	if err != nil {
		m.logger.Log(log.LevelError, "msg", "Failed to marshal log data", "error", err)
		return
	}
	
	// 脱敏日志内容
	sanitizedContent := m.sanitizer.SanitizeLogMessage(string(jsonData))
	
	// 限制日志长度
	if m.config.MaxLogLength > 0 && len(sanitizedContent) > m.config.MaxLogLength {
		sanitizedContent = sanitizedContent[:m.config.MaxLogLength] + "...[TRUNCATED]"
	}
	
	// 记录日志
	m.logger.Log(log.LevelInfo, "msg", fmt.Sprintf("Sanitized %s log", logType), "data", sanitizedContent)
}

// isExcludedPath 检查是否为排除路径
func (m *LogSanitizeMiddleware) isExcludedPath(operation string) bool {
	for _, excludePath := range m.config.ExcludePaths {
		// 使用前缀匹配而不是包含匹配，避免误匹配
		if strings.HasPrefix(operation, excludePath) {
			return true
		}
	}
	return false
}

// isSensitiveHeader 检查是否为敏感请求头
func (m *LogSanitizeMiddleware) isSensitiveHeader(headerName string) bool {
	sensitiveHeaders := []string{
		"authorization",
		"cookie",
		"set-cookie",
		"x-api-key",
		"x-auth-token",
		"x-access-token",
		"x-refresh-token",
		"password",
		"secret",
		"token",
	}
	
	headerLower := strings.ToLower(headerName)
	for _, sensitive := range sensitiveHeaders {
		if strings.Contains(headerLower, sensitive) {
			return true
		}
	}
	
	return false
}

// SensitiveLogHelper 敏感信息日志助手
type SensitiveLogHelper struct {
	sanitizer LogSanitizer
	rules     map[string]AnonymizeRule
}

// NewSensitiveLogHelper 创建敏感信息日志助手
func NewSensitiveLogHelper(customRules ...map[string]AnonymizeRule) *SensitiveLogHelper {
	rules := GetDefaultRules()
	
	// 合并自定义规则
	for _, customRule := range customRules {
		for key, rule := range customRule {
			rules[key] = rule
		}
	}
	
	return &SensitiveLogHelper{
		sanitizer: NewLogSanitizer(),
		rules:     rules,
	}
}

// SanitizeForLog 为日志脱敏数据
func (h *SensitiveLogHelper) SanitizeForLog(data interface{}) string {
	return h.sanitizer.SanitizeLogContent(data)
}

// SanitizeMessage 脱敏消息
func (h *SensitiveLogHelper) SanitizeMessage(message string) string {
	return h.sanitizer.SanitizeLogMessage(message)
}

// ContainsSensitive 检查是否包含敏感信息
func (h *SensitiveLogHelper) ContainsSensitive(content string) bool {
	return h.sanitizer.ContainsSensitiveInfo(content)
}

// LogSafeValue 创建日志安全的值包装器
type LogSafeValue struct {
	value     interface{}
	sanitizer LogSanitizer
}

// NewLogSafeValue 创建日志安全值
func NewLogSafeValue(value interface{}) *LogSafeValue {
	return &LogSafeValue{
		value:     value,
		sanitizer: NewLogSanitizer(),
	}
}

// String 实现fmt.Stringer接口
func (v *LogSafeValue) String() string {
	return v.sanitizer.SanitizeLogContent(v.value)
}

// LogSafeString 实现LogSafeStringer接口
func (v *LogSafeValue) LogSafeString() string {
	return v.String()
}

// MarshalJSON 实现json.Marshaler接口
func (v *LogSafeValue) MarshalJSON() ([]byte, error) {
	sanitized := v.sanitizer.SanitizeLogContent(v.value)
	return json.Marshal(sanitized)
}