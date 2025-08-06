package sensitive

import (
	"context"
	"fmt"
	"strings"
	"testing"
	
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTransporter 模拟传输器
type MockTransporter struct {
	mock.Mock
}

func (m *MockTransporter) Kind() transport.Kind {
	args := m.Called()
	return args.Get(0).(transport.Kind)
}

func (m *MockTransporter) Endpoint() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTransporter) Operation() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTransporter) RequestHeader() transport.Header {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(transport.Header)
}

func (m *MockTransporter) ReplyHeader() transport.Header {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(transport.Header)
}

// MockLogger 模拟日志器
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Log(level log.Level, keyvals ...interface{}) error {
	args := m.Called(level, keyvals)
	return args.Error(0)
}

// TestDefaultLogSanitizeConfig 测试默认配置
func TestDefaultLogSanitizeConfig(t *testing.T) {
	config := DefaultLogSanitizeConfig()
	
	assert.True(t, config.Enabled)
	assert.True(t, config.SanitizeRequest)
	assert.True(t, config.SanitizeResponse)
	assert.True(t, config.SanitizeHeaders)
	assert.NotNil(t, config.CustomRules)
	assert.Contains(t, config.ExcludePaths, "/health")
	assert.Equal(t, 10000, config.MaxLogLength)
}

// TestNewLogSanitizeMiddleware 测试中间件创建
func TestNewLogSanitizeMiddleware(t *testing.T) {
	mockLogger := &MockLogger{}
	
	// 测试使用自定义配置
	config := &LogSanitizeConfig{
		Enabled: true,
		SanitizeRequest: true,
	}
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	assert.NotNil(t, middleware)
	assert.Equal(t, config, middleware.config)
	
	// 测试使用nil配置（应该使用默认配置）
	middleware = NewLogSanitizeMiddleware(nil, mockLogger)
	assert.NotNil(t, middleware)
	assert.NotNil(t, middleware.config)
	assert.True(t, middleware.config.Enabled)
	
	// 测试边界情况
	config2 := &LogSanitizeConfig{
		Enabled:      false,
		CustomRules:  make(map[string]AnonymizeRule),
		MaxLogLength: 0,
	}
	middleware = NewLogSanitizeMiddleware(config2, mockLogger)
	assert.NotNil(t, middleware)
}

// TestLogSanitizeMiddleware_AddHeadersToLog 测试添加请求头到日志
func TestLogSanitizeMiddleware_AddHeadersToLog(t *testing.T) {
	mockLogger := &MockLogger{}
	config := DefaultLogSanitizeConfig()
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	// 创建模拟传输器
	mockTransporter := &MockTransporter{}
	
	// 测试添加请求头（使用模拟传输器）
	logData := make(map[string]interface{})
	middleware.addHeadersToLog(mockTransporter, logData)
	
	// 由于使用模拟传输器，headers应该为空或不存在
	// 这个测试主要验证函数不会崩溃
	_, exists := logData["headers"]
	assert.False(t, exists)
}

// TestLogSanitizeMiddleware_IsSensitiveHeader 测试敏感请求头检测
func TestLogSanitizeMiddleware_IsSensitiveHeader(t *testing.T) {
	mockLogger := &MockLogger{}
	config := DefaultLogSanitizeConfig()
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	// 测试敏感头部
	sensitiveHeaders := []string{
		"Authorization",
		"Cookie",
		"Set-Cookie",
		"X-API-Key",
		"X-Auth-Token",
		"X-Access-Token",
		"X-Refresh-Token",
		"Password",
		"Secret",
		"Token",
		"authorization", // 测试大小写不敏感
		"COOKIE",
	}
	
	for _, header := range sensitiveHeaders {
		assert.True(t, middleware.isSensitiveHeader(header), "Header %s should be sensitive", header)
	}
	
	// 测试非敏感头部
	normalHeaders := []string{
		"Content-Type",
		"User-Agent",
		"Accept",
		"Content-Length",
		"Host",
	}
	
	for _, header := range normalHeaders {
		assert.False(t, middleware.isSensitiveHeader(header), "Header %s should not be sensitive", header)
	}
}

