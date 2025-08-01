package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	v1 "kratos-boilerplate/api/auth/v1"

	"github.com/go-kratos/kratos/v2/errors"
)

// APIClient 封装HTTP API调用
type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAPIClient 创建新的API客户端
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HealthCheck 检查服务健康状态
func (c *APIClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetCaptcha 获取验证码
func (c *APIClient) GetCaptcha(ctx context.Context, req *v1.GetCaptchaRequest) (*v1.GetCaptchaReply, error) {
	url := fmt.Sprintf("%s/api/v1/auth/captcha?captcha_type=%s&target=%s", c.baseURL, req.CaptchaType, req.Target)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp)
	}

	var result v1.GetCaptchaReply
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// VerifyCaptcha 验证验证码
func (c *APIClient) VerifyCaptcha(ctx context.Context, req *v1.VerifyCaptchaRequest) (*v1.VerifyCaptchaReply, error) {
	var result v1.VerifyCaptchaReply
	err := c.postJSON(ctx, "/api/v1/auth/captcha/verify", req, &result)
	return &result, err
}

// Register 用户注册
func (c *APIClient) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterReply, error) {
	var result v1.RegisterReply
	err := c.postJSON(ctx, "/api/v1/auth/register", req, &result)
	return &result, err
}

// Login 用户登录
func (c *APIClient) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginReply, error) {
	var result v1.LoginReply
	err := c.postJSON(ctx, "/api/v1/auth/login", req, &result)
	return &result, err
}

// Logout 用户退出
func (c *APIClient) Logout(ctx context.Context, req *v1.LogoutRequest) (*v1.LogoutReply, error) {
	var result v1.LogoutReply
	err := c.postJSON(ctx, "/api/v1/auth/logout", req, &result)
	return &result, err
}

// RefreshToken 刷新令牌
func (c *APIClient) RefreshToken(ctx context.Context, req *v1.RefreshTokenRequest) (*v1.RefreshTokenReply, error) {
	var result v1.RefreshTokenReply
	err := c.postJSON(ctx, "/api/v1/auth/refresh", req, &result)
	return &result, err
}

// GetLockStatus 获取账户锁定状态
func (c *APIClient) GetLockStatus(ctx context.Context, req *v1.LockStatusRequest) (*v1.LockStatusReply, error) {
	url := fmt.Sprintf("%s/api/v1/auth/lock-status/%s", c.baseURL, req.Username)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleHTTPError(resp)
	}

	var result v1.LockStatusReply
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CleanupUser 清理测试用户数据
func (c *APIClient) CleanupUser(ctx context.Context, username string) error {
	// 这里应该调用管理API来清理用户数据
	// 在实际项目中，你可能需要实现一个专门的清理端点
	// 或者直接操作数据库
	return nil
}

// postJSON 发送JSON POST请求的通用方法
func (c *APIClient) postJSON(ctx context.Context, path string, reqBody interface{}, respBody interface{}) error {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.handleHTTPError(resp)
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return err
		}
	}

	return nil
}

// handleHTTPError 处理HTTP错误响应
func (c *APIClient) handleHTTPError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	// 尝试解析Kratos错误格式
	var kratosErr struct {
		Code    int32  `json:"code"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	}

	if json.Unmarshal(body, &kratosErr) == nil && kratosErr.Code != 0 {
		return errors.New(int(kratosErr.Code), kratosErr.Reason, kratosErr.Message)
	}

	// 如果不是Kratos错误格式，返回HTTP错误
	return &HTTPError{
		StatusCode: resp.StatusCode,
		Message:    string(body),
	}
}

// HTTPError 表示HTTP错误
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// IsHTTPError 检查错误是否为指定状态码的HTTP错误
func IsHTTPError(err error, statusCode int) bool {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.StatusCode == statusCode
	}
	return false
}

// GetTestServerURL 获取测试服务器URL
func GetTestServerURL() string {
	if url := os.Getenv("TEST_SERVER_URL"); url != "" {
		return url
	}
	return "http://localhost:8000" // 默认URL
}

// WaitForServer 等待服务器启动
func WaitForServer(baseURL string, timeout time.Duration) error {
	client := NewAPIClient(baseURL)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server to start")
		case <-ticker.C:
			if err := client.HealthCheck(ctx); err == nil {
				return nil
			}
		}
	}
}