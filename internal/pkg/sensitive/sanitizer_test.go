package sensitive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestLogSanitizer_NewLogSanitizer 测试创建日志脱敏器
func TestLogSanitizer_NewLogSanitizer(t *testing.T) {
	sanitizer := NewLogSanitizer()
	assert.NotNil(t, sanitizer)
}

// TestLogSanitizer_SanitizeLogMessage 测试日志消息脱敏
func TestLogSanitizer_SanitizeLogMessage(t *testing.T) {
	sanitizer := NewLogSanitizer()

	tests := []struct {
		name     string
		message  string
		contains []string // 不应该包含的内容
	}{
		{
			name:     "email_sanitization",
			message:  "用户邮箱：test@example.com",
			contains: []string{"test@example.com"},
		},
		{
			name:     "phone_sanitization",
			message:  "手机号码：13812345678",
			contains: []string{"13812345678"},
		},
		{
			name:     "id_card_sanitization",
			message:  "身份证号：110101199001011234",
			contains: []string{"110101199001011234"},
		},
		{
			name:     "multiple_sensitive_data",
			message:  "用户信息：邮箱user@test.com，手机13812345678，身份证110101199001011234",
			contains: []string{"user@test.com", "13812345678", "110101199001011234"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeLogMessage(tt.message)
			for _, sensitive := range tt.contains {
				assert.NotContains(t, result, sensitive)
			}
		})
	}
}