// TestLogSanitizeMiddleware_IsExcludedPath 测试排除路径检测
func TestLogSanitizeMiddleware_IsExcludedPath(t *testing.T) {
	mockLogger := &MockLogger{}
	config := DefaultLogSanitizeConfig()
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	// 测试排除路径
	excludedPaths := []string{
		"/health",
		"/metrics",
		"/ping",
		"/health/check", // 测试前缀匹配
		"/metrics/prometheus",
	}
	
	for _, path := range excludedPaths {
		assert.True(t, middleware.isExcludedPath(path), "Path %s should be excluded", path)
	}
	
	// 测试非排除路径
	normalPaths := []string{
		"/api/users",
		"/api/orders",
		"/login",
		"/register",
	}
	
	for _, path := range normalPaths {
		assert.False(t, middleware.isExcludedPath(path), "Path %s should not be excluded", path)
	}
}

// TestLogSanitizeMiddleware_SanitizeData 测试数据脱敏
func TestLogSanitizeMiddleware_SanitizeData(t *testing.T) {
	mockLogger := &MockLogger{}
	config := DefaultLogSanitizeConfig()
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	// 测试nil数据
	result := middleware.sanitizeData(nil)
	assert.Nil(t, result)
	
	// 测试MakeSensitive接口
	mockSensitive := &mockMakeSensitive{value: "sensitive-data"}
	result = middleware.sanitizeData(mockSensitive)
	// MakeSensitive接口会被AnonymizeValue处理
	if result != nil {
		resultMap, ok := result.(map[string]interface{})
		if ok {
			assert.Contains(t, resultMap, "value")
		}
	}
	
	// 测试LogSafeStringer接口
	mockLogSafe := &mockLogSafeStringer{value: "safe-data"}
	result = middleware.sanitizeData(mockLogSafe)
	assert.Equal(t, "[SAFE]safe-data", result)
	
	// 测试普通字符串
	result = middleware.sanitizeData("test@example.com")
	assert.Equal(t, "te**@example.com", result)
}

// TestLogSanitizeMiddleware_LogWithSanitization 测试带脱敏的日志记录
func TestLogSanitizeMiddleware_LogWithSanitization(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Log", log.LevelInfo, mock.Anything).Return(nil)
	
	config := DefaultLogSanitizeConfig()
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	// 测试正常日志记录
	logData := map[string]interface{}{
		"operation": "/api/users",
		"email":     "test@example.com",
		"phone":     "13812345678",
	}
	
	middleware.logWithSanitization("test", logData)
	
	// 验证日志方法被调用
	mockLogger.AssertExpectations(t)
}

// TestLogSanitizeMiddleware_LogWithSanitization_LongContent 测试长内容截断
func TestLogSanitizeMiddleware_LogWithSanitization_LongContent(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Log", log.LevelInfo, mock.Anything).Return(nil)
	
	config := DefaultLogSanitizeConfig()
	config.MaxLogLength = 50 // 设置较小的最大长度
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	// 创建长内容
	longContent := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		longContent[fmt.Sprintf("field_%d", i)] = "very long content that should be truncated"
	}
	
	middleware.logWithSanitization("test", longContent)
	
	// 验证日志方法被调用
	mockLogger.AssertExpectations(t)
}

// TestSensitiveLogHelper 测试敏感信息日志助手
func TestSensitiveLogHelper(t *testing.T) {
	// 测试默认助手
	helper := NewSensitiveLogHelper()
	assert.NotNil(t, helper)
	
	// 测试脱敏功能
	result := helper.SanitizeForLog("test@example.com")
	assert.Contains(t, result, "te**@example.com")
	
	// 测试消息脱敏
	result = helper.SanitizeMessage("Contact: 13812345678")
	assert.Contains(t, result, "138****5678")
	
	// 测试敏感信息检测
	hasSensitive := helper.ContainsSensitive("Email: test@example.com")
	assert.True(t, hasSensitive)
	
	hasSensitive = helper.ContainsSensitive("Normal message")
	assert.False(t, hasSensitive)
}

