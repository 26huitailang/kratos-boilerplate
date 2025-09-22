package feature

import (
	"context"
	"fmt"
	"time"
)

// 功能开关错误代码常量
const (
	ErrCodeFeatureNotFound        = "FEATURE_NOT_FOUND"
	ErrCodeInvalidConfig          = "INVALID_CONFIG"
	ErrCodeStrategyNotSupported   = "STRATEGY_NOT_SUPPORTED"
	ErrCodeRepositoryError        = "REPOSITORY_ERROR"
	ErrCodeEvaluationError        = "EVALUATION_ERROR"
	ErrCodeConfigValidationError  = "CONFIG_VALIDATION_ERROR"
	ErrCodePermissionDenied       = "PERMISSION_DENIED"
	ErrCodeRateLimitExceeded      = "RATE_LIMIT_EXCEEDED"
	ErrCodeCircuitBreakerOpen     = "CIRCUIT_BREAKER_OPEN"
	ErrCodeDependencyError        = "DEPENDENCY_ERROR"
	ErrCodeAuditLogError          = "AUDIT_LOG_ERROR"
	ErrCodeMetricsError           = "METRICS_ERROR"
	ErrCodeCacheError             = "CACHE_ERROR"
	ErrCodeNetworkError           = "NETWORK_ERROR"
	ErrCodeTimeoutError           = "TIMEOUT_ERROR"
)

