package biz

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"

	"kratos-boilerplate/internal/pkg/sensitive"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrUserExists          = errors.New("user already exists")
	ErrPasswordIncorrect   = errors.New("password incorrect")
	ErrCaptchaRequired     = errors.New("captcha required")
	ErrCaptchaInvalid      = errors.New("captcha invalid")
	ErrCaptchaExpired      = errors.New("captcha expired")
	ErrAccountLocked       = errors.New("account locked")
	ErrTokenInvalid        = errors.New("token invalid")
	ErrTokenExpired        = errors.New("token expired")
	ErrRefreshTokenInvalid = errors.New("refresh token invalid")
	ErrRefreshTokenReused  = errors.New("refresh token reused")
	ErrTotpCodeInvalid     = errors.New("totp code invalid")
	ErrTotpRequired        = errors.New("totp code required")
)

// User 用户模型
type User struct {
	ID         int64
	Username   string
	Password   string
	Email      string
	Phone      string
	Name       string
	TotpSecret string // TOTP 密钥
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// GetSensitiveFields 获取敏感字段列表
func (u *User) GetSensitiveFields() []string {
	return []string{"email", "phone", "name", "password", "totp_secret"}
}

// Anonymize 脱敏处理
func (u *User) Anonymize() interface{} {
	if u == nil {
		return nil
	}

	// 获取脱敏规则
	rules := u.GetAnonymizeRules()
	anonymizer := sensitive.NewAnonymizer()

	return map[string]interface{}{
		"id":          u.ID,
		"username":    u.Username,   // 用户名不脱敏
		"password":    "[REDACTED]", // 密码完全隐藏
		"email":       anonymizer.AnonymizeString(u.Email, rules["email"]),
		"phone":       anonymizer.AnonymizeString(u.Phone, rules["phone"]),
		"name":        anonymizer.AnonymizeString(u.Name, rules["name"]),
		"totp_secret": "[REDACTED]", // TOTP密钥完全隐藏
		"created_at":  u.CreatedAt,
		"updated_at":  u.UpdatedAt,
	}
}

// GetAnonymizeRules 获取脱敏规则
func (u *User) GetAnonymizeRules() map[string]sensitive.AnonymizeRule {
	rules := sensitive.GetDefaultRules()

	// 可以根据需要自定义规则
	rules["password"] = sensitive.AnonymizeRule{
		FieldName: "password",
		CustomFunc: func(string) string {
			return "[REDACTED]"
		},
	}

	rules["totp_secret"] = sensitive.AnonymizeRule{
		FieldName: "totp_secret",
		CustomFunc: func(string) string {
			return "[REDACTED]"
		},
	}

	return rules
}

// LogSafeString 实现LogSafeStringer接口
func (u *User) LogSafeString() string {
	if u == nil {
		return "<nil>"
	}

	anonymized := u.Anonymize()
	return fmt.Sprintf("%+v", anonymized)
}

// UserInfo 用户信息（匿名化后的）
type UserInfo struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"` // 匿名化的邮箱
	Phone     string    `json:"phone"` // 匿名化的手机号
	Name      string    `json:"name"`  // 匿名化的姓名
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Captcha 验证码
type Captcha struct {
	ID       string
	Code     string
	Type     string // 验证码类型: "sms", "email", "image"
	Target   string // 短信或邮件的接收者
	ExpireAt time.Time
	Used     bool
}

// TokenPair 令牌对
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // 访问令牌有效期（秒）
}

// AccountLock 账户锁定信息
type AccountLock struct {
	Username       string
	FailedAttempts int32
	LockUntil      time.Time
	LastAttempt    time.Time
}

// UserRepo 用户仓储接口
type UserRepo interface {
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, username string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByPhone(ctx context.Context, phone string) (*User, error)
	GetUserByName(ctx context.Context, name string) (*User, error)
	UpdateUser(ctx context.Context, user *User) error

	// 账户锁定相关
	GetLock(ctx context.Context, username string) (*AccountLock, error)
	SaveLock(ctx context.Context, lock *AccountLock) error
	RemoveLock(ctx context.Context, username string) error

	// 令牌相关
	SaveRefreshToken(ctx context.Context, username, tokenID string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, tokenID string) (string, bool, error)
	InvalidateRefreshToken(ctx context.Context, tokenID string) error
	InvalidateAllRefreshTokens(ctx context.Context, username string) error

	// 验证码相关
	SaveCaptcha(ctx context.Context, captcha *Captcha) error
	GetCaptcha(ctx context.Context, captchaID string) (*Captcha, error)
	MarkCaptchaUsed(ctx context.Context, captchaID string) error
}

// CaptchaRepo 验证码存储接口
type CaptchaRepo interface {
	SaveCaptcha(ctx context.Context, captcha *Captcha) error
	GetCaptcha(ctx context.Context, captchaID string) (*Captcha, error)
	MarkCaptchaUsed(ctx context.Context, captchaID string) error
}

