package captcha

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"time"

	"kratos-boilerplate/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// 图形验证码配置
var (
	captchaHeight = 60
	captchaWidth  = 240
	captchaLength = 6
	// 图形验证码默认过期时间（分钟）
	captchaExpiration = 5 * time.Minute
)

// DefaultService 默认验证码服务实现
type DefaultService struct {
	repo   biz.UserRepo
	log    *log.Helper
	config *Config
}

// Config 验证码配置
type Config struct {
	// 验证码类型启用设置
	EnableSMS   bool
	EnableEmail bool
	EnableImage bool

	// 验证码过期时间
	Expiration time.Duration

	// 短信验证码配置
	SMSProvider  string
	SMSAPIKey    string
	SMSAPISecret string

	// 邮件验证码配置
	EmailSMTP     string
	EmailPort     int
	EmailUsername string
	EmailPassword string
	EmailFrom     string
}

// DefaultConfig 默认配置
var DefaultConfig = &Config{
	EnableSMS:   true,
	EnableEmail: true,
	EnableImage: true,
	Expiration:  captchaExpiration,
	SMSProvider: "mock", // 实际项目需替换
	EmailSMTP:   "smtp.example.com",
	EmailPort:   587,
	EmailFrom:   "noreply@example.com",
}

// NewCaptchaService 创建新的验证码服务
func NewCaptchaService(repo biz.UserRepo, cfg *Config, logger log.Logger) biz.CaptchaService {
	if cfg == nil {
		cfg = DefaultConfig
	}
	if cfg.Expiration == 0 {
		cfg.Expiration = captchaExpiration
	}
	return &DefaultService{
		repo:   repo,
		log:    log.NewHelper(logger),
		config: cfg,
	}
}

// Generate 生成验证码
func (s *DefaultService) Generate(ctx context.Context, captchaType, target string) (*biz.Captcha, error) {
	var code string
	var imageData string
	var err error

	// 检查验证码类型是否启用
	switch captchaType {
	case "sms":
		if !s.config.EnableSMS {
			return nil, fmt.Errorf("短信验证码功能未启用")
		}
		code, err = s.generateSMSCode(target)
	case "email":
		if !s.config.EnableEmail {
			return nil, fmt.Errorf("邮件验证码功能未启用")
		}
		code, err = s.generateEmailCode(target)
	case "image":
		if !s.config.EnableImage {
			return nil, fmt.Errorf("图片验证码功能未启用")
		}
		code, imageData, err = s.generateImageCode()
	default:
		return nil, fmt.Errorf("不支持的验证码类型: %s", captchaType)
	}

	if err != nil {
		return nil, err
	}

	// 生成验证码ID
	captchaID, err := generateRandomString(32)
	if err != nil {
		return nil, err
	}

	// 创建验证码记录
	captcha := &biz.Captcha{
		ID:       captchaID,
		Code:     code,
		Type:     captchaType,
		Target:   target,
		ExpireAt: time.Now().Add(s.config.Expiration),
		Used:     false,
	}

	// 保存验证码
	if err := s.repo.SaveCaptcha(ctx, captcha); err != nil {
		return nil, fmt.Errorf("保存验证码失败: %v", err)
	}

	// 返回给客户端的验证码不包含实际Code
	clientCaptcha := &biz.Captcha{
		ID:       captchaID,
		Type:     captchaType,
		Target:   target,
		ExpireAt: captcha.ExpireAt,
	}

	// 如果是图片验证码，返回图片数据
	if captchaType == "image" && imageData != "" {
		clientCaptcha.Code = imageData
	}

	return clientCaptcha, nil
}

// Verify 验证验证码
func (s *DefaultService) Verify(ctx context.Context, captchaID, captchaCode string) (bool, error) {
	if captchaID == "" || captchaCode == "" {
		return false, biz.ErrCaptchaInvalid
	}

	// 获取验证码
	captcha, err := s.repo.GetCaptcha(ctx, captchaID)
	if err != nil {
		return false, fmt.Errorf("获取验证码失败: %v", err)
	}

	// 检查验证码是否已过期
	if captcha.ExpireAt.Before(time.Now()) {
		return false, biz.ErrCaptchaExpired
	}

	// 检查验证码是否已使用
	if captcha.Used {
		return false, fmt.Errorf("验证码已使用")
	}

	// 验证码比对(不区分大小写)
	if !equalFold(captcha.Code, captchaCode) {
		return false, nil
	}

	// 标记验证码为已使用
	if err := s.repo.MarkCaptchaUsed(ctx, captchaID); err != nil {
		s.log.Warnf("标记验证码为已使用失败: %v", err)
	}

	return true, nil
}

// generateSMSCode 生成短信验证码
func (s *DefaultService) generateSMSCode(phone string) (string, error) {
	// 生成随机验证码
	code, err := generateNumericCode(6)
	if err != nil {
		return "", err
	}

	// 实际项目中，应该调用短信服务商的API发送短信
	s.log.Infof("【模拟发送短信】目标: %s, 验证码: %s", phone, code)

	return code, nil
}

// generateEmailCode 生成邮件验证码
func (s *DefaultService) generateEmailCode(email string) (string, error) {
	// 生成随机验证码
	code, err := generateNumericCode(6)
	if err != nil {
		return "", err
	}

	// 实际项目中，应该调用邮件服务发送邮件
	s.log.Infof("【模拟发送邮件】目标: %s, 验证码: %s", email, code)

	return code, nil
}

// generateImageCode 生成图片验证码
func (s *DefaultService) generateImageCode() (string, string, error) {
	// 简化实现，不使用外部库
	// 生成随机验证码
	code, err := generateRandomCode(captchaLength, "1234567890abcdefghijklmnopqrstuvwxyz")
	if err != nil {
		return "", "", err
	}

	// 实际项目中应该使用图形库生成验证码图片
	// 这里简单返回一个编码后的字符串表示图片
	imageData := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString([]byte("mock-image-data")))

	s.log.Infof("【模拟生成图片验证码】验证码: %s", code)

	return code, imageData, nil
}

// 生成指定长度的随机数字验证码
func generateNumericCode(length int) (string, error) {
	const charset = "0123456789"
	return generateRandomCode(length, charset)
}

// 生成指定长度和字符集的随机验证码
func generateRandomCode(length int, charset string) (string, error) {
	code := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		code[i] = charset[randomIndex.Int64()]
	}

	return string(code), nil
}

// 生成随机字符串
func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:length], nil
}

// 不区分大小写比较字符串
func equalFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		c1 := s[i]
		c2 := t[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 'a' - 'A'
		}
		if c2 >= 'A' && c2 <= 'Z' {
			c2 += 'a' - 'A'
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}
