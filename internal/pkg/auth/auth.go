// Package auth 提供企业级认证鉴权解决方案
package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang-jwt/jwt/v5"
)

// TokenType 令牌类型
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
	TokenTypeAPI     TokenType = "api"
)

// Subject 认证主体
type Subject struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Attributes  map[string]string `json:"attributes"`
	Permissions []string          `json:"permissions"`
	Roles       []string          `json:"roles"`
	ExpiresAt   time.Time         `json:"expires_at"`
}

// Token 令牌信息
type Token struct {
	Type      TokenType `json:"type"`
	Value     string    `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
	Subject   *Subject  `json:"subject"`
}

// AuthStrategy 认证策略接口
type AuthStrategy interface {
	Authenticate(ctx context.Context, credentials interface{}) (*Subject, error)
	GetName() string
}

// TokenManager 令牌管理器接口
type TokenManager interface {
	GenerateToken(ctx context.Context, subject *Subject, tokenType TokenType) (*Token, error)
	VerifyToken(ctx context.Context, tokenValue string) (*Subject, error)
	RevokeToken(ctx context.Context, tokenValue string) error
	RefreshToken(ctx context.Context, refreshToken string) (*Token, error)
}

// AuthManager 认证管理器接口
type AuthManager interface {
	RegisterStrategy(strategy AuthStrategy) error
	Authenticate(ctx context.Context, strategyName string, credentials interface{}) (*Subject, error)
	GetTokenManager() TokenManager
	CheckPermission(ctx context.Context, subject *Subject, resource string, action string) error
}

// AuthConfig 认证配置
type AuthConfig struct {
	JWT JWTConfig `yaml:"jwt"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret        string        `yaml:"secret"`
	AccessExpiry  time.Duration `yaml:"access_expiry"`
	RefreshExpiry time.Duration `yaml:"refresh_expiry"`
	Issuer        string        `yaml:"issuer"`
	SigningMethod string        `yaml:"signing_method"`
}

