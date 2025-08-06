package sensitive

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

// mockLogger 模拟日志器
type mockLogger struct {
	buffer *bytes.Buffer
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		buffer: &bytes.Buffer{},
	}
}

func (m *mockLogger) Log(level log.Level, keyvals ...interface{}) error {
	data := make(map[string]interface{})
	for i := 0; i < len(keyvals)-1; i += 2 {
		if key, ok := keyvals[i].(string); ok {
			data[key] = keyvals[i+1]
		}
	}
	
	jsonData, _ := json.Marshal(data)
	m.buffer.Write(jsonData)
	m.buffer.WriteString("\n")
	return nil
}

func (m *mockLogger) GetOutput() string {
	return m.buffer.String()
}

func (m *mockLogger) Clear() {
	m.buffer.Reset()
}

func TestStructuredLogger_Basic(t *testing.T) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = false // 先禁用脱敏测试基本功能
	
	slogger := NewStructuredLogger(helper, config)
	
	// 测试基本的结构化日志
	slogger.Infow("test message", "key1", "value1", "key2", "value2")
	
	output := mockLog.GetOutput()
	if !strings.Contains(output, "key1") || !strings.Contains(output, "value1") {
		t.Errorf("Expected output to contain key1 and value1, got: %s", output)
	}
}

func TestStructuredLogger_Sanitization(t *testing.T) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = true
	
	slogger := NewStructuredLogger(helper, config)
	
	// 测试敏感信息脱敏
	slogger.Infow("user login", "password", "secret123", "email", "test@example.com")
	
	output := mockLog.GetOutput()
	
	// 密码应该被脱敏
	if strings.Contains(output, "secret123") {
		t.Errorf("Password should be sanitized, but found in output: %s", output)
	}
	
	// 邮箱应该被脱敏
	if strings.Contains(output, "test@example.com") {
		t.Errorf("Email should be sanitized, but found in output: %s", output)
	}
}

func TestStructuredLogger_CustomRules(t *testing.T) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = true
	config.CustomRules["custom_field"] = AnonymizeRule{
		FieldName: "custom_field",
		KeepStart: 2,
		KeepEnd:   2,
		MaskChar:  "#",
	}
	
	slogger := NewStructuredLogger(helper, config)
	
	// 测试自定义脱敏规则
	slogger.Infow("custom test", "custom_field", "1234567890")
	
	output := mockLog.GetOutput()
	
	// 应该按照自定义规则脱敏
	if strings.Contains(output, "1234567890") {
		t.Errorf("Custom field should be sanitized, but found original value in output: %s", output)
	}
}

func TestStructuredLogger_AutoDetection(t *testing.T) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = true
	config.AutoDetect = true
	
	slogger := NewStructuredLogger(helper, config)
	
	// 测试自动检测敏感信息
	slogger.Infow("user info", "data", "My email is john@example.com and phone is 13812345678")
	
	output := mockLog.GetOutput()
	
	// 邮箱和手机号应该被自动检测并脱敏
	if strings.Contains(output, "john@example.com") {
		t.Errorf("Email should be auto-detected and sanitized: %s", output)
	}
	
	if strings.Contains(output, "13812345678") {
		t.Errorf("Phone should be auto-detected and sanitized: %s", output)
	}
}

func TestStructuredLogger_WithContext(t *testing.T) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = false
	
	slogger := NewStructuredLogger(helper, config)
	
	// 测试WithContext
	ctx := context.WithValue(context.Background(), "trace_id", "12345")
	contextLogger := slogger.WithContext(ctx)
	
	contextLogger.Infow("context test", "key", "value")
	
	output := mockLog.GetOutput()
	if !strings.Contains(output, "key") {
		t.Errorf("Expected output to contain key, got: %s", output)
	}
}

func TestStructuredLogger_WithFields(t *testing.T) {
	mockLog := newMockLogger()
	
	config := DefaultStructuredLogConfig()
	config.Enabled = false
	
	slogger := NewStructuredLoggerFromLogger(mockLog, config)
	
	// 测试WithFields
	fieldsLogger := slogger.WithFields("service", "test", "version", "1.0")
	
	fieldsLogger.Infow("fields test", "key", "value")
	
	output := mockLog.GetOutput()
	if !strings.Contains(output, "service") || !strings.Contains(output, "test") {
		t.Errorf("Expected output to contain service and test, got: %s", output)
	}
}

func TestStructuredLogger_CompatibilityMethods(t *testing.T) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = false
	
	slogger := NewStructuredLogger(helper, config)
	
	// 测试兼容性方法
	slogger.Info("info message")
	slogger.Infof("info message with format: %s", "test")
	slogger.Debug("debug message")
	slogger.Debugf("debug message with format: %s", "test")
	slogger.Error("error message")
	slogger.Errorf("error message with format: %s", "test")
	slogger.Warn("warn message")
	slogger.Warnf("warn message with format: %s", "test")
	
	output := mockLog.GetOutput()
	if len(output) == 0 {
		t.Error("Expected some output from compatibility methods")
	}
}

func TestStructuredLogger_TruncateValues(t *testing.T) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = true
	config.MaxValueLength = 10
	config.TruncateThreshold = 5
	
	slogger := NewStructuredLogger(helper, config)
	
	// 测试值截断
	longValue := strings.Repeat("a", 20)
	slogger.Infow("truncate test", "long_value", longValue)
	
	output := mockLog.GetOutput()
	
	// 长值应该被截断
	if strings.Contains(output, longValue) {
		t.Errorf("Long value should be truncated, but found full value in output: %s", output)
	}
	
	if !strings.Contains(output, "...") {
		t.Errorf("Expected truncation indicator (...) in output: %s", output)
	}
}

func TestStructuredLogger_UpdateRules(t *testing.T) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = true
	
	slogger := NewStructuredLogger(helper, config)
	
	// 测试更新规则
	newRules := map[string]AnonymizeRule{
		"new_field": {
			FieldName: "new_field",
			KeepStart: 1,
			KeepEnd:   1,
			MaskChar:  "X",
		},
	}
	
	slogger.UpdateRules(newRules)
	
	// 测试新规则是否生效
	slogger.Infow("rule test", "new_field", "123456")
	
	output := mockLog.GetOutput()
	
	// 应该按照新规则脱敏
	if strings.Contains(output, "123456") {
		t.Errorf("New field should be sanitized with new rule, but found original value: %s", output)
	}
}

func TestStructuredLogger_AddSensitiveKey(t *testing.T) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = true
	
	slogger := NewStructuredLogger(helper, config)
	
	// 添加新的敏感字段
	slogger.AddSensitiveKey("custom_sensitive")
	
	// 测试新敏感字段是否被脱敏
	slogger.Infow("sensitive test", "custom_sensitive", "sensitive_data")
	
	output := mockLog.GetOutput()
	
	// 新敏感字段应该被脱敏
	if strings.Contains(output, "sensitive_data") {
		t.Errorf("Custom sensitive field should be sanitized: %s", output)
	}
}

func BenchmarkStructuredLogger_Infow(b *testing.B) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = true
	
	slogger := NewStructuredLogger(helper, config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slogger.Infow("benchmark test", "key1", "value1", "key2", "value2", "password", "secret123")
	}
}

func BenchmarkStructuredLogger_InfowDisabled(b *testing.B) {
	mockLog := newMockLogger()
	helper := log.NewHelper(mockLog)
	
	config := DefaultStructuredLogConfig()
	config.Enabled = false
	
	slogger := NewStructuredLogger(helper, config)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		slogger.Infow("benchmark test", "key1", "value1", "key2", "value2", "password", "secret123")
	}
}