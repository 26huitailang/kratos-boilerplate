package service

import (
	"context"
	"os"
	"testing"
	"time"

	v1 "kratos-boilerplate/api/auth/v1"
	"kratos-boilerplate/internal/biz"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/go-kratos/kratos/v2/metadata"
)

// MockAuthUsecase 是 AuthUsecase 的 mock 实现
type MockAuthUsecase struct {
	mock.Mock
}

// TestAuthService_Logout 测试用户退出登录功能
func TestAuthService_Logout(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	logger := log.NewStdLogger(os.Stdout)
	service := NewAuthService(mockUC, logger)

	tests := []struct {
		name           string
		request        *v1.LogoutRequest
		ctxSetup       func() context.Context
		mockSetup      func()
		expectedError  error
		expectedReply  *v1.LogoutReply
	}{
		{
			name:    "成功退出登录",
			request: &v1.LogoutRequest{},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{
					"Authorization": {"Bearer valid_token_123"},
				})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup: func() {
				mockUC.On("Logout", mock.Anything, "valid_token_123").Return(nil)
			},
			expectedError: nil,
			expectedReply: &v1.LogoutReply{Success: true},
		},
		{
			name:    "缺少元数据",
			request: &v1.LogoutRequest{},
			ctxSetup: func() context.Context {
				return context.Background()
			},
			mockSetup:     func() {},
			expectedError: errors.Unauthorized("UNAUTHORIZED", "未授权访问"),
			expectedReply: nil,
		},
		{
			name:    "缺少访问令牌",
			request: &v1.LogoutRequest{},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup:     func() {},
			expectedError: errors.Unauthorized("TOKEN_MISSING", "缺少访问令牌"),
			expectedReply: nil,
		},
		{
			name:    "访问令牌格式错误",
			request: &v1.LogoutRequest{},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{
					"Authorization": {"InvalidFormat"},
				})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup:     func() {},
			expectedError: errors.Unauthorized("INVALID_TOKEN_FORMAT", "访问令牌格式错误"),
			expectedReply: nil,
		},
		{
			name:    "访问令牌无效",
			request: &v1.LogoutRequest{},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{
					"Authorization": {"Bearer invalid_token"},
				})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup: func() {
				mockUC.On("Logout", mock.Anything, "invalid_token").Return(biz.ErrTokenInvalid)
			},
			expectedError: errors.Unauthorized("TOKEN_INVALID", "访问令牌无效"),
			expectedReply: nil,
		},
		{
			name:    "访问令牌已过期",
			request: &v1.LogoutRequest{},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{
					"Authorization": {"Bearer expired_token"},
				})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup: func() {
				mockUC.On("Logout", mock.Anything, "expired_token").Return(biz.ErrTokenExpired)
			},
			expectedError: errors.Unauthorized("TOKEN_EXPIRED", "访问令牌已过期"),
			expectedReply: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置 mock
			mockUC.ExpectedCalls = nil
			mockUC.Calls = nil

			// 设置 mock 期望
			tt.mockSetup()

			// 设置上下文
			ctx := tt.ctxSetup()

			// 执行测试
			reply, err := service.Logout(ctx, tt.request)

			// 验证结果
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, reply)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedReply, reply)
			}

			// 验证 mock 调用
			mockUC.AssertExpectations(t)
		})
	}
}