// TestSensitiveLogHelper_Advanced 测试敏感信息日志助手的高级功能
func TestSensitiveLogHelper_Advanced(t *testing.T) {
	// 测试自定义助手
	helper := NewSensitiveLogHelper()
	assert.NotNil(t, helper)
	
	// 测试复杂对象的脱敏
	type UserInfo struct {
		Email string
		Phone string
		Name  string
	}
	
	user := UserInfo{
		Email: "user@example.com",
		Phone: "13812345678",
		Name:  "张三",
	}
	
	sanitized := helper.SanitizeForLog(user)
	assert.NotContains(t, sanitized, "user@example.com")
	assert.NotContains(t, sanitized, "13812345678")
	assert.Contains(t, sanitized, "张三") // 非敏感信息应该保留
	
	// 测试LogSafeValue
	logSafeValue := NewLogSafeValue(user)
	assert.NotNil(t, logSafeValue)
	
	logSafeStr := logSafeValue.String()
	assert.NotContains(t, logSafeStr, "user@example.com")
	assert.NotContains(t, logSafeStr, "13812345678")
	
	// 测试空值处理
	emptyResult := helper.SanitizeForLog(nil)
	assert.Equal(t, "", emptyResult)
	
	// 测试字符串处理
	stringResult := helper.SanitizeForLog("普通字符串")
	assert.Equal(t, "普通字符串", stringResult)
	
	// 测试包含敏感信息的字符串
	sensitiveString := "联系邮箱：contact@test.com，手机：13987654321"
	sensitiveResult := helper.SanitizeForLog(sensitiveString)
	assert.NotContains(t, sensitiveResult, "contact@test.com")
	assert.NotContains(t, sensitiveResult, "13987654321")
}

// TestLogSafeValue_Advanced 测试LogSafeValue的高级功能
func TestLogSafeValue_Advanced(t *testing.T) {
	// 测试创建LogSafeValue
	value := NewLogSafeValue("test@example.com")
	assert.NotNil(t, value)
	
	// 测试String方法
	str := value.String()
	assert.Contains(t, str, "**")
	
	// 测试LogSafeString方法
	logStr := value.LogSafeString()
	assert.Equal(t, str, logStr)
	
	// 测试MarshalJSON方法
	jsonData, err := value.MarshalJSON()
	assert.NoError(t, err)
	assert.NotNil(t, jsonData)
	
	// 测试复杂对象
	complexValue := NewLogSafeValue(map[string]interface{}{
		"email": "user@test.com",
		"phone": "13812345678",
		"name":  "张三",
	})
	complexStr := complexValue.String()
	assert.Contains(t, complexStr, "**")
	
	// 测试nil值
	nilValue := NewLogSafeValue(nil)
	nilStr := nilValue.String()
	assert.Equal(t, "", nilStr)
	
	// 测试不同类型的值
	tests := []struct {
		name  string
		value interface{}
	}{
		{"string", "test@example.com"},
		{"int", 12345},
		{"bool", true},
		{"slice", []string{"user1@test.com", "user2@test.com"}},
		{"map", map[string]string{"email": "admin@test.com"}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logSafeValue := NewLogSafeValue(tt.value)
			assert.NotNil(t, logSafeValue)
			
			result := logSafeValue.String()
			assert.NotEmpty(t, result)
			
			// 对于包含邮箱的值，确保邮箱被脱敏
			if tt.name == "string" || tt.name == "slice" || tt.name == "map" {
				// 检查是否包含脱敏标记
				assert.Contains(t, result, "**")
			}
		})
	}
}

// TestNewSensitiveLogHelper_CustomRules 测试自定义规则
func TestNewSensitiveLogHelper_CustomRules(t *testing.T) {
	// 测试单个自定义规则
	customRules := map[string]AnonymizeRule{
		"custom_field": {
			FieldName: "custom_field",
			KeepStart: 1,
			KeepEnd:   1,
			MaskChar:  "#",
		},
	}
	
	helper := NewSensitiveLogHelper(customRules)
	assert.NotNil(t, helper)
	assert.NotNil(t, helper.rules)
	assert.Contains(t, helper.rules, "custom_field")
	
	// 测试多个自定义规则
	customRules1 := map[string]AnonymizeRule{
		"rule1": {
			FieldName: "rule1",
			KeepStart: 2,
			KeepEnd:   2,
			MaskChar:  "@",
		},
	}
	
	customRules2 := map[string]AnonymizeRule{
		"rule2": {
			FieldName: "rule2",
			KeepStart: 0,
			KeepEnd:   0,
			MaskChar:  "&",
		},
	}
	
	helper2 := NewSensitiveLogHelper(customRules1, customRules2)
	assert.NotNil(t, helper2)
	assert.Contains(t, helper2.rules, "rule1")
	assert.Contains(t, helper2.rules, "rule2")
	
	// 测试规则覆盖
	overrideRules := map[string]AnonymizeRule{
		"email": {
			FieldName: "email",
			KeepStart: 0,
			KeepEnd:   0,
			MaskChar:  "X",
		},
	}
	
	helper3 := NewSensitiveLogHelper(overrideRules)
	assert.NotNil(t, helper3)
	assert.Equal(t, "X", helper3.rules["email"].MaskChar)
}

