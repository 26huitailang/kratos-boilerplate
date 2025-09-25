package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
)

// ErrorCode 错误码类型
type ErrorCode int

// 错误码定义
const (
	// 系统错误 10000-19999
	ErrCodeInternalError     ErrorCode = 10000
	ErrCodeServiceUnavailable ErrorCode = 10001
	ErrCodeTimeout           ErrorCode = 10002
	ErrCodeDatabaseError     ErrorCode = 10003
	ErrCodeCacheError        ErrorCode = 10004
	ErrCodeConfigError       ErrorCode = 10005
	
	// 参数错误 20000-29999
	ErrCodeInvalidParams     ErrorCode = 20000
	ErrCodeMissingParams     ErrorCode = 20001
	ErrCodeInvalidFormat     ErrorCode = 20002
	ErrCodeInvalidRange      ErrorCode = 20003
	ErrCodeDuplicateValue    ErrorCode = 20004
	ErrCodeInvalidOperation  ErrorCode = 20005
	
	// 认证错误 30000-39999
	ErrCodeUnauthorized      ErrorCode = 30000
	ErrCodeInvalidToken      ErrorCode = 30001
	ErrCodeTokenExpired      ErrorCode = 30002
	ErrCodePermissionDenied  ErrorCode = 30003
	ErrCodeAccountLocked     ErrorCode = 30004
	ErrCodeInvalidCredentials ErrorCode = 30005
	
	// 业务错误 40000-49999
	ErrCodeResourceNotFound  ErrorCode = 40000
	ErrCodeResourceExists    ErrorCode = 40001
	ErrCodeOperationFailed   ErrorCode = 40002
	ErrCodeBusinessLogic     ErrorCode = 40003
	ErrCodeQuotaExceeded     ErrorCode = 40004
	ErrCodeWorkflowError     ErrorCode = 40005
	
	// 外部错误 50000-59999
	ErrCodeExternalService   ErrorCode = 50000
	ErrCodeThirdPartyAPI     ErrorCode = 50001
	ErrCodeNetworkError      ErrorCode = 50002
	ErrCodeUpstreamError     ErrorCode = 50003
	ErrCodeIntegrationError  ErrorCode = 50004
	ErrCodeWebhookError      ErrorCode = 50005
)

// BaseError 基础错误结构
type BaseError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	Timestamp  time.Time              `json:"timestamp"`
	TraceID    string                 `json:"trace_id,omitempty"`
	Operation  string                 `json:"operation,omitempty"`
	HTTPStatus int                    `json:"-"`
}