// FeatureError 功能开关系统的错误类型
type FeatureError struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Feature   FeatureFlag `json:"feature,omitempty"`
	Operation string      `json:"operation,omitempty"`
	Cause     error       `json:"cause,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Error 实现 error 接口
func (e *FeatureError) Error() string {
	if e.Feature != "" {
		return fmt.Sprintf("[%s] %s (feature: %s): %s", e.Code, e.Operation, e.Feature, e.Message)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Operation, e.Message)
}

// Unwrap 实现错误链
func (e *FeatureError) Unwrap() error {
	return e.Cause
}

// Is 判断错误类型
func (e *FeatureError) Is(target error) bool {
	if fe, ok := target.(*FeatureError); ok {
		return e.Code == fe.Code
	}
	return false
}

// NewFeatureError 创建新的功能开关错误
func NewFeatureError(code, operation, message string) *FeatureError {
	return &FeatureError{
		Code:      code,
		Operation: operation,
		Message:   message,
		Details:   make(map[string]interface{}),
	}
}

// NewFeatureErrorWithFeature 创建包含功能标识的错误
func NewFeatureErrorWithFeature(code, operation, message string, feature FeatureFlag) *FeatureError {
	return &FeatureError{
		Code:      code,
		Operation: operation,
		Message:   message,
		Feature:   feature,
		Details:   make(map[string]interface{}),
	}
}

// NewFeatureErrorWithCause 创建包含原因的错误
func NewFeatureErrorWithCause(code, operation, message string, cause error) *FeatureError {
	return &FeatureError{
		Code:      code,
		Operation: operation,
		Message:   message,
		Cause:     cause,
		Details:   make(map[string]interface{}),
	}
}

// WithFeature 设置功能标识
func (e *FeatureError) WithFeature(feature FeatureFlag) *FeatureError {
	e.Feature = feature
	return e
}

// WithCause 设置错误原因
func (e *FeatureError) WithCause(cause error) *FeatureError {
	e.Cause = cause
	return e
}

// WithDetail 添加详细信息
func (e *FeatureError) WithDetail(key string, value interface{}) *FeatureError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// IsRetryable 判断错误是否可重试
func (e *FeatureError) IsRetryable() bool {
	switch e.Code {
	case ErrCodeNetworkError, ErrCodeTimeoutError, ErrCodeRepositoryError:
		return true
	default:
		return false
	}
}

// IsClientError 判断是否为客户端错误
func (e *FeatureError) IsClientError() bool {
	switch e.Code {
	case ErrCodeFeatureNotFound, ErrCodeInvalidConfig, ErrCodeConfigValidationError, ErrCodePermissionDenied:
		return true
	default:
		return false
	}
}

// IsServerError 判断是否为服务端错误
func (e *FeatureError) IsServerError() bool {
	switch e.Code {
	case ErrCodeRepositoryError, ErrCodeEvaluationError, ErrCodeAuditLogError, ErrCodeMetricsError, ErrCodeDependencyError:
		return true
	default:
		return false
	}
}

// 预定义错误创建函数

// ErrFeatureNotFound 功能不存在错误
func ErrFeatureNotFound(feature FeatureFlag) *FeatureError {
	return NewFeatureErrorWithFeature(
		ErrCodeFeatureNotFound,
		"check_feature",
		"feature not found",
		feature,
	)
}

// ErrInvalidConfig 无效配置错误
func ErrInvalidConfig(feature FeatureFlag, reason string) *FeatureError {
	return NewFeatureErrorWithFeature(
		ErrCodeInvalidConfig,
		"validate_config",
		reason,
		feature,
	)
}

// ErrStrategyNotSupported 策略不支持错误
func ErrStrategyNotSupported(strategy FeatureStrategy) *FeatureError {
	return NewFeatureError(
		ErrCodeStrategyNotSupported,
		"evaluate_strategy",
		fmt.Sprintf("strategy %s is not supported", strategy),
	).WithDetail("strategy", strategy)
}

// ErrRepositoryError 存储库错误
func ErrRepositoryError(operation string, cause error) *FeatureError {
	return NewFeatureErrorWithCause(
		ErrCodeRepositoryError,
		operation,
		"repository operation failed",
		cause,
	)
}

// ErrEvaluationError 评估错误
func ErrEvaluationError(feature FeatureFlag, cause error) *FeatureError {
	return NewFeatureErrorWithCause(
		ErrCodeEvaluationError,
		"evaluate_feature",
		"failed to evaluate feature",
		cause,
	).WithFeature(feature)
}

// ErrConfigValidationError 配置验证错误
func ErrConfigValidationError(field, reason string) *FeatureError {
	return NewFeatureError(
		ErrCodeConfigValidationError,
		"validate_config",
		fmt.Sprintf("validation failed for field %s: %s", field, reason),
	).WithDetail("field", field).WithDetail("reason", reason)
}

// ErrPermissionDenied 权限拒绝错误
func ErrPermissionDenied(operation string, feature FeatureFlag) *FeatureError {
	return NewFeatureErrorWithFeature(
		ErrCodePermissionDenied,
		operation,
		"permission denied",
		feature,
	)
}

// ErrRateLimitExceeded 速率限制错误
func ErrRateLimitExceeded(feature FeatureFlag) *FeatureError {
	return NewFeatureErrorWithFeature(
		ErrCodeRateLimitExceeded,
		"check_rate_limit",
		"rate limit exceeded",
		feature,
	)
}

// ErrCircuitBreakerOpen 熔断器打开错误
func ErrCircuitBreakerOpen(feature FeatureFlag) *FeatureError {
	return NewFeatureErrorWithFeature(
		ErrCodeCircuitBreakerOpen,
		"check_circuit_breaker",
		"circuit breaker is open",
		feature,
	)
}

// ErrDependencyError 依赖错误
func ErrDependencyError(dependency string, cause error) *FeatureError {
	return NewFeatureErrorWithCause(
		ErrCodeDependencyError,
		"check_dependency",
		fmt.Sprintf("dependency %s failed", dependency),
		cause,
	).WithDetail("dependency", dependency)
}

// ErrorCollector 错误收集器
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector 创建错误收集器
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add 添加错误
func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors 是否有错误
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Errors 获取所有错误
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// Error 实现 error 接口，返回组合错误信息
func (ec *ErrorCollector) Error() string {
	if len(ec.errors) == 0 {
		return ""
	}
	
	if len(ec.errors) == 1 {
		return ec.errors[0].Error()
	}
	
	msg := fmt.Sprintf("multiple errors (%d): ", len(ec.errors))
	for i, err := range ec.errors {
		if i > 0 {
			msg += "; "
		}
		msg += err.Error()
	}
	return msg
}

// First 获取第一个错误
func (ec *ErrorCollector) First() error {
	if len(ec.errors) == 0 {
		return nil
	}
	return ec.errors[0]
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	// HandleError 处理错误
	HandleError(ctx context.Context, err error) error
	// ShouldRetry 判断是否应该重试
	ShouldRetry(err error) bool
	// GetRetryDelay 获取重试延迟时间
	GetRetryDelay(attempt int) time.Duration
}

// DefaultErrorHandler 默认错误处理器
type DefaultErrorHandler struct {
	maxRetries int
	baseDelay  time.Duration
}

// NewDefaultErrorHandler 创建默认错误处理器
func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{
		maxRetries: 3,
		baseDelay:  100 * time.Millisecond,
	}
}

// HandleError 处理错误
func (h *DefaultErrorHandler) HandleError(ctx context.Context, err error) error {
	if featureErr, ok := err.(*FeatureError); ok {
		// 记录错误详情
		logFeatureError(ctx, featureErr)
		return featureErr
	}
	
	// 将非功能开关错误包装为功能开关错误
	return NewFeatureErrorWithCause(
		ErrCodeDependencyError,
		"unknown_operation", 
		"unknown error occurred",
		err,
	)
}

// ShouldRetry 判断是否应该重试
func (h *DefaultErrorHandler) ShouldRetry(err error) bool {
	if featureErr, ok := err.(*FeatureError); ok {
		return featureErr.IsRetryable()
	}
	return false
}

// GetRetryDelay 获取重试延迟时间
func (h *DefaultErrorHandler) GetRetryDelay(attempt int) time.Duration {
	// 指数退避策略
	delay := h.baseDelay * time.Duration(1<<uint(attempt))
	// 最大延迟不超过30秒
	if delay > 30*time.Second {
		delay = 30*time.Second
	}
	return delay
}

// logFeatureError 记录功能开关错误
func logFeatureError(ctx context.Context, err *FeatureError) {
	// 这里可以集成具体的日志系统
	// 暂时使用简单的日志记录
	fmt.Printf("FeatureError: code=%s, operation=%s, feature=%s, message=%s\n", 
		err.Code, err.Operation, err.Feature, err.Message)
}