// TestLogSanitizer_ContainsSensitiveInfo 测试敏感信息检测
func TestLogSanitizer_ContainsSensitiveInfo(t *testing.T) {
	sanitizer := NewLogSanitizer()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "contains_email",
			content:  "联系我们：support@example.com",
			expected: true,
		},
		{
			name:     "contains_phone",
			content:  "客服电话：13812345678",
			expected: true,
		},
		{
			name:     "contains_id_card",
			content:  "身份证：110101199001011234",
			expected: true,
		},
		{
			name:     "no_sensitive_info",
			content:  "这是一条普通的日志消息",
			expected: false,
		},
		{
			name:     "empty_content",
			content:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.ContainsSensitiveInfo(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestLogSanitizer_SanitizeLogContent 测试日志内容脱敏
func TestLogSanitizer_SanitizeLogContent(t *testing.T) {
	sanitizer := NewLogSanitizer()

	// 测试字符串
	result := sanitizer.SanitizeLogContent("用户邮箱：test@example.com")
	assert.NotContains(t, result, "test@example.com")

	// 测试结构体
	type User struct {
		Email string
		Phone string
		Name  string
	}

	user := User{
		Email: "user@example.com",
		Phone: "13812345678",
		Name:  "张三",
	}

	result = sanitizer.SanitizeLogContent(user)
	assert.NotContains(t, result, "user@example.com")
	assert.NotContains(t, result, "13812345678")
	assert.Contains(t, result, "张三") // 非敏感信息应该保留
}

// TestLogSanitizer_SanitizeLogContent_ComplexObjects 测试复杂对象的日志内容脱敏
func TestLogSanitizer_SanitizeLogContent_ComplexObjects(t *testing.T) {
	sanitizer := NewLogSanitizer()
	
	// 测试嵌套结构体
	type Address struct {
		Street string
		City   string
	}
	
	type User struct {
		Email   string
		Phone   string
		Address Address
	}
	
	user := User{
		Email: "test@example.com",
		Phone: "13812345678",
		Address: Address{
			Street: "某某街道123号",
			City:   "北京市",
		},
	}
	
	result := sanitizer.SanitizeLogContent(user)
	assert.NotEmpty(t, result)
	assert.NotContains(t, result, "test@example.com")
	assert.NotContains(t, result, "13812345678")
}

// TestLogSanitizer_SanitizeLogContent_Slice 测试切片的日志内容脱敏
func TestLogSanitizer_SanitizeLogContent_Slice(t *testing.T) {
	sanitizer := NewLogSanitizer()
	
	// 测试字符串切片
	emails := []string{"user1@example.com", "user2@example.com", "normal_text"}
	result := sanitizer.SanitizeLogContent(emails)
	assert.NotEmpty(t, result)
	assert.NotContains(t, result, "user1@example.com")
	assert.NotContains(t, result, "user2@example.com")
	assert.Contains(t, result, "normal_text")
	
	// 测试结构体切片
	type Contact struct {
		Email string
		Phone string
	}
	
	contacts := []Contact{
		{Email: "contact1@example.com", Phone: "13812345678"},
		{Email: "contact2@example.com", Phone: "15987654321"},
	}
	
	result = sanitizer.SanitizeLogContent(contacts)
	assert.NotEmpty(t, result)
	assert.NotContains(t, result, "contact1@example.com")
	assert.NotContains(t, result, "contact2@example.com")
}

// TestLogSanitizer_SanitizeLogContent_Map 测试映射的日志内容脱敏
func TestLogSanitizer_SanitizeLogContent_Map(t *testing.T) {
	sanitizer := NewLogSanitizer()
	
	// 测试字符串映射
	userData := map[string]interface{}{
		"email":    "user@example.com",
		"phone":    "13812345678",
		"name":     "张三",
		"age":      25,
		"address":  "北京市朝阳区某某街道123号",
	}
	
	result := sanitizer.SanitizeLogContent(userData)
	assert.NotEmpty(t, result)
	assert.NotContains(t, result, "user@example.com")
	assert.NotContains(t, result, "13812345678")
	assert.Contains(t, result, "25") // 非敏感数据应该保留
}

// TestLogSanitizer_SanitizeLogMessage_EdgeCases 测试日志消息脱敏的边界情况
func TestLogSanitizer_SanitizeLogMessage_EdgeCases(t *testing.T) {
	sanitizer := NewLogSanitizer()
	
	tests := []struct {
		name     string
		message  string
		expected bool // 是否应该被脱敏
	}{
		{
			name:     "empty_message",
			message:  "",
			expected: false,
		},
		{
			name:     "whitespace_only",
			message:  "   \n\t  ",
			expected: false,
		},
		{
			name:     "normal_message",
			message:  "这是一条普通的日志消息",
			expected: false,
		},
		{
			name:     "message_with_email",
			message:  "用户登录失败：test@example.com",
			expected: true,
		},
		{
			name:     "message_with_phone",
			message:  "发送短信到：13812345678",
			expected: true,
		},
		{
			name:     "message_with_multiple_sensitive",
			message:  "用户信息：邮箱user@test.com，手机13812345678",
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizer.SanitizeLogMessage(tt.message)
			
			if tt.expected {
				// 应该被脱敏，结果应该与原始消息不同
				assert.NotEqual(t, tt.message, result)
			} else {
				// 不应该被脱敏，结果应该与原始消息相同
				assert.Equal(t, tt.message, result)
			}
		})
	}
}

// TestLogSanitizer_ContainsSensitiveInfo_Performance 测试敏感信息检测的性能
func TestLogSanitizer_ContainsSensitiveInfo_Performance(t *testing.T) {
	sanitizer := NewLogSanitizer()
	
	// 生成大量文本
	largeText := ""
	for i := 0; i < 10000; i++ {
		largeText += "这是一段普通的文本内容，用于性能测试。"
	}
	
	// 在末尾添加敏感信息
	largeText += "联系邮箱：performance@test.com"
	
	start := time.Now()
	result := sanitizer.ContainsSensitiveInfo(largeText)
	duration := time.Since(start)
	
	assert.True(t, result)
	assert.Less(t, duration, 100*time.Millisecond) // 确保性能在可接受范围内
}

// TestLogSanitizer_SanitizeLogContent_SpecialTypes 测试特殊类型的日志内容脱敏
func TestLogSanitizer_SanitizeLogContent_SpecialTypes(t *testing.T) {
	sanitizer := NewLogSanitizer()
	
	// 测试nil值
	result := sanitizer.SanitizeLogContent(nil)
	assert.Equal(t, "", result)
	
	// 测试数字类型
	result = sanitizer.SanitizeLogContent(12345)
	assert.Contains(t, result, "12345")
	
	// 测试布尔类型
	result = sanitizer.SanitizeLogContent(true)
	assert.Contains(t, result, "true")
	
	// 测试浮点数
	result = sanitizer.SanitizeLogContent(123.456)
	assert.Contains(t, result, "123.456")
	
	// 测试指针
	value := "test"
	result = sanitizer.SanitizeLogContent(&value)
	assert.NotEmpty(t, result)
}