// Error 实现error接口
func (e *BaseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 实现errors.Unwrap接口
func (e *BaseError) Unwrap() error {
	return e.Cause
}

// Is 实现errors.Is接口
func (e *BaseError) Is(target error) bool {
	if t, ok := target.(*BaseError); ok {
		return e.Code == t.Code
	}
	return false
}

// ToJSON 转换为JSON格式
func (e *BaseError) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// WithCause 添加原因错误
func (e *BaseError) WithCause(cause error) *BaseError {
	newErr := *e
	newErr.Cause = cause
	return &newErr
}

// WithDetail 添加详细信息
func (e *BaseError) WithDetail(key string, value interface{}) *BaseError {
	newErr := *e
	if newErr.Details == nil {
		newErr.Details = make(map[string]interface{})
	}
	newErr.Details[key] = value
	return &newErr
}

// WithTraceID 添加追踪ID
func (e *BaseError) WithTraceID(traceID string) *BaseError {
	newErr := *e
	newErr.TraceID = traceID
	return &newErr
}

// WithOperation 添加操作信息
func (e *BaseError) WithOperation(operation string) *BaseError {
	newErr := *e
	newErr.Operation = operation
	return &newErr
}

// NewError 创建新错误
func NewError(code ErrorCode, message string) *BaseError {
	return &BaseError{
		Code:       code,
		Message:    message,
		Details:    make(map[string]interface{}),
		Timestamp:  time.Now(),
		HTTPStatus: getHTTPStatus(code),
	}
}

// NewErrorf 创建格式化错误
func NewErrorf(code ErrorCode, format string, args ...interface{}) *BaseError {
	return NewError(code, fmt.Sprintf(format, args...))
}

// NewErrorWithCause 创建带原因的错误
func NewErrorWithCause(code ErrorCode, message string, cause error) *BaseError {
	return NewError(code, message).WithCause(cause)
}

// WrapError 包装已有错误
func WrapError(err error, code ErrorCode, message string) *BaseError {
	if err == nil {
		return nil
	}
	
	// 如果已经是BaseError，更新信息
	if baseErr, ok := err.(*BaseError); ok {
		return baseErr.WithCause(err)
	}
	
	return NewErrorWithCause(code, message, err)
}

// getHTTPStatus 获取HTTP状态码
func getHTTPStatus(code ErrorCode) int {
	switch {
	case code >= 10000 && code < 20000: // 系统错误
		return http.StatusInternalServerError
	case code >= 20000 && code < 30000: // 参数错误
		return http.StatusBadRequest
	case code >= 30000 && code < 40000: // 认证错误
		if code == ErrCodePermissionDenied {
			return http.StatusForbidden
		}
		return http.StatusUnauthorized
	case code >= 40000 && code < 50000: // 业务错误
		if code == ErrCodeResourceNotFound {
			return http.StatusNotFound
		}
		if code == ErrCodeResourceExists {
			return http.StatusConflict
		}
		if code == ErrCodeQuotaExceeded {
			return http.StatusTooManyRequests
		}
		return http.StatusBadRequest
	case code >= 50000 && code < 60000: // 外部错误
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// 预定义错误构造函数

// 系统错误
func InternalError(message string) *BaseError {
	return NewError(ErrCodeInternalError, message)
}

func ServiceUnavailable(message string) *BaseError {
	return NewError(ErrCodeServiceUnavailable, message)
}

func Timeout(message string) *BaseError {
	return NewError(ErrCodeTimeout, message)
}

func DatabaseError(message string) *BaseError {
	return NewError(ErrCodeDatabaseError, message)
}

func CacheError(message string) *BaseError {
	return NewError(ErrCodeCacheError, message)
}

func ConfigError(message string) *BaseError {
	return NewError(ErrCodeConfigError, message)
}

// 参数错误
func InvalidParams(message string) *BaseError {
	return NewError(ErrCodeInvalidParams, message)
}

func MissingParams(field string) *BaseError {
	return NewError(ErrCodeMissingParams, fmt.Sprintf("missing required parameter: %s", field))
}

func InvalidFormat(field string, expected string) *BaseError {
	return NewError(ErrCodeInvalidFormat, fmt.Sprintf("invalid format for %s, expected: %s", field, expected))
}

func InvalidRange(field string, min, max interface{}) *BaseError {
	return NewError(ErrCodeInvalidRange, fmt.Sprintf("value for %s out of range [%v, %v]", field, min, max))
}

func DuplicateValue(field string) *BaseError {
	return NewError(ErrCodeDuplicateValue, fmt.Sprintf("duplicate value for %s", field))
}

func InvalidOperation(operation string) *BaseError {
	return NewError(ErrCodeInvalidOperation, fmt.Sprintf("invalid operation: %s", operation))
}

// 认证错误
func Unauthorized(message string) *BaseError {
	return NewError(ErrCodeUnauthorized, message)
}

func InvalidToken(message string) *BaseError {
	return NewError(ErrCodeInvalidToken, message)
}

func TokenExpired() *BaseError {
	return NewError(ErrCodeTokenExpired, "token has expired")
}

func PermissionDenied(resource string) *BaseError {
	return NewError(ErrCodePermissionDenied, fmt.Sprintf("permission denied for resource: %s", resource))
}

func AccountLocked(reason string) *BaseError {
	return NewError(ErrCodeAccountLocked, fmt.Sprintf("account locked: %s", reason))
}

func InvalidCredentials() *BaseError {
	return NewError(ErrCodeInvalidCredentials, "invalid username or password")
}

// 业务错误
func ResourceNotFound(resource string, id interface{}) *BaseError {
	return NewError(ErrCodeResourceNotFound, fmt.Sprintf("%s not found: %v", resource, id))
}

func ResourceExists(resource string, id interface{}) *BaseError {
	return NewError(ErrCodeResourceExists, fmt.Sprintf("%s already exists: %v", resource, id))
}

func OperationFailed(operation string, reason string) *BaseError {
	return NewError(ErrCodeOperationFailed, fmt.Sprintf("operation %s failed: %s", operation, reason))
}

func BusinessLogic(message string) *BaseError {
	return NewError(ErrCodeBusinessLogic, message)
}

func QuotaExceeded(resource string, limit interface{}) *BaseError {
	return NewError(ErrCodeQuotaExceeded, fmt.Sprintf("quota exceeded for %s, limit: %v", resource, limit))
}

func WorkflowError(stage string, message string) *BaseError {
	return NewError(ErrCodeWorkflowError, fmt.Sprintf("workflow error at %s: %s", stage, message))
}

// 外部错误
func ExternalService(service string, message string) *BaseError {
	return NewError(ErrCodeExternalService, fmt.Sprintf("external service %s error: %s", service, message))
}

func ThirdPartyAPI(api string, message string) *BaseError {
	return NewError(ErrCodeThirdPartyAPI, fmt.Sprintf("third party API %s error: %s", api, message))
}

func NetworkError(message string) *BaseError {
	return NewError(ErrCodeNetworkError, message)
}

func UpstreamError(upstream string, message string) *BaseError {
	return NewError(ErrCodeUpstreamError, fmt.Sprintf("upstream %s error: %s", upstream, message))
}

func IntegrationError(integration string, message string) *BaseError {
	return NewError(ErrCodeIntegrationError, fmt.Sprintf("integration %s error: %s", integration, message))
}

func WebhookError(webhook string, message string) *BaseError {
	return NewError(ErrCodeWebhookError, fmt.Sprintf("webhook %s error: %s", webhook, message))
}

// ErrorCollector 错误收集器
type ErrorCollector struct {
	errors []*BaseError
}

// NewErrorCollector 创建错误收集器
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]*BaseError, 0),
	}
}

