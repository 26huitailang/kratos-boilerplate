package captcha

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"kratos-boilerplate/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 模拟UserRepo
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) CreateUser(ctx context.Context, user *biz.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) GetUser(ctx context.Context, username string) (*biz.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.User), args.Error(1)
}

func (m *mockUserRepo) GetUserByEmail(ctx context.Context, email string) (*biz.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.User), args.Error(1)
}

func (m *mockUserRepo) GetUserByPhone(ctx context.Context, phone string) (*biz.User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.User), args.Error(1)
}

func (m *mockUserRepo) GetUserByName(ctx context.Context, name string) (*biz.User, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.User), args.Error(1)
}

func (m *mockUserRepo) UpdateUser(ctx context.Context, user *biz.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) GetLock(ctx context.Context, username string) (*biz.AccountLock, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.AccountLock), args.Error(1)
}

func (m *mockUserRepo) SaveLock(ctx context.Context, lock *biz.AccountLock) error {
	args := m.Called(ctx, lock)
	return args.Error(0)
}

func (m *mockUserRepo) RemoveLock(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}

func (m *mockUserRepo) SaveRefreshToken(ctx context.Context, username, tokenID string, expiresAt time.Time) error {
	args := m.Called(ctx, username, tokenID, expiresAt)
	return args.Error(0)
}

func (m *mockUserRepo) GetRefreshToken(ctx context.Context, tokenID string) (string, bool, error) {
	args := m.Called(ctx, tokenID)
	return args.String(0), args.Bool(1), args.Error(2)
}

func (m *mockUserRepo) InvalidateRefreshToken(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *mockUserRepo) InvalidateAllRefreshTokens(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}

func (m *mockUserRepo) SaveCaptcha(ctx context.Context, captcha *biz.Captcha) error {
	args := m.Called(ctx, captcha)
	return args.Error(0)
}

func (m *mockUserRepo) GetCaptcha(ctx context.Context, captchaID string) (*biz.Captcha, error) {
	args := m.Called(ctx, captchaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.Captcha), args.Error(1)
}

func (m *mockUserRepo) MarkCaptchaUsed(ctx context.Context, captchaID string) error {
	args := m.Called(ctx, captchaID)
	return args.Error(0)
}

// 表驱动TDD测试示例
func TestMultiply_TableDriven(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"2*3", 2, 3, 6},
		{"4*5", 4, 5, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a * tt.b
			assert.Equal(t, tt.want, got)
		})
	}
}

// 测试用例
func TestGenerate_Image(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	repo.On("SaveCaptcha", mock.Anything, mock.AnythingOfType("*biz.Captcha")).Return(nil)

	// 创建验证码服务
	config := DefaultConfig
	service := NewCaptchaService(repo, config, logger)

	// 执行测试
	result, err := service.Generate(context.Background(), "image", "")

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "image", result.Type)
	assert.NotEmpty(t, result.ID)
	assert.NotEmpty(t, result.Code) // 对于图片验证码，这应该是base64的图片数据

	repo.AssertExpectations(t)
}

func TestGenerate_SMS(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	repo.On("SaveCaptcha", mock.Anything, mock.AnythingOfType("*biz.Captcha")).Return(nil)

	// 创建验证码服务
	config := DefaultConfig
	service := NewCaptchaService(repo, config, logger)

	// 执行测试
	result, err := service.Generate(context.Background(), "sms", "13800138000")

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "sms", result.Type)
	assert.Equal(t, "13800138000", result.Target)
	assert.NotEmpty(t, result.ID)
	assert.Empty(t, result.Code) // 短信验证码不应该返回实际验证码内容

	repo.AssertExpectations(t)
}

func TestGenerate_Email(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	repo.On("SaveCaptcha", mock.Anything, mock.AnythingOfType("*biz.Captcha")).Return(nil)

	// 创建验证码服务
	config := DefaultConfig
	service := NewCaptchaService(repo, config, logger)

	// 执行测试
	result, err := service.Generate(context.Background(), "email", "test@example.com")

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "email", result.Type)
	assert.Equal(t, "test@example.com", result.Target)
	assert.NotEmpty(t, result.ID)
	assert.Empty(t, result.Code) // 邮件验证码不应该返回实际验证码内容

	repo.AssertExpectations(t)
}

func TestGenerate_UnsupportedType(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 创建验证码服务
	config := DefaultConfig
	service := NewCaptchaService(repo, config, logger)

	// 执行测试
	result, err := service.Generate(context.Background(), "unsupported", "")

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不支持的验证码类型")
}

