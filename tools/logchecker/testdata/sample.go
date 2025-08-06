package testdata

import (
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
)

type Service struct {
	log *log.Helper
}

// 应该被检测的问题代码
func (s *Service) BadExamples() {
	// 格式化日志方法 - 应该被检测
	s.log.Infof("User %s logged in", "john")
	s.log.Debugf("Processing request %d", 123)
	s.log.Errorf("Failed to connect to database: %v", fmt.Errorf("connection timeout"))
	s.log.Warnf("High memory usage: %d%%", 85)
	
	// 包含敏感信息 - 应该被检测
	s.log.Info("User password updated successfully")
	s.log.Debug("JWT token generated for user")
	s.log.Error("Authentication failed for email user@example.com")
	
	// 长消息 - 应该被检测
	s.log.Info("This is a very long log message that exceeds the minimum length threshold and should be detected by the log checker tool")
	
	// 多参数调用 - 应该被检测
	s.log.Info("Operation completed", "duration", "5s", "status", "success")
}

// 正确的结构化日志使用
func (s *Service) GoodExamples() {
	// 结构化日志方法 - 不应该被检测
	s.log.Infow("User logged in", "username", "john", "ip", "192.168.1.1")
	s.log.Debugw("Processing request", "request_id", 123, "method", "GET")
	s.log.Errorw("Database connection failed", "error", "connection timeout", "retry_count", 3)
	s.log.Warnw("High memory usage", "usage_percent", 85, "threshold", 80)
}

// 白名单方法 - 可能不被检测（取决于配置）
func (s *Service) WhitelistExamples() {
	fmt.Printf("Debug output: %v\n", "some value")
}

// 简单日志 - 可能不被检测
func (s *Service) SimpleExamples() {
	s.log.Info("Server started")
	s.log.Debug("Cache cleared")
	s.log.Error("Shutdown")
}