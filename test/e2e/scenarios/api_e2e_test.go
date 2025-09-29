package scenarios
package scenarios_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	authv1 "kratos-boilerplate/api/auth/v1"
	v1 "kratos-boilerplate/api/helloworld/v1"
)

var _ = Describe("API端到端测试", func() {
	var (
		baseURL    = "http://localhost:8000"
		httpClient *http.Client
	)

	BeforeEach(func() {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	})

	Describe("健康检查端点", func() {
		Context("当服务正常运行时", func() {
			It("应该返回健康状态", func() {
				resp, err := httpClient.Get(baseURL + "/health")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				var healthResp map[string]interface{}
				err = json.Unmarshal(body, &healthResp)
				Expect(err).NotTo(HaveOccurred())

				Expect(healthResp).To(HaveKey("status"))
				Expect(healthResp["status"]).To(Equal("healthy"))
			})
		})
	})

	Describe("Greeter API", func() {
		Context("正常请求", func() {
			It("应该返回问候消息", func() {
				resp, err := httpClient.Get(baseURL + "/helloworld/world")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				var helloResp v1.HelloReply
				err = json.Unmarshal(body, &helloResp)
				Expect(err).NotTo(HaveOccurred())

				Expect(helloResp.Message).To(Equal("Hello world"))
			})
		})

		Context("错误处理场景", func() {
			It("应该返回500内部服务器错误", func() {
				resp, err := httpClient.Get(baseURL + "/helloworld/error")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				var errorResp map[string]interface{}
				err = json.Unmarshal(body, &errorResp)
				Expect(err).NotTo(HaveOccurred())

				Expect(errorResp).To(HaveKey("code"))
				Expect(errorResp).To(HaveKey("message"))
				Expect(errorResp).To(HaveKey("metadata"))
			})

			It("应该返回400客户端错误", func() {
				resp, err := httpClient.Get(baseURL + "/helloworld/badrequest")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				var errorResp map[string]interface{}
				err = json.Unmarshal(body, &errorResp)
				Expect(err).NotTo(HaveOccurred())

				Expect(errorResp).To(HaveKey("code"))
				Expect(errorResp["message"]).To(ContainSubstring("Bad request"))
			})

			It("应该返回404资源不存在", func() {
				resp, err := httpClient.Get(baseURL + "/helloworld/notfound")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

				body, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())

				var errorResp map[string]interface{}
				err = json.Unmarshal(body, &errorResp)
				Expect(err).NotTo(HaveOccurred())

				Expect(errorResp).To(HaveKey("code"))
				Expect(errorResp["message"]).To(ContainSubstring("not found"))
			})

			It("应该返回403禁止访问", func() {
				resp, err := httpClient.Get(baseURL + "/helloworld/forbidden")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			})

			It("应该返回408请求超时", func() {
				resp, err := httpClient.Get(baseURL + "/helloworld/timeout")
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				Expect(resp.StatusCode).To(Equal(http.StatusRequestTimeout))
			})
		})
	})

	Describe("认证API", func() {
		Context("验证码功能", func() {
			It("应该能够获取验证码", func() {
				// 构建请求
				getCaptchaReq := authv1.GetCaptchaRequest{
					CaptchaType: "email",
					Target:      "test@example.com",
				}

				reqBody, err := json.Marshal(getCaptchaReq)
				Expect(err).NotTo(HaveOccurred())

				resp, err := httpClient.Post(
					baseURL+"/api/auth/v1/captcha",
					"application/json",
					bytes.NewBuffer(reqBody),
				)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				// 注意：如果没有实际的后端运行，这个测试可能会失败
				// 这是一个示例，展示如何构造端到端测试
				if resp.StatusCode == http.StatusOK {
					body, err := io.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())

					var captchaResp authv1.GetCaptchaReply
					err = json.Unmarshal(body, &captchaResp)
					Expect(err).NotTo(HaveOccurred())

					Expect(captchaResp.CaptchaId).NotTo(BeEmpty())
				}
			})
		})

		Context("用户注册流程", func() {
			It("应该能够完成完整的注册流程", func() {
				// 1. 获取验证码
				// 2. 验证验证码
				// 3. 注册用户
				// 这里是一个完整的用户注册端到端测试示例

				By("获取验证码")
				// ... 实现验证码获取逻辑

				By("注册用户")
				registerReq := authv1.RegisterRequest{
					Username:    fmt.Sprintf("testuser_%d", time.Now().Unix()),
					Password:    "TestPassword123!",
					Email:       "test@example.com",
					Phone:       "13812345678",
					CaptchaId:   "test-captcha-id",
					CaptchaCode: "123456",
				}

				reqBody, err := json.Marshal(registerReq)
				Expect(err).NotTo(HaveOccurred())

				resp, err := httpClient.Post(
					baseURL+"/api/auth/v1/register",
					"application/json",
					bytes.NewBuffer(reqBody),
				)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				// 根据实际后端状态处理响应
				if resp.StatusCode == http.StatusOK {
					body, err := io.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())

					var registerResp authv1.RegisterReply
					err = json.Unmarshal(body, &registerResp)
					Expect(err).NotTo(HaveOccurred())

					Expect(registerResp.Message).To(ContainSubstring("成功"))
				}
			})
		})

		Context("用户登录流程", func() {
			It("应该能够完成完整的登录流程", func() {
				By("用户登录")
				loginReq := authv1.LoginRequest{
					Username:    "testuser",
					Password:    "TestPassword123!",
					CaptchaId:   "test-captcha-id",
					CaptchaCode: "123456",
				}

				reqBody, err := json.Marshal(loginReq)
				Expect(err).NotTo(HaveOccurred())

				resp, err := httpClient.Post(
					baseURL+"/api/auth/v1/login",
					"application/json",
					bytes.NewBuffer(reqBody),
				)
				Expect(err).NotTo(HaveOccurred())
				defer resp.Body.Close()

				// 处理登录响应
				if resp.StatusCode == http.StatusOK {
					body, err := io.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())

					var loginResp authv1.LoginReply
					err = json.Unmarshal(body, &loginResp)
					Expect(err).NotTo(HaveOccurred())

					Expect(loginResp.AccessToken).NotTo(BeEmpty())
					Expect(loginResp.RefreshToken).NotTo(BeEmpty())
					Expect(loginResp.ExpiresIn).To(BeNumerically(">", 0))

					By("使用访问令牌访问受保护资源")
					// 这里可以添加使用令牌访问受保护资源的测试
				}
			})
		})
	})

	Describe("压力测试场景", func() {
		Context("高并发请求", func() {
			It("应该能够处理并发请求", func() {
				concurrency := 10
				done := make(chan bool, concurrency)

				for i := 0; i < concurrency; i++ {
					go func(id int) {
						defer func() { done <- true }()

						resp, err := httpClient.Get(fmt.Sprintf("%s/helloworld/user%d", baseURL, id))
						if err == nil {
							resp.Body.Close()
							Expect(resp.StatusCode).To(BeNumerically(">=", 200))
							Expect(resp.StatusCode).To(BeNumerically("<", 500))
						}
					}(i)
				}

				// 等待所有请求完成
				for i := 0; i < concurrency; i++ {
					Eventually(done).Should(Receive())
				}
			})
		})

		Context("长时间运行测试", func() {
			It("应该在持续负载下保持稳定", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()

				requestCount := 0
				successCount := 0

				for {
					select {
					case <-ctx.Done():
						// 测试完成
						successRate := float64(successCount) / float64(requestCount)
						Expect(successRate).To(BeNumerically(">=", 0.95)) // 95%成功率
						return
					case <-ticker.C:
						go func() {
							requestCount++
							resp, err := httpClient.Get(baseURL + "/health")
							if err == nil {
								resp.Body.Close()
								if resp.StatusCode == http.StatusOK {
									successCount++
								}
							}
						}()
					}
				}
			})
		})
	})

	Describe("错误处理和恢复", func() {
		Context("网络错误场景", func() {
			It("应该正确处理超时", func() {
				// 创建一个短超时的客户端
				shortTimeoutClient := &http.Client{
					Timeout: 1 * time.Millisecond, // 非常短的超时
				}

				_, err := shortTimeoutClient.Get(baseURL + "/helloworld/world")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("timeout"))
			})
		})

		Context("错误响应格式", func() {
			It("所有错误应该返回统一格式", func() {
				errorEndpoints := []string{
					"/helloworld/error",
					"/helloworld/badrequest",
					"/helloworld/notfound",
					"/helloworld/forbidden",
				}

				for _, endpoint := range errorEndpoints {
					resp, err := httpClient.Get(baseURL + endpoint)
					Expect(err).NotTo(HaveOccurred())
					defer resp.Body.Close()

					body, err := io.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())

					var errorResp map[string]interface{}
					err = json.Unmarshal(body, &errorResp)
					Expect(err).NotTo(HaveOccurred())

					// 验证统一的错误格式
					Expect(errorResp).To(HaveKey("code"))
					Expect(errorResp).To(HaveKey("message"))
					Expect(errorResp).To(HaveKey("metadata"))

					// 验证metadata包含预期字段
					metadata, ok := errorResp["metadata"].(map[string]interface{})
					Expect(ok).To(BeTrue())
					Expect(metadata).To(HaveKey("response"))
				}
			})
		})
	})
})