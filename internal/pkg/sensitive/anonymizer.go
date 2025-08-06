package sensitive

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// anonymizer 脱敏处理器实现
type anonymizer struct {
	defaultRules map[string]AnonymizeRule
	detector     SensitiveDetector
}

// NewAnonymizer 创建新的脱敏处理器
func NewAnonymizer() Anonymizer {
	return &anonymizer{
		defaultRules: GetDefaultRules(),
		detector:     NewSensitiveDetector(),
	}
}

// AnonymizeString 脱敏字符串
func (a *anonymizer) AnonymizeString(value string, rule AnonymizeRule) string {
	if value == "" {
		return value
	}
	
	// 如果有自定义函数，优先使用
	if rule.CustomFunc != nil {
		return rule.CustomFunc(value)
	}
	
	// 使用通用规则
	return anonymizeWithRule(value, rule)
}

// AnonymizeObject 脱敏对象
func (a *anonymizer) AnonymizeObject(obj MakeSensitive) interface{} {
	if obj == nil {
		return nil
	}
	
	return obj.Anonymize()
}

// AnonymizeBatch 批量脱敏
func (a *anonymizer) AnonymizeBatch(objects []MakeSensitive) []interface{} {
	result := make([]interface{}, len(objects))
	
	for i, obj := range objects {
		result[i] = a.AnonymizeObject(obj)
	}
	
	return result
}

// AnonymizeValue 脱敏任意值
func (a *anonymizer) AnonymizeValue(value interface{}, rules map[string]AnonymizeRule) interface{} {
	if value == nil {
		return nil
	}
	
	// 如果实现了MakeSensitive接口，直接调用
	if sensitive, ok := value.(MakeSensitive); ok {
		return a.AnonymizeObject(sensitive)
	}
	
	// 使用反射处理结构体
	return a.anonymizeValueByReflection(value, rules)
}

// anonymizeValueByReflection 使用反射脱敏值
func (a *anonymizer) anonymizeValueByReflection(value interface{}, rules map[string]AnonymizeRule) interface{} {
	v := reflect.ValueOf(value)
	t := reflect.TypeOf(value)
	
	// 处理指针
	if t.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
		t = t.Elem()
	}
	
	switch t.Kind() {
	case reflect.Struct:
		return a.anonymizeStruct(v, t, rules)
	case reflect.Slice, reflect.Array:
		return a.anonymizeSlice(v, rules)
	case reflect.Map:
		return a.anonymizeMap(v, rules)
	case reflect.String:
		return a.anonymizeStringValue(v.String(), rules)
	default:
		return value
	}
}

// anonymizeStruct 脱敏结构体
func (a *anonymizer) anonymizeStruct(v reflect.Value, t reflect.Type, rules map[string]AnonymizeRule) interface{} {
	result := make(map[string]interface{})
	
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		
		// 跳过未导出的字段
		if !field.CanInterface() {
			continue
		}
		
		fieldName := strings.ToLower(fieldType.Name)
		fieldValue := field.Interface()
		
		// 检查是否有对应的脱敏规则
		if rule, exists := rules[fieldName]; exists {
			if field.Kind() == reflect.String {
				result[fieldType.Name] = a.AnonymizeString(field.String(), rule)
			} else {
				result[fieldType.Name] = a.AnonymizeValue(fieldValue, rules)
			}
		} else {
			// 递归处理嵌套结构
			result[fieldType.Name] = a.AnonymizeValue(fieldValue, rules)
		}
	}
	
	return result
}

// anonymizeSlice 脱敏切片
func (a *anonymizer) anonymizeSlice(v reflect.Value, rules map[string]AnonymizeRule) interface{} {
	result := make([]interface{}, v.Len())
	
	for i := 0; i < v.Len(); i++ {
		result[i] = a.AnonymizeValue(v.Index(i).Interface(), rules)
	}
	
	return result
}

// anonymizeMap 脱敏映射
func (a *anonymizer) anonymizeMap(v reflect.Value, rules map[string]AnonymizeRule) interface{} {
	result := make(map[interface{}]interface{})
	
	for _, key := range v.MapKeys() {
		mapValue := v.MapIndex(key)
		result[key.Interface()] = a.AnonymizeValue(mapValue.Interface(), rules)
	}
	
	return result
}

// anonymizeStringValue 脱敏字符串值
func (a *anonymizer) anonymizeStringValue(value string, rules map[string]AnonymizeRule) string {
	// 检测字符串中的敏感信息
	detected := a.detector.DetectAll(value)
	
	result := value
	for fieldType, matches := range detected {
		if rule, exists := rules[fieldType]; exists {
			for _, match := range matches {
				anonymized := a.AnonymizeString(match, rule)
				result = strings.ReplaceAll(result, match, anonymized)
			}
		}
	}
	
	return result
}

// fieldAnonymizer 字段脱敏器实现
type fieldAnonymizer struct {
	anonymizer Anonymizer
}

// NewFieldAnonymizer 创建字段脱敏器
func NewFieldAnonymizer() FieldAnonymizer {
	return &fieldAnonymizer{
		anonymizer: NewAnonymizer(),
	}
}

// AnonymizeEmail 脱敏邮箱
func (f *fieldAnonymizer) AnonymizeEmail(email string) string {
	return f.anonymizer.AnonymizeString(email, EmailRule)
}

// AnonymizePhone 脱敏手机号
func (f *fieldAnonymizer) AnonymizePhone(phone string) string {
	return f.anonymizer.AnonymizeString(phone, PhoneRule)
}

// AnonymizeName 脱敏姓名
func (f *fieldAnonymizer) AnonymizeName(name string) string {
	return f.anonymizer.AnonymizeString(name, NameRule)
}

