package sensitive

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEmailRule 测试邮箱脱敏规则
func TestEmailRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid email", "test@example.com", "te**@example.com"},
		{"short username", "a@example.com", "a@example.com"},
		{"two char username", "ab@example.com", "ab@example.com"},
		{"long username", "verylongusername@example.com", "ve**************@example.com"},
		{"invalid email", "invalid-email", "invalid-email"},
		{"empty string", "", ""},
		{"no @ symbol", "testexample.com", "testexample.com"},
		{"multiple @ symbols", "test@example@com", "test@example@com"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := EmailRule.CustomFunc(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestPhoneRule 测试手机号脱敏规则
func TestPhoneRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid phone", "13812345678", "138****5678"},
		{"another valid phone", "15987654321", "159****4321"},
		{"invalid phone - too short", "1381234567", "1381234567"},
		{"invalid phone - too long", "138123456789", "138123456789"},
		{"invalid phone - wrong prefix", "12812345678", "12812345678"},
		{"empty string", "", ""},
		{"non-numeric", "13a12345678", "13a12345678"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := PhoneRule.CustomFunc(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestNameRule 测试姓名脱敏规则
func TestNameRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single char", "张", "张"},
		{"two chars", "张三", "张*"},
		{"three chars", "张三丰", "张*丰"},
		{"four chars", "欧阳修文", "欧**文"},
		{"english name", "John", "J**n"},
		{"long name", "司马相如", "司**如"},
		{"empty string", "", ""},
		{"whitespace", "  ", "  "},
		{"name with spaces", " 张三 ", "张*"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := NameRule.CustomFunc(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestIDCardRule 测试身份证号脱敏规则
func TestIDCardRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid 18-digit ID", "110101199001011234", "110101********1234"},
		{"valid ID with X", "11010119900101123X", "110101********123X"},
		{"invalid ID - too short", "11010119900101123", "11010119900101123"},
		{"invalid ID - too long", "1101011990010112345", "1101011990010112345"},
		{"invalid ID - wrong format", "000000199001011234", "000000199001011234"},
		{"empty string", "", ""},
		{"non-numeric", "11010119900101123a", "11010119900101123a"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IDCardRule.CustomFunc(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestBankCardRule 测试银行卡号脱敏规则
func TestBankCardRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid 16-digit card", "6222021234567890", "6222********7890"},
		{"valid 19-digit card", "6222021234567890123", "6222***********0123"},
		{"short card number", "1234567", "1234567"},
		{"invalid card - starts with 0", "0222021234567890", "0222021234567890"},
		{"empty string", "", ""},
		{"non-numeric", "622202123456789a", "622202123456789a"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := BankCardRule.CustomFunc(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestAddressRule 测试地址脱敏规则
func TestAddressRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal address", "北京市朝阳区建国门外大街1号", "北京市朝阳区********"},
		{"short address", "北京市", "北京市"},
		{"six char address", "北京市朝阳区", "北京市朝阳区"},
		{"long address", "广东省深圳市南山区科技园南区深南大道10000号", "广东省深圳市******************"},
		{"empty string", "", ""},
		{"whitespace", "  ", "  "},
		{"address with spaces", " 北京市朝阳区建国门外大街1号 ", "北京市朝阳区********"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := AddressRule.CustomFunc(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestGetDefaultRules 测试获取默认规则
func TestGetDefaultRules(t *testing.T) {
	rules := GetDefaultRules()
	
	assert.NotEmpty(t, rules)
	assert.Contains(t, rules, "email")
	assert.Contains(t, rules, "phone")
	assert.Contains(t, rules, "name")
	assert.Contains(t, rules, "id_card")
	assert.Contains(t, rules, "bank_card")
	assert.Contains(t, rules, "address")
	
	// 验证规则内容
	emailRule := rules["email"]
	assert.Equal(t, "email", emailRule.FieldName)
	assert.Equal(t, 2, emailRule.KeepStart)
	assert.Equal(t, 4, emailRule.KeepEnd)
	assert.Equal(t, "*", emailRule.MaskChar)
	assert.NotNil(t, emailRule.CustomFunc)
}

// TestGetRuleByFieldName 测试根据字段名获取规则
func TestGetRuleByFieldName(t *testing.T) {
	tests := []struct {
		fieldName string
		expected  bool
	}{
		{"email", true},
		{"EMAIL", true}, // 测试大小写不敏感
		{"Email", true},
		{"phone", true},
		{"name", true},
		{"id_card", true},
		{"bank_card", true},
		{"address", true},
		{"unknown_field", false},
		{"", false},
	}

	for _, test := range tests {
		t.Run(test.fieldName, func(t *testing.T) {
			rule, exists := GetRuleByFieldName(test.fieldName)
			assert.Equal(t, test.expected, exists)
			if exists {
				assert.NotEmpty(t, rule.FieldName)
			}
		})
	}
}

// TestValidationFunctions 测试验证函数
func TestValidationFunctions(t *testing.T) {
	// 测试邮箱验证
	emailTests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name@domain.co.uk", true},
		{"user+tag@example.org", true},
		{"invalid-email", false},
		{"@example.com", false},
		{"test@", false},
		{"", false},
	}

	for _, test := range emailTests {
		t.Run("email_"+test.email, func(t *testing.T) {
			result := isValidEmail(test.email)
			assert.Equal(t, test.valid, result)
		})
	}

	// 测试手机号验证
	phoneTests := []struct {
		phone string
		valid bool
	}{
		{"13812345678", true},
		{"15987654321", true},
		{"19876543210", true},
		{"12812345678", false}, // 不是1[3-9]开头
		{"1381234567", false},  // 长度不对
		{"138123456789", false}, // 长度不对
		{"", false},
	}

	for _, test := range phoneTests {
		t.Run("phone_"+test.phone, func(t *testing.T) {
			result := isValidPhone(test.phone)
			assert.Equal(t, test.valid, result)
		})
	}

	// 测试身份证号验证
	idCardTests := []struct {
		idCard string
		valid  bool
	}{
		{"110101199001011234", true},
		{"11010119900101123X", true},
		{"11010119900101123x", true},
		{"000000199001011234", false}, // 不能以0开头
		{"11010119900101123", false},  // 长度不对
		{"11010119900101123a", false}, // 最后一位只能是数字或X/x
		{"", false},
	}

	for _, test := range idCardTests {
		t.Run("idcard_"+test.idCard, func(t *testing.T) {
			result := isValidIDCard(test.idCard)
			assert.Equal(t, test.valid, result)
		})
	}

	// 测试银行卡号验证
	bankCardTests := []struct {
		bankCard string
		valid    bool
	}{
		{"6222021234567890", true},
		{"6222021234567890123", true},
		{"123456789012", true}, // 12位最短
		{"12345678901234567890", true}, // 20位最长
		{"0222021234567890", false}, // 不能以0开头
		{"12345678901", false},      // 少于12位
		{"123456789012345678901", false}, // 超过20位
		{"", false},
	}

	for _, test := range bankCardTests {
		t.Run("bankcard_"+test.bankCard, func(t *testing.T) {
			result := isValidBankCard(test.bankCard)
			assert.Equal(t, test.valid, result)
		})
	}
}

// TestAnonymizeWithRule 测试通用脱敏规则
func TestAnonymizeWithRule(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		rule     AnonymizeRule
		expected string
	}{
		{
			name:  "normal case",
			value: "1234567890",
			rule:  AnonymizeRule{KeepStart: 3, KeepEnd: 3, MaskChar: "*"},
			expected: "123****890",
		},
		{
			name:  "short value",
			value: "12345",
			rule:  AnonymizeRule{KeepStart: 3, KeepEnd: 3, MaskChar: "*"},
			expected: "12345", // 长度不够，不脱敏
		},
		{
			name:  "custom mask char",
			value: "abcdefghij",
			rule:  AnonymizeRule{KeepStart: 2, KeepEnd: 2, MaskChar: "#"},
			expected: "ab######ij",
		},
		{
			name:  "empty mask char",
			value: "abcdefghij",
			rule:  AnonymizeRule{KeepStart: 2, KeepEnd: 2, MaskChar: ""},
			expected: "ab******ij", // 默认使用*
		},
		{
			name:  "zero keep start",
			value: "1234567890",
			rule:  AnonymizeRule{KeepStart: 0, KeepEnd: 3, MaskChar: "*"},
			expected: "*******890",
		},
		{
			name:  "zero keep end",
			value: "1234567890",
			rule:  AnonymizeRule{KeepStart: 3, KeepEnd: 0, MaskChar: "*"},
			expected: "123*******",
		},
		{
			name:  "unicode characters",
			value: "你好世界测试",
			rule:  AnonymizeRule{KeepStart: 2, KeepEnd: 2, MaskChar: "*"},
			expected: "你好**测试",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := anonymizeWithRule(test.value, test.rule)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestCreateCustomRule 测试创建自定义规则
func TestCreateCustomRule(t *testing.T) {
	customFunc := func(value string) string {
		return "[CUSTOM]" + value
	}
	
	rule := CreateCustomRule("custom_field", 1, 2, "#", customFunc)
	
	assert.Equal(t, "custom_field", rule.FieldName)
	assert.Equal(t, 1, rule.KeepStart)
	assert.Equal(t, 2, rule.KeepEnd)
	assert.Equal(t, "#", rule.MaskChar)
	assert.NotNil(t, rule.CustomFunc)
	
	// 测试自定义函数
	result := rule.CustomFunc("test")
	assert.Equal(t, "[CUSTOM]test", result)
}

// TestMergeRules 测试合并规则
func TestMergeRules(t *testing.T) {
	rules1 := map[string]AnonymizeRule{
		"field1": {FieldName: "field1", KeepStart: 1, KeepEnd: 1},
		"field2": {FieldName: "field2", KeepStart: 2, KeepEnd: 2},
	}
	
	rules2 := map[string]AnonymizeRule{
		"field2": {FieldName: "field2_updated", KeepStart: 3, KeepEnd: 3}, // 覆盖field2
		"field3": {FieldName: "field3", KeepStart: 4, KeepEnd: 4},
	}
	
	merged := MergeRules(rules1, rules2)
	
	assert.Len(t, merged, 3)
	assert.Contains(t, merged, "field1")
	assert.Contains(t, merged, "field2")
	assert.Contains(t, merged, "field3")
	
	// 验证field2被覆盖
	assert.Equal(t, "field2_updated", merged["field2"].FieldName)
	assert.Equal(t, 3, merged["field2"].KeepStart)
	
	// 验证其他字段正确
	assert.Equal(t, "field1", merged["field1"].FieldName)
	assert.Equal(t, "field3", merged["field3"].FieldName)
}

// TestMergeRules_EmptyInput 测试合并空规则
func TestMergeRules_EmptyInput(t *testing.T) {
	// 测试空输入
	merged := MergeRules()
	assert.Empty(t, merged)
	
	// 测试单个规则集
	rules := map[string]AnonymizeRule{
		"field1": {FieldName: "field1"},
	}
	merged = MergeRules(rules)
	assert.Len(t, merged, 1)
	assert.Contains(t, merged, "field1")
	
	// 测试包含空规则集
	emptyRules := map[string]AnonymizeRule{}
	merged = MergeRules(rules, emptyRules)
	assert.Len(t, merged, 1)
	assert.Contains(t, merged, "field1")
}

// TestRuleEdgeCases 测试规则边界情况
func TestRuleEdgeCases(t *testing.T) {
	// 测试空字符串
	result := EmailRule.CustomFunc("")
	assert.Equal(t, "", result)
	
	result = PhoneRule.CustomFunc("")
	assert.Equal(t, "", result)
	
	result = NameRule.CustomFunc("")
	assert.Equal(t, "", result)
	
	result = IDCardRule.CustomFunc("")
	assert.Equal(t, "", result)
	
	result = BankCardRule.CustomFunc("")
	assert.Equal(t, "", result)
	
	result = AddressRule.CustomFunc("")
	assert.Equal(t, "", result)
	
	// 测试特殊字符
	specialChars := "!@#$%^&*()"
	result = NameRule.CustomFunc(specialChars)
	assert.NotEmpty(t, result)
	
	// 测试非常长的字符串
	longString := strings.Repeat("a", 1000)
	result = NameRule.CustomFunc(longString)
	assert.NotEmpty(t, result)
	assert.True(t, len(result) <= len(longString))
}