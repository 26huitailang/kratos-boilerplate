package auth

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// MockUserRepository 模拟用户仓储
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

// TestJWTTokenManager JWT令牌管理器测试
func TestJWTTokenManager(t *testing.T) {
	config := &JWTConfig{
		Secret:        "test-secret",
		AccessExpiry:  time.Hour,
		RefreshExpiry: 24 * time.Hour,
		Issuer:        "test",
		SigningMethod: "HS256",
	}
	
	logger := log.NewHelper(log.DefaultLogger)
	manager := NewJWTTokenManager(config, logger)
	
	t.Run("GenerateAndVerifyToken", func(t *testing.T) {
		subject := &Subject{
			ID:          "user123",
			Type:        "user",
			Permissions: []string{"read:posts", "write:posts"},
			Roles:       []string{"user"},
			ExpiresAt:   time.Now().Add(time.Hour),
		}
		
		token, err := manager.GenerateToken(context.Background(), subject, TokenTypeAccess)
		assert.NoError(t, err)
		assert.NotEmpty(t, token.Value)
		assert.Equal(t, TokenTypeAccess, token.Type)
		
		verifiedSubject, err := manager.VerifyToken(context.Background(), token.Value)
		assert.NoError(t, err)
		assert.Equal(t, subject.ID, verifiedSubject.ID)
		assert.Equal(t, string(TokenTypeAccess), verifiedSubject.Type) // JWT中的Type是令牌类型
		assert.Equal(t, subject.Permissions, verifiedSubject.Permissions)
		assert.Equal(t, subject.Roles, verifiedSubject.Roles)
	})
	
	t.Run("InvalidToken", func(t *testing.T) {
		_, err := manager.VerifyToken(context.Background(), "invalid-token")
		assert.Error(t, err)
	})
}

// TestPasswordAuthStrategy 密码认证策略测试
func TestPasswordAuthStrategy(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	logger := log.NewHelper(log.DefaultLogger)
	strategy := NewPasswordAuthStrategy(mockUserRepo, logger)
	
	t.Run("SuccessfulAuthentication", func(t *testing.T) {
		password := "password123"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		
		user := &User{
			ID:          "user123",
			Username:    "testuser",
			Password:    string(hashedPassword),
			Roles:       []string{"user"},
			Permissions: []string{"read:posts"},
			Status:      "active",
		}
		
		mockUserRepo.On("GetByUsername", mock.Anything, "testuser").Return(user, nil)
		
		credentials := &PasswordCredentials{
			Username: "testuser",
			Password: password,
		}
		
		subject, err := strategy.Authenticate(context.Background(), credentials)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, subject.ID)
		assert.Equal(t, "user", subject.Type)
		assert.Equal(t, user.Roles, subject.Roles)
		assert.Equal(t, user.Permissions, subject.Permissions)
		
		mockUserRepo.AssertExpectations(t)
	})
	
	t.Run("InvalidPassword", func(t *testing.T) {
		password := "password123"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		
		user := &User{
			ID:       "user123",
			Username: "testuser",
			Password: string(hashedPassword),
			Status:   "active",
		}
		
		mockUserRepo.On("GetByUsername", mock.Anything, "testuser").Return(user, nil)
		
		credentials := &PasswordCredentials{
			Username: "testuser",
			Password: "wrongpassword",
		}
		
		_, err := strategy.Authenticate(context.Background(), credentials)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
		
		mockUserRepo.AssertExpectations(t)
	})
}

// TestDefaultAuthManager 默认认证管理器测试
func TestDefaultAuthManager(t *testing.T) {
	config := &AuthConfig{
		JWT: JWTConfig{
			Secret:        "test-secret",
			AccessExpiry:  time.Hour,
			RefreshExpiry: 24 * time.Hour,
			Issuer:        "test",
			SigningMethod: "HS256",
		},
	}
	
	manager := NewAuthManager(config, log.DefaultLogger)
	
	t.Run("RegisterStrategy", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		logger := log.NewHelper(log.DefaultLogger)
		strategy := NewPasswordAuthStrategy(mockUserRepo, logger)
		
		err := manager.RegisterStrategy(strategy)
		assert.NoError(t, err)
	})
	
	t.Run("CheckPermission", func(t *testing.T) {
		subject := &Subject{
			ID:          "user123",
			Permissions: []string{"posts:read", "comments:write"},
		}
		
		// 允许的权限
		err := manager.CheckPermission(context.Background(), subject, "posts", "read")
		assert.NoError(t, err)
		
		// 拒绝的权限
		err = manager.CheckPermission(context.Background(), subject, "admin", "delete")
		assert.Error(t, err)
		
		// 通配符权限
		subject.Permissions = []string{"*"}
		err = manager.CheckPermission(context.Background(), subject, "any", "action")
		assert.NoError(t, err)
	})
}

// BenchmarkJWTTokenGeneration JWT令牌生成性能测试
func BenchmarkJWTTokenGeneration(b *testing.B) {
	config := &JWTConfig{
		Secret:        "test-secret",
		AccessExpiry:  time.Hour,
		RefreshExpiry: 24 * time.Hour,
		Issuer:        "test",
		SigningMethod: "HS256",
	}
	
	logger := log.NewHelper(log.DefaultLogger)
	manager := NewJWTTokenManager(config, logger)
	
	subject := &Subject{
		ID:          "user123",
		Type:        "user",
		Permissions: []string{"read:posts", "write:posts"},
		Roles:       []string{"user"},
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GenerateToken(context.Background(), subject, TokenTypeAccess)
		if err != nil {
			b.Fatal(err)
		}
	}
}