// AnonymizeIDCard 脱敏身份证号
func (f *fieldAnonymizer) AnonymizeIDCard(idCard string) string {
	return f.anonymizer.AnonymizeString(idCard, IDCardRule)
}

// AnonymizeBankCard 脱敏银行卡号
func (f *fieldAnonymizer) AnonymizeBankCard(bankCard string) string {
	return f.anonymizer.AnonymizeString(bankCard, BankCardRule)
}

// AnonymizeAddress 脱敏地址
func (f *fieldAnonymizer) AnonymizeAddress(address string) string {
	return f.anonymizer.AnonymizeString(address, AddressRule)
}

// logSanitizer 日志脱敏器实现
type logSanitizer struct {
	anonymizer Anonymizer
	detector   SensitiveDetector
	rules      map[string]AnonymizeRule
}

// NewLogSanitizer 创建日志脱敏器
func NewLogSanitizer() LogSanitizer {
	return &logSanitizer{
		anonymizer: NewAnonymizer(),
		detector:   NewSensitiveDetector(),
		rules:      GetDefaultRules(),
	}
}

// SanitizeLogContent 脱敏日志内容
func (l *logSanitizer) SanitizeLogContent(content interface{}) string {
	if content == nil {
		return ""
	}
	
	// 如果实现了MakeSensitive接口
	if sensitive, ok := content.(MakeSensitive); ok {
		anonymized := l.anonymizer.AnonymizeObject(sensitive)
		if jsonBytes, err := json.Marshal(anonymized); err == nil {
			return string(jsonBytes)
		}
	}
	
	// 如果实现了LogSafeStringer接口
	if logSafe, ok := content.(LogSafeStringer); ok {
		return logSafe.LogSafeString()
	}
	
	// 转换为字符串并脱敏
	var str string
	switch v := content.(type) {
	case string:
		str = v
	case fmt.Stringer:
		str = v.String()
	default:
		if jsonBytes, err := json.Marshal(content); err == nil {
			str = string(jsonBytes)
		} else {
			str = fmt.Sprintf("%+v", content)
		}
	}
	
	return l.SanitizeLogMessage(str)
}

// SanitizeLogMessage 脱敏日志消息
func (l *logSanitizer) SanitizeLogMessage(message string) string {
	if message == "" {
		return message
	}
	
	// 检测并脱敏敏感信息
	detected := l.detector.DetectAll(message)
	
	result := message
	for fieldType, matches := range detected {
		if rule, exists := l.rules[fieldType]; exists {
			for _, match := range matches {
				anonymized := l.anonymizer.AnonymizeString(match, rule)
				result = strings.ReplaceAll(result, match, anonymized)
			}
		}
	}
	
	return result
}

// ContainsSensitiveInfo 检查是否包含敏感信息
func (l *logSanitizer) ContainsSensitiveInfo(content string) bool {
	detected := l.detector.DetectAll(content)
	return len(detected) > 0
}

// sensitiveDetector 敏感信息检测器实现
type sensitiveDetector struct {
	emailRegex    *regexp.Regexp
	phoneRegex    *regexp.Regexp
	idCardRegex   *regexp.Regexp
	bankCardRegex *regexp.Regexp
}

// NewSensitiveDetector 创建敏感信息检测器
func NewSensitiveDetector() SensitiveDetector {
	return &sensitiveDetector{
		emailRegex:    regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		phoneRegex:    regexp.MustCompile(`1[3-9]\d{9}`),
		idCardRegex:   regexp.MustCompile(`[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]`),
		bankCardRegex: regexp.MustCompile(`[1-9]\d{11,19}`),
	}
}

// DetectSensitiveInfo 检测特定类型的敏感信息
func (s *sensitiveDetector) DetectSensitiveInfo(text string, infoType string) []string {
	switch strings.ToLower(infoType) {
	case "email":
		return s.DetectEmail(text)
	case "phone":
		return s.DetectPhone(text)
	case "idcard", "id_card":
		return s.DetectIDCard(text)
	case "bankcard", "bank_card":
		return s.DetectBankCard(text)
	default:
		return nil
	}
}

// HasSensitiveInfo 检查是否包含敏感信息
func (s *sensitiveDetector) HasSensitiveInfo(text string, infoType string) bool {
	matches := s.DetectSensitiveInfo(text, infoType)
	return len(matches) > 0
}

// DetectEmail 检测邮箱
func (s *sensitiveDetector) DetectEmail(text string) []string {
	return s.emailRegex.FindAllString(text, -1)
}

// DetectPhone 检测手机号
func (s *sensitiveDetector) DetectPhone(text string) []string {
	return s.phoneRegex.FindAllString(text, -1)
}

// DetectIDCard 检测身份证号
func (s *sensitiveDetector) DetectIDCard(text string) []string {
	return s.idCardRegex.FindAllString(text, -1)
}

// DetectBankCard 检测银行卡号
func (s *sensitiveDetector) DetectBankCard(text string) []string {
	return s.bankCardRegex.FindAllString(text, -1)
}

// DetectAll 检测所有敏感信息
func (s *sensitiveDetector) DetectAll(text string) map[string][]string {
	result := make(map[string][]string)
	
	if emails := s.DetectEmail(text); len(emails) > 0 {
		result["email"] = emails
	}
	
	if phones := s.DetectPhone(text); len(phones) > 0 {
		result["phone"] = phones
	}
	
	if idCards := s.DetectIDCard(text); len(idCards) > 0 {
		result["id_card"] = idCards
	}
	
	if bankCards := s.DetectBankCard(text); len(bankCards) > 0 {
		result["bank_card"] = bankCards
	}
	
	return result
}