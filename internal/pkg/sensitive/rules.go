package sensitive

import (
	"regexp"
	"strings"
)

// 预定义的脱敏规则
var (
	// 邮箱脱敏规则
	EmailRule = AnonymizeRule{
		FieldName: "email",
		KeepStart: 2,
		KeepEnd:   4, // 保留@domain.com部分
		MaskChar:  "*",
		CustomFunc: func(email string) string {
			if !isValidEmail(email) {
				return email
			}
			parts := strings.Split(email, "@")
			if len(parts) != 2 {
				return email
			}
			username := parts[0]
			domain := parts[1]
			
			if len(username) <= 2 {
				return email
			}
			
			maskedUsername := username[:2] + strings.Repeat("*", len(username)-2)
			return maskedUsername + "@" + domain
		},
	}
	
	// 手机号脱敏规则
	PhoneRule = AnonymizeRule{
		FieldName: "phone",
		KeepStart: 3,
		KeepEnd:   4,
		MaskChar:  "*",
		CustomFunc: func(phone string) string {
			if !isValidPhone(phone) {
				return phone
			}
			// 中国手机号格式：138****8000
			if len(phone) == 11 {
				return phone[:3] + "****" + phone[7:]
			}
			// 其他格式使用通用规则
			rule := AnonymizeRule{KeepStart: 3, KeepEnd: 4, MaskChar: "*"}
			return anonymizeWithRule(phone, rule)
		},
	}
	
	// 姓名脱敏规则
	NameRule = AnonymizeRule{
		FieldName: "name",
		KeepStart: 1,
		KeepEnd:   1,
		MaskChar:  "*",
		CustomFunc: func(name string) string {
			runes := []rune(strings.TrimSpace(name))
			if len(runes) <= 1 {
				return name
			}
			if len(runes) == 2 {
				return string(runes[0]) + "*"
			}
			// 保留首尾字符，中间用*替代
			maskLength := len(runes) - 2
			return string(runes[0]) + strings.Repeat("*", maskLength) + string(runes[len(runes)-1])
		},
	}
	
	// 身份证号脱敏规则
	IDCardRule = AnonymizeRule{
		FieldName: "id_card",
		KeepStart: 6,
		KeepEnd:   4,
		MaskChar:  "*",
		CustomFunc: func(idCard string) string {
			if !isValidIDCard(idCard) {
				return idCard
			}
			// 身份证号格式：110101********1234
			if len(idCard) == 18 {
				return idCard[:6] + "********" + idCard[14:]
			}
			rule := AnonymizeRule{KeepStart: 6, KeepEnd: 4, MaskChar: "*"}
			return anonymizeWithRule(idCard, rule)
		},
	}
	
	// 银行卡号脱敏规则
	BankCardRule = AnonymizeRule{
		FieldName: "bank_card",
		KeepStart: 4,
		KeepEnd:   4,
		MaskChar:  "*",
		CustomFunc: func(bankCard string) string {
			if !isValidBankCard(bankCard) {
				return bankCard
			}
			// 银行卡号格式：6222********1234
			if len(bankCard) >= 8 {
				maskLength := len(bankCard) - 8
				return bankCard[:4] + strings.Repeat("*", maskLength) + bankCard[len(bankCard)-4:]
			}
			rule := AnonymizeRule{KeepStart: 4, KeepEnd: 4, MaskChar: "*"}
			return anonymizeWithRule(bankCard, rule)
		},
	}
	
	// 地址脱敏规则
	AddressRule = AnonymizeRule{
		FieldName: "address",
		KeepStart: 6,
		KeepEnd:   0,
		MaskChar:  "*",
		CustomFunc: func(address string) string {
			runes := []rune(strings.TrimSpace(address))
			if len(runes) <= 6 {
				return address
			}
			// 保留前6个字符（通常是省市区），后面用*替代
			maskLength := len(runes) - 6
			return string(runes[:6]) + strings.Repeat("*", maskLength)
		},
	}
)

// GetDefaultRules 获取默认脱敏规则集合
func GetDefaultRules() map[string]AnonymizeRule {
	return map[string]AnonymizeRule{
		"email":     EmailRule,
		"phone":     PhoneRule,
		"name":      NameRule,
		"id_card":   IDCardRule,
		"bank_card": BankCardRule,
		"address":   AddressRule,
	}
}

// GetRuleByFieldName 根据字段名获取脱敏规则
func GetRuleByFieldName(fieldName string) (AnonymizeRule, bool) {
	rules := GetDefaultRules()
	rule, exists := rules[strings.ToLower(fieldName)]
	return rule, exists
}

// 验证函数
var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex    = regexp.MustCompile(`^1[3-9]\d{9}$`)
	idCardRegex   = regexp.MustCompile(`^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`)
	bankCardRegex = regexp.MustCompile(`^[1-9]\d{11,19}$`)
)

// isValidEmail 验证邮箱格式
func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// isValidPhone 验证手机号格式
func isValidPhone(phone string) bool {
	return phoneRegex.MatchString(phone)
}

// isValidIDCard 验证身份证号格式
func isValidIDCard(idCard string) bool {
	return idCardRegex.MatchString(idCard)
}

// isValidBankCard 验证银行卡号格式
func isValidBankCard(bankCard string) bool {
	return bankCardRegex.MatchString(bankCard)
}

// anonymizeWithRule 使用规则进行脱敏
func anonymizeWithRule(value string, rule AnonymizeRule) string {
	runes := []rune(value)
	length := len(runes)
	
	if length <= rule.KeepStart+rule.KeepEnd {
		return value
	}
	
	maskChar := rule.MaskChar
	if maskChar == "" {
		maskChar = "*"
	}
	
	start := string(runes[:rule.KeepStart])
	end := string(runes[length-rule.KeepEnd:])
	maskLength := length - rule.KeepStart - rule.KeepEnd
	mask := strings.Repeat(maskChar, maskLength)
	
	return start + mask + end
}

// CreateCustomRule 创建自定义脱敏规则
func CreateCustomRule(fieldName string, keepStart, keepEnd int, maskChar string, customFunc func(string) string) AnonymizeRule {
	return AnonymizeRule{
		FieldName:  fieldName,
		KeepStart:  keepStart,
		KeepEnd:    keepEnd,
		MaskChar:   maskChar,
		CustomFunc: customFunc,
	}
}

// MergeRules 合并脱敏规则
func MergeRules(rules ...map[string]AnonymizeRule) map[string]AnonymizeRule {
	merged := make(map[string]AnonymizeRule)
	
	for _, ruleMap := range rules {
		for key, rule := range ruleMap {
			merged[key] = rule
		}
	}
	
	return merged
}