package sensitive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSensitiveDetector_DetectAll 测试检测所有敏感信息
func TestSensitiveDetector_DetectAll(t *testing.T) {
	detector := NewSensitiveDetector()
	
	tests := []struct {
		name     string
		text     string
		expected map[string][]string
	}{
		{
			name: "email_detection",
			text: "请联系我们：admin@example.com",
			expected: map[string][]string{
				"email": {"admin@example.com"},
			},
		},
		{
			name: "phone_detection",
			text: "电话号码：13812345678",
			expected: map[string][]string{
				"phone": {"13812345678"},
			},
		},
		{
			name: "multiple_types",
			text: "联系方式：邮箱test@example.com，手机13812345678",
			expected: map[string][]string{
				"email": {"test@example.com"},
				"phone": {"13812345678"},
			},
		},
		{
			name:     "no_sensitive_info",
			text:     "这是一段普通的文本",
			expected: map[string][]string{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectAll(tt.text)
			assert.Equal(t, len(tt.expected), len(result))
			
			for fieldType, expectedMatches := range tt.expected {
				actualMatches, exists := result[fieldType]
				assert.True(t, exists, "Expected field type %s not found", fieldType)
				assert.Equal(t, expectedMatches, actualMatches)
			}
		})
	}
}

// TestSensitiveDetector_DetectEmail 测试检测邮箱
func TestSensitiveDetector_DetectEmail(t *testing.T) {
	detector := NewSensitiveDetector()
	
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "single_email",
			text:     "邮箱：user@test.com",
			expected: []string{"user@test.com"},
		},
		{
			name:     "multiple_emails",
			text:     "联系方式：admin@test.com 或者 support@example.org",
			expected: []string{"admin@test.com", "support@example.org"},
		},
		{
			name:     "no_email",
			text:     "这是一段普通文本",
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectSensitiveInfo(tt.text, "email")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSensitiveDetector_DetectPhone 测试检测手机号
func TestSensitiveDetector_DetectPhone(t *testing.T) {
	detector := NewSensitiveDetector()
	
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "single_phone",
			text:     "联系电话：13812345678",
			expected: []string{"13812345678"},
		},
		{
			name:     "multiple_phones",
			text:     "电话：13812345678 或 15987654321",
			expected: []string{"13812345678", "15987654321"},
		},
		{
			name:     "no_phone",
			text:     "这是一段普通文本",
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectSensitiveInfo(tt.text, "phone")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSensitiveDetector_DetectIDCard 测试检测身份证号
func TestSensitiveDetector_DetectIDCard(t *testing.T) {
	detector := NewSensitiveDetector()
	
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "single_id_card",
			text:     "身份证：110101199001011234",
			expected: []string{"110101199001011234"},
		},
		{
			name:     "id_card_with_x",
			text:     "身份证号码：11010119900101123X",
			expected: []string{"11010119900101123X"},
		},
		{
			name:     "no_id_card",
			text:     "这是一段普通文本",
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectSensitiveInfo(tt.text, "idcard")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSensitiveDetector_DetectBankCard 测试检测银行卡号
func TestSensitiveDetector_DetectBankCard(t *testing.T) {
	detector := NewSensitiveDetector()
	
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "single_bank_card",
			text:     "银行卡：6222021234567890123",
			expected: []string{"6222021234567890123"},
		},
		{
			name:     "no_bank_card",
			text:     "这是一段普通文本",
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectSensitiveInfo(tt.text, "bankcard")
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSensitiveDetector_DetectAll_EdgeCases 测试检测所有敏感信息的边界情况
func TestSensitiveDetector_DetectAll_EdgeCases(t *testing.T) {
	detector := NewSensitiveDetector()
	
	// 测试空字符串
	result := detector.DetectAll("")
	assert.Empty(t, result)
	
	// 测试只有空格的字符串
	result = detector.DetectAll("   ")
	assert.Empty(t, result)
	
	// 测试包含多种敏感信息的复杂字符串
	complexText := `
		用户信息：
		姓名：张三
		邮箱：zhang.san@example.com
		手机：13812345678
		身份证：110101199001011234
		银行卡：6222021234567890123
		地址：北京市朝阳区某某街道123号
	`
	
	result = detector.DetectAll(complexText)
	assert.NotEmpty(t, result)
	
	// 验证检测到的敏感信息类型
	if emailMatches, exists := result["email"]; exists {
		assert.Contains(t, emailMatches, "zhang.san@example.com")
	}
	
	if phoneMatches, exists := result["phone"]; exists {
		assert.Contains(t, phoneMatches, "13812345678")
	}
	
	if idMatches, exists := result["idcard"]; exists {
		assert.Contains(t, idMatches, "110101199001011234")
	}
	
	if bankMatches, exists := result["bankcard"]; exists {
		assert.Contains(t, bankMatches, "6222021234567890123")
	}
}

// TestSensitiveDetector_DetectAll_Performance 测试检测所有敏感信息的性能
func TestSensitiveDetector_DetectAll_Performance(t *testing.T) {
	detector := NewSensitiveDetector()
	
	// 测试大量文本的检测性能
	largeText := ""
	for i := 0; i < 1000; i++ {
		largeText += "这是一段普通的文本，不包含敏感信息。"
	}
	
	// 在大量文本中插入敏感信息
	largeText += "联系邮箱：test@example.com 手机：13812345678"
	
	start := time.Now()
	result := detector.DetectAll(largeText)
	duration := time.Since(start)
	
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "email")
	assert.Contains(t, result, "phone")
	assert.Less(t, duration, time.Second) // 确保性能在可接受范围内
}

// TestSensitiveDetector_DetectAll_ComplexText 测试检测复杂文本中的所有敏感信息
func TestSensitiveDetector_DetectAll_ComplexText(t *testing.T) {
	detector := NewSensitiveDetector()
	
	complexText := `
		用户信息：
		姓名：张三
		邮箱：zhangsan@example.com
		手机：13812345678
		身份证：110101199001011234
		银行卡：6222021234567890123
		备用邮箱：backup@test.com
		备用手机：15987654321
	`
	
	result := detector.DetectAll(complexText)
	
	// 验证检测到的邮箱
	assert.Contains(t, result, "email")
	assert.Contains(t, result["email"], "zhangsan@example.com")
	assert.Contains(t, result["email"], "backup@test.com")
	
	// 验证检测到的手机号
	assert.Contains(t, result, "phone")
	assert.Contains(t, result["phone"], "13812345678")
	assert.Contains(t, result["phone"], "15987654321")
	
	// 验证检测到的身份证
	assert.Contains(t, result, "id_card")
	assert.Contains(t, result["id_card"], "110101199001011234")
	
	// 验证检测到的银行卡
	assert.Contains(t, result, "bank_card")
	assert.Contains(t, result["bank_card"], "6222021234567890123")
}