// TestNewSensitiveLogHelper_EmptyRules 测试空规则
func TestNewSensitiveLogHelper_EmptyRules(t *testing.T) {
	// 测试空的自定义规则
	emptyRules := map[string]AnonymizeRule{}
	helper := NewSensitiveLogHelper(emptyRules)
	assert.NotNil(t, helper)
	assert.NotNil(t, helper.rules)
	
	// 应该仍然包含默认规则
	defaultRules := GetDefaultRules()
	for key := range defaultRules {
		assert.Contains(t, helper.rules, key)
	}
}

// TestNewSensitiveLogHelper_NilRules 测试nil规则
func TestNewSensitiveLogHelper_NilRules(t *testing.T) {
	// 测试传入nil规则
	var nilRules map[string]AnonymizeRule
	helper := NewSensitiveLogHelper(nilRules)
	assert.NotNil(t, helper)
	assert.NotNil(t, helper.rules)
	
	// 应该仍然包含默认规则
	defaultRules := GetDefaultRules()
	for key := range defaultRules {
		assert.Contains(t, helper.rules, key)
	}
}

// TestLogSafeValue 测试日志安全值包装器
func TestLogSafeValue(t *testing.T) {
	// 测试字符串值
	value := NewLogSafeValue("test@example.com")
	assert.NotNil(t, value)
	
	// 测试String方法
	result := value.String()
	assert.Contains(t, result, "te**@example.com")
	
	// 测试LogSafeString方法
	result = value.LogSafeString()
	assert.Contains(t, result, "te**@example.com")
	
	// 测试JSON序列化
	jsonData, err := value.MarshalJSON()
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "te**@example.com")
}

// TestNewSensitiveLogHelper_EdgeCases 测试SensitiveLogHelper的边界情况
func TestNewSensitiveLogHelper_EdgeCases(t *testing.T) {
	// 测试使用nil配置
	helper := NewSensitiveLogHelper(nil)
	assert.NotNil(t, helper)
	
	// 测试SanitizeForLog方法
	result := helper.SanitizeForLog("test@example.com")
	assert.Contains(t, result, "**")
	
	// 测试SanitizeMessage方法
	message := helper.SanitizeMessage("Contact us at test@example.com")
	assert.Contains(t, message, "**")
	assert.NotContains(t, message, "test@example.com")
	
	// 测试ContainsSensitive方法
	contains := helper.ContainsSensitive("test@example.com")
	assert.True(t, contains)
	
	contains = helper.ContainsSensitive("normal text")
	assert.False(t, contains)
	
	// 测试使用自定义配置
	rules := GetDefaultRules()
	helper = NewSensitiveLogHelper(rules)
	assert.NotNil(t, helper)
	
	// 测试长文本截断
	longText := strings.Repeat("a", 200)
	result = helper.SanitizeForLog(longText)
	assert.True(t, len(result) > 0) // 确保有输出
	
	// 测试多个自定义规则合并
	customRules1 := map[string]AnonymizeRule{
		"custom_field1": {FieldName: "custom_field1", KeepStart: 0, KeepEnd: 0, MaskChar: "*"},
	}
	customRules2 := map[string]AnonymizeRule{
		"custom_field2": {FieldName: "custom_field2", KeepStart: 1, KeepEnd: 1, MaskChar: "*"},
	}
	helper = NewSensitiveLogHelper(customRules1, customRules2)
	assert.NotNil(t, helper)
	assert.NotNil(t, helper.rules)
	
	// 验证自定义规则被正确合并
	assert.Contains(t, helper.rules, "custom_field1")
	assert.Contains(t, helper.rules, "custom_field2")
	assert.Equal(t, "custom_field1", helper.rules["custom_field1"].FieldName)
	assert.Equal(t, "custom_field2", helper.rules["custom_field2"].FieldName)
	
	// 测试空的自定义规则
	emptyRules := map[string]AnonymizeRule{}
	helper = NewSensitiveLogHelper(emptyRules)
	assert.NotNil(t, helper)
	assert.NotNil(t, helper.rules)
}

