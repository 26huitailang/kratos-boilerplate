package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	kratosErrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// ErrorHandlerMiddleware 统一错误处理中间件
type ErrorHandlerMiddleware struct {
	logger        *log.Helper
	enableDetails bool
	enableStack   bool
}

// NewErrorHandlerMiddleware 创建错误处理中间件
func NewErrorHandlerMiddleware(logger log.Logger, enableDetails, enableStack bool) *ErrorHandlerMiddleware {
	return &ErrorHandlerMiddleware{
		logger:        log.NewHelper(log.With(logger, "component", "error_handler")),
		enableDetails: enableDetails,
		enableStack:   enableStack,
	}
}

// Middleware 返回中间件函数
func (e *ErrorHandlerMiddleware) Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			reply, err := handler(ctx, req)
			if err != nil {
				// 处理并格式化错误
				return reply, e.handleError(ctx, err)
			}
			return reply, nil
		}
	}
}

// handleError 处理错误
func (e *ErrorHandlerMiddleware) handleError(ctx context.Context, err error) error {
	// 获取请求信息
	var operation, method, path string
	var traceID string

	if tr, ok := transport.FromServerContext(ctx); ok {
		operation = tr.Operation()
		if httpTr, ok := tr.(interface{ Request() *http.Request }); ok {
			req := httpTr.Request()
			method = req.Method
			path = req.URL.Path
		}

		// 尝试从header获取trace ID
		if headers := tr.RequestHeader(); headers != nil {
			traceID = headers.Get("X-Trace-ID")
			if traceID == "" {
				traceID = headers.Get("X-Request-ID")
			}
		}
	}

	// 从context获取trace ID
	if traceID == "" {
		if id := ctx.Value("trace_id"); id != nil {
			if tid, ok := id.(string); ok {
				traceID = tid
			}
		}
	}

	// 转换为统一错误格式
	baseErr := e.convertError(err)

	// 添加请求上下文信息
	if operation != "" {
		baseErr = baseErr.WithOperation(operation)
	}
	if traceID != "" {
		baseErr = baseErr.WithTraceID(traceID)
	}

	// 记录错误日志
	e.logError(baseErr, method, path, traceID)

	// 转换为Kratos错误返回
	return e.toKratosError(baseErr)
}

// convertError 转换错误为BaseError
func (e *ErrorHandlerMiddleware) convertError(err error) *BaseError {
	if err == nil {
		return nil
	}

	// 如果已经是BaseError，直接返回
	if baseErr, ok := err.(*BaseError); ok {
		return baseErr
	}

	// 如果是Kratos错误，转换格式
	if kratosErr := kratosErrors.FromError(err); kratosErr != nil {
		code := parseKratosErrorCode(kratosErr.Code)
		baseErr := NewError(code, kratosErr.Message)

		if kratosErr.Reason != "" {
			baseErr = baseErr.WithDetail("reason", kratosErr.Reason)
		}

		return baseErr.WithCause(err)
	}

	// 检查特定错误类型
	errMsg := err.Error()
	switch {
	case strings.Contains(errMsg, "context canceled"):
		return NewError(ErrCodeTimeout, "Request was canceled").WithCause(err)
	case strings.Contains(errMsg, "context deadline exceeded"):
		return NewError(ErrCodeTimeout, "Request timeout").WithCause(err)
	case strings.Contains(errMsg, "connection refused"):
		return NewError(ErrCodeExternalService, "Service temporarily unavailable").WithCause(err)
	case strings.Contains(errMsg, "no such host"):
		return NewError(ErrCodeNetworkError, "Network connectivity issue").WithCause(err)
	default:
		// 其他错误包装为内部错误
		return InternalError("An internal error occurred").WithCause(err)
	}
}

// logError 记录错误日志
func (e *ErrorHandlerMiddleware) logError(baseErr *BaseError, method, path, traceID string) {
	// 根据错误级别选择日志级别
	level := log.LevelError
	if baseErr.Code >= 20000 && baseErr.Code < 30000 {
		// 参数错误使用warn级别
		level = log.LevelWarn
	} else if baseErr.Code >= 40000 && baseErr.Code < 50000 {
		// 业务错误使用info级别
		level = log.LevelInfo
	}

	logFields := []interface{}{
		"error_code", baseErr.Code,
		"error_message", baseErr.Message,
		"http_status", baseErr.HTTPStatus,
		"method", method,
		"path", path,
		"trace_id", traceID,
		"operation", baseErr.Operation,
		"timestamp", baseErr.Timestamp,
	}

	// 添加详细信息
	if e.enableDetails && len(baseErr.Details) > 0 {
		if detailsJSON, err := json.Marshal(baseErr.Details); err == nil {
			logFields = append(logFields, "details", string(detailsJSON))
		}
	}

	// 添加原始错误
	if baseErr.Cause != nil {
		logFields = append(logFields, "cause", baseErr.Cause.Error())
	}

	e.logger.Log(level, logFields...)
}