func TestGenerate_SaveFailed(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为 - 保存失败
	repo.On("SaveCaptcha", mock.Anything, mock.AnythingOfType("*biz.Captcha")).Return(assert.AnError)

	// 创建验证码服务
	config := DefaultConfig
	service := NewCaptchaService(repo, config, logger)

	// 执行测试
	result, err := service.Generate(context.Background(), "image", "")

	// 验证结果
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "保存验证码失败")

	repo.AssertExpectations(t)
}

func TestVerify_Success(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	captcha := &biz.Captcha{
		ID:       "captcha123",
		Code:     "123456",
		Type:     "image",
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     false,
	}

	repo.On("GetCaptcha", mock.Anything, "captcha123").Return(captcha, nil)
	repo.On("MarkCaptchaUsed", mock.Anything, "captcha123").Return(nil)

	// 创建验证码服务
	config := DefaultConfig
	service := NewCaptchaService(repo, config, logger)

	// 执行测试
	result, err := service.Verify(context.Background(), "captcha123", "123456")

	// 验证结果
	assert.NoError(t, err)
	assert.True(t, result)

	repo.AssertExpectations(t)
}

func TestVerify_InvalidCode(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	captcha := &biz.Captcha{
		ID:       "captcha123",
		Code:     "123456",
		Type:     "image",
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     false,
	}

	repo.On("GetCaptcha", mock.Anything, "captcha123").Return(captcha, nil)

	// 创建验证码服务
	config := DefaultConfig
	service := NewCaptchaService(repo, config, logger)

	// 执行测试
	result, err := service.Verify(context.Background(), "captcha123", "wrongcode")

	// 验证结果
	assert.NoError(t, err)
	assert.False(t, result)

	repo.AssertExpectations(t)
}

func TestVerify_Expired(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为 - 验证码已过期
	captcha := &biz.Captcha{
		ID:       "captcha123",
		Code:     "123456",
		Type:     "image",
		ExpireAt: time.Now().Add(-5 * time.Minute), // 过期的验证码
		Used:     false,
	}

	repo.On("GetCaptcha", mock.Anything, "captcha123").Return(captcha, nil)

	// 创建验证码服务
	config := DefaultConfig
	service := NewCaptchaService(repo, config, logger)

	// 执行测试
	result, err := service.Verify(context.Background(), "captcha123", "123456")

	// 验证结果
	assert.Error(t, err)
	assert.Equal(t, biz.ErrCaptchaExpired, err)
	assert.False(t, result)

	repo.AssertExpectations(t)
}

func TestVerify_AlreadyUsed(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为 - 验证码已使用
	captcha := &biz.Captcha{
		ID:       "captcha123",
		Code:     "123456",
		Type:     "image",
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     true, // 已使用的验证码
	}

	repo.On("GetCaptcha", mock.Anything, "captcha123").Return(captcha, nil)

	// 创建验证码服务
	config := DefaultConfig
	service := NewCaptchaService(repo, config, logger)

	// 执行测试
	result, err := service.Verify(context.Background(), "captcha123", "123456")

	// 验证结果
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "验证码已使用")
	assert.False(t, result)

	repo.AssertExpectations(t)
}

// 测试配置相关功能
func TestNewCaptchaService_WithNilConfig(t *testing.T) {
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	service := NewCaptchaService(repo, nil, logger)
	assert.NotNil(t, service)
}

func TestNewCaptchaService_WithZeroExpiration(t *testing.T) {
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)
	config := &Config{
		EnableSMS:   true,
		EnableEmail: true,
		EnableImage: true,
		Expiration:  0, // 零值过期时间
	}

	service := NewCaptchaService(repo, config, logger)
	assert.NotNil(t, service)
}

// 测试禁用功能的情况
func TestGenerate_SMSDisabled(t *testing.T) {
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)
	config := &Config{
		EnableSMS:   false, // 禁用短信
		EnableEmail: true,
		EnableImage: true,
		Expiration:  5 * time.Minute,
	}

	service := NewCaptchaService(repo, config, logger)
	result, err := service.Generate(context.Background(), "sms", "13800138000")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "短信验证码功能未启用")
}

func TestGenerate_EmailDisabled(t *testing.T) {
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)
	config := &Config{
		EnableSMS:   true,
		EnableEmail: false, // 禁用邮件
		EnableImage: true,
		Expiration:  5 * time.Minute,
	}

	service := NewCaptchaService(repo, config, logger)
	result, err := service.Generate(context.Background(), "email", "test@example.com")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "邮件验证码功能未启用")
}

func TestGenerate_ImageDisabled(t *testing.T) {
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)
	config := &Config{
		EnableSMS:   true,
		EnableEmail: true,
		EnableImage: false, // 禁用图片
		Expiration:  5 * time.Minute,
	}

	service := NewCaptchaService(repo, config, logger)
	result, err := service.Generate(context.Background(), "image", "")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "图片验证码功能未启用")
}