// CaptchaService 验证码服务接口
type CaptchaService interface {
	Generate(ctx context.Context, captchaType, target string) (*Captcha, error)
	Verify(ctx context.Context, captchaID, captchaCode string) (bool, error)
}

// AuthConfig 认证配置
type AuthConfig struct {
	// JWT配置
	JWTSecretKey           string
	AccessTokenExpiration  time.Duration
	RefreshTokenExpiration time.Duration

	// 验证码配置
	CaptchaEnabled    bool
	CaptchaExpiration time.Duration

	// 账户锁定配置
	MaxLoginAttempts int32
	LockDuration     time.Duration

	// TOTP配置
	TOTPEnabled bool
}

// 设置默认配置
var DefaultAuthConfig = AuthConfig{
	JWTSecretKey:           "your-secret-key",
	AccessTokenExpiration:  15 * time.Minute,
	RefreshTokenExpiration: 7 * 24 * time.Hour,
	CaptchaEnabled:         true,
	CaptchaExpiration:      5 * time.Minute,
	MaxLoginAttempts:       5,
	LockDuration:           30 * time.Minute,
	TOTPEnabled:            false,
}

// AuthUsecase defines the interface for authentication use cases.
type AuthUsecase interface {
	Register(ctx context.Context, username, password, email, phone, captchaID, captchaCode string) error
	Login(ctx context.Context, username, password, captchaID, captchaCode, totpCode string) (*TokenPair, error)
	Logout(ctx context.Context, accessToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	GetCaptcha(ctx context.Context, captchaType, target string) (*Captcha, error)
	VerifyCaptcha(ctx context.Context, captchaID, captchaCode string) (bool, error)
	GetLockStatus(ctx context.Context, username string) (*AccountLock, error)
	Now() time.Time
	GetMaxLoginAttempts() int32
}

// authUsecase is the implementation of AuthUsecase.
type authUsecase struct {
	repo           UserRepo
	captchaService CaptchaService
	config         AuthConfig
	log            *log.Helper
	// 内存缓存黑名单的访问令牌
	tokenBlacklist sync.Map
}

// NewAuthUsecase creates a new authUsecase instance.
func NewAuthUsecase(repo UserRepo, captchaService CaptchaService, config AuthConfig, logger log.Logger) AuthUsecase {
	if config.JWTSecretKey == "" {
		config = DefaultAuthConfig
	}
	return &authUsecase{
		repo:           repo,
		captchaService: captchaService,
		config:         config,
		log:            log.NewHelper(logger),
	}
}

// Register 用户注册
func (uc *authUsecase) Register(ctx context.Context, username, password, email, phone string, captchaID, captchaCode string) error {
	// 验证验证码
	if uc.config.CaptchaEnabled {
		if captchaID == "" || captchaCode == "" {
			return ErrCaptchaRequired
		}
		valid, err := uc.captchaService.Verify(ctx, captchaID, captchaCode)
		if err != nil {
			return err
		}
		if !valid {
			return ErrCaptchaInvalid
		}
	}

	// 检查用户是否已存在
	_, err := uc.repo.GetUser(ctx, username)
	if err == nil {
		return ErrUserExists
	} else if err != ErrUserNotFound {
		return fmt.Errorf("查询用户失败: %v", err)
	}

	// 检查邮箱是否已使用
	if email != "" {
		_, err = uc.repo.GetUserByEmail(ctx, email)
		if err == nil {
			return fmt.Errorf("邮箱已被注册")
		} else if err != ErrUserNotFound {
			return fmt.Errorf("查询邮箱失败: %v", err)
		}
	}

	// 检查手机号是否已使用
	if phone != "" {
		_, err = uc.repo.GetUserByPhone(ctx, phone)
		if err == nil {
			return fmt.Errorf("手机号已被注册")
		} else if err != ErrUserNotFound {
			return fmt.Errorf("查询手机号失败: %v", err)
		}
	}

	// 验证密码强度
	if err := validatePassword(password); err != nil {
		return err
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %v", err)
	}

	// 创建用户
	now := time.Now()
	if err := uc.repo.CreateUser(ctx, &User{
		Username:  username,
		Password:  string(hashedPassword),
		Email:     email,
		Phone:     phone,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("创建用户失败: %v", err)
	}

	return nil
}

// Login 用户登录
func (uc *authUsecase) Login(ctx context.Context, username, password, captchaID, captchaCode, totpCode string) (*TokenPair, error) {
	// 检查账户是否被锁定
	lock, err := uc.repo.GetLock(ctx, username)
	if err != nil && err != ErrUserNotFound {
		return nil, fmt.Errorf("查询账户锁定状态失败: %v", err)
	}

	if lock != nil && lock.LockUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	// 验证验证码
	if uc.config.CaptchaEnabled {
		if captchaID == "" || captchaCode == "" {
			return nil, ErrCaptchaRequired
		}
		valid, err := uc.captchaService.Verify(ctx, captchaID, captchaCode)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrCaptchaInvalid
		}
	}

	// 获取用户
	user, err := uc.repo.GetUser(ctx, username)
	if err != nil {
		// 记录失败尝试
		uc.recordFailedAttempt(ctx, username)
		return nil, ErrUserNotFound
	}

	// 验证密码
	if err := bcryptCompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		// 记录失败尝试
		uc.recordFailedAttempt(ctx, username)
		return nil, ErrPasswordIncorrect
	}

	// 如果启用了TOTP，验证TOTP码
	if uc.config.TOTPEnabled && user.TotpSecret != "" {
		if totpCode == "" {
			return nil, ErrTotpRequired
		}
		if !uc.verifyTOTP(user.TotpSecret, totpCode) {
			// 记录失败尝试
			uc.recordFailedAttempt(ctx, username)
			return nil, ErrTotpCodeInvalid
		}
	}

	// 清除账户锁定
	if lock != nil {
		if err := uc.repo.RemoveLock(ctx, username); err != nil {
			uc.log.Warnf("移除账户锁定失败: %v", err)
		}
	}

	// 生成JWT令牌对
	tokenPair, err := uc.generateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %v", err)
	}

	return tokenPair, nil
}

