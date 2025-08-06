package sensitive

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockMakeSensitive 实现MakeSensitive接口的模拟结构体
type mockMakeSensitive struct {
	value string
}

func (m mockMakeSensitive) GetSensitiveFields() []string {
	return []string{"value"}
}

func (m mockMakeSensitive) Anonymize() interface{} {
	return map[string]string{"value": "[SENSITIVE]" + m.value}
}

func (m mockMakeSensitive) GetAnonymizeRules() map[string]AnonymizeRule {
	return map[string]AnonymizeRule{
		"value": {FieldName: "value", KeepStart: 1, KeepEnd: 1, MaskChar: "*"},
	}
}

func (m mockMakeSensitive) MakeSensitive() string {
	return m.value
}

// mockLogSafeStringer 实现LogSafeStringer接口的模拟结构体
type mockLogSafeStringer struct {
	value string
}

func (m mockLogSafeStringer) LogSafeString() string {
	return "[SAFE]" + m.value
}

// TestNewAnonymizer 测试创建匿名化器
func TestNewAnonymizer(t *testing.T) {
	anonymizer := NewAnonymizer()
	assert.NotNil(t, anonymizer)
}

// TestAnonymizer_AnonymizeString 测试字符串脱敏
func TestAnonymizer_AnonymizeString(t *testing.T) {
	anonymizer := NewAnonymizer()
	
	tests := []struct {
		name     string
		value    string
		rule     AnonymizeRule
		expected string
	}{
		{"email rule", "test@example.com", EmailRule, "te**@example.com"},
		{"phone rule", "13812345678", PhoneRule, "138****5678"},
		{"name rule", "张三丰", NameRule, "张*丰"},
		{"custom rule", "sensitive_data", AnonymizeRule{KeepStart: 3, KeepEnd: 2, MaskChar: "*"}, "sen*********ta"},
		{"empty value", "", EmailRule, ""},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := anonymizer.AnonymizeString(test.value, test.rule)
			assert.Equal(t, test.expected, result)
		})
	}
}

// TestAnonymizer_AnonymizeObject 测试对象脱敏
func TestAnonymizer_AnonymizeObject(t *testing.T) {
	anonymizer := NewAnonymizer()
	
	// 测试MakeSensitive接口
	mockObj := mockMakeSensitive{value: "secret"}
	result := anonymizer.AnonymizeObject(mockObj)
	assert.NotNil(t, result)
	
	// 验证脱敏结果
	resultMap, ok := result.(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "[SENSITIVE]secret", resultMap["value"])
	
	// 测试nil对象
	result = anonymizer.AnonymizeObject(nil)
	assert.Nil(t, result)
}

// TestAnonymizer_AnonymizeBatch 测试批量脱敏
func TestAnonymizer_AnonymizeBatch(t *testing.T) {
	anonymizer := NewAnonymizer()
	
	// 创建测试对象
	obj1 := mockMakeSensitive{value: "secret1"}
	obj2 := mockMakeSensitive{value: "secret2"}
	objects := []MakeSensitive{obj1, obj2}
	
	results := anonymizer.AnonymizeBatch(objects)
	assert.Len(t, results, 2)
	
	// 验证第一个结果
	result1, ok := results[0].(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "[SENSITIVE]secret1", result1["value"])
	
	// 验证第二个结果
	result2, ok := results[1].(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "[SENSITIVE]secret2", result2["value"])
}

// TestAnonymizer_AnonymizeValue 测试任意值脱敏
func TestAnonymizer_AnonymizeValue(t *testing.T) {
	anonymizer := NewAnonymizer()
	rules := GetDefaultRules()
	
	tests := []struct {
		name     string
		value    interface{}
		expected interface{}
	}{
		{"string value", "test_string", "test_string"}, // 没有敏感信息，保持不变
		{"int value", 123, 123},
		{"bool value", true, true},
		{"float value", 3.14, 3.14},
		{"nil value", nil, nil},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := anonymizer.AnonymizeValue(test.value, rules)
			assert.Equal(t, test.expected, result)
		})
	}
	
	// 测试MakeSensitive接口
	mockSensitive := mockMakeSensitive{value: "secret"}
	result := anonymizer.AnonymizeValue(mockSensitive, rules)
	resultMap, ok := result.(map[string]string)
	assert.True(t, ok)
	assert.Equal(t, "[SENSITIVE]secret", resultMap["value"])
}

// TestNewFieldAnonymizer 测试创建字段匿名化器
func TestNewFieldAnonymizer(t *testing.T) {
	fieldAnonymizer := NewFieldAnonymizer()
	assert.NotNil(t, fieldAnonymizer)
}

