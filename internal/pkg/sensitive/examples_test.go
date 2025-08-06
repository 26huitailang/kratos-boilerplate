package sensitive

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
)

// TestBasicUsage 基本使用示例
func TestBasicUsage(t *testing.T) {
	// 创建标准输出日志器
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	
	// 创建结构化日志器
	slogger := NewStructuredLoggerQuick(logger)
	
	// 使用结构化日志
	slogger.Infow("用户登录",
		"user_id", 12345,
		"username", "john_doe",
		"ip", "192.168.1.100",
		"timestamp", "2024-01-15T10:30:00Z",
	)
	
	// Output:
	// 结构化日志输出，包含所有字段
}

// TestWithSanitization 带脱敏功能的示例
func TestWithSanitization(t *testing.T) {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	
	// 创建带脱敏功能的结构化日志器
	sensitiveKeys := []string{"password", "token", "secret", "credit_card"}
	slogger := NewStructuredLoggerWithSanitization(logger, sensitiveKeys)
	
	// 敏感信息会被自动脱敏
	slogger.Infow("用户注册",
		"username", "john_doe",
		"email", "john@example.com",
		"password", "mySecretPassword123", // 会被脱敏
		"phone", "13812345678",           // 会被自动检测并脱敏
	)
	
	// Output:
	// 敏感信息被脱敏的日志输出
}

// TestCustomRules 自定义脱敏规则示例
func TestCustomRules(t *testing.T) {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	
	// 定义自定义脱敏规则
	customRules := map[string]AnonymizeRule{
		"user_id": {
			FieldName: "user_id",
			KeepStart: 2,
			KeepEnd:   2,
			MaskChar:  "#",
		},
		"order_id": {
			FieldName: "order_id",
			CustomFunc: func(value string) string {
				return "ORDER_" + value[len(value)-4:] // 只保留后4位
			},
		},
	}
	
	// 创建带自定义规则的日志器
	slogger := NewStructuredLoggerWithCustomRules(logger, customRules)
	
	slogger.Infow("订单处理",
		"user_id", "1234567890",     // 会按自定义规则脱敏: 12####90
		"order_id", "ORD20240115001", // 会按自定义函数脱敏: ORDER_5001
		"amount", 99.99,
	)
	
	// Output:
	// 按自定义规则脱敏的日志输出
}

// TestProductionUsage 生产环境使用示例
func TestProductionUsage(t *testing.T) {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service", "user-service",
		"version", "v1.0.0",
	)
	
	// 创建生产环境日志器（启用所有安全特性）
	slogger := CreateProductionLogger(logger)
	
	// 添加上下文信息
	ctx := context.WithValue(context.Background(), "trace_id", "abc123")
	contextLogger := slogger.WithContext(ctx)
	
	// 添加通用字段
	serviceLogger := contextLogger.WithFields(
		"module", "authentication",
		"operation", "login",
	)
	
	// 记录业务日志
	serviceLogger.Infow("用户登录成功",
		"user_id", 12345,
		"username", "john_doe",
		"login_method", "password",
		"client_ip", "192.168.1.100",
		"user_agent", "Mozilla/5.0...",
	)
	
	serviceLogger.Errorw("登录失败",
		"username", "invalid_user",
		"reason", "invalid_credentials",
		"attempt_count", 3,
		"client_ip", "192.168.1.200",
	)
	
	// Output:
	// 生产环境格式的结构化日志
}

// TestMigration 从现有代码迁移示例
func TestMigration(t *testing.T) {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	
	// 现有的log.Helper
	existingHelper := log.NewHelper(logger)
	
	// 包装现有Helper为结构化日志器
	slogger := WrapExistingHelper(existingHelper)
	
	// 现在可以使用结构化日志方法
	slogger.Infow("迁移示例",
		"old_method", "helper.Infof",
		"new_method", "slogger.Infow",
		"benefits", []string{"结构化", "脱敏", "类型安全"},
	)
	
	// 同时保持兼容性，仍可使用原有方法
	slogger.Infof("兼容性示例: %s", "仍然可用")
	
	// Output:
	// 迁移后的日志输出
}

// TestErrorHandling 错误处理示例
func TestErrorHandling(t *testing.T) {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	
	slogger := CreateProductionLogger(logger)
	
	// 模拟业务错误
	err := fmt.Errorf("数据库连接失败")
	
	// 记录错误日志
	slogger.Errorw("业务操作失败",
		"operation", "create_user",
		"error", err.Error(),
		"user_data", map[string]interface{}{
			"username": "john_doe",
			"email":    "john@example.com", // 会被脱敏
			"phone":    "13812345678",      // 会被脱敏
		},
		"retry_count", 3,
		"timestamp", "2024-01-15T10:30:00Z",
	)
	
	// 记录警告日志
	slogger.Warnw("性能警告",
		"operation", "query_users",
		"duration_ms", 5000,
		"threshold_ms", 3000,
		"query", "SELECT * FROM users WHERE ...",
		"affected_rows", 10000,
	)
	
	// Output:
	// 错误和警告的结构化日志
}

// TestDifferentEnvironments 不同环境配置示例
func TestDifferentEnvironments(t *testing.T) {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	
	// 开发环境：禁用脱敏，便于调试
	devLogger := CreateDevelopmentLogger(logger)
	devLogger.Infow("开发环境日志",
		"password", "plaintext_password", // 开发环境不脱敏
		"debug_info", "详细调试信息",
	)
	
	// 测试环境：启用部分脱敏
	testLogger := CreateTestLogger(logger)
	testLogger.Infow("测试环境日志",
		"password", "test_password", // 测试环境会脱敏
		"test_data", "测试数据",
	)
	
	// 生产环境：启用完整脱敏
	prodLogger := CreateProductionLogger(logger)
	prodLogger.Infow("生产环境日志",
		"password", "prod_password", // 生产环境完全脱敏
		"sensitive_data", "敏感数据",
	)
	
	// Output:
	// 不同环境的日志输出
}

// TestPerformanceOptimization 性能优化示例
func TestPerformanceOptimization(t *testing.T) {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	
	// 创建配置，优化性能
	config := &StructuredLogConfig{
		Enabled:           true,
		AutoDetect:        false, // 禁用自动检测以提高性能
		CustomRules:       GetDefaultRules(),
		SensitiveKeys:     []string{"password", "token"}, // 只检测关键敏感字段
		MaxValueLength:    500,                           // 限制值长度
		TruncateThreshold: 300,
	}
	
	slogger := NewStructuredLoggerFromLogger(logger, config)
	
	// 高频日志记录
	for i := 0; i < 1000; i++ {
		slogger.Infow("高频操作",
			"operation_id", i,
			"status", "success",
			"duration_ms", 10+i%50,
		)
	}
	
	// Output:
	// 优化后的高性能日志输出
}