// TestAuthService_RefreshToken 测试刷新令牌功能
func TestAuthService_RefreshToken(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	logger := log.NewStdLogger(os.Stdout)
	service := NewAuthService(mockUC, logger)

	tests := []struct {
		name           string
		request        *v1.RefreshTokenRequest
		mockSetup      func()
		expectedError  error
		expectedReply  *v1.RefreshTokenReply
	}{
		{
			name:    "成功刷新令牌",
			request: &v1.RefreshTokenRequest{RefreshToken: "valid_refresh_token"},
			mockSetup: func() {
				mockTokenPair := &biz.TokenPair{
					AccessToken:  "new_access_token",
					RefreshToken: "new_refresh_token",
					ExpiresIn:    3600,
				}
				mockUC.On("RefreshToken", mock.Anything, "valid_refresh_token").Return(mockTokenPair, nil)
			},
			expectedError: nil,
			expectedReply: &v1.RefreshTokenReply{
				AccessToken:  "new_access_token",
				RefreshToken: "new_refresh_token",
				ExpiresIn:    3600,
			},
		},
		{
			name:          "刷新令牌为空",
			request:       &v1.RefreshTokenRequest{RefreshToken: ""},
			mockSetup:     func() {},
			expectedError: errors.BadRequest("REFRESH_TOKEN_REQUIRED", "刷新令牌不能为空"),
			expectedReply: nil,
		},
		{
			name:    "刷新令牌无效",
			request: &v1.RefreshTokenRequest{RefreshToken: "invalid_refresh_token"},
			mockSetup: func() {
				mockUC.On("RefreshToken", mock.Anything, "invalid_refresh_token").Return(nil, biz.ErrTokenInvalid)
			},
			expectedError: errors.Unauthorized("TOKEN_INVALID", "刷新令牌无效"),
			expectedReply: nil,
		},
		{
			name:    "刷新令牌已过期",
			request: &v1.RefreshTokenRequest{RefreshToken: "expired_refresh_token"},
			mockSetup: func() {
				mockUC.On("RefreshToken", mock.Anything, "expired_refresh_token").Return(nil, biz.ErrTokenExpired)
			},
			expectedError: errors.Unauthorized("TOKEN_EXPIRED", "刷新令牌已过期"),
			expectedReply: nil,
		},
		{
			name:    "用户不存在",
			request: &v1.RefreshTokenRequest{RefreshToken: "token_for_nonexistent_user"},
			mockSetup: func() {
				mockUC.On("RefreshToken", mock.Anything, "token_for_nonexistent_user").Return(nil, biz.ErrUserNotFound)
			},
			expectedError: errors.NotFound("USER_NOT_FOUND", "用户不存在"),
			expectedReply: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置 mock
			mockUC.ExpectedCalls = nil
			mockUC.Calls = nil

			// 设置 mock 期望
			tt.mockSetup()

			// 执行测试
			reply, err := service.RefreshToken(context.Background(), tt.request)

			// 验证结果
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, reply)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedReply, reply)
			}

			// 验证 mock 调用
			mockUC.AssertExpectations(t)
		})
	}
}

