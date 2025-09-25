package auth

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// Example 认证模块使用示例
func Example() {
	// 1. 创建认证配置
	config := &AuthConfig{
		JWT: JWTConfig{
			Secret:        "your-secret-key",
			AccessExpiry:  time.Hour,
			RefreshExpiry: 24 * time.Hour,
			Issuer:        "your-app",
			SigningMethod: "HS256",
		},
	}

	// 2. 创建认证管理器
	authManager := NewAuthManager(config, log.DefaultLogger)

	// 3. 注册认证策略
	// 注意：这里需要提供UserRepository的实现
	// userRepo := &YourUserRepository{}
	// passwordStrategy := NewPasswordAuthStrategy(userRepo, log.NewHelper(log.DefaultLogger))
	// authManager.RegisterStrategy(passwordStrategy)

	// 4. 执行认证（示例）
	ctx := context.Background()
	
	// 创建一个示例用户
	subject := &Subject{
		ID:          "user123",
		Type:        "user",
		Permissions: []string{"posts:read", "posts:write"},
		Roles:       []string{"user"},
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	// 5. 生成令牌
	tokenManager := authManager.GetTokenManager()
	accessToken, err := tokenManager.GenerateToken(ctx, subject, TokenTypeAccess)
	if err != nil {
		log.Errorf("failed to generate token: %v", err)
		return
	}

	log.Infof("Generated access token: %s", accessToken.Value[:20]+"...")

	// 6. 验证令牌
	verifiedSubject, err := tokenManager.VerifyToken(ctx, accessToken.Value)
	if err != nil {
		log.Errorf("failed to verify token: %v", err)
		return
	}

	log.Infof("Token verification successful for user: %s", verifiedSubject.ID)

	// 7. 检查权限
	err = authManager.CheckPermission(ctx, verifiedSubject, "posts", "read")
	if err != nil {
		log.Errorf("permission denied: %v", err)
		return
	}

	log.Info("Permission check passed")

	// 8. 生成刷新令牌
	refreshToken, err := tokenManager.GenerateToken(ctx, subject, TokenTypeRefresh)
	if err != nil {
		log.Errorf("failed to generate refresh token: %v", err)
		return
	}

	// 9. 使用刷新令牌获取新的访问令牌
	newAccessToken, err := tokenManager.RefreshToken(ctx, refreshToken.Value)
	if err != nil {
		log.Errorf("failed to refresh token: %v", err)
		return
	}

	log.Infof("Token refresh successful, new token: %s", newAccessToken.Value[:20]+"...")
}