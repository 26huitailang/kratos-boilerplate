package service

import (
	"context"
	"encoding/base64"
	"math/rand"
	"time"
	"xredline/internal/biz"
)

type captchaService struct {
	repo biz.CaptchaRepo
}

func NewCaptchaService(repo biz.CaptchaRepo) biz.CaptchaService {
	return &captchaService{
		repo: repo,
	}
}

func (s *captchaService) Generate(ctx context.Context, captchaType, target string) (*biz.Captcha, error) {
	// 生成随机验证码
	code := generateRandomCode(6)

	captcha := &biz.Captcha{
		ID:       generateRandomString(32),
		Code:     code,
		Type:     captchaType,
		Target:   target,
		ExpireAt: time.Now().Add(5 * time.Minute),
		Used:     false,
	}

	if err := s.repo.SaveCaptcha(ctx, captcha); err != nil {
		return nil, err
	}

	return captcha, nil
}

func (s *captchaService) Verify(ctx context.Context, captchaID, captchaCode string) (bool, error) {
	captcha, err := s.repo.GetCaptcha(ctx, captchaID)
	if err != nil {
		return false, err
	}

	if captcha.Used {
		return false, biz.ErrCaptchaInvalid
	}

	if time.Now().After(captcha.ExpireAt) {
		return false, biz.ErrCaptchaExpired
	}

	if captcha.Code != captchaCode {
		return false, biz.ErrCaptchaInvalid
	}

	if err := s.repo.MarkCaptchaUsed(ctx, captchaID); err != nil {
		return false, err
	}

	return true, nil
}

func generateRandomCode(length int) string {
	const digits = "0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = digits[rand.Intn(len(digits))]
	}
	return string(b)
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)[:length]
}