// Logout 退出登录
func (uc *authUsecase) Logout(ctx context.Context, accessToken string) error {
	// 将访问令牌加入黑名单
	claims, err := uc.parseAccessToken(accessToken)
	if err != nil {
		return err
	}

	// 获取过期时间，将令牌加入黑名单直到过期
	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return fmt.Errorf("无效的令牌过期时间")
	}
	exp := time.Unix(int64(expFloat), 0)
	uc.tokenBlacklist.Store(accessToken, exp)

	// 清理令牌黑名单中已过期的条目
	uc.cleanupTokenBlacklist()

	username := claims["username"].(string)
	// 可选：使所有刷新令牌无效
	if err := uc.repo.InvalidateAllRefreshTokens(ctx, username); err != nil {
		uc.log.Warnf("使所有刷新令牌无效失败: %v", err)
	}

	return nil
}

// RefreshToken 刷新令牌
func (uc *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// 解析刷新令牌
	claims, err := uc.parseRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	tokenID := claims["jti"].(string)
	username := claims["username"].(string)

	// 检查令牌是否已使用
	_, used, err := uc.repo.GetRefreshToken(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("验证刷新令牌失败: %v", err)
	}

	if used {
		// 如果令牌已被使用，可能是令牌被盗用，使所有令牌失效
		if err := uc.repo.InvalidateAllRefreshTokens(ctx, username); err != nil {
			uc.log.Warnf("令牌被重用，使所有令牌无效失败: %v", err)
		}
		return nil, ErrRefreshTokenReused
	}

	// 将当前刷新令牌标记为已使用
	if err := uc.repo.InvalidateRefreshToken(ctx, tokenID); err != nil {
		uc.log.Warnf("将刷新令牌标记为已使用失败: %v", err)
	}

	// 获取用户
	user, err := uc.repo.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("获取用户失败: %v", err)
	}

	// 生成新的令牌对
	tokenPair, err := uc.generateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("生成新令牌失败: %v", err)
	}

	return tokenPair, nil
}

// GetCaptcha 获取验证码
func (uc *authUsecase) GetCaptcha(ctx context.Context, captchaType, target string) (*Captcha, error) {
	return uc.captchaService.Generate(ctx, captchaType, target)
}

// VerifyCaptcha 验证验证码
func (uc *authUsecase) VerifyCaptcha(ctx context.Context, captchaID, captchaCode string) (bool, error) {
	return uc.captchaService.Verify(ctx, captchaID, captchaCode)
}

// GetLockStatus 获取账户锁定状态
func (uc *authUsecase) GetLockStatus(ctx context.Context, username string) (*AccountLock, error) {
	lock, err := uc.repo.GetLock(ctx, username)
	if err != nil {
		if err == ErrUserNotFound {
			// 用户不存在或未锁定，返回空锁定状态
			return &AccountLock{
				Username:       username,
				FailedAttempts: 0,
				LockUntil:      time.Time{},
				LastAttempt:    time.Time{},
			}, nil
		}
		return nil, fmt.Errorf("获取账户锁定状态失败: %v", err)
	}

	// 如果锁定时间已过，但记录仍存在，清除锁定
	if lock != nil && lock.LockUntil.Before(time.Now()) {
		if err := uc.repo.RemoveLock(ctx, username); err != nil {
			uc.log.Warnf("移除过期锁定记录失败: %v", err)
		}
		// 返回未锁定状态
		return &AccountLock{
			Username:       username,
			FailedAttempts: 0,
			LockUntil:      time.Time{},
			LastAttempt:    time.Time{},
		}, nil
	}

	return lock, nil
}

