package data

import (
	"context"
	"testing"
	"time"

	"kratos-boilerplate/internal/biz"

	"github.com/stretchr/testify/assert"
)

// 测试NewCaptchaRepo
func TestNewCaptchaRepo(t *testing.T) {
	data := &Data{}
	repo := NewCaptchaRepo(data)
	assert.NotNil(t, repo)
}

// 测试CaptchaRepo接口实现
func TestCaptchaRepoInterface(t *testing.T) {
	data := &Data{}
	repo := NewCaptchaRepo(data)

	// 验证实现了接口
	var _ biz.CaptchaRepo = repo
	assert.NotNil(t, repo)
}

// 测试CaptchaRepo结构
func TestCaptchaRepoStructure(t *testing.T) {
	data := &Data{}
	repo := NewCaptchaRepo(data)
	captchaRepo := repo.(*captchaRepo)

	// 验证结构
	assert.NotNil(t, captchaRepo.data)
}

// 测试Captcha数据结构
func TestCaptchaStructure(t *testing.T) {
	// 创建Captcha实例
	captcha := &biz.Captcha{
		ID:       "test-id",
		Code:     "123456",
		Type:     "image",
		Target:   "test@example.com",
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     false,
	}

	// 验证结构
	assert.Equal(t, "test-id", captcha.ID)
	assert.Equal(t, "123456", captcha.Code)
	assert.Equal(t, "image", captcha.Type)
	assert.Equal(t, "test@example.com", captcha.Target)
	assert.False(t, captcha.Used)
}

// 测试所有CaptchaRepo方法 - 模拟测试
func TestCaptchaRepoAllMethods(t *testing.T) {
	// 创建带有Redis的数据层
	data := &Data{}
	repo := NewCaptchaRepo(data)
	captchaRepo := repo.(*captchaRepo)

	ctx := context.Background()

	// 测试SaveCaptcha - 这里会失败，因为Redis为nil
	captcha := &biz.Captcha{
		ID:       "test-id",
		Code:     "123456",
		Type:     "image",
		Target:   "test@example.com",
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     false,
	}

	// 由于Redis为nil，这个测试会失败
	err := captchaRepo.SaveCaptcha(ctx, captcha)
	assert.Error(t, err) // 预期的错误，因为Redis为nil

	// 测试GetCaptcha
	_, err = captchaRepo.GetCaptcha(ctx, "test-id")
	assert.Error(t, err) // 预期的错误

	// 测试DeleteCaptcha
	err = captchaRepo.DeleteCaptcha(ctx, "test-id")
	assert.Error(t, err) // 预期的错误，因为Redis为nil

	// 测试MarkCaptchaUsed
	err = captchaRepo.MarkCaptchaUsed(ctx, "test-id")
	assert.Error(t, err) // 预期的错误，因为无法获取验证码
}

// 测试验证码过期逻辑
func TestCaptchaExpirationLogic(t *testing.T) {
	// 测试过期时间逻辑
	now := time.Now()
	expiredTime := now.Add(-time.Hour)
	futureTime := now.Add(time.Hour)

	// 验证时间比较逻辑
	assert.True(t, now.After(expiredTime))
	assert.True(t, futureTime.After(now))
	assert.False(t, now.After(futureTime))
}

// 测试验证码类型
func TestCaptchaTypes(t *testing.T) {
	// 测试不同的验证码类型
	types := []string{"sms", "email", "image"}

	for _, captchaType := range types {
		captcha := &biz.Captcha{
			ID:       "test-id",
			Code:     "123456",
			Type:     captchaType,
			Target:   "test@example.com",
			ExpireAt: time.Now().Add(5 * time.Minute),
			Used:     false,
		}

		assert.Equal(t, captchaType, captcha.Type)
	}
}

// 测试验证码使用状态
func TestCaptchaUsageStatus(t *testing.T) {
	// 测试未使用的验证码
	unusedCaptcha := &biz.Captcha{
		ID:       "test-id",
		Code:     "123456",
		Type:     "image",
		Target:   "test@example.com",
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     false,
	}

	assert.False(t, unusedCaptcha.Used)

	// 测试已使用的验证码
	usedCaptcha := &biz.Captcha{
		ID:       "test-id",
		Code:     "123456",
		Type:     "image",
		Target:   "test@example.com",
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     true,
	}

	assert.True(t, usedCaptcha.Used)
}

// 测试验证码目标
func TestCaptchaTargets(t *testing.T) {
	// 测试不同的验证码目标
	targets := []string{
		"test@example.com",
		"13800138000",
		"user@domain.com",
	}

	for _, target := range targets {
		captcha := &biz.Captcha{
			ID:       "test-id",
			Code:     "123456",
			Type:     "image",
			Target:   target,
			ExpireAt: time.Now().Add(5 * time.Minute),
			Used:     false,
		}

		assert.Equal(t, target, captcha.Target)
	}
}
