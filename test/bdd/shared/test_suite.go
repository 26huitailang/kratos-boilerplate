package shared

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"kratos-boilerplate/internal/biz"
	"kratos-boilerplate/internal/conf"
	"kratos-boilerplate/internal/data"
	"kratos-boilerplate/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestSuite 提供BDD测试的基础设施
type TestSuite struct {
	Logger   log.Logger
	Ctx      context.Context
	Data     *data.Data
	Cleanup  func()
	Services *Services
}

// Services 包含所有服务实例
type Services struct {
	AuthService    *service.AuthService
	GreeterService *service.GreeterService
}

// MockRepositories 包含所有Mock仓储
type MockRepositories struct {
	UserRepo         *MockUserRepo
	CaptchaService   *MockCaptchaService
	OperationLogRepo *MockOperationLogRepo
	GreeterRepo      *MockGreeterRepo
}

// NewTestSuite 创建新的测试套件
func NewTestSuite() *TestSuite {
	return &TestSuite{
		Logger: log.NewStdLogger(os.Stdout),
		Ctx:    context.Background(),
	}
}

// SetupWithMocks 使用Mock设置测试环境
func (ts *TestSuite) SetupWithMocks() *MockRepositories {
	mocks := &MockRepositories{
		UserRepo:         NewMockUserRepo(),
		CaptchaService:   NewMockCaptchaService(),
		OperationLogRepo: NewMockOperationLogRepo(),
		GreeterRepo:      NewMockGreeterRepo(),
	}

	// 创建默认配置
	config := biz.DefaultAuthConfig

	// 创建业务逻辑层
	authUsecase := biz.NewAuthUsecase(mocks.UserRepo, mocks.CaptchaService, config, ts.Logger)
	greeterUsecase := biz.NewGreeterUsecase(mocks.GreeterRepo, ts.Logger)

	// 创建服务层
	ts.Services = &Services{
		AuthService:    service.NewAuthService(authUsecase, ts.Logger),
		GreeterService: service.NewGreeterService(greeterUsecase),
	}

	return mocks
}

// SetupWithRealData 使用真实数据层设置测试环境
func (ts *TestSuite) SetupWithRealData(dataConfig *conf.Data) error {
	var err error
	ts.Data, ts.Cleanup, err = data.NewData(dataConfig, ts.Logger)
	if err != nil {
		return fmt.Errorf("failed to setup data layer: %w", err)
	}

	// 创建仓储
	userRepo, err := data.NewUserRepo(ts.Data, ts.Logger)
	if err != nil {
		return fmt.Errorf("failed to create user repo: %w", err)
	}

	captchaRepo := data.NewCaptchaRepo(ts.Data)
	greeterRepo := data.NewGreeterRepo(ts.Data, ts.Logger)

	// 创建认证配置
	authConfig := biz.DefaultAuthConfig

	// 创建验证码服务（实现CaptchaService接口）
	// 注意：在真实环境中，这里应该使用实际的验证码服务实现
	// 这里我们使用captchaRepo作为CaptchaService的简单实现
	captchaService := &simpleCaptchaService{repo: captchaRepo}

	// 创建业务逻辑层
	authUsecase := biz.NewAuthUsecase(userRepo, captchaService, authConfig, ts.Logger)
	greeterUsecase := biz.NewGreeterUsecase(greeterRepo, ts.Logger)

	// 创建服务层
	ts.Services = &Services{
		AuthService:    service.NewAuthService(authUsecase, ts.Logger),
		GreeterService: service.NewGreeterService(greeterUsecase),
	}

	return nil
}

// simpleCaptchaService 简单的验证码服务实现，用于测试
type simpleCaptchaService struct {
	repo biz.CaptchaRepo
}

func (s *simpleCaptchaService) Generate(ctx context.Context, captchaType, target string) (*biz.Captcha, error) {
	// 生成简单的测试验证码
	captcha := &biz.Captcha{
		ID:       fmt.Sprintf("test_%d", time.Now().UnixNano()),
		Code:     "123456", // 固定验证码用于测试
		Type:     captchaType,
		Target:   target,
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     false,
	}
	
	// 保存到仓储
	err := s.repo.SaveCaptcha(ctx, captcha)
	if err != nil {
		return nil, err
	}
	
	return captcha, nil
}

func (s *simpleCaptchaService) Verify(ctx context.Context, captchaID, captchaCode string) (bool, error) {
	// 从仓储获取验证码
	captcha, err := s.repo.GetCaptcha(ctx, captchaID)
	if err != nil {
		return false, err
	}
	
	// 检查验证码是否已使用
	if captcha.Used {
		return false, biz.ErrCaptchaInvalid
	}
	
	// 检查验证码是否过期
	if time.Now().After(captcha.ExpireAt) {
		return false, biz.ErrCaptchaExpired
	}
	
	// 验证验证码
	if captcha.Code != captchaCode {
		return false, nil
	}
	
	// 标记验证码为已使用
	err = s.repo.MarkCaptchaUsed(ctx, captchaID)
	if err != nil {
		return false, err
	}
	
	return true, nil
}

// Teardown 清理测试环境
func (ts *TestSuite) Teardown() {
	if ts.Cleanup != nil {
		ts.Cleanup()
	}
}

// WaitForCondition 等待条件满足
func (ts *TestSuite) WaitForCondition(condition func() bool, timeout time.Duration, message string) {
	Eventually(condition, timeout, 100*time.Millisecond).Should(BeTrue(), message)
}

// AssertEventually 断言条件最终满足
func (ts *TestSuite) AssertEventually(condition func() bool, timeout time.Duration, message string) {
	Eventually(condition, timeout, 100*time.Millisecond).Should(BeTrue(), message)
}

// AssertConsistently 断言条件持续满足
func (ts *TestSuite) AssertConsistently(condition func() bool, duration time.Duration, message string) {
	Consistently(condition, duration, 100*time.Millisecond).Should(BeTrue(), message)
}

// CreateTestContext 创建带有超时的测试上下文
func (ts *TestSuite) CreateTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ts.Ctx, timeout)
}

// LogInfo 记录信息日志
func (ts *TestSuite) LogInfo(msg string, keyvals ...interface{}) {
	ts.Logger.Log(log.LevelInfo, "msg", msg, keyvals)
}

// LogError 记录错误日志
func (ts *TestSuite) LogError(msg string, err error, keyvals ...interface{}) {
	keyvals = append(keyvals, "error", err)
	ts.Logger.Log(log.LevelError, "msg", msg, keyvals)
}

// BeforeEachSetup 在每个测试用例前执行的通用设置
func BeforeEachSetup() *TestSuite {
	ts := NewTestSuite()
	GinkgoWriter.Printf("Setting up test suite at %s\n", time.Now().Format(time.RFC3339))
	return ts
}

// AfterEachTeardown 在每个测试用例后执行的通用清理
func AfterEachTeardown(ts *TestSuite) {
	GinkgoWriter.Printf("Tearing down test suite at %s\n", time.Now().Format(time.RFC3339))
	if ts != nil {
		ts.Teardown()
	}
}

// RegisterBDDSuite 注册BDD测试套件
func RegisterBDDSuite(t *testing.T, suiteName string) {
	RegisterFailHandler(Fail)
	RunSpecs(t, suiteName)
}