// 生成令牌对
func (uc *authUsecase) generateTokens(ctx context.Context, user *User) (*TokenPair, error) {
	now := time.Now()

	// 生成access token
	accessExp := now.Add(uc.config.AccessTokenExpiration)
	accessClaims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      accessExp.Unix(),
		"iat":      now.Unix(),
		"type":     "access",
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	signedAccessToken, err := accessToken.SignedString([]byte(uc.config.JWTSecretKey))
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %v", err)
	}

	// 生成refresh token
	refreshExp := now.Add(uc.config.RefreshTokenExpiration)
	tokenID := generateRandomString(32)
	refreshClaims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      refreshExp.Unix(),
		"iat":      now.Unix(),
		"type":     "refresh",
		"jti":      tokenID, // 令牌ID，用于标识刷新令牌
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	signedRefreshToken, err := refreshToken.SignedString([]byte(uc.config.JWTSecretKey))
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %v", err)
	}

	// 存储刷新令牌
	if err := uc.repo.SaveRefreshToken(ctx, user.Username, tokenID, refreshExp); err != nil {
		return nil, fmt.Errorf("保存刷新令牌失败: %v", err)
	}

	return &TokenPair{
		AccessToken:  signedAccessToken,
		RefreshToken: signedRefreshToken,
		ExpiresIn:    int64(uc.config.AccessTokenExpiration.Seconds()),
	}, nil
}

// 解析访问令牌
func (uc *authUsecase) parseAccessToken(tokenStr string) (jwt.MapClaims, error) {
	// 检查令牌是否在黑名单中
	if _, ok := uc.tokenBlacklist.Load(tokenStr); ok {
		return nil, ErrTokenInvalid
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(uc.config.JWTSecretKey), nil
	})

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, ErrTokenExpired
			}
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid || claims["type"] != "access" {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// 解析刷新令牌
func (uc *authUsecase) parseRefreshToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(uc.config.JWTSecretKey), nil
	})

	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return nil, ErrTokenExpired
			}
		}
		return nil, ErrRefreshTokenInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid || claims["type"] != "refresh" {
		return nil, ErrRefreshTokenInvalid
	}

	return claims, nil
}

// 记录失败的登录尝试
func (uc *authUsecase) recordFailedAttempt(ctx context.Context, username string) {
	// 获取当前锁定状态
	lock, err := uc.repo.GetLock(ctx, username)
	if err != nil {
		if err != ErrUserNotFound {
			uc.log.Errorf("获取账户锁定状态失败: %v", err)
			return
		}
		// 不存在则创建新的锁定记录
		lock = &AccountLock{
			Username:       username,
			FailedAttempts: 0,
			LastAttempt:    time.Now(),
		}
	}

	now := time.Now()
	lock.FailedAttempts++
	lock.LastAttempt = now

	// 达到最大尝试次数，进行锁定
	if lock.FailedAttempts >= uc.config.MaxLoginAttempts {
		lock.LockUntil = now.Add(uc.config.LockDuration)
	}

	// 保存锁定记录
	if err := uc.repo.SaveLock(ctx, lock); err != nil {
		uc.log.Errorf("保存账户锁定状态失败: %v", err)
	}
}

// 清理令牌黑名单中的过期条目
func (uc *authUsecase) cleanupTokenBlacklist() {
	now := time.Now()
	uc.tokenBlacklist.Range(func(key, value interface{}) bool {
		exp := value.(time.Time)
		if exp.Before(now) {
			uc.tokenBlacklist.Delete(key)
		}
		return true
	})
}

// 验证TOTP码
func (uc *authUsecase) verifyTOTP(secret, code string) bool {
	// 实际项目中应该使用TOTP库实现验证
	// 这里简单模拟
	return code == "123456" // 临时模拟，实际应使用如 github.com/pquerna/otp
}

// 验证密码强度
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("密码长度至少为8位")
	}

	// 实现其他密码强度检查
	// TODO: 实现更完整的密码策略验证

	return nil
}

// 生成随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

// Now 返回当前时间，便于测试中模拟时间
func (uc *authUsecase) Now() time.Time {
	return time.Now()
}

// GetMaxLoginAttempts 返回最大登录尝试次数
func (uc *authUsecase) GetMaxLoginAttempts() int32 {
	return uc.config.MaxLoginAttempts
}

// 使用变量包装bcrypt.CompareHashAndPassword函数，便于测试
var bcryptCompareHashAndPassword = bcrypt.CompareHashAndPassword
