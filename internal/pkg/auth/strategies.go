package auth

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt"
)

// PasswordCredentials 密码凭证
type PasswordCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// User 用户实体
type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	Status      string   `json:"status"`
}

// UserRepository 用户仓储接口
type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*User, error)
}

// PasswordAuthStrategy 密码认证策略
type PasswordAuthStrategy struct {
	userRepo UserRepository
	logger   *log.Helper
}

// NewPasswordAuthStrategy 创建密码认证策略
func NewPasswordAuthStrategy(userRepo UserRepository, logger *log.Helper) AuthStrategy {
	return &PasswordAuthStrategy{
		userRepo: userRepo,
		logger:   logger,
	}
}

// GetName 获取策略名称
func (s *PasswordAuthStrategy) GetName() string {
	return "password"
}

// Authenticate 执行认证
func (s *PasswordAuthStrategy) Authenticate(ctx context.Context, credentials interface{}) (*Subject, error) {
	creds, ok := credentials.(*PasswordCredentials)
	if !ok {
		return nil, fmt.Errorf("invalid credentials type for password auth")
	}
	
	if creds.Username == "" || creds.Password == "" {
		return nil, fmt.Errorf("username and password are required")
	}
	
	user, err := s.userRepo.GetByUsername(ctx, creds.Username)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("failed to get user: %v", err)
		return nil, fmt.Errorf("authentication failed")
	}
	
	if user.Status != "active" {
		return nil, fmt.Errorf("user account is %s", user.Status)
	}
	
	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password)); err != nil {
		s.logger.WithContext(ctx).Warnf("password mismatch for user: %s", user.Username)
		return nil, fmt.Errorf("authentication failed")
	}
	
	subject := &Subject{
		ID:          user.ID,
		Type:        "user",
		Permissions: user.Permissions,
		Roles:       user.Roles,
	}
	
	s.logger.WithContext(ctx).Infof("password authentication successful for user: %s", user.Username)
	return subject, nil
}