package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// ContextKey 上下文键类型
type ContextKey string

const (
	// SubjectKey 主体上下文键
	SubjectKey ContextKey = "auth.subject"
)

// AuthMiddlewareConfig 认证中间件配置
type AuthMiddlewareConfig struct {
	TokenManager TokenManager
	SkipPaths    []string
	HeaderName   string
	TokenPrefix  string
	Logger       log.Logger
}

// DefaultAuthMiddlewareConfig 默认认证中间件配置
func DefaultAuthMiddlewareConfig() *AuthMiddlewareConfig {
	return &AuthMiddlewareConfig{
		SkipPaths:   []string{"/api/v1/auth/login", "/health"},
		HeaderName:  "Authorization",
		TokenPrefix: "Bearer ",
	}
}

// AuthMiddleware 创建认证中间件
func AuthMiddleware(config *AuthMiddlewareConfig) middleware.Middleware {
	logger := log.NewHelper(log.With(config.Logger, "middleware", "auth"))
	
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}
			
			// 检查是否跳过认证
			if shouldSkip(tr, config) {
				return handler(ctx, req)
			}
			
			// 提取令牌
			token, err := extractToken(tr, config)
			if err != nil {
				logger.WithContext(ctx).Warnf("failed to extract token: %v", err)
				return nil, errors.Unauthorized("AUTH_TOKEN_MISSING", "Authentication token is required")
			}
			
			// 验证令牌
			subject, err := config.TokenManager.VerifyToken(ctx, token)
			if err != nil {
				logger.WithContext(ctx).Warnf("token verification failed: %v", err)
				return nil, errors.Unauthorized("AUTH_TOKEN_INVALID", "Invalid authentication token")
			}
			
			// 将主体放入上下文
			ctx = context.WithValue(ctx, SubjectKey, subject)
			
			logger.WithContext(ctx).Debugf("authentication successful for subject: %s", subject.ID)
			return handler(ctx, req)
		}
	}
}

// GetSubjectFromContext 从上下文获取主体
func GetSubjectFromContext(ctx context.Context) *Subject {
	if subject, ok := ctx.Value(SubjectKey).(*Subject); ok {
		return subject
	}
	return nil
}

// extractToken 提取令牌
func extractToken(tr transport.Transporter, config *AuthMiddlewareConfig) (string, error) {
	if header, ok := tr.(interface{ RequestHeader() transport.Header }); ok {
		auth := header.RequestHeader().Get(config.HeaderName)
		if auth != "" {
			if strings.HasPrefix(auth, config.TokenPrefix) {
				return strings.TrimPrefix(auth, config.TokenPrefix), nil
			}
			return auth, nil
		}
	}
	
	return "", fmt.Errorf("token not found")
}

// shouldSkip 检查是否应该跳过认证
func shouldSkip(tr transport.Transporter, config *AuthMiddlewareConfig) bool {
	// 简化实现：检查跳过路径
	for _, skipPath := range config.SkipPaths {
		// 这里应该从transport中获取实际路径进行匹配
		// 由于简化实现，暂时返回false
		_ = skipPath
	}
	return false
}