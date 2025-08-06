package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// logVisitor AST访问器，用于检测日志方法调用
type logVisitor struct {
	checker  *LogChecker
	filename string
	issues   []Issue
}

// Visit 实现ast.Visitor接口
func (v *logVisitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.CallExpr:
		v.checkLogCall(n)
	}
	return v
}

// checkLogCall 检查函数调用是否为日志方法
func (v *logVisitor) checkLogCall(call *ast.CallExpr) {
	// 检查选择器表达式 (如 log.Infof)
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		methodName := sel.Sel.Name
		
		// 检查是否为格式化日志方法
		if v.isFormattedLogMethod(methodName) {
			// 检查是否应该使用结构化日志
			if v.shouldUseStructuredLog(call) {
				v.addIssue(call, methodName)
			}
		}
	}
}

// isFormattedLogMethod 检查是否为格式化日志方法
func (v *logVisitor) isFormattedLogMethod(methodName string) bool {
	formattedMethods := []string{
		"Infof", "Debugf", "Errorf", "Warnf", "Warningf",
		"Info", "Debug", "Error", "Warn", "Warning",
		"Printf", "Print", "Println",
	}
	
	for _, method := range formattedMethods {
		if methodName == method {
			return true
		}
	}
	return false
}

// shouldUseStructuredLog 判断是否应该使用结构化日志
func (v *logVisitor) shouldUseStructuredLog(call *ast.CallExpr) bool {
	// 如果参数数量大于1，可能包含结构化数据
	if len(call.Args) > 1 {
		return true
	}
	
	// 检查第一个参数是否包含格式化占位符
	if len(call.Args) > 0 {
		if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
			// 移除引号
			value := strings.Trim(lit.Value, "`\"")
			// 如果包含%占位符，建议使用结构化日志
			if strings.Contains(value, "%") {
				return true
			}
			// 如果消息较长或包含敏感信息关键词，建议使用结构化日志
			if len(value) > 50 || v.containsSensitiveKeywords(value) {
				return true
			}
		}
	}
	
	return false
}

// containsSensitiveKeywords 检查是否包含敏感信息关键词
func (v *logVisitor) containsSensitiveKeywords(text string) bool {
	sensitiveKeywords := []string{
		"password", "token", "secret", "key", "auth",
		"email", "phone", "mobile", "card", "id",
		"user", "account", "login", "credential",
	}
	
	lowerText := strings.ToLower(text)
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}
	return false
}

// addIssue 添加问题到列表
func (v *logVisitor) addIssue(call *ast.CallExpr, methodName string) {
	pos := v.checker.fset.Position(call.Pos())
	
	// 生成建议的结构化方法名
	structuredMethod := v.getStructuredMethodName(methodName)
	
	issue := Issue{
		File:        v.filename,
		Line:        pos.Line,
		Column:      pos.Column,
		Method:      methodName,
		Suggestion:  structuredMethod,
		Description: v.generateDescription(methodName, structuredMethod),
		Severity:    v.getSeverity(methodName),
	}
	
	v.issues = append(v.issues, issue)
}

// getStructuredMethodName 获取对应的结构化日志方法名
func (v *logVisitor) getStructuredMethodName(methodName string) string {
	mapping := map[string]string{
		"Infof":     "Infow",
		"Info":      "Infow",
		"Debugf":    "Debugw",
		"Debug":     "Debugw",
		"Errorf":    "Errorw",
		"Error":     "Errorw",
		"Warnf":     "Warnw",
		"Warn":      "Warnw",
		"Warningf":  "Warnw",
		"Warning":   "Warnw",
		"Printf":    "Infow",
		"Print":     "Infow",
		"Println":   "Infow",
	}
	
	if structured, exists := mapping[methodName]; exists {
		return structured
	}
	return "Infow" // 默认建议
}

// generateDescription 生成问题描述
func (v *logVisitor) generateDescription(oldMethod, newMethod string) string {
	return fmt.Sprintf("使用了格式化日志方法 '%s'，建议使用结构化日志方法 '%s' 以获得更好的日志脱敏和分析能力", oldMethod, newMethod)
}

// getSeverity 获取问题严重程度
func (v *logVisitor) getSeverity(methodName string) string {
	// Error级别的日志问题严重程度更高
	if strings.Contains(strings.ToLower(methodName), "error") {
		return "high"
	}
	// Warn级别的日志问题中等严重程度
	if strings.Contains(strings.ToLower(methodName), "warn") {
		return "medium"
	}
	// 其他级别为低严重程度
	return "low"
}