// Add 添加错误
func (ec *ErrorCollector) Add(err error) {
	if err == nil {
		return
	}
	
	if baseErr, ok := err.(*BaseError); ok {
		ec.errors = append(ec.errors, baseErr)
	} else {
		ec.errors = append(ec.errors, NewErrorWithCause(ErrCodeInternalError, "unknown error", err))
	}
}

// HasErrors 是否有错误
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Errors 获取所有错误
func (ec *ErrorCollector) Errors() []*BaseError {
	return ec.errors
}

// First 获取第一个错误
func (ec *ErrorCollector) First() *BaseError {
	if len(ec.errors) == 0 {
		return nil
	}
	return ec.errors[0]
}

// Error 实现error接口
func (ec *ErrorCollector) Error() string {
	if len(ec.errors) == 0 {
		return ""
	}
	
	if len(ec.errors) == 1 {
		return ec.errors[0].Error()
	}
	
	return fmt.Sprintf("multiple errors (%d): %s", len(ec.errors), ec.errors[0].Error())
}

// ToMultiError 转换为多错误格式
func (ec *ErrorCollector) ToMultiError() *MultiError {
	if len(ec.errors) == 0 {
		return nil
	}
	
	return &MultiError{
		Errors: ec.errors,
		Count:  len(ec.errors),
	}
}

// MultiError 多错误结构
type MultiError struct {
	Errors []*BaseError `json:"errors"`
	Count  int          `json:"count"`
}

// Error 实现error接口
func (me *MultiError) Error() string {
	if me.Count == 0 {
		return ""
	}
	
	if me.Count == 1 {
		return me.Errors[0].Error()
	}
	
	return fmt.Sprintf("multiple errors (%d): %s", me.Count, me.Errors[0].Error())
}

