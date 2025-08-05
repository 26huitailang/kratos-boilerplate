package middleware

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOperationLogRepo 模拟操作日志仓库
type MockOperationLogRepo struct {
	mock.Mock
}

func (m *MockOperationLogRepo) Create(ctx context.Context, log *OperationLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

// MockTransporter 模拟传输器
type MockTransporter struct {
	mock.Mock
}

func (m *MockTransporter) Kind() transport.Kind {
	args := m.Called()
	return args.Get(0).(transport.Kind)
}

func (m *MockTransporter) Endpoint() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTransporter) Operation() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTransporter) RequestHeader() transport.Header {
	args := m.Called()
	return args.Get(0).(transport.Header)
}

func (m *MockTransporter) ReplyHeader() transport.Header {
	args := m.Called()
	return args.Get(0).(transport.Header)
}

// TestOperationLogMiddleware 测试操作日志中间件
func TestOperationLogMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		ctxSetup       func() context.Context
		mockSetup      func(*MockOperationLogRepo)
		handlerSetup   func() middleware.Handler
		request        interface{}
		expectedReply  interface{}
		expectedError  error
	}{
		{
			name: "成功记录操作日志",
			ctxSetup: func() context.Context {
				mockTransporter := new(MockTransporter)
				mockTransporter.On("Operation").Return("/api/v1/auth/login")
				mockTransporter.On("Kind").Return(transport.KindHTTP)
				return transport.NewServerContext(context.Background(), mockTransporter)
			},
			mockSetup: func(mockRepo *MockOperationLogRepo) {
				expectedLog := &OperationLog{
					Path:   "/api/v1/auth/login",
					Method: "http",
				}
				mockRepo.On("Create", mock.Anything, expectedLog).Return(nil)
			},
			handlerSetup: func() middleware.Handler {
				return func(ctx context.Context, req interface{}) (interface{}, error) {
					return "success", nil
				}
			},
			request:       "test_request",
			expectedReply: "success",
			expectedError: nil,
		},
		{
			name: "无传输器上下文",
			ctxSetup: func() context.Context {
				return context.Background()
			},
			mockSetup: func(mockRepo *MockOperationLogRepo) {
				// 不应该调用 Create 方法
			},
			handlerSetup: func() middleware.Handler {
				return func(ctx context.Context, req interface{}) (interface{}, error) {
					return "success", nil
				}
			},
			request:       "test_request",
			expectedReply: "success",
			expectedError: nil,
		},
		{
			name: "处理器返回错误",
			ctxSetup: func() context.Context {
				mockTransporter := new(MockTransporter)
				mockTransporter.On("Operation").Return("/api/v1/auth/register")
				mockTransporter.On("Kind").Return(transport.KindGRPC)
				return transport.NewServerContext(context.Background(), mockTransporter)
			},
			mockSetup: func(mockRepo *MockOperationLogRepo) {
				expectedLog := &OperationLog{
					Path:   "/api/v1/auth/register",
					Method: "grpc",
				}
				mockRepo.On("Create", mock.Anything, expectedLog).Return(nil)
			},
			handlerSetup: func() middleware.Handler {
				return func(ctx context.Context, req interface{}) (interface{}, error) {
					return nil, assert.AnError
				}
			},
			request:       "test_request",
			expectedReply: nil,
			expectedError: assert.AnError,
		},
		{
			name: "日志记录失败但不影响主流程",
			ctxSetup: func() context.Context {
				mockTransporter := new(MockTransporter)
				mockTransporter.On("Operation").Return("/api/v1/user/profile")
				mockTransporter.On("Kind").Return(transport.KindHTTP)
				return transport.NewServerContext(context.Background(), mockTransporter)
			},
			mockSetup: func(mockRepo *MockOperationLogRepo) {
				expectedLog := &OperationLog{
					Path:   "/api/v1/user/profile",
					Method: "http",
				}
				// 模拟日志记录失败
				mockRepo.On("Create", mock.Anything, expectedLog).Return(assert.AnError)
			},
			handlerSetup: func() middleware.Handler {
				return func(ctx context.Context, req interface{}) (interface{}, error) {
					return "profile_data", nil
				}
			},
			request:       "test_request",
			expectedReply: "profile_data",
			expectedError: nil, // 日志失败不应影响主流程
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock 仓库
			mockRepo := new(MockOperationLogRepo)
			tt.mockSetup(mockRepo)

			// 创建中间件
			middleware := OperationLogMiddleware(mockRepo)

			// 创建处理器
			handler := tt.handlerSetup()

			// 包装处理器
			wrappedHandler := middleware(handler)

			// 设置上下文
			ctx := tt.ctxSetup()

			// 执行测试
			reply, err := wrappedHandler(ctx, tt.request)

			// 验证结果
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err)
				assert.Equal(t, tt.expectedReply, reply)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedReply, reply)
			}

			// 验证 mock 调用
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestOperationLog 测试操作日志结构
func TestOperationLog(t *testing.T) {
	log := &OperationLog{
		Path:   "/api/v1/test",
		Method: "HTTP",
	}

	assert.Equal(t, "/api/v1/test", log.Path)
	assert.Equal(t, "HTTP", log.Method)
}