package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected int
	}{
		{"internal error", ErrCodeInternalError, 10000},
		{"invalid params", ErrCodeInvalidParams, 20000},
		{"unauthorized", ErrCodeUnauthorized, 30000},
		{"resource not found", ErrCodeResourceNotFound, 40000},
		{"external service", ErrCodeExternalService, 50000},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, int(tt.code))
		})
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrCodeInvalidParams, "test message")
	
	assert.Equal(t, ErrCodeInvalidParams, err.Code)
	assert.Equal(t, "test message", err.Message)
	assert.NotNil(t, err.Details)
	assert.False(t, err.Timestamp.IsZero())
	assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
	assert.Nil(t, err.Cause)
}

func TestNewErrorf(t *testing.T) {
	err := NewErrorf(ErrCodeInvalidParams, "test message: %s", "value")
	
	assert.Equal(t, ErrCodeInvalidParams, err.Code)
	assert.Equal(t, "test message: value", err.Message)
}

func TestNewErrorWithCause(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := NewErrorWithCause(ErrCodeInternalError, "wrapped error", cause)
	
	assert.Equal(t, ErrCodeInternalError, err.Code)
	assert.Equal(t, "wrapped error", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestBaseError_Error(t *testing.T) {
	// 测试没有原因的错误
	err1 := NewError(ErrCodeInvalidParams, "test message")
	assert.Equal(t, "[20000] test message", err1.Error())
	
	// 测试有原因的错误
	cause := fmt.Errorf("original error")
	err2 := NewErrorWithCause(ErrCodeInternalError, "wrapped error", cause)
	assert.Equal(t, "[10000] wrapped error: original error", err2.Error())
}

func TestBaseError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := NewErrorWithCause(ErrCodeInternalError, "wrapped error", cause)
	
	assert.Equal(t, cause, err.Unwrap())
}

func TestBaseError_Is(t *testing.T) {
	err1 := NewError(ErrCodeInvalidParams, "message 1")
	err2 := NewError(ErrCodeInvalidParams, "message 2")
	err3 := NewError(ErrCodeInternalError, "message 3")
	
	assert.True(t, err1.Is(err2))
	assert.False(t, err1.Is(err3))
	assert.False(t, err1.Is(fmt.Errorf("other error")))
}

func TestBaseError_ToJSON(t *testing.T) {
	now := time.Now()
	err := &BaseError{
		Code:      ErrCodeInvalidParams,
		Message:   "test message",
		Details:   map[string]interface{}{"key": "value"},
		Timestamp: now,
		TraceID:   "trace-123",
		Operation: "test-operation",
	}
	
	data, jsonErr := err.ToJSON()
	require.NoError(t, jsonErr)
	
	var result map[string]interface{}
	jsonErr = json.Unmarshal(data, &result)
	require.NoError(t, jsonErr)
	
	assert.Equal(t, float64(20000), result["code"])
	assert.Equal(t, "test message", result["message"])
	assert.Equal(t, "trace-123", result["trace_id"])
	assert.Equal(t, "test-operation", result["operation"])
	assert.NotNil(t, result["details"])
}

func TestBaseError_WithMethods(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := NewError(ErrCodeInvalidParams, "test message")
	
	// 测试WithCause
	errWithCause := err.WithCause(cause)
	assert.Equal(t, cause, errWithCause.Cause)
	assert.NotSame(t, err, errWithCause) // 应该是新的实例
	
	// 测试WithDetail
	errWithDetail := err.WithDetail("key", "value")
	assert.Equal(t, "value", errWithDetail.Details["key"])
	assert.NotSame(t, err, errWithDetail)
	
	// 测试WithTraceID
	errWithTrace := err.WithTraceID("trace-123")
	assert.Equal(t, "trace-123", errWithTrace.TraceID)
	assert.NotSame(t, err, errWithTrace)
	
	// 测试WithOperation
	errWithOp := err.WithOperation("test-op")
	assert.Equal(t, "test-op", errWithOp.Operation)
	assert.NotSame(t, err, errWithOp)
}

func TestWrapError(t *testing.T) {
	// 测试nil错误
	wrapped := WrapError(nil, ErrCodeInternalError, "wrapper message")
	assert.Nil(t, wrapped)
	
	// 测试普通错误
	originalErr := fmt.Errorf("original error")
	wrapped = WrapError(originalErr, ErrCodeInternalError, "wrapper message")
	assert.Equal(t, ErrCodeInternalError, wrapped.Code)
	assert.Equal(t, "wrapper message", wrapped.Message)
	assert.Equal(t, originalErr, wrapped.Cause)
	
	// 测试BaseError
	baseErr := NewError(ErrCodeInvalidParams, "base error")
	wrapped = WrapError(baseErr, ErrCodeInternalError, "wrapper message")
	assert.Equal(t, ErrCodeInvalidParams, wrapped.Code) // 保持原有的错误码
	assert.Equal(t, baseErr, wrapped.Cause)
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected int
	}{
		{"internal error", ErrCodeInternalError, http.StatusInternalServerError},
		{"invalid params", ErrCodeInvalidParams, http.StatusBadRequest},
		{"unauthorized", ErrCodeUnauthorized, http.StatusUnauthorized},
		{"permission denied", ErrCodePermissionDenied, http.StatusForbidden},
		{"resource not found", ErrCodeResourceNotFound, http.StatusNotFound},
		{"resource exists", ErrCodeResourceExists, http.StatusConflict},
		{"quota exceeded", ErrCodeQuotaExceeded, http.StatusTooManyRequests},
		{"external service", ErrCodeExternalService, http.StatusBadGateway},
		{"unknown code", ErrorCode(99999), http.StatusInternalServerError},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := getHTTPStatus(tt.code)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	// 测试系统错误
	assert.Equal(t, ErrCodeInternalError, InternalError("test").Code)
	assert.Equal(t, ErrCodeServiceUnavailable, ServiceUnavailable("test").Code)
	assert.Equal(t, ErrCodeTimeout, Timeout("test").Code)
	assert.Equal(t, ErrCodeDatabaseError, DatabaseError("test").Code)
	assert.Equal(t, ErrCodeCacheError, CacheError("test").Code)
	assert.Equal(t, ErrCodeConfigError, ConfigError("test").Code)
	
	// 测试参数错误
	assert.Equal(t, ErrCodeInvalidParams, InvalidParams("test").Code)
	assert.Equal(t, ErrCodeMissingParams, MissingParams("field").Code)
	assert.Equal(t, ErrCodeInvalidFormat, InvalidFormat("field", "format").Code)
	assert.Equal(t, ErrCodeInvalidRange, InvalidRange("field", 1, 10).Code)
	assert.Equal(t, ErrCodeDuplicateValue, DuplicateValue("field").Code)
	assert.Equal(t, ErrCodeInvalidOperation, InvalidOperation("op").Code)
	
	// 测试认证错误
	assert.Equal(t, ErrCodeUnauthorized, Unauthorized("test").Code)
	assert.Equal(t, ErrCodeInvalidToken, InvalidToken("test").Code)
	assert.Equal(t, ErrCodeTokenExpired, TokenExpired().Code)
	assert.Equal(t, ErrCodePermissionDenied, PermissionDenied("resource").Code)
	assert.Equal(t, ErrCodeAccountLocked, AccountLocked("reason").Code)
	assert.Equal(t, ErrCodeInvalidCredentials, InvalidCredentials().Code)
	
	// 测试业务错误
	assert.Equal(t, ErrCodeResourceNotFound, ResourceNotFound("user", 123).Code)
	assert.Equal(t, ErrCodeResourceExists, ResourceExists("user", 123).Code)
	assert.Equal(t, ErrCodeOperationFailed, OperationFailed("create", "reason").Code)
	assert.Equal(t, ErrCodeBusinessLogic, BusinessLogic("test").Code)
	assert.Equal(t, ErrCodeQuotaExceeded, QuotaExceeded("resource", 100).Code)
	assert.Equal(t, ErrCodeWorkflowError, WorkflowError("stage", "message").Code)
	
	// 测试外部错误
	assert.Equal(t, ErrCodeExternalService, ExternalService("service", "message").Code)
	assert.Equal(t, ErrCodeThirdPartyAPI, ThirdPartyAPI("api", "message").Code)
	assert.Equal(t, ErrCodeNetworkError, NetworkError("message").Code)
	assert.Equal(t, ErrCodeUpstreamError, UpstreamError("upstream", "message").Code)
	assert.Equal(t, ErrCodeIntegrationError, IntegrationError("integration", "message").Code)
	assert.Equal(t, ErrCodeWebhookError, WebhookError("webhook", "message").Code)
}

func TestErrorCollector(t *testing.T) {
	collector := NewErrorCollector()
	
	// 初始状态
	assert.False(t, collector.HasErrors())
	assert.Empty(t, collector.Errors())
	assert.Nil(t, collector.First())
	assert.Empty(t, collector.Error())
	
	// 添加错误
	err1 := NewError(ErrCodeInvalidParams, "error 1")
	err2 := fmt.Errorf("error 2")
	
	collector.Add(err1)
	collector.Add(err2)
	collector.Add(nil) // 应该被忽略
	
	// 验证状态
	assert.True(t, collector.HasErrors())
	assert.Len(t, collector.Errors(), 2)
	assert.Equal(t, err1, collector.First())
	assert.Contains(t, collector.Error(), "error 1")
	
	// 测试多错误
	multiError := collector.ToMultiError()
	assert.NotNil(t, multiError)
	assert.Equal(t, 2, multiError.Count)
	assert.Len(t, multiError.Errors, 2)
}

func TestMultiError(t *testing.T) {
	err1 := NewError(ErrCodeInvalidParams, "error 1")
	err2 := NewError(ErrCodeInternalError, "error 2")
	
	// 测试单个错误
	multiError := &MultiError{
		Errors: []*BaseError{err1},
		Count:  1,
	}
	assert.Equal(t, "[20000] error 1", multiError.Error())
	
	// 测试多个错误
	multiError = &MultiError{
		Errors: []*BaseError{err1, err2},
		Count:  2,
	}
	assert.Contains(t, multiError.Error(), "multiple errors (2)")
	assert.Contains(t, multiError.Error(), "error 1")
	
	// 测试空错误
	multiError = &MultiError{
		Errors: []*BaseError{},
		Count:  0,
	}
	assert.Empty(t, multiError.Error())
	
	// 测试JSON序列化
	data, err := multiError.ToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestErrorResponse(t *testing.T) {
	baseErr := NewError(ErrCodeInvalidParams, "test error")
	response := NewErrorResponse(baseErr, "trace-123", "/api/test", "POST")
	
	assert.Equal(t, baseErr, response.Error)
	assert.Equal(t, "trace-123", response.TraceID)
	assert.Equal(t, "/api/test", response.Path)
	assert.Equal(t, "POST", response.Method)
	
	// 测试JSON序列化
	data, err := response.ToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestDefaultErrorMiddleware(t *testing.T) {
	middleware := &DefaultErrorMiddleware{}
	
	// 测试nil错误
	result := middleware.HandleError(nil)
	assert.Nil(t, result)
	
	// 测试BaseError
	baseErr := NewError(ErrCodeInvalidParams, "test error")
	result = middleware.HandleError(baseErr)
	assert.Equal(t, baseErr, result)
	
	// 测试Kratos错误
	kratosErr := errors.BadRequest("INVALID_PARAM", "invalid parameter")
	result = middleware.HandleError(kratosErr)
	assert.Equal(t, ErrCodeInvalidParams, result.Code)
	assert.Contains(t, result.Message, "invalid parameter")
	
	// 测试普通错误
	normalErr := fmt.Errorf("normal error")
	result = middleware.HandleError(normalErr)
	assert.Equal(t, ErrCodeInternalError, result.Code)
	assert.Equal(t, "normal error", result.Message)
	assert.Equal(t, normalErr, result.Cause)
}

func TestParseKratosErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		code     int32
		expected ErrorCode
	}{
		{"bad request", 400, ErrCodeInvalidParams},
		{"unauthorized", 401, ErrCodeUnauthorized},
		{"forbidden", 403, ErrCodePermissionDenied},
		{"not found", 404, ErrCodeResourceNotFound},
		{"conflict", 409, ErrCodeResourceExists},
		{"too many requests", 429, ErrCodeQuotaExceeded},
		{"internal server error", 500, ErrCodeInternalError},
		{"bad gateway", 502, ErrCodeUpstreamError},
		{"service unavailable", 503, ErrCodeServiceUnavailable},
		{"gateway timeout", 504, ErrCodeTimeout},
		{"unknown", 999, ErrCodeInternalError},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKratosErrorCode(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKratosErrorConversion(t *testing.T) {
	// 测试ToKratosError
	baseErr := NewError(ErrCodeInvalidParams, "test message")
	kratosErr := ToKratosError(baseErr)
	assert.NotNil(t, kratosErr)
	assert.Equal(t, int32(400), kratosErr.Code)
	assert.Equal(t, "20000", kratosErr.Reason)
	assert.Equal(t, "test message", kratosErr.Message)
	
	// 测试nil转换
	kratosErr = ToKratosError(nil)
	assert.Nil(t, kratosErr)
	
	// 测试FromKratosError
	kratosErr = errors.BadRequest("INVALID_PARAM", "invalid parameter")
	baseErr = FromKratosError(kratosErr)
	assert.NotNil(t, baseErr)
	assert.Equal(t, ErrCodeInvalidParams, baseErr.Code)
	assert.Equal(t, "invalid parameter", baseErr.Message)
	assert.Equal(t, "INVALID_PARAM", baseErr.Details["reason"])
	
	// 测试nil转换
	baseErr = FromKratosError(nil)
	assert.Nil(t, baseErr)
}

func TestErrorClassification(t *testing.T) {
	// 测试可重试错误
	retryableErrors := []*BaseError{
		Timeout("timeout"),
		ServiceUnavailable("unavailable"),
		NetworkError("network error"),
		UpstreamError("upstream", "error"),
	}
	
	for _, err := range retryableErrors {
		assert.True(t, IsRetryable(err), "Expected %v to be retryable", err.Code)
	}
	
	// 测试不可重试错误
	nonRetryableErrors := []*BaseError{
		InvalidParams("invalid"),
		PermissionDenied("denied"),
		ResourceNotFound("user", 123),
	}
	
	for _, err := range nonRetryableErrors {
		assert.False(t, IsRetryable(err), "Expected %v to not be retryable", err.Code)
	}
	
	// 测试永久性错误
	permanentErrors := []*BaseError{
		InvalidParams("invalid"),
		PermissionDenied("denied"),
		ResourceNotFound("user", 123),
		InvalidCredentials(),
	}
	
	for _, err := range permanentErrors {
		assert.True(t, IsPermanent(err), "Expected %v to be permanent", err.Code)
	}
	
	// 测试非永久性错误
	nonPermanentErrors := []*BaseError{
		Timeout("timeout"),
		ServiceUnavailable("unavailable"),
		DatabaseError("db error"),
	}
	
	for _, err := range nonPermanentErrors {
		assert.False(t, IsPermanent(err), "Expected %v to not be permanent", err.Code)
	}
}

func TestGetErrorCode(t *testing.T) {
	// 测试BaseError
	baseErr := NewError(ErrCodeInvalidParams, "test")
	assert.Equal(t, ErrCodeInvalidParams, GetErrorCode(baseErr))
	
	// 测试普通错误
	normalErr := fmt.Errorf("normal error")
	assert.Equal(t, ErrCodeInternalError, GetErrorCode(normalErr))
}

func TestGetHTTPStatusHelper(t *testing.T) {
	// 测试BaseError
	baseErr := NewError(ErrCodeInvalidParams, "test")
	assert.Equal(t, http.StatusBadRequest, GetHTTPStatus(baseErr))
	
	// 测试普通错误
	normalErr := fmt.Errorf("normal error")
	assert.Equal(t, http.StatusInternalServerError, GetHTTPStatus(normalErr))
}

// BenchmarkNewError 性能基准测试
func BenchmarkNewError(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewError(ErrCodeInvalidParams, "test message")
	}
}

func BenchmarkErrorWithCause(b *testing.B) {
	cause := fmt.Errorf("original error")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewErrorWithCause(ErrCodeInternalError, "wrapped error", cause)
	}
}

func BenchmarkErrorJSON(b *testing.B) {
	err := NewError(ErrCodeInvalidParams, "test message").
		WithDetail("key", "value").
		WithTraceID("trace-123")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = err.ToJSON()
	}
}