// 测试验证码验证的边界情况
func TestVerify_EmptyParams(t *testing.T) {
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)
	service := NewCaptchaService(repo, DefaultConfig, logger)

	// 测试空的验证码ID
	result, err := service.Verify(context.Background(), "", "123456")
	assert.Error(t, err)
	assert.Equal(t, biz.ErrCaptchaInvalid, err)
	assert.False(t, result)

	// 测试空的验证码
	result, err = service.Verify(context.Background(), "captcha123", "")
	assert.Error(t, err)
	assert.Equal(t, biz.ErrCaptchaInvalid, err)
	assert.False(t, result)
}

func TestVerify_GetCaptchaFailed(t *testing.T) {
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为 - 获取验证码失败
	repo.On("GetCaptcha", mock.Anything, "captcha123").Return(nil, assert.AnError)

	service := NewCaptchaService(repo, DefaultConfig, logger)
	result, err := service.Verify(context.Background(), "captcha123", "123456")

	assert.Error(t, err)
	assert.False(t, result)
	assert.Contains(t, err.Error(), "获取验证码失败")

	repo.AssertExpectations(t)
}

func TestVerify_MarkUsedFailed(t *testing.T) {
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	captcha := &biz.Captcha{
		ID:       "captcha123",
		Code:     "123456",
		Type:     "image",
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     false,
	}

	repo.On("GetCaptcha", mock.Anything, "captcha123").Return(captcha, nil)
	repo.On("MarkCaptchaUsed", mock.Anything, "captcha123").Return(assert.AnError) // 标记失败

	service := NewCaptchaService(repo, DefaultConfig, logger)
	result, err := service.Verify(context.Background(), "captcha123", "123456")

	// 即使标记失败，验证仍然成功
	assert.NoError(t, err)
	assert.True(t, result)

	repo.AssertExpectations(t)
}

// 测试工具函数
func TestGenerateNumericCode(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length_4", 4},
		{"length_6", 6},
		{"length_8", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := generateNumericCode(tt.length)
			assert.NoError(t, err)
			assert.Len(t, code, tt.length)
			// 验证只包含数字
			for _, char := range code {
				assert.True(t, char >= '0' && char <= '9')
			}
		})
	}
}

func TestGenerateRandomCode(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		charset string
	}{
		{"numeric", 6, "0123456789"},
		{"alpha", 8, "abcdefghijklmnopqrstuvwxyz"},
		{"alphanumeric", 10, "0123456789abcdefghijklmnopqrstuvwxyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := generateRandomCode(tt.length, tt.charset)
			assert.NoError(t, err)
			assert.Len(t, code, tt.length)
			// 验证字符都在指定字符集中
			for _, char := range code {
				assert.Contains(t, tt.charset, string(char))
			}
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []int{8, 16, 32, 64}

	for _, length := range tests {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			str, err := generateRandomString(length)
			assert.NoError(t, err)
			assert.Len(t, str, length)
			// 验证生成的字符串不为空
			assert.NotEmpty(t, str)
		})
	}
}

func TestEqualFold(t *testing.T) {
	tests := []struct {
		name     string
		s, t     string
		expected bool
	}{
		{"same_case", "hello", "hello", true},
		{"different_case", "Hello", "hello", true},
		{"mixed_case", "HeLLo", "hEllO", true},
		{"different_strings", "hello", "world", false},
		{"different_length", "hello", "hell", false},
		{"empty_strings", "", "", true},
		{"one_empty", "hello", "", false},
		{"numbers", "123", "123", true},
		{"mixed_alphanumeric", "Hello123", "hello123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := equalFold(tt.s, tt.t)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// 测试并发安全性
func TestConcurrentGenerate(t *testing.T) {
	repo := new(mockUserRepo)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	repo.On("SaveCaptcha", mock.Anything, mock.AnythingOfType("*biz.Captcha")).Return(nil)

	service := NewCaptchaService(repo, DefaultConfig, logger)

	// 并发生成验证码
	const goroutines = 10
	results := make(chan *biz.Captcha, goroutines)
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			result, err := service.Generate(context.Background(), "image", "")
			if err != nil {
				errors <- err
			} else {
				results <- result
			}
		}()
	}

	// 收集结果
	var captchas []*biz.Captcha
	for i := 0; i < goroutines; i++ {
		select {
		case result := <-results:
			captchas = append(captchas, result)
		case err := <-errors:
			t.Fatalf("并发生成验证码失败: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("并发测试超时")
		}
	}

	// 验证所有验证码ID都是唯一的
	ids := make(map[string]bool)
	for _, captcha := range captchas {
		assert.False(t, ids[captcha.ID], "验证码ID重复: %s", captcha.ID)
		ids[captcha.ID] = true
	}

	repo.AssertExpectations(t)
}