// JWTClaims JWT Claims
type JWTClaims struct {
	jwt.RegisteredClaims
	Type        string            `json:"type"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	Permissions []string          `json:"permissions,omitempty"`
	Roles       []string          `json:"roles,omitempty"`
}

// DefaultAuthManager 默认认证管理器
type DefaultAuthManager struct {
	config       *AuthConfig
	strategies   map[string]AuthStrategy
	tokenManager TokenManager
	logger       *log.Helper
}

// NewAuthManager 创建认证管理器
func NewAuthManager(config *AuthConfig, logger log.Logger) AuthManager {
	helper := log.NewHelper(log.With(logger, "module", "auth"))
	tokenManager := NewJWTTokenManager(&config.JWT, helper)
	
	return &DefaultAuthManager{
		config:       config,
		strategies:   make(map[string]AuthStrategy),
		tokenManager: tokenManager,
		logger:       helper,
	}
}

// RegisterStrategy 注册认证策略
func (m *DefaultAuthManager) RegisterStrategy(strategy AuthStrategy) error {
	if strategy == nil {
		return fmt.Errorf("strategy cannot be nil")
	}
	
	name := strategy.GetName()
	if name == "" {
		return fmt.Errorf("strategy name cannot be empty")
	}
	
	m.strategies[name] = strategy
	m.logger.Infof("registered auth strategy: %s", name)
	return nil
}

// Authenticate 执行认证
func (m *DefaultAuthManager) Authenticate(ctx context.Context, strategyName string, credentials interface{}) (*Subject, error) {
	strategy, exists := m.strategies[strategyName]
	if !exists {
		return nil, fmt.Errorf("unknown auth strategy: %s", strategyName)
	}
	
	subject, err := strategy.Authenticate(ctx, credentials)
	if err != nil {
		m.logger.WithContext(ctx).Errorf("authentication failed for strategy %s: %v", strategyName, err)
		return nil, err
	}
	
	m.logger.WithContext(ctx).Infof("authentication successful for subject: %s", subject.ID)
	return subject, nil
}

// GetTokenManager 获取令牌管理器
func (m *DefaultAuthManager) GetTokenManager() TokenManager {
	return m.tokenManager
}

// CheckPermission 检查权限
func (m *DefaultAuthManager) CheckPermission(ctx context.Context, subject *Subject, resource string, action string) error {
	permission := fmt.Sprintf("%s:%s", resource, action)
	
	for _, perm := range subject.Permissions {
		if perm == "*" || perm == permission {
			return nil
		}
		if matchWildcard(perm, permission) {
			return nil
		}
	}
	
	return fmt.Errorf("permission denied: %s", permission)
}

// JWTTokenManager JWT令牌管理器
type JWTTokenManager struct {
	config *JWTConfig
	logger *log.Helper
}

// NewJWTTokenManager 创建JWT令牌管理器
func NewJWTTokenManager(config *JWTConfig, logger *log.Helper) TokenManager {
	return &JWTTokenManager{
		config: config,
		logger: logger,
	}
}

// GenerateToken 生成令牌
func (m *JWTTokenManager) GenerateToken(ctx context.Context, subject *Subject, tokenType TokenType) (*Token, error) {
	now := time.Now()
	tokenID := fmt.Sprintf("%d", now.UnixNano())
	
	var expiry time.Duration
	switch tokenType {
	case TokenTypeAccess:
		expiry = m.config.AccessExpiry
	case TokenTypeRefresh:
		expiry = m.config.RefreshExpiry
	case TokenTypeAPI:
		expiry = 365 * 24 * time.Hour
	default:
		expiry = m.config.AccessExpiry
	}
	
	expiresAt := now.Add(expiry)
	
	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   subject.ID,
			Issuer:    m.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
		Type:        string(tokenType),
		Attributes:  subject.Attributes,
		Permissions: subject.Permissions,
		Roles:       subject.Roles,
	}
	
	token := jwt.NewWithClaims(m.getSigningMethod(), claims)
	tokenString, err := token.SignedString([]byte(m.config.Secret))
	if err != nil {
		m.logger.WithContext(ctx).Errorf("failed to sign token: %v", err)
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	
	return &Token{
		Type:      tokenType,
		Value:     tokenString,
		ExpiresAt: expiresAt,
		Subject:   subject,
	}, nil
}

// VerifyToken 验证令牌
func (m *JWTTokenManager) VerifyToken(ctx context.Context, tokenValue string) (*Subject, error) {
	token, err := jwt.ParseWithClaims(tokenValue, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != m.getSigningMethod() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.Secret), nil
	})
	
	if err != nil {
		m.logger.WithContext(ctx).Errorf("failed to parse token: %v", err)
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	
	subject := &Subject{
		ID:          claims.Subject,
		Type:        claims.Type,
		Attributes:  claims.Attributes,
		Permissions: claims.Permissions,
		Roles:       claims.Roles,
		ExpiresAt:   claims.ExpiresAt.Time,
	}
	
	return subject, nil
}

// RevokeToken 撤销令牌
func (m *JWTTokenManager) RevokeToken(ctx context.Context, tokenValue string) error {
	// 简化实现：实际应该使用黑名单存储
	m.logger.WithContext(ctx).Infof("token revoked")
	return nil
}

// RefreshToken 刷新令牌
func (m *JWTTokenManager) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	subject, err := m.VerifyToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}
	
	// 生成新的访问令牌
	newToken, err := m.GenerateToken(ctx, subject, TokenTypeAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new token: %w", err)
	}
	
	return newToken, nil
}

// getSigningMethod 获取签名方法
func (m *JWTTokenManager) getSigningMethod() jwt.SigningMethod {
	switch m.config.SigningMethod {
	case "HS256":
		return jwt.SigningMethodHS256
	case "HS384":
		return jwt.SigningMethodHS384
	case "HS512":
		return jwt.SigningMethodHS512
	default:
		return jwt.SigningMethodHS256
	}
}

// matchWildcard 通配符匹配
func matchWildcard(pattern string, str string) bool {
	if pattern == "*" {
		return true
	}
	
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(str) >= len(prefix) && str[:len(prefix)] == prefix
	}
	
	return false
}