// TestFieldAnonymizer_Methods 测试字段脱敏器方法
func TestFieldAnonymizer_Methods(t *testing.T) {
	fieldAnonymizer := NewFieldAnonymizer()
	
	// 测试邮箱脱敏
	result := fieldAnonymizer.AnonymizeEmail("test@example.com")
	assert.Equal(t, "te**@example.com", result)
	
	// 测试手机号脱敏
	result = fieldAnonymizer.AnonymizePhone("13812345678")
	assert.Equal(t, "138****5678", result)
	
	// 测试姓名脱敏
	result = fieldAnonymizer.AnonymizeName("张三丰")
	assert.Equal(t, "张*丰", result)
	
	// 测试身份证脱敏
	result = fieldAnonymizer.AnonymizeIDCard("110101199001011234")
	assert.Equal(t, "110101********1234", result)
	
	// 测试银行卡脱敏
	result = fieldAnonymizer.AnonymizeBankCard("6222021234567890")
	assert.Equal(t, "6222********7890", result)
	
	// 测试地址脱敏
	result = fieldAnonymizer.AnonymizeAddress("北京市朝阳区建国门外大街1号")
	assert.Equal(t, "北京市朝阳区********", result)
}

// TestNewLogSanitizer 测试创建日志脱敏器
func TestNewLogSanitizer(t *testing.T) {
	logSanitizer := NewLogSanitizer()
	assert.NotNil(t, logSanitizer)
}

// TestLogSanitizer_Methods 测试日志脱敏器方法
func TestLogSanitizer_Methods(t *testing.T) {
	logSanitizer := NewLogSanitizer()
	
	// 测试SanitizeLogContent
	result := logSanitizer.SanitizeLogContent("User email: test@example.com")
	assert.Contains(t, result, "te**@example.com")
	
	// 测试SanitizeLogMessage
	result = logSanitizer.SanitizeLogMessage("Contact: 13812345678")
	assert.Contains(t, result, "138****5678")
	
	// 测试ContainsSensitiveInfo
	hasSensitive := logSanitizer.ContainsSensitiveInfo("Email: test@example.com")
	assert.True(t, hasSensitive)
	
	// 测试LogSafeStringer接口
	mockLogSafe := mockLogSafeStringer{value: "test"}
	result = logSanitizer.SanitizeLogContent(mockLogSafe)
	assert.Equal(t, "[SAFE]test", result)
	
	// 测试nil内容
	result = logSanitizer.SanitizeLogContent(nil)
	assert.Equal(t, "", result)
	
	// 测试空字符串
	result = logSanitizer.SanitizeLogMessage("")
	assert.Equal(t, "", result)
	
	// 测试不包含敏感信息
	hasSensitive = logSanitizer.ContainsSensitiveInfo("Normal message")
	assert.False(t, hasSensitive)
	
	hasSensitive = logSanitizer.ContainsSensitiveInfo("Normal log message")
	assert.False(t, hasSensitive)
}

// TestStruct 测试结构体
type TestStruct struct {
	Email    string
	Phone    string
	Name     string
	Age      int
	private  string // 未导出字段
}

// TestAnonymizer_AnonymizeStruct 测试结构体脱敏
func TestAnonymizer_AnonymizeStruct(t *testing.T) {
	anonymizer := NewAnonymizer()
	rules := GetDefaultRules()
	
	// 创建测试结构体
	testStruct := TestStruct{
		Email:   "test@example.com",
		Phone:   "13812345678",
		Name:    "张三",
		Age:     25,
		private: "private_data",
	}
	
	// 测试结构体脱敏
	result := anonymizer.AnonymizeValue(testStruct, rules)
	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok)
	
	// 验证脱敏结果
	assert.Equal(t, "te**@example.com", resultMap["Email"])
	assert.Equal(t, "138****5678", resultMap["Phone"])
	assert.Equal(t, "张*", resultMap["Name"])
	assert.Equal(t, 25, resultMap["Age"])
	
	// 验证私有字段被跳过
	_, exists := resultMap["private"]
	assert.False(t, exists)
}

