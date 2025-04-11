package captcha

import (
	"context"
	"os"
	"testing"
	"time"

	"xredline/internal/biz"

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
