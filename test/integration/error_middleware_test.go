package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kratos-boilerplate/internal/pkg/errors"

	"github.com/go-kratos/kratos/v2/log"
)

// TestErrorMiddlewareIntegration 测试错误处理中间件集成
func TestErrorMiddlewareIntegration(t *testing.T) {
	logger := log.NewStdLogger(nil)

	t.Run("ErrorMiddlewareCreation", func(t *testing.T) {
		// 创建错误处理中间件
		errorHandler := errors.NewErrorHandlerMiddleware(logger, true, false)
		validationErrorHandler := errors.NewValidationErrorMiddleware(logger)
		rateLimitErrorHandler := errors.NewRateLimitErrorMiddleware(logger)

		// 验证中间件创建成功
		assert.NotNil(t, errorHandler)
		assert.NotNil(t, validationErrorHandler)
		assert.NotNil(t, rateLimitErrorHandler)

		// 验证中间件方法可用
		assert.NotNil(t, errorHandler.Middleware())
		assert.NotNil(t, validationErrorHandler.Middleware())
		assert.NotNil(t, rateLimitErrorHandler.Middleware())
	})

	t.Run("ErrorFormatConsistency", func(t *testing.T) {
		// 测试不同类型的错误都有一致的格式
		testCases := []struct {
			name     string
			errorFn  func() *errors.BaseError
			expected errors.ErrorCode
		}{
			{
				name:     "InvalidParams",
				errorFn:  func() *errors.BaseError { return errors.InvalidParams("test param") },
				expected: errors.ErrCodeInvalidParams,
			},
			{
				name:     "Unauthorized",
				errorFn:  func() *errors.BaseError { return errors.Unauthorized("test auth") },
				expected: errors.ErrCodeUnauthorized,
			},
			{
				name:     "InternalError",
				errorFn:  func() *errors.BaseError { return errors.InternalError("test internal") },
				expected: errors.ErrCodeInternalError,
			},
			{
				name:     "NetworkError",
				errorFn:  func() *errors.BaseError { return errors.NetworkError("test network") },
				expected: errors.ErrCodeNetworkError,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := tc.errorFn()

				assert.Equal(t, tc.expected, err.Code)
				assert.NotEmpty(t, err.Message)
				assert.NotZero(t, err.Timestamp)
				assert.NotZero(t, err.HTTPStatus)

				// 验证错误字符串格式
				errStr := err.Error()
				assert.Contains(t, errStr, "[")
				assert.Contains(t, errStr, "]")
			})
		}
	})

	t.Run("ErrorCollector", func(t *testing.T) {
		collector := errors.NewErrorCollector()
		assert.False(t, collector.HasErrors())

		// 添加错误
		err1 := errors.InvalidParams("param1")
		err2 := errors.InvalidParams("param2")

		collector.Add(err1)
		collector.Add(err2)

		assert.True(t, collector.HasErrors())
		assert.Len(t, collector.Errors(), 2)
		assert.Equal(t, err1, collector.First())

		// 测试多错误转换
		multiError := collector.ToMultiError()
		assert.NotNil(t, multiError)
		assert.Equal(t, 2, multiError.Count)
	})
}

// TestMiddlewareConfiguration 测试中间件配置
func TestMiddlewareConfiguration(t *testing.T) {
	logger := log.NewStdLogger(nil)

	t.Run("MiddlewareParameters", func(t *testing.T) {
		// 测试不同参数配置
		configs := []struct {
			name          string
			enableDetails bool
			enableStack   bool
		}{
			{"detailed_with_stack", true, true},
			{"detailed_no_stack", true, false},
			{"simple_no_stack", false, false},
			{"simple_with_stack", false, true},
		}

		for _, config := range configs {
			t.Run(config.name, func(t *testing.T) {
				middleware := errors.NewErrorHandlerMiddleware(
					logger,
					config.enableDetails,
					config.enableStack,
				)
				assert.NotNil(t, middleware)
				assert.NotNil(t, middleware.Middleware())
			})
		}
	})
}