// TestAnonymizer_AnonymizeSlice 测试切片脱敏
func TestAnonymizer_AnonymizeSlice(t *testing.T) {
	anonymizer := NewAnonymizer()
	rules := GetDefaultRules()
	
	// 测试字符串切片
	emails := []string{"test1@example.com", "test2@example.com", "normal_string"}
	result := anonymizer.AnonymizeValue(emails, rules)
	resultSlice, ok := result.([]interface{})
	assert.True(t, ok)
	assert.Len(t, resultSlice, 3)
	
	// 验证脱敏结果
	assert.Equal(t, "te***@example.com", resultSlice[0])
	assert.Equal(t, "te***@example.com", resultSlice[1])
	assert.Equal(t, "normal_string", resultSlice[2])
	
	// 测试结构体切片
	structs := []TestStruct{
		{Email: "user1@test.com", Phone: "13800000001"},
		{Email: "user2@test.com", Phone: "13800000002"},
	}
	result = anonymizer.AnonymizeValue(structs, rules)
	resultSlice, ok = result.([]interface{})
	assert.True(t, ok)
	assert.Len(t, resultSlice, 2)
	
	// 验证第一个结构体
	firstStruct, ok := resultSlice[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "us***@test.com", firstStruct["Email"])
	assert.Equal(t, "138****0001", firstStruct["Phone"])
	
	// 测试空切片
	emptySlice := []string{}
	result = anonymizer.AnonymizeValue(emptySlice, rules)
	resultSlice, ok = result.([]interface{})
	assert.True(t, ok)
	assert.Len(t, resultSlice, 0)
}

// TestAnonymizer_AnonymizeMap 测试映射脱敏
func TestAnonymizer_AnonymizeMap(t *testing.T) {
	anonymizer := NewAnonymizer()
	rules := GetDefaultRules()
	
	// 测试字符串映射
	testMap := map[string]string{
		"email1": "test1@example.com",
		"email2": "test2@example.com",
		"normal": "normal_value",
	}
	
	result := anonymizer.AnonymizeValue(testMap, rules)
	resultMap, ok := result.(map[interface{}]interface{})
	assert.True(t, ok)
	
	// 验证脱敏结果
	assert.Equal(t, "te***@example.com", resultMap["email1"])
	assert.Equal(t, "te***@example.com", resultMap["email2"])
	assert.Equal(t, "normal_value", resultMap["normal"])
	
	// 测试空映射
	emptyMap := map[string]string{}
	result = anonymizer.AnonymizeValue(emptyMap, rules)
	resultMap, ok = result.(map[interface{}]interface{})
	assert.True(t, ok)
	assert.Len(t, resultMap, 0)
}

// TestAnonymizer_AnonymizeNestedStruct 测试嵌套结构体脱敏
func TestAnonymizer_AnonymizeNestedStruct(t *testing.T) {
	anonymizer := NewAnonymizer()
	rules := GetDefaultRules()
	
	// 定义嵌套结构体
	type Address struct {
		Street string
		City   string
	}
	
	type User struct {
		Email   string
		Phone   string
		Address Address
	}
	
	// 创建测试数据
	user := User{
		Email: "user@test.com",
		Phone: "13812345678",
		Address: Address{
			Street: "Main Street 123",
			City:   "Beijing",
		},
	}
	
	// 测试嵌套结构体脱敏
	result := anonymizer.AnonymizeValue(user, rules)
	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok)
	
	// 验证顶层字段
	assert.Equal(t, "us**@test.com", resultMap["Email"])
	assert.Equal(t, "138****5678", resultMap["Phone"])
	
	// 验证嵌套结构体
	addressMap, ok := resultMap["Address"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "Main Street 123", addressMap["Street"])
	assert.Equal(t, "Beijing", addressMap["City"])
}

// TestNewSensitiveDetector 测试创建敏感信息检测器
func TestNewSensitiveDetector(t *testing.T) {
	detector := NewSensitiveDetector()
	assert.NotNil(t, detector)
}

// TestSensitiveDetector_Methods 测试敏感信息检测器方法
func TestSensitiveDetector_Methods(t *testing.T) {
	detector := NewSensitiveDetector()
	
	// 测试邮箱检测
	emails := detector.DetectSensitiveInfo("Contact us at test@example.com for help", "email")
	assert.Len(t, emails, 1)
	assert.Equal(t, "test@example.com", emails[0])
	
	// 测试手机号检测
	phones := detector.DetectSensitiveInfo("Call me at 13812345678", "phone")
	assert.Len(t, phones, 1)
	assert.Equal(t, "13812345678", phones[0])
	
	// 测试身份证检测
	idCards := detector.DetectSensitiveInfo("My ID is 110101199001011234", "idcard")
	assert.Len(t, idCards, 1)
	assert.Equal(t, "110101199001011234", idCards[0])
	
	// 测试银行卡检测
	bankCards := detector.DetectSensitiveInfo("Card number: 6222021234567890", "bankcard")
	assert.Len(t, bankCards, 1)
	assert.Equal(t, "6222021234567890", bankCards[0])
	
	// 测试HasSensitiveInfo方法
	assert.True(t, detector.HasSensitiveInfo("test@example.com", "email"))
	assert.False(t, detector.HasSensitiveInfo("normal text", "email"))
	
	// 测试检测所有敏感信息
	allDetected := detector.DetectAll("Email: test@example.com, Phone: 13812345678")
	assert.Contains(t, allDetected, "email")
	assert.Contains(t, allDetected, "phone")
	assert.Len(t, allDetected["email"], 1)
	assert.Len(t, allDetected["phone"], 1)
	
	// 测试无敏感信息
	allDetected = detector.DetectAll("This is a normal message")
	assert.Empty(t, allDetected)
}