// ToJSON 转换为JSON
func (me *MultiError) ToJSON() ([]byte, error) {
	return json.Marshal(me)
}

// ErrorResponse HTTP错误响应
type ErrorResponse struct {
	Error   *BaseError `json:"error"`
	TraceID string     `json:"trace_id,omitempty"`
	Path    string     `json:"path,omitempty"`
	Method  string     `json:"method,omitempty"`
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(err *BaseError, traceID, path, method string) *ErrorResponse {
	return &ErrorResponse{
		Error:   err,
		TraceID: traceID,
		Path:    path,
		Method:  method,
	}
}

// ToJSON 转换为JSON
func (er *ErrorResponse) ToJSON() ([]byte, error) {
	return json.Marshal(er)
}

// ErrorMiddleware 错误处理中间件接口
type ErrorMiddleware interface {
	HandleError(err error) *BaseError
}

// DefaultErrorMiddleware 默认错误处理中间件
type DefaultErrorMiddleware struct{}

// HandleError 处理错误
func (m *DefaultErrorMiddleware) HandleError(err error) *BaseError {
	if err == nil {
		return nil
	}
	
	// 如果已经是BaseError，直接返回
	if baseErr, ok := err.(*BaseError); ok {
		return baseErr
	}
	
	// 如果是Kratos错误，转换格式
	if kratosErr := errors.FromError(err); kratosErr != nil {
		code := parseKratosErrorCode(kratosErr.Code)
		return NewError(code, kratosErr.Message).WithDetail("reason", kratosErr.Reason)
	}
	
	// 其他错误包装为内部错误
	return InternalError(err.Error()).WithCause(err)
}

// parseKratosErrorCode 解析Kratos错误码
func parseKratosErrorCode(code int32) ErrorCode {
	switch code {
	case 400:
		return ErrCodeInvalidParams
	case 401:
		return ErrCodeUnauthorized
	case 403:
		return ErrCodePermissionDenied
	case 404:
		return ErrCodeResourceNotFound
	case 409:
		return ErrCodeResourceExists
	case 429:
		return ErrCodeQuotaExceeded
	case 500:
		return ErrCodeInternalError
	case 502:
		return ErrCodeUpstreamError
	case 503:
		return ErrCodeServiceUnavailable
	case 504:
		return ErrCodeTimeout
	default:
		return ErrCodeInternalError
	}
}

// ToKratosError 转换为Kratos错误
func ToKratosError(err *BaseError) *errors.Error {
	if err == nil {
		return nil
	}
	
	return errors.New(int(err.HTTPStatus), strconv.Itoa(int(err.Code)), err.Message)
}

// FromKratosError 从Kratos错误转换
func FromKratosError(err *errors.Error) *BaseError {
	if err == nil {
		return nil
	}
	
	code := parseKratosErrorCode(err.Code)
	return NewError(code, err.Message).WithDetail("reason", err.Reason)
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	if baseErr, ok := err.(*BaseError); ok {
		switch baseErr.Code {
		case ErrCodeTimeout, ErrCodeServiceUnavailable, ErrCodeNetworkError, ErrCodeUpstreamError:
			return true
		}
	}
	return false
}

// IsPermanent 判断错误是否是永久性的
func IsPermanent(err error) bool {
	if baseErr, ok := err.(*BaseError); ok {
		switch baseErr.Code {
		case ErrCodeInvalidParams, ErrCodePermissionDenied, ErrCodeResourceNotFound, ErrCodeInvalidCredentials:
			return true
		}
	}
	return false
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) ErrorCode {
	if baseErr, ok := err.(*BaseError); ok {
		return baseErr.Code
	}
	return ErrCodeInternalError
}

// GetHTTPStatus 获取HTTP状态码
func GetHTTPStatus(err error) int {
	if baseErr, ok := err.(*BaseError); ok {
		return baseErr.HTTPStatus
	}
	return http.StatusInternalServerError
}