// TestAuthService_LockStatus 测试账户锁定状态功能
func TestAuthService_LockStatus(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	logger := log.NewStdLogger(os.Stdout)
	service := NewAuthService(mockUC, logger)

	tests := []struct {
		name           string
		request        *v1.LockStatusRequest
		ctxSetup       func() context.Context
		mockSetup      func()
		expectedError  error
		expectedReply  *v1.LockStatusReply
	}{
		{
			name:    "成功获取锁定状态",
			request: &v1.LockStatusRequest{Username: "testuser"},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{
					"Authorization": {"Bearer valid_token_123"},
				})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup: func() {
				mockLockInfo := &biz.AccountLock{
					Username:       "valid_user",
					FailedAttempts: 0,
					LockUntil:      time.Time{},
					LastAttempt:    time.Time{},
				}
				mockUC.On("GetLockStatus", mock.Anything, "testuser").Return(mockLockInfo, nil)
				mockUC.On("Now").Return(time.Now())
				mockUC.On("GetMaxLoginAttempts").Return(int32(5))
			},
			expectedError: nil,
			expectedReply: &v1.LockStatusReply{
				Locked:         false,
				UnlockTime:     0,
				FailedAttempts: 0,
				MaxAttempts:    5,
			},
		},
		{
			name:    "账户已锁定",
			request: &v1.LockStatusRequest{Username: "testuser"},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{
					"Authorization": {"Bearer locked_user_token"},
				})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup: func() {
				lockTime := time.Now()
				unlockTime := lockTime.Add(30 * time.Minute)
				mockLockInfo := &biz.AccountLock{
					Username:       "locked_user",
					FailedAttempts: 5,
					LockUntil:      unlockTime,
					LastAttempt:    lockTime,
				}
				mockUC.On("GetLockStatus", mock.Anything, "testuser").Return(mockLockInfo, nil)
				mockUC.On("Now").Return(time.Now())
				mockUC.On("GetMaxLoginAttempts").Return(int32(5))
			},
			expectedError: nil,
			expectedReply: &v1.LockStatusReply{
				Locked:         true,
				UnlockTime:     time.Now().Add(30 * time.Minute).Unix(),
				FailedAttempts: 5,
				MaxAttempts:    5,
			},
		},
		{
			name:    "缺少元数据",
			request: &v1.LockStatusRequest{Username: "testuser"},
			ctxSetup: func() context.Context {
				return context.Background()
			},
			mockSetup:     func() {},
			expectedError: errors.Unauthorized("UNAUTHORIZED", "未授权访问"),
			expectedReply: nil,
		},
		{
			name:    "缺少访问令牌",
			request: &v1.LockStatusRequest{},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup:     func() {},
			expectedError: errors.Unauthorized("TOKEN_MISSING", "缺少访问令牌"),
			expectedReply: nil,
		},
		{
			name:    "访问令牌格式错误",
			request: &v1.LockStatusRequest{},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{
					"Authorization": {"InvalidFormat"},
				})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup:     func() {},
			expectedError: errors.Unauthorized("INVALID_TOKEN_FORMAT", "访问令牌格式错误"),
			expectedReply: nil,
		},
		{
			name:    "访问令牌无效",
			request: &v1.LockStatusRequest{},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{
					"Authorization": {"Bearer invalid_token"},
				})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup: func() {
				mockUC.On("GetLockStatus", mock.Anything, "").Return(nil, biz.ErrTokenInvalid)
			},
			expectedError: errors.Unauthorized("TOKEN_INVALID", "访问令牌无效"),
			expectedReply: nil,
		},
		{
			name:    "用户不存在",
			request: &v1.LockStatusRequest{},
			ctxSetup: func() context.Context {
				md := metadata.New(map[string][]string{
					"Authorization": {"Bearer token_for_nonexistent_user"},
				})
				return metadata.NewServerContext(context.Background(), md)
			},
			mockSetup: func() {
				mockUC.On("GetLockStatus", mock.Anything, "").Return(nil, biz.ErrUserNotFound)
			},
			expectedError: errors.NotFound("USER_NOT_FOUND", "用户不存在"),
			expectedReply: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置 mock
			mockUC.ExpectedCalls = nil
			mockUC.Calls = nil

			// 设置 mock 期望
			tt.mockSetup()

			// 设置上下文
			ctx := tt.ctxSetup()

			// 执行测试
			reply, err := service.LockStatus(ctx, tt.request)

			// 验证结果
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, reply)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedReply, reply)
			}

			// 验证 mock 调用
			mockUC.AssertExpectations(t)
		})
	}
}

// TestAuthService_GetCaptcha 测试获取验证码功能
func TestAuthService_GetCaptcha(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	logger := log.NewStdLogger(os.Stdout)
	service := NewAuthService(mockUC, logger)

	tests := []struct {
		name           string
		request        *v1.GetCaptchaRequest
		mockSetup      func()
		expectedError  error
		expectedReply  *v1.GetCaptchaReply
	}{
		{
			name:    "成功获取验证码",
			request: &v1.GetCaptchaRequest{},
			mockSetup: func() {
				mockCaptcha := &biz.Captcha{
					ID:    "captcha_123",
					Code:     "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
				}
				mockUC.On("GetCaptcha", mock.Anything, mock.Anything, mock.Anything).Return(mockCaptcha, nil)
			},
			expectedError: nil,
			expectedReply: &v1.GetCaptchaReply{
				CaptchaId: "captcha_123",
				ImageData: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
			},
		},
		{
			name:    "生成验证码失败",
			request: &v1.GetCaptchaRequest{},
			mockSetup: func() {
				mockUC.On("GetCaptcha", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.InternalServer("CAPTCHA_GENERATION_FAILED", "验证码生成失败"))
			},
			expectedError: errors.InternalServer("CAPTCHA_GENERATION_FAILED", "验证码生成失败"),
			expectedReply: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置 mock
			mockUC.ExpectedCalls = nil
			mockUC.Calls = nil

			// 设置 mock 期望
			tt.mockSetup()

			// 执行测试
			reply, err := service.GetCaptcha(context.Background(), tt.request)

			// 验证结果
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, reply)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedReply, reply)
			}

			// 验证 mock 调用
			mockUC.AssertExpectations(t)
		})
	}
}

