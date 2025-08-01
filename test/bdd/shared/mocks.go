package shared

import (
	"context"
	"time"

	"kratos-boilerplate/internal/biz"

	"github.com/stretchr/testify/mock"
)

// MockUserRepo 模拟用户仓储
type MockUserRepo struct {
	mock.Mock
}

func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{}
}

func (m *MockUserRepo) CreateUser(ctx context.Context, user *biz.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) GetUser(ctx context.Context, username string) (*biz.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.User), args.Error(1)
}

func (m *MockUserRepo) GetUserByEmail(ctx context.Context, email string) (*biz.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.User), args.Error(1)
}

func (m *MockUserRepo) GetUserByPhone(ctx context.Context, phone string) (*biz.User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.User), args.Error(1)
}

func (m *MockUserRepo) GetUserByName(ctx context.Context, name string) (*biz.User, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.User), args.Error(1)
}

func (m *MockUserRepo) UpdateUser(ctx context.Context, user *biz.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepo) GetLock(ctx context.Context, username string) (*biz.AccountLock, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.AccountLock), args.Error(1)
}

func (m *MockUserRepo) SaveLock(ctx context.Context, lock *biz.AccountLock) error {
	args := m.Called(ctx, lock)
	return args.Error(0)
}

func (m *MockUserRepo) RemoveLock(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}

func (m *MockUserRepo) SaveRefreshToken(ctx context.Context, username, tokenID string, expiresAt time.Time) error {
	args := m.Called(ctx, username, tokenID, expiresAt)
	return args.Error(0)
}

func (m *MockUserRepo) GetRefreshToken(ctx context.Context, tokenID string) (string, bool, error) {
	args := m.Called(ctx, tokenID)
	return args.String(0), args.Bool(1), args.Error(2)
}

func (m *MockUserRepo) InvalidateRefreshToken(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *MockUserRepo) InvalidateAllRefreshTokens(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}

func (m *MockUserRepo) SaveCaptcha(ctx context.Context, captcha *biz.Captcha) error {
	args := m.Called(ctx, captcha)
	return args.Error(0)
}

func (m *MockUserRepo) GetCaptcha(ctx context.Context, captchaID string) (*biz.Captcha, error) {
	args := m.Called(ctx, captchaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.Captcha), args.Error(1)
}

func (m *MockUserRepo) MarkCaptchaUsed(ctx context.Context, captchaID string) error {
	args := m.Called(ctx, captchaID)
	return args.Error(0)
}

// MockCaptchaRepo 模拟验证码仓储
type MockCaptchaRepo struct {
	mock.Mock
}

func NewMockCaptchaRepo() *MockCaptchaRepo {
	return &MockCaptchaRepo{}
}

func (m *MockCaptchaRepo) SaveCaptcha(ctx context.Context, captcha *biz.Captcha) error {
	args := m.Called(ctx, captcha)
	return args.Error(0)
}

func (m *MockCaptchaRepo) GetCaptcha(ctx context.Context, captchaID string) (*biz.Captcha, error) {
	args := m.Called(ctx, captchaID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.Captcha), args.Error(1)
}

func (m *MockCaptchaRepo) MarkCaptchaUsed(ctx context.Context, captchaID string) error {
	args := m.Called(ctx, captchaID)
	return args.Error(0)
}

// MockCaptchaService 模拟验证码服务
type MockCaptchaService struct {
	mock.Mock
}

func NewMockCaptchaService() *MockCaptchaService {
	return &MockCaptchaService{}
}

func (m *MockCaptchaService) Generate(ctx context.Context, captchaType, target string) (*biz.Captcha, error) {
	args := m.Called(ctx, captchaType, target)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.Captcha), args.Error(1)
}

func (m *MockCaptchaService) Verify(ctx context.Context, captchaID, captchaCode string) (bool, error) {
	args := m.Called(ctx, captchaID, captchaCode)
	return args.Bool(0), args.Error(1)
}

// MockOperationLogRepo 模拟操作日志仓储
type MockOperationLogRepo struct {
	mock.Mock
}

func NewMockOperationLogRepo() *MockOperationLogRepo {
	return &MockOperationLogRepo{}
}

// MockGreeterRepo 模拟Greeter仓储
type MockGreeterRepo struct {
	mock.Mock
}

func NewMockGreeterRepo() *MockGreeterRepo {
	return &MockGreeterRepo{}
}

func (m *MockGreeterRepo) Save(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	args := m.Called(ctx, g)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.Greeter), args.Error(1)
}

func (m *MockGreeterRepo) Update(ctx context.Context, g *biz.Greeter) (*biz.Greeter, error) {
	args := m.Called(ctx, g)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.Greeter), args.Error(1)
}

func (m *MockGreeterRepo) FindByID(ctx context.Context, id int64) (*biz.Greeter, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.Greeter), args.Error(1)
}

func (m *MockGreeterRepo) ListByHello(ctx context.Context, hello string) ([]*biz.Greeter, error) {
	args := m.Called(ctx, hello)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*biz.Greeter), args.Error(1)
}

func (m *MockGreeterRepo) ListAll(ctx context.Context) ([]*biz.Greeter, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*biz.Greeter), args.Error(1)
}