// toKratosError 转换为Kratos错误
func (e *ErrorHandlerMiddleware) toKratosError(baseErr *BaseError) error {
	if baseErr == nil {
		return nil
	}

	// 创建错误响应
	errorResp := map[string]interface{}{
		"code":      baseErr.Code,
		"message":   baseErr.Message,
		"timestamp": baseErr.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 添加追踪ID
	if baseErr.TraceID != "" {
		errorResp["trace_id"] = baseErr.TraceID
	}

	// 添加详细信息（生产环境可能需要过滤敏感信息）
	if e.enableDetails && len(baseErr.Details) > 0 {
		errorResp["details"] = baseErr.Details
	}

	// 添加操作信息
	if baseErr.Operation != "" {
		errorResp["operation"] = baseErr.Operation
	}

	// 创建Kratos错误
	kratosErr := kratosErrors.New(
		baseErr.HTTPStatus,
		string(rune(baseErr.Code)),
		baseErr.Message,
	)

	// 添加元数据
	if metadata := kratosErr.Metadata; metadata == nil {
		kratosErr.Metadata = make(map[string]string)
	}

	kratosErr.Metadata["error_code"] = string(rune(baseErr.Code))
	kratosErr.Metadata["trace_id"] = baseErr.TraceID

	if respJSON, err := json.Marshal(errorResp); err == nil {
		kratosErr.Metadata["response"] = string(respJSON)
	}

	return kratosErr
}

// ValidationErrorMiddleware 验证错误处理中间件
type ValidationErrorMiddleware struct {
	logger *log.Helper
}

// NewValidationErrorMiddleware 创建验证错误中间件
func NewValidationErrorMiddleware(logger log.Logger) *ValidationErrorMiddleware {
	return &ValidationErrorMiddleware{
		logger: log.NewHelper(log.With(logger, "component", "validation_error")),
	}
}

// Middleware 返回中间件函数
func (v *ValidationErrorMiddleware) Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			reply, err := handler(ctx, req)
			if err != nil {
				// 检查是否是验证错误
				if validationErr := v.extractValidationError(err); validationErr != nil {
					return reply, validationErr
				}
			}
			return reply, err
		}
	}
}

// extractValidationError 提取验证错误
func (v *ValidationErrorMiddleware) extractValidationError(err error) error {
	errMsg := err.Error()

	// 检查常见的验证错误模式
	switch {
	case strings.Contains(errMsg, "required"):
		field := v.extractFieldName(errMsg)
		return InvalidParams(fmt.Sprintf("Field '%s' is required", field))

	case strings.Contains(errMsg, "invalid format"):
		field := v.extractFieldName(errMsg)
		return InvalidFormat(field, "valid format")

	case strings.Contains(errMsg, "out of range"):
		field := v.extractFieldName(errMsg)
		return InvalidRange(field, "min", "max")

	case strings.Contains(errMsg, "too long") || strings.Contains(errMsg, "too short"):
		field := v.extractFieldName(errMsg)
		return InvalidFormat(field, "proper length")

	default:
		return nil
	}
}

// extractFieldName 从错误消息中提取字段名
func (v *ValidationErrorMiddleware) extractFieldName(errMsg string) string {
	// 简单的字段名提取逻辑，实际项目中可能需要更复杂的解析
	parts := strings.Split(errMsg, " ")
	for _, part := range parts {
		if strings.Contains(part, ".") || strings.Contains(part, "_") {
			return strings.Trim(part, " :'\"")
		}
	}
	return "unknown"
}

// RateLimitErrorMiddleware 限流错误处理中间件
type RateLimitErrorMiddleware struct {
	logger *log.Helper
}

// NewRateLimitErrorMiddleware 创建限流错误中间件
func NewRateLimitErrorMiddleware(logger log.Logger) *RateLimitErrorMiddleware {
	return &RateLimitErrorMiddleware{
		logger: log.NewHelper(log.With(logger, "component", "rate_limit_error")),
	}
}

// Middleware 返回中间件函数
func (r *RateLimitErrorMiddleware) Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			reply, err := handler(ctx, req)
			if err != nil {
				// 检查是否是限流错误
				if kratosErr := kratosErrors.FromError(err); kratosErr != nil && kratosErr.Code == 429 {
					return reply, QuotaExceeded("request rate", "per minute").
						WithDetail("retry_after", "60").
						WithDetail("suggestion", "Please slow down your requests and try again later")
				}
			}
			return reply, err
		}
	}
}