// TestAuthService_VerifyCaptcha 测试验证码验证功能
func TestAuthService_VerifyCaptcha(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	logger := log.NewStdLogger(os.Stdout)
	service := NewAuthService(mockUC, logger)

	tests := []struct {
		name           string
		request        *v1.VerifyCaptchaRequest
		mockSetup      func()
		expectedError  error
		expectedReply  *v1.VerifyCaptchaReply
	}{
		{
			name: "验证码验证成功",
			request: &v1.VerifyCaptchaRequest{
				CaptchaId:   "captcha_123",
				CaptchaCode: "ABCD",
			},
			mockSetup: func() {
				mockUC.On("VerifyCaptcha", mock.Anything, "captcha_123", "ABCD").Return(true, nil)
			},
			expectedError: nil,
			expectedReply: &v1.VerifyCaptchaReply{Success: true},
		},
		{
			name: "验证码错误",
			request: &v1.VerifyCaptchaRequest{
				CaptchaId:   "captcha_123",
				CaptchaCode: "WRONG",
			},
			mockSetup: func() {
				mockUC.On("VerifyCaptcha", mock.Anything, "captcha_123", "WRONG").Return(false, nil)
			},
			expectedError: nil,
			expectedReply: &v1.VerifyCaptchaReply{Success: false},
		},
		{
			name: "验证码ID为空",
			request: &v1.VerifyCaptchaRequest{
				CaptchaId:   "",
				CaptchaCode: "ABCD",
			},
			mockSetup:     func() {},
			expectedError: errors.BadRequest("CAPTCHA_ID_REQUIRED", "验证码ID不能为空"),
			expectedReply: nil,
		},
		{
			name: "验证码为空",
			request: &v1.VerifyCaptchaRequest{
				CaptchaId:   "captcha_123",
				CaptchaCode: "",
			},
			mockSetup:     func() {},
			expectedError: errors.BadRequest("CAPTCHA_CODE_REQUIRED", "验证码不能为空"),
			expectedReply: nil,
		},
		{
			name: "验证码已过期",
			request: &v1.VerifyCaptchaRequest{
				CaptchaId:   "expired_captcha",
				CaptchaCode: "ABCD",
			},
			mockSetup: func() {
				mockUC.On("VerifyCaptcha", mock.Anything, "expired_captcha", "ABCD").Return(false, biz.ErrCaptchaExpired)
			},
			expectedError: errors.BadRequest("CAPTCHA_EXPIRED", "验证码已过期"),
			expectedReply: nil,
		},
		{
			name: "验证码不存在",
			request: &v1.VerifyCaptchaRequest{
				CaptchaId:   "nonexistent_captcha",
				CaptchaCode: "ABCD",
			},
			mockSetup: func() {
				mockUC.On("VerifyCaptcha", mock.Anything, "nonexistent_captcha", "ABCD").Return(false, biz.ErrCaptchaInvalid)
			},
			expectedError: errors.BadRequest("CAPTCHA_INVALID", "验证码无效"),
			expectedReply: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置 mock
			mockUC.ExpectedCalls = nil
			mockUC.Calls = nil

			// 设置 mock 期望
			tt.mockSetup()

			// 执行测试
			reply, err := service.VerifyCaptcha(context.Background(), tt.request)

			// 验证结果
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, reply)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedReply, reply)
			}

			// 验证 mock 调用
			mockUC.AssertExpectations(t)
		})
	}
}

func (m *MockAuthUsecase) Register(ctx context.Context, username, password, email, phone, captchaID, captchaCode string) error {
	args := m.Called(ctx, username, password, email, phone, captchaID, captchaCode)
	return args.Error(0)
}

func (m *MockAuthUsecase) Login(ctx context.Context, username, password, captchaID, captchaCode, totpCode string) (*biz.TokenPair, error) {
	args := m.Called(ctx, username, password, captchaID, captchaCode, totpCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.TokenPair), args.Error(1)
}

func (m *MockAuthUsecase) Logout(ctx context.Context, accessToken string) error {
	args := m.Called(ctx, accessToken)
	return args.Error(0)
}

func (m *MockAuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*biz.TokenPair, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.TokenPair), args.Error(1)
}

func (m *MockAuthUsecase) GetCaptcha(ctx context.Context, captchaType, target string) (*biz.Captcha, error) {
	args := m.Called(ctx, captchaType, target)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.Captcha), args.Error(1)
}