// TestAnonymizer_ConcurrentAccess 测试并发访问
func TestAnonymizer_ConcurrentAccess(t *testing.T) {
	anonymizer := NewAnonymizer()
	var wg sync.WaitGroup
	results := make([]string, 10)
	
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			email := fmt.Sprintf("user%d@example.com", index)
			results[index] = anonymizer.AnonymizeString(email, EmailRule)
		}(i)
	}
	
	wg.Wait()
	
	// 验证所有结果都正确脱敏
	for i, result := range results {
		assert.Contains(t, result, "@example.com")
		assert.Contains(t, result, "**")
		t.Logf("Result %d: %s", i, result)
	}
}

// TestAnonymizer_AnonymizePointer 测试指针脱敏
func TestAnonymizer_AnonymizePointer(t *testing.T) {
	anonymizer := NewAnonymizer()
	rules := GetDefaultRules()
	
	// 测试nil指针
	var nilPtr *TestStruct
	result := anonymizer.AnonymizeValue(nilPtr, rules)
	assert.Nil(t, result)
	
	// 测试非nil指针
	testStruct := &TestStruct{
		Email: "test@example.com",
		Phone: "13812345678",
		Name:  "张三",
		Age:   25,
	}
	
	result = anonymizer.AnonymizeValue(testStruct, rules)
	assert.NotNil(t, result)
	
	resultMap, ok := result.(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, resultMap, "Email")
	assert.Contains(t, resultMap, "Phone")
	assert.Contains(t, resultMap, "Name")
	assert.Contains(t, resultMap, "Age")
}

// TestAnonymizer_EdgeCases 测试边界情况
func TestAnonymizer_EdgeCases(t *testing.T) {
	anonymizer := NewAnonymizer()
	rules := GetDefaultRules()
	
	// 测试空值
	result := anonymizer.AnonymizeValue(nil, rules)
	assert.Nil(t, result)
	
	// 测试基本类型
	intValue := 123
	result = anonymizer.AnonymizeValue(intValue, rules)
	assert.Equal(t, intValue, result)
	
	floatValue := 123.45
	result = anonymizer.AnonymizeValue(floatValue, rules)
	assert.Equal(t, floatValue, result)
	
	boolValue := true
	result = anonymizer.AnonymizeValue(boolValue, rules)
	assert.Equal(t, boolValue, result)
	
	// 测试空字符串
	emptyString := ""
	result = anonymizer.AnonymizeString(emptyString, EmailRule)
	assert.Equal(t, emptyString, result)
}

// TestFieldAnonymizer_AllMethods 测试字段脱敏器所有方法
func TestFieldAnonymizer_AllMethods(t *testing.T) {
	fieldAnonymizer := NewFieldAnonymizer()
	
	// 测试邮箱脱敏
	email := "test@example.com"
	anonymizedEmail := fieldAnonymizer.AnonymizeEmail(email)
	assert.NotEqual(t, email, anonymizedEmail)
	assert.Contains(t, anonymizedEmail, "*")
	
	// 测试手机号脱敏
	phone := "13812345678"
	anonymizedPhone := fieldAnonymizer.AnonymizePhone(phone)
	assert.NotEqual(t, phone, anonymizedPhone)
	assert.Contains(t, anonymizedPhone, "*")
	
	// 测试姓名脱敏
	name := "张三"
	anonymizedName := fieldAnonymizer.AnonymizeName(name)
	assert.NotEqual(t, name, anonymizedName)
	assert.Contains(t, anonymizedName, "*")
	
	// 测试地址脱敏
	address := "北京市朝阳区某某街道123号"
	anonymizedAddress := fieldAnonymizer.AnonymizeAddress(address)
	assert.NotEqual(t, address, anonymizedAddress)
	assert.Contains(t, anonymizedAddress, "*")
}