package service

import (
	"context"
	"testing"
	"time"

	v1 "kratos-boilerplate/api/auth/v1"
	"kratos-boilerplate/internal/biz"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

// 1. 定义 Mock AuthUsecase
// 这个 mock 结构模拟了 biz.AuthUsecase 的行为，使我们能独立测试 service 层。
type mockAuthUsecase struct {
	mock.Mock
}

func (m *mockAuthUsecase) Login(ctx context.Context, username, password, captchaID, captchaCode, totpCode string) (*biz.TokenPair, error) {
	args := m.Called(ctx, username, password, captchaID, captchaCode, totpCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.TokenPair), args.Error(1)
}

func (m *mockAuthUsecase) Register(ctx context.Context, username, password, email, phone, captchaID, captchaCode string) error {
	args := m.Called(ctx, username, password, email, phone, captchaID, captchaCode)
	return args.Error(0)
}

func (m *mockAuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*biz.TokenPair, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.TokenPair), args.Error(1)
}

func (m *mockAuthUsecase) Logout(ctx context.Context, accessToken string) error {
	args := m.Called(ctx, accessToken)
	return args.Error(0)
}

func (m *mockAuthUsecase) GetCaptcha(ctx context.Context, captchaType, target string) (*biz.Captcha, error) {
	args := m.Called(ctx, captchaType, target)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.Captcha), args.Error(1)
}

func (m *mockAuthUsecase) VerifyCaptcha(ctx context.Context, captchaID, captchaCode string) (bool, error) {
	args := m.Called(ctx, captchaID, captchaCode)
	return args.Bool(0), args.Error(1)
}

func (m *mockAuthUsecase) GetLockStatus(ctx context.Context, username string) (*biz.AccountLock, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*biz.AccountLock), args.Error(1)
}

func (m *mockAuthUsecase) Now() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *mockAuthUsecase) GetMaxLoginAttempts() int32 {
	args := m.Called()
	return args.Get(0).(int32)
}

// TestAuthServiceBDD 是 Ginkgo 测试套件的入口
func TestAuthServiceBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AuthService BDD Suite")
}

// 2. 编写 BDD 测试描述
var _ = Describe("AuthService", func() {
	var (
		authService *AuthService
		mockUsecase *mockAuthUsecase
		ctx         context.Context
	)

	// BeforeEach 在每个 "It" 测试用例执行前运行
	// 用于创建干净的测试环境
	BeforeEach(func() {
		ctx = context.Background()
		mockUsecase = new(mockAuthUsecase)
		authService = NewAuthService(mockUsecase, log.DefaultLogger)
	})

	// Describe 块描述一个具体的功能，例如 "Login"
	Describe("User Login", func() {
		// Context 块描述一个特定的场景
		Context("with valid credentials and captcha", func() {
			// It 块描述该场景下的预期行为
			It("should return a token pair and no error", func() {
				// a. 准备 (Arrange)
				loginReq := &v1.LoginRequest{
					Username:    "testuser",
					Password:    "password123",
					CaptchaId:   "captcha-id",
					CaptchaCode: "123456",
				}
				expectedTokenPair := &biz.TokenPair{
					AccessToken:  "fake-access-token",
					RefreshToken: "fake-refresh-token",
					ExpiresIn:    3600,
				}

				// 设置 mock 预期：当 Login 方法被以特定参数调用时，
				// 返回预设的 token pair 和 nil 错误。
				mockUsecase.On("Login", ctx, loginReq.Username, loginReq.Password, loginReq.CaptchaId, loginReq.CaptchaCode, "").
					Return(expectedTokenPair, nil)

				// b. 执行 (Act)
				tokenPair, err := authService.Login(ctx, loginReq)

				// c. 断言 (Assert)
				// 使用 Gomega 进行断言，可读性更强
				Expect(err).NotTo(HaveOccurred()) // 期望没有错误发生
				Expect(tokenPair).NotTo(BeNil())  // 期望 tokenPair 不为 nil
				Expect(tokenPair.AccessToken).To(Equal(expectedTokenPair.AccessToken))
				Expect(tokenPair.RefreshToken).To(Equal(expectedTokenPair.RefreshToken))
			})
		})

		Context("with an incorrect password", func() {
			It("should return a password incorrect error", func() {
				// a. 准备
				loginReq := &v1.LoginRequest{
					Username:    "testuser",
					Password:    "wrong-password",
					CaptchaId:   "captcha-id",
					CaptchaCode: "123456",
				}
				expectedErr := errors.Unauthorized("PASSWORD_INCORRECT", "密码错误") // 预期的业务错误

				// 设置 mock 预期：返回 nil 和密码错误
				mockUsecase.On("Login", ctx, loginReq.Username, loginReq.Password, loginReq.CaptchaId, loginReq.CaptchaCode, "").
					Return(nil, biz.ErrPasswordIncorrect)

				// b. 执行
				tokenPair, err := authService.Login(ctx, loginReq)

				// c. 断言
				Expect(tokenPair).To(BeNil())      // 期望 tokenPair 为 nil
				Expect(err).To(HaveOccurred())     // 期望有错误发生
				Expect(err.Error()).To(Equal(expectedErr.Error())) // 期望错误类型是我们预期的业务错误
			})
		})

		Context("with an invalid captcha code", func() {
			It("should return a captcha invalid error", func() {
				// a. 准备
				loginReq := &v1.LoginRequest{
					Username:    "testuser",
					Password:    "password123",
					CaptchaId:   "captcha-id",
					CaptchaCode: "wrong-code",
				}
				expectedErr := errors.BadRequest("CAPTCHA_INVALID", "验证码无效")

				// 设置 mock 预期：返回 nil 和验证码错误
				mockUsecase.On("Login", ctx, loginReq.Username, loginReq.Password, loginReq.CaptchaId, loginReq.CaptchaCode, "").
					Return(nil, biz.ErrCaptchaInvalid)

				// b. 执行
				tokenPair, err := authService.Login(ctx, loginReq)

				// c. 断言
				Expect(tokenPair).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(expectedErr.Error()))
			})
		})
	})
})
