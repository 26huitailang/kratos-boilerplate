//go:build e2e

package scenarios_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	v1 "kratos-boilerplate/api/auth/v1"
	"kratos-boilerplate/test/e2e/helpers"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestAuthE2E 是认证端到端测试套件的入口
func TestAuthE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth E2E Suite")
}

var _ = Describe("认证端到端测试", func() {
	var (
		client    *helpers.APIClient
		baseURL   string
		testUser  *helpers.TestUser
		ctx       context.Context
	)

	BeforeSuite(func() {
		// 设置测试环境
		baseURL = helpers.GetTestServerURL()
		client = helpers.NewAPIClient(baseURL)
		ctx = context.Background()

		// 等待服务启动
		Eventually(func() error {
			return client.HealthCheck(ctx)
		}, 30*time.Second, 1*time.Second).Should(Succeed())
	})

	BeforeEach(func() {
		// 为每个测试创建唯一的测试用户
		testUser = helpers.NewTestUser()
	})

	AfterEach(func() {
		// 清理测试数据
		if testUser != nil {
			client.CleanupUser(ctx, testUser.Username)
		}
	})

	Describe("用户注册流程", func() {
		Context("完整的注册流程", func() {
			It("应该能够成功注册新用户", func() {
				By("获取图片验证码")
				captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(captchaResp.CaptchaId).NotTo(BeEmpty())
				Expect(captchaResp.ImageData).NotTo(BeEmpty())

				By("使用验证码注册用户")
				registerResp, err := client.Register(ctx, &v1.RegisterRequest{
					Username:    testUser.Username,
					Password:    testUser.Password,
					Email:       testUser.Email,
					Phone:       testUser.Phone,
					CaptchaId:   captchaResp.CaptchaId,
					CaptchaCode: "123456", // 在测试环境中使用固定验证码
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(registerResp).NotTo(BeNil())

				By("验证用户已成功创建")
				// 尝试登录验证用户是否创建成功
				captchaResp2, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())

				loginResp, err := client.Login(ctx, &v1.LoginRequest{
					Username:    testUser.Username,
					Password:    testUser.Password,
					CaptchaId:   captchaResp2.CaptchaId,
					CaptchaCode: "123456",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(loginResp.AccessToken).NotTo(BeEmpty())
				Expect(loginResp.RefreshToken).NotTo(BeEmpty())
			})
		})

		Context("注册验证", func() {
			It("应该拒绝重复的用户名", func() {
				By("首次注册用户")
				captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = client.Register(ctx, &v1.RegisterRequest{
					Username:    testUser.Username,
					Password:    testUser.Password,
					Email:       testUser.Email,
					Phone:       testUser.Phone,
					CaptchaId:   captchaResp.CaptchaId,
					CaptchaCode: "123456",
				})
				Expect(err).NotTo(HaveOccurred())

				By("尝试使用相同用户名再次注册")
				captchaResp2, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = client.Register(ctx, &v1.RegisterRequest{
					Username:    testUser.Username,
					Password:    "different_password",
					Email:       "different@example.com",
					Phone:       "13800138001",
					CaptchaId:   captchaResp2.CaptchaId,
					CaptchaCode: "123456",
				})
				Expect(err).To(HaveOccurred())
				Expect(helpers.IsHTTPError(err, http.StatusBadRequest)).To(BeTrue())
			})

			It("应该拒绝无效的验证码", func() {
				captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = client.Register(ctx, &v1.RegisterRequest{
					Username:    testUser.Username,
					Password:    testUser.Password,
					Email:       testUser.Email,
					Phone:       testUser.Phone,
					CaptchaId:   captchaResp.CaptchaId,
					CaptchaCode: "wrong_code",
				})
				Expect(err).To(HaveOccurred())
				Expect(helpers.IsHTTPError(err, http.StatusBadRequest)).To(BeTrue())
			})
		})
	})

	Describe("用户登录流程", func() {
		BeforeEach(func() {
			// 为登录测试预先创建用户
			captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
				CaptchaType: "image",
				Target:      "",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Register(ctx, &v1.RegisterRequest{
				Username:    testUser.Username,
				Password:    testUser.Password,
				Email:       testUser.Email,
				Phone:       testUser.Phone,
				CaptchaId:   captchaResp.CaptchaId,
				CaptchaCode: "123456",
			})
			Expect(err).NotTo(HaveOccurred())
		})

		Context("成功登录", func() {
			It("应该返回有效的令牌", func() {
				captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())

				loginResp, err := client.Login(ctx, &v1.LoginRequest{
					Username:    testUser.Username,
					Password:    testUser.Password,
					CaptchaId:   captchaResp.CaptchaId,
					CaptchaCode: "123456",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(loginResp.AccessToken).NotTo(BeEmpty())
				Expect(loginResp.RefreshToken).NotTo(BeEmpty())
				Expect(loginResp.ExpiresIn).To(BeNumerically(">", 0))

				// 保存令牌供后续测试使用
				testUser.AccessToken = loginResp.AccessToken
				testUser.RefreshToken = loginResp.RefreshToken
			})
		})

		Context("登录失败", func() {
			It("应该拒绝错误的密码", func() {
				captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = client.Login(ctx, &v1.LoginRequest{
					Username:    testUser.Username,
					Password:    "wrong_password",
					CaptchaId:   captchaResp.CaptchaId,
					CaptchaCode: "123456",
				})
				Expect(err).To(HaveOccurred())
				Expect(helpers.IsHTTPError(err, http.StatusUnauthorized)).To(BeTrue())
			})

			It("应该在多次失败后锁定账户", func() {
				// 多次尝试错误密码
				for i := 0; i < 5; i++ {
					captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
						CaptchaType: "image",
						Target:      "",
					})
					Expect(err).NotTo(HaveOccurred())

					_, err = client.Login(ctx, &v1.LoginRequest{
						Username:    testUser.Username,
						Password:    "wrong_password",
						CaptchaId:   captchaResp.CaptchaId,
						CaptchaCode: "123456",
					})
					Expect(err).To(HaveOccurred())
				}

				// 检查账户锁定状态
				lockStatus, err := client.GetLockStatus(ctx, &v1.LockStatusRequest{
					Username: testUser.Username,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(lockStatus.IsLocked).To(BeTrue())
				Expect(lockStatus.FailedAttempts).To(BeNumerically(">=", 5))

				// 即使使用正确密码也应该被拒绝
				captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = client.Login(ctx, &v1.LoginRequest{
					Username:    testUser.Username,
					Password:    testUser.Password,
					CaptchaId:   captchaResp.CaptchaId,
					CaptchaCode: "123456",
				})
				Expect(err).To(HaveOccurred())
				Expect(helpers.IsHTTPError(err, http.StatusForbidden)).To(BeTrue())
			})
		})
	})

	Describe("令牌管理", func() {
		var accessToken, refreshToken string

		BeforeEach(func() {
			// 创建用户并登录
			captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
				CaptchaType: "image",
				Target:      "",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Register(ctx, &v1.RegisterRequest{
				Username:    testUser.Username,
				Password:    testUser.Password,
				Email:       testUser.Email,
				Phone:       testUser.Phone,
				CaptchaId:   captchaResp.CaptchaId,
				CaptchaCode: "123456",
			})
			Expect(err).NotTo(HaveOccurred())

			captchaResp2, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
				CaptchaType: "image",
				Target:      "",
			})
			Expect(err).NotTo(HaveOccurred())

			loginResp, err := client.Login(ctx, &v1.LoginRequest{
				Username:    testUser.Username,
				Password:    testUser.Password,
				CaptchaId:   captchaResp2.CaptchaId,
				CaptchaCode: "123456",
			})
			Expect(err).NotTo(HaveOccurred())

			accessToken = loginResp.AccessToken
			refreshToken = loginResp.RefreshToken
		})

		Context("刷新令牌", func() {
			It("应该能够使用刷新令牌获取新的访问令牌", func() {
				refreshResp, err := client.RefreshToken(ctx, &v1.RefreshTokenRequest{
					RefreshToken: refreshToken,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(refreshResp.AccessToken).NotTo(BeEmpty())
				Expect(refreshResp.RefreshToken).NotTo(BeEmpty())
				Expect(refreshResp.AccessToken).NotTo(Equal(accessToken)) // 新令牌应该不同
			})
		})

		Context("退出登录", func() {
			It("应该能够成功退出登录", func() {
				logoutResp, err := client.Logout(ctx, &v1.LogoutRequest{
					AccessToken: accessToken,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(logoutResp).NotTo(BeNil())

				// 退出后，刷新令牌应该无效
				_, err = client.RefreshToken(ctx, &v1.RefreshTokenRequest{
					RefreshToken: refreshToken,
				})
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("验证码功能", func() {
		Context("图片验证码", func() {
			It("应该能够获取图片验证码", func() {
				captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(captchaResp.CaptchaId).NotTo(BeEmpty())
				Expect(captchaResp.ImageData).NotTo(BeEmpty())

				// 验证图片数据是base64格式
				Expect(captchaResp.ImageData).To(HavePrefix("data:image/"))
			})
		})

		Context("验证码验证", func() {
			It("应该能够验证正确的验证码", func() {
				captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())

				verifyResp, err := client.VerifyCaptcha(ctx, &v1.VerifyCaptchaRequest{
					CaptchaId:   captchaResp.CaptchaId,
					CaptchaCode: "123456", // 测试环境固定验证码
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(verifyResp.Success).To(BeTrue())
			})

			It("应该拒绝错误的验证码", func() {
				captchaResp, err := client.GetCaptcha(ctx, &v1.GetCaptchaRequest{
					CaptchaType: "image",
					Target:      "",
				})
				Expect(err).NotTo(HaveOccurred())

				verifyResp, err := client.VerifyCaptcha(ctx, &v1.VerifyCaptchaRequest{
					CaptchaId:   captchaResp.CaptchaId,
					CaptchaCode: "wrong_code",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(verifyResp.Success).To(BeFalse())
			})
		})
	})
})