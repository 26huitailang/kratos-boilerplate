package sensitive

// MakeSensitive 敏感信息脱敏接口
type MakeSensitive interface {
	// 获取敏感字段列表
	GetSensitiveFields() []string
	
	// 脱敏处理
	Anonymize() interface{}
	
	// 获取脱敏规则
	GetAnonymizeRules() map[string]AnonymizeRule
}

// LogSafeStringer 日志安全字符串接口
type LogSafeStringer interface {
	// 返回日志安全的字符串表示
	LogSafeString() string
}

// AnonymizeRule 脱敏规则
type AnonymizeRule struct {
	FieldName  string                `json:"field_name"`  // 字段名
	KeepStart  int                   `json:"keep_start"`  // 保留开头字符数
	KeepEnd    int                   `json:"keep_end"`    // 保留结尾字符数
	MaskChar   string                `json:"mask_char"`   // 掩码字符，默认为*
	CustomFunc func(string) string   `json:"-"`           // 自定义脱敏函数
}

// Anonymizer 脱敏处理器接口
type Anonymizer interface {
	// 脱敏字符串
	AnonymizeString(value string, rule AnonymizeRule) string
	
	// 脱敏对象
	AnonymizeObject(obj MakeSensitive) interface{}
	
	// 批量脱敏
	AnonymizeBatch(objects []MakeSensitive) []interface{}
	
	// 脱敏任意值
	AnonymizeValue(value interface{}, rules map[string]AnonymizeRule) interface{}
}

// FieldAnonymizer 字段脱敏器接口
type FieldAnonymizer interface {
	// 脱敏邮箱
	AnonymizeEmail(email string) string
	
	// 脱敏手机号
	AnonymizePhone(phone string) string
	
	// 脱敏姓名
	AnonymizeName(name string) string
	
	// 脱敏身份证号
	AnonymizeIDCard(idCard string) string
	
	// 脱敏银行卡号
	AnonymizeBankCard(bankCard string) string
	
	// 脱敏地址
	AnonymizeAddress(address string) string
}

// LogSanitizer 日志脱敏器接口
type LogSanitizer interface {
	// 脱敏日志内容
	SanitizeLogContent(content interface{}) string
	
	// 脱敏日志消息
	SanitizeLogMessage(message string) string
	
	// 检查是否包含敏感信息
	ContainsSensitiveInfo(content string) bool
}

// SensitiveDetector 敏感信息检测器接口
type SensitiveDetector interface {
	// 检测邮箱
	DetectEmail(text string) []string
	
	// 检测手机号
	DetectPhone(text string) []string
	
	// 检测身份证号
	DetectIDCard(text string) []string
	
	// 检测银行卡号
	DetectBankCard(text string) []string
	
	// 检测所有敏感信息
	DetectAll(text string) map[string][]string
}