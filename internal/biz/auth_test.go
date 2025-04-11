package biz

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 模拟UserRepo
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) CreateUser(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) GetUser(ctx context.Context, username string) (*User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *mockUserRepo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *mockUserRepo) GetUserByPhone(ctx context.Context, phone string) (*User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *mockUserRepo) GetUserByName(ctx context.Context, name string) (*User, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *mockUserRepo) UpdateUser(ctx context.Context, user *User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) GetLock(ctx context.Context, username string) (*AccountLock, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountLock), args.Error(1)
}

func (m *mockUserRepo) SaveLock(ctx context.Context, lock *AccountLock) error {
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

func (m *mockUserRepo) SaveCaptcha(ctx context.Context, captcha *Captcha) error {
	args := m.Called(ctx, captcha)
	return args.Error(0)
}

func (m *mockUserRepo) GetCaptcha(ctx context.Context, captchaID string) (*Captcha, error) {
	args := m.Called(ctx, captchaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Captcha), args.Error(1)
}

func (m *mockUserRepo) MarkCaptchaUsed(ctx context.Context, captchaID string) error {
	args := m.Called(ctx, captchaID)
	return args.Error(0)
}

// 模拟CaptchaService
type mockCaptchaService struct {
	mock.Mock
}

func (m *mockCaptchaService) Generate(ctx context.Context, captchaType, target string) (*Captcha, error) {
	args := m.Called(ctx, captchaType, target)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Captcha), args.Error(1)
}

func (m *mockCaptchaService) Verify(ctx context.Context, captchaID, captchaCode string) (bool, error) {
	args := m.Called(ctx, captchaID, captchaCode)
	return args.Bool(0), args.Error(1)
}

