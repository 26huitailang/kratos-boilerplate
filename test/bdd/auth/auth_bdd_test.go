package auth_test

import (
	"context"
	"testing"
	"time"

	v1 "kratos-boilerplate/api/auth/v1"
	"kratos-boilerplate/internal/biz"
	"kratos-boilerplate/test/bdd/shared"

	"github.com/go-kratos/kratos/v2/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

// TestAuthServiceBDD 是认证服务BDD测试套件的入口
func TestAuthServiceBDD(t *testing.T) {
	shared.RegisterBDDSuite(t, "AuthService BDD Suite")
}

var _ = Describe("认证服务", func() {
	var (
		testSuite *shared.TestSuite
		mocks     *shared.MockRepositories
		ctx       context.Context
	)

	BeforeEach(func() {
		testSuite = shared.BeforeEachSetup()
		mocks = testSuite.SetupWithMocks()
		ctx = testSuite.Ctx
	})

	AfterEach(func() {
		shared.AfterEachTeardown(testSuite)
	})

	Describe("用户登录", func() {
		Context("当提供有效的用户名和密码时", func() {
			It("应该返回访问令牌", func() {
				// 准备测试数据
				loginReq := &v1.LoginRequest{
					Username:    "testuser",
					Password:    "password123",
					CaptchaId:   "captcha-123",
					CaptchaCode: "123456",
					TotpCode:    "", // 不使用TOTP
				}

				user := &biz.User{
					ID:       1,
					Username: "testuser",
					Password: "$2a$10$oX.ALctDnUvY.7OGzQbU/OmVtBMr0KzOyrl3XVppNfuWvt7EoW9M6", // bcrypt哈希 for "password123"
					Email:    "test@example.com",
				}

				// 设置Mock期望
				mocks.UserRepo.On("GetUser", ctx, "testuser").Return(user, nil)
				mocks.UserRepo.On("GetLock", ctx, "testuser").Return(nil, biz.ErrUserNotFound)
				mocks.CaptchaService.On("Verify", ctx, "captcha-123", "123456").Return(true, nil)
				mocks.UserRepo.On("SaveRefreshToken", ctx, "testuser", mock.AnythingOfType("string"), mock.AnythingOfType("time.Time")).Return(nil)

				// 执行测试
				response, err := testSuite.Services.AuthService.Login(ctx, loginReq)

				// 验证结果
				Expect(err).NotTo(HaveOccurred())
				Expect(response).NotTo(BeNil())
				Expect(response.AccessToken).NotTo(BeEmpty())
				Expect(response.RefreshToken).NotTo(BeEmpty())
				Expect(response.ExpiresIn).To(BeNumerically(">", 0))

				// 验证Mock调用
				mocks.UserRepo.AssertExpectations(GinkgoT())
			})
		})

		Context("当提供错误的密码时", func() {
			It("应该返回密码错误", func() {
				// 准备测试数据
				loginReq := &v1.LoginRequest{
					Username:    "testuser",
					Password:    "wrongpassword",
					CaptchaId:   "captcha-123",
					CaptchaCode: "123456",
					TotpCode:    "",
				}

				user := &biz.User{
					ID:       1,
					Username: "testuser",
					Password: "$2a$10$hashedpassword", // bcrypt哈希
					Email:    "test@example.com",
				}

				// 设置Mock期望
				mocks.UserRepo.On("GetUser", ctx, "testuser").Return(user, nil)
				mocks.UserRepo.On("GetLock", ctx, "testuser").Return(nil, biz.ErrUserNotFound)
				mocks.CaptchaService.On("Verify", ctx, "captcha-123", "123456").Return(true, nil)
				mocks.UserRepo.On("SaveLock", ctx, mock.AnythingOfType("*biz.AccountLock")).Return(nil)

				// 执行测试
				response, err := testSuite.Services.AuthService.Login(ctx, loginReq)

				// 验证结果
				Expect(response).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(errors.IsUnauthorized(err)).To(BeTrue())

				// 验证Mock调用
				mocks.UserRepo.AssertExpectations(GinkgoT())
			})
		})

		Context("当验证码无效时", func() {
			It("应该返回验证码错误", func() {
				// 准备测试数据
				loginReq := &v1.LoginRequest{
					Username:    "testuser",
					Password:    "password123",
					CaptchaId:   "captcha-123",
					CaptchaCode: "wrongcode",
					TotpCode:    "",
				}

				// 设置Mock期望
				// 验证码验证失败时，不会执行GetUser，所以只需要设置GetLock和CaptchaService的期望
				mocks.UserRepo.On("GetLock", ctx, "testuser").Return(nil, biz.ErrUserNotFound)
				mocks.CaptchaService.On("Verify", ctx, "captcha-123", "wrongcode").Return(false, nil)

				// 执行测试
				response, err := testSuite.Services.AuthService.Login(ctx, loginReq)

				// 验证结果
				Expect(response).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(errors.IsBadRequest(err)).To(BeTrue())

				// 验证Mock调用
				mocks.UserRepo.AssertExpectations(GinkgoT())
				mocks.CaptchaService.AssertExpectations(GinkgoT())
			})
		})

		Context("当账户被锁定时", func() {
			It("应该返回账户锁定错误", func() {
				// 准备测试数据
				loginReq := &v1.LoginRequest{
					Username:    "testuser",
					Password:    "password123",
					CaptchaId:   "captcha-123",
					CaptchaCode: "123456",
					TotpCode:    "",
				}

				accountLock := &biz.AccountLock{
					Username:       "testuser",
					FailedAttempts: 5,
					LockUntil:      time.Now().Add(30 * time.Minute),
					LastAttempt:    time.Now(),
				}

				// 设置Mock期望
				// 当账户被锁定时，只会检查锁定状态，不会执行后续的验证码验证和获取用户
				mocks.UserRepo.On("GetLock", ctx, "testuser").Return(accountLock, nil)

				// 执行测试
				response, err := testSuite.Services.AuthService.Login(ctx, loginReq)

				// 验证结果
				Expect(response).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(errors.IsForbidden(err)).To(BeTrue())

				// 验证Mock调用
				mocks.UserRepo.AssertExpectations(GinkgoT())
				mocks.CaptchaService.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("用户注册", func() {
		Context("当提供有效的注册信息时", func() {
			It("应该成功创建用户", func() {
				// 准备测试数据
				registerReq := &v1.RegisterRequest{
					Username:    "newuser",
					Password:    "password123",
					Email:       "newuser@example.com",
					Phone:       "13800138000",
					CaptchaId:   "captcha-123",
					CaptchaCode: "123456",
				}

				// 设置Mock期望
				mocks.UserRepo.On("GetUser", ctx, "newuser").Return(nil, biz.ErrUserNotFound)
				mocks.UserRepo.On("GetUserByEmail", ctx, "newuser@example.com").Return(nil, biz.ErrUserNotFound)
				mocks.UserRepo.On("GetUserByPhone", ctx, "13800138000").Return(nil, biz.ErrUserNotFound)
				mocks.CaptchaService.On("Verify", ctx, "captcha-123", "123456").Return(true, nil)
				mocks.UserRepo.On("CreateUser", ctx, mock.AnythingOfType("*biz.User")).Return(nil)

				// 执行测试
				response, err := testSuite.Services.AuthService.Register(ctx, registerReq)

				// 验证结果
				Expect(err).NotTo(HaveOccurred())
				Expect(response).NotTo(BeNil())

				// 验证Mock调用
				mocks.UserRepo.AssertExpectations(GinkgoT())
				mocks.CaptchaService.AssertExpectations(GinkgoT())
			})
		})

		Context("当用户名已存在时", func() {
			It("应该返回用户已存在错误", func() {
				// 准备测试数据
				registerReq := &v1.RegisterRequest{
					Username:    "existinguser",
					Password:    "password123",
					Email:       "existing@example.com",
					Phone:       "13800138000",
					CaptchaId:   "captcha-123",
					CaptchaCode: "123456",
				}

				existingUser := &biz.User{
					ID:       1,
					Username: "existinguser",
					Email:    "existing@example.com",
				}

				// 设置Mock期望
				mocks.CaptchaService.On("Verify", ctx, "captcha-123", "123456").Return(true, nil)
				mocks.UserRepo.On("GetUser", ctx, "existinguser").Return(existingUser, nil)

				// 执行测试
				response, err := testSuite.Services.AuthService.Register(ctx, registerReq)

				// 验证结果
				Expect(response).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(errors.IsBadRequest(err)).To(BeTrue())

				// 验证Mock调用
				mocks.UserRepo.AssertExpectations(GinkgoT())
			})
		})
	})

	Describe("获取验证码", func() {
		Context("当请求图片验证码时", func() {
			It("应该返回验证码信息", func() {
				// 准备测试数据
				captchaReq := &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				}

				expectedCaptcha := &biz.Captcha{
					ID:       "captcha-123",
					Code:     "123456",
					Type:     "image",
					ExpireAt: time.Now().Add(5 * time.Minute),
				}

				// 设置Mock期望
				mocks.CaptchaService.On("Generate", ctx, "image", "").Return(expectedCaptcha, nil)

				// 执行测试
				response, err := testSuite.Services.AuthService.GetCaptcha(ctx, captchaReq)

				// 验证结果
				Expect(err).NotTo(HaveOccurred())
				Expect(response).NotTo(BeNil())
				Expect(response.CaptchaId).To(Equal("captcha-123"))
				Expect(response.ImageData).NotTo(BeEmpty())

				// 验证Mock调用
				mocks.CaptchaService.AssertExpectations(GinkgoT())
			})
		})
	})
})