func (m *MockAuthUsecase) VerifyCaptcha(ctx context.Context, captchaID, captchaCode string) (bool, error) {
	args := m.Called(ctx, captchaID, captchaCode)
	return args.Bool(0), args.Error(1)
}

func (m *MockAuthUsecase) GetLockStatus(ctx context.Context, username string) (*biz.AccountLock, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.AccountLock), args.Error(1)
}

func (m *MockAuthUsecase) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *MockAuthUsecase) GetMaxLoginAttempts() int32 {
	args := m.Called()
	return args.Get(0).(int32)
}

// TestAuthService_Register 测试用户注册功能
func TestAuthService_Register(t *testing.T) {
	mockUC := new(MockAuthUsecase)
	logger := log.NewStdLogger(os.Stdout)
	service := NewAuthService(mockUC, logger)

	tests := []struct {
		name           string
		request        *v1.RegisterRequest
		mockSetup      func()
		expectedError  error
		expectedReply  *v1.RegisterReply
	}{
		{
			name: "成功注册",
			request: &v1.RegisterRequest{
				Username:    "testuser",
				Password:    "password123",
				Email:       "test@example.com",
				Phone:       "13800138000",
				CaptchaId:   "captcha123",
				CaptchaCode: "1234",
			},
			mockSetup: func() {
				mockUC.On("Register", mock.Anything, "testuser", "password123", "test@example.com", "13800138000", "captcha123", "1234").Return(nil)
			},
			expectedError: nil,
			expectedReply: &v1.RegisterReply{Message: "注册成功"},
		},
		{
			name: "用户已存在",
			request: &v1.RegisterRequest{
				Username:    "existinguser",
				Password:    "password123",
				Email:       "existing@example.com",
				Phone:       "13800138001",
				CaptchaId:   "captcha123",
				CaptchaCode: "1234",
			},
			mockSetup: func() {
				mockUC.On("Register", mock.Anything, "existinguser", "password123", "existing@example.com", "13800138001", "captcha123", "1234").Return(biz.ErrUserExists)
			},
			expectedError: errors.BadRequest("USER_EXISTS", "用户名已存在"),
			expectedReply: nil,
		},
		{
			name: "验证码必填",
			request: &v1.RegisterRequest{
				Username: "testuser",
				Password: "password123",
				Email:    "test@example.com",
				Phone:    "13800138000",
			},
			mockSetup: func() {
				mockUC.On("Register", mock.Anything, "testuser", "password123", "test@example.com", "13800138000", "", "").Return(biz.ErrCaptchaRequired)
			},
			expectedError: errors.BadRequest("CAPTCHA_REQUIRED", "验证码必填"),
			expectedReply: nil,
		},
		{
			name: "验证码无效",
			request: &v1.RegisterRequest{
				Username:    "testuser",
				Password:    "password123",
				Email:       "test@example.com",
				Phone:       "13800138000",
				CaptchaId:   "captcha123",
				CaptchaCode: "wrong",
			},
			mockSetup: func() {
				mockUC.On("Register", mock.Anything, "testuser", "password123", "test@example.com", "13800138000", "captcha123", "wrong").Return(biz.ErrCaptchaInvalid)
			},
			expectedError: errors.BadRequest("CAPTCHA_INVALID", "验证码无效"),
			expectedReply: nil,
		},
		{
			name: "验证码已过期",
			request: &v1.RegisterRequest{
				Username:    "testuser",
				Password:    "password123",
				Email:       "test@example.com",
				Phone:       "13800138000",
				CaptchaId:   "expired123",
				CaptchaCode: "1234",
			},
			mockSetup: func() {
				mockUC.On("Register", mock.Anything, "testuser", "password123", "test@example.com", "13800138000", "expired123", "1234").Return(biz.ErrCaptchaExpired)
			},
			expectedError: errors.BadRequest("CAPTCHA_EXPIRED", "验证码已过期"),
			expectedReply: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 重置 mock
			mockUC.ExpectedCalls = nil
			mockUC.Calls = nil

			// 设置 mock 期望
			tt.mockSetup()

			// 执行测试
			reply, err := service.Register(context.Background(), tt.request)

			// 验证结果
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				assert.Nil(t, reply)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedReply, reply)
			}

			// 验证 mock 调用
			mockUC.AssertExpectations(t)
		})
	}
}