// 创建测试用例
func TestRegister_Success(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	repo.On("GetUser", mock.Anything, "testuser").Return(nil, ErrUserNotFound)
	repo.On("GetUserByEmail", mock.Anything, "test@example.com").Return(nil, ErrUserNotFound)
	repo.On("GetUserByPhone", mock.Anything, "13800138000").Return(nil, ErrUserNotFound)
	repo.On("CreateUser", mock.Anything, mock.AnythingOfType("*biz.User")).Return(nil)
	captchaService.On("Verify", mock.Anything, "captcha123", "123456").Return(true, nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	err := uc.Register(context.Background(), "testuser", "Password123", "test@example.com", "13800138000", "captcha123", "123456")

	// 验证结果
	assert.NoError(t, err)
	repo.AssertExpectations(t)
	captchaService.AssertExpectations(t)
}

func TestRegister_UserExists(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为 - 用户已存在
	existingUser := &User{ID: 1, Username: "testuser"}
	repo.On("GetUser", mock.Anything, "testuser").Return(existingUser, nil)
	captchaService.On("Verify", mock.Anything, "captcha123", "123456").Return(true, nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	err := uc.Register(context.Background(), "testuser", "Password123", "test@example.com", "13800138000", "captcha123", "123456")

	// 验证结果
	assert.Equal(t, ErrUserExists, err)
	repo.AssertExpectations(t)
	captchaService.AssertExpectations(t)
}

func TestRegister_CaptchaInvalid(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为 - 验证码无效
	captchaService.On("Verify", mock.Anything, "captcha123", "123456").Return(false, nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	err := uc.Register(context.Background(), "testuser", "Password123", "test@example.com", "13800138000", "captcha123", "123456")

	// 验证结果
	assert.Equal(t, ErrCaptchaInvalid, err)
	captchaService.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 生成加密密码
	hashedPassword := "$2a$10$abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01" // 模拟加密后的密码

	// 配置模拟行为
	user := &User{
		ID:       1,
		Username: "testuser",
		Password: hashedPassword,
	}

	// 不使用 GetLock 返回 nil，这会导致 SaveLock 不会被调用
	// 而是返回一个有效的锁对象，但锁定时间已过期
	lock := &AccountLock{
		Username:       "testuser",
		FailedAttempts: 1,
		LockUntil:      time.Time{},                // 未锁定
		LastAttempt:    time.Now().Add(-time.Hour), // 上次尝试是很久以前
	}
	repo.On("GetLock", mock.Anything, "testuser").Return(lock, nil)
	repo.On("RemoveLock", mock.Anything, "testuser").Return(nil)
	captchaService.On("Verify", mock.Anything, "captcha123", "123456").Return(true, nil)
	repo.On("GetUser", mock.Anything, "testuser").Return(user, nil)
	repo.On("SaveRefreshToken", mock.Anything, "testuser", mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	config.JWTSecretKey = "test-secret-key" // 使用固定的测试密钥
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	// 修改bcrypt.CompareHashAndPassword的行为进行测试
	// 在实际测试中，我们需要使用真实的bcrypt密码，或者使用测试替身
	originalVerifyPassword := bcryptCompareHashAndPassword
	defer func() { bcryptCompareHashAndPassword = originalVerifyPassword }()

	// 临时替换密码验证函数
	bcryptCompareHashAndPassword = func(hashedPassword, password []byte) error {
		return nil // 模拟密码验证成功
	}

	tokenPair, err := uc.Login(context.Background(), "testuser", "Password123", "captcha123", "123456", "")

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, tokenPair)
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)
	assert.Greater(t, tokenPair.ExpiresIn, int64(0))

	repo.AssertExpectations(t)
	captchaService.AssertExpectations(t)
}

func TestLogin_AccountLocked(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为 - 账户锁定
	lock := &AccountLock{
		Username:       "testuser",
		FailedAttempts: 5,
		LockUntil:      time.Now().Add(time.Hour), // 账户锁定1小时
		LastAttempt:    time.Now(),
	}

	repo.On("GetLock", mock.Anything, "testuser").Return(lock, nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	tokenPair, err := uc.Login(context.Background(), "testuser", "Password123", "captcha123", "123456", "")

	// 验证结果
	assert.Equal(t, ErrAccountLocked, err)
	assert.Nil(t, tokenPair)

	repo.AssertExpectations(t)
}

func TestLogin_InvalidPassword(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	hashedPassword := "$2a$10$abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01" // 模拟加密后的密码
	user := &User{
		ID:       1,
		Username: "testuser",
		Password: hashedPassword,
	}

	repo.On("GetLock", mock.Anything, "testuser").Return(nil, ErrUserNotFound)
	captchaService.On("Verify", mock.Anything, "captcha123", "123456").Return(true, nil)
	repo.On("GetUser", mock.Anything, "testuser").Return(user, nil)
	repo.On("SaveLock", mock.Anything, mock.AnythingOfType("*biz.AccountLock")).Return(nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	// 修改bcrypt.CompareHashAndPassword的行为进行测试
	originalVerifyPassword := bcryptCompareHashAndPassword
	defer func() { bcryptCompareHashAndPassword = originalVerifyPassword }()

	// 临时替换密码验证函数
	bcryptCompareHashAndPassword = func(hashedPassword, password []byte) error {
		return ErrPasswordIncorrect // 模拟密码验证失败
	}

	tokenPair, err := uc.Login(context.Background(), "testuser", "WrongPassword", "captcha123", "123456", "")

	// 验证结果
	assert.Equal(t, ErrPasswordIncorrect, err)
	assert.Nil(t, tokenPair)

	repo.AssertExpectations(t)
	captchaService.AssertExpectations(t)
}

func TestRefreshToken_Success(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	user := &User{
		ID:       1,
		Username: "testuser",
	}

	repo.On("GetRefreshToken", mock.Anything, mock.AnythingOfType("string")).Return("testuser", false, nil)
	repo.On("InvalidateRefreshToken", mock.Anything, mock.AnythingOfType("string")).Return(nil)
	repo.On("GetUser", mock.Anything, "testuser").Return(user, nil)
	repo.On("SaveRefreshToken", mock.Anything, "testuser", mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	config.JWTSecretKey = "test-secret-key" // 使用固定的测试密钥
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	// 生成一个有效的刷新令牌用于测试
	refreshToken, _ := generateTestRefreshToken("testuser", 1, "test-secret-key")

	tokenPair, err := uc.RefreshToken(context.Background(), refreshToken)

	// 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, tokenPair)
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)

	repo.AssertExpectations(t)
}

func TestLogout_Success(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	repo.On("InvalidateAllRefreshTokens", mock.Anything, "testuser").Return(nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	config.JWTSecretKey = "test-secret-key" // 使用固定的测试密钥
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	// 生成一个有效的访问令牌用于测试
	accessToken, _ := generateTestAccessToken("testuser", 1, "test-secret-key")

	err := uc.Logout(context.Background(), accessToken)

	// 验证结果
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestGetCaptcha_Success(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	captcha := &Captcha{
		ID:       "captcha123",
		Code:     "123456",
		Type:     "image",
		ExpireAt: time.Now().Add(5 * time.Minute),
	}

	captchaService.On("Generate", mock.Anything, "image", "").Return(captcha, nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	result, err := uc.GetCaptcha(context.Background(), "image", "")

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, captcha, result)

	captchaService.AssertExpectations(t)
}

func TestVerifyCaptcha_Success(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	captchaService.On("Verify", mock.Anything, "captcha123", "123456").Return(true, nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	valid, err := uc.VerifyCaptcha(context.Background(), "captcha123", "123456")

	// 验证结果
	assert.NoError(t, err)
	assert.True(t, valid)

	captchaService.AssertExpectations(t)
}

func TestGetLockStatus_Success(t *testing.T) {
	// 准备测试依赖
	repo := new(mockUserRepo)
	captchaService := new(mockCaptchaService)
	logger := log.NewStdLogger(os.Stdout)

	// 配置模拟行为
	lock := &AccountLock{
		Username:       "testuser",
		FailedAttempts: 3,
		LockUntil:      time.Time{}, // 未锁定
		LastAttempt:    time.Now().Add(-time.Hour),
	}

	repo.On("GetLock", mock.Anything, "testuser").Return(lock, nil)
	repo.On("RemoveLock", mock.Anything, "testuser").Return(nil)

	// 创建用例并执行
	config := DefaultAuthConfig
	uc := NewAuthUsecase(repo, captchaService, config, logger)

	result, err := uc.GetLockStatus(context.Background(), "testuser")

	// 验证结果
	assert.NoError(t, err)
	// 由于锁定时间已过，应该返回未锁定状态
	expectedResult := &AccountLock{
		Username:       "testuser",
		FailedAttempts: 0,
		LockUntil:      time.Time{},
		LastAttempt:    time.Time{},
	}
	assert.Equal(t, expectedResult, result)

	repo.AssertExpectations(t)
}

// 辅助函数 - 生成测试用访问令牌
func generateTestAccessToken(username string, userID int64, secretKey string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      now.Add(15 * time.Minute).Unix(),
		"iat":      now.Unix(),
		"type":     "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// 辅助函数 - 生成测试用刷新令牌
func generateTestRefreshToken(username string, userID int64, secretKey string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      now.Add(7 * 24 * time.Hour).Unix(),
		"iat":      now.Unix(),
		"type":     "refresh",
		"jti":      "test-token-id",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}