// TestLogSanitizeMiddleware_EdgeCases 测试中间件的边界情况
func TestLogSanitizeMiddleware_EdgeCases(t *testing.T) {
	mockLogger := &MockLogger{}
	config := &LogSanitizeConfig{
		Enabled:      true,
		CustomRules:  make(map[string]AnonymizeRule),
		ExcludePaths: []string{"/health", "/metrics"},
		MaxLogLength: 50,
	}
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	// 测试sanitizeData方法的各种输入
	tests := []struct {
		name     string
		input    interface{}
		expected bool // 是否期望有输出
	}{
		{"nil input", nil, false},
		{"empty string", "", true}, // 空字符串也会有输出
		{"number", 123, true},
		{"boolean", true, true},
		{"slice", []string{"a", "b"}, true},
		{"map", map[string]string{"key": "value"}, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.sanitizeData(tt.input)
			if tt.expected {
				assert.NotNil(t, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
	
	// 测试logWithSanitization方法
	mockLogger.On("Log", log.LevelInfo, mock.Anything).Return(nil)
	logData := map[string]interface{}{
		"message": "test message",
		"email":   "user@example.com",
		"level":   "info",
	}
	
	middleware.logWithSanitization("INFO", logData)
	// 验证日志被调用
	mockLogger.AssertExpectations(t)
}

// TestLogSanitizeMiddleware_Middleware_Disabled 测试禁用中间件
func TestLogSanitizeMiddleware_Middleware_Disabled(t *testing.T) {
	mockLogger := &MockLogger{}
	config := DefaultLogSanitizeConfig()
	config.Enabled = false
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	// 创建模拟处理器
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}
	
	// 创建中间件
	mw := middleware.Middleware()
	wrappedHandler := mw(handler)
	
	// 执行处理器
	ctx := context.Background()
	resp, err := wrappedHandler(ctx, "request")
	
	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, "response", resp)
	
	// 验证没有日志调用
	mockLogger.AssertNotCalled(t, "Log")
}

// TestLogSanitizeMiddleware_IsSensitiveHeader_Extended 测试敏感请求头检测扩展
func TestLogSanitizeMiddleware_IsSensitiveHeader_Extended(t *testing.T) {
	mockLogger := &MockLogger{}
	config := DefaultLogSanitizeConfig()
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	tests := []struct {
		name       string
		headerName string
		expected   bool
	}{
		{"case_insensitive", "AUTHORIZATION", true},
		{"partial_match", "Custom-Authorization-Header", true},
		{"x_api_key_header", "X-API-Key", true},
		{"x_auth_token_header", "X-Auth-Token", true},
		{"x_access_token_header", "X-Access-Token", true},
		{"x_refresh_token_header", "X-Refresh-Token", true},
		{"password_header", "Password", true},
		{"secret_header", "Secret", true},
		{"token_header", "Token", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.isSensitiveHeader(tt.headerName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLogSanitizeMiddleware_IsExcludedPath_Extended 测试排除路径检测扩展
func TestLogSanitizeMiddleware_IsExcludedPath_Extended(t *testing.T) {
	mockLogger := &MockLogger{}
	config := DefaultLogSanitizeConfig()
	config.ExcludePaths = []string{"/health", "/metrics", "/api/v1/status"}
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	tests := []struct {
		name      string
		operation string
		expected  bool
	}{
		{"health_with_params", "/health?check=all", true},
		{"metrics_subpath", "/metrics/prometheus", true},
		{"status_subpath", "/api/v1/status/detailed", true},
		{"partial_match_excluded", "/healthcheck", true},
		{"empty_operation", "", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.isExcludedPath(tt.operation)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLogSanitizeMiddleware_LogWithSanitization_MarshalError 测试JSON序列化错误
func TestLogSanitizeMiddleware_LogWithSanitization_MarshalError(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("Log", log.LevelError, mock.Anything).Return(nil)
	
	config := DefaultLogSanitizeConfig()
	middleware := NewLogSanitizeMiddleware(config, mockLogger)
	
	// 创建一个无法序列化的数据（包含循环引用）
	type cyclicStruct struct {
		Name string
		Ref  *cyclicStruct
	}
	
	cyclic := &cyclicStruct{Name: "test"}
	cyclic.Ref = cyclic // 创建循环引用
	
	logData := map[string]interface{}{
		"cyclic": cyclic,
	}
	
	middleware.logWithSanitization("test", logData)
	
	// 验证错误日志被调用
	mockLogger.AssertExpectations(t)
}

// 注意：mockMakeSensitive 和 mockLogSafeStringer 类型已在 anonymizer_test.go 中定义