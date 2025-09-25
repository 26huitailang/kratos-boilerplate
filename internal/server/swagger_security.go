package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/swagger-api/openapiv2"

	"kratos-boilerplate/internal/pkg/auth"
	"kratos-boilerplate/internal/pkg/feature"
)

// SwaggerUISecurityConfig Swagger UI安全配置
type SwaggerUISecurityConfig struct {
	// 是否启用认证保护
	EnableAuth bool `json:"enable_auth"`
	
	// 允许的IP地址列表
	AllowedIPs []string `json:"allowed_ips"`
	
	// 允许的用户类型
	AllowedUserTypes []string `json:"allowed_user_types"`
	
	// 是否记录访问日志
	EnableAuditLog bool `json:"enable_audit_log"`
	
	// 自定义访问路径（默认为 /q/）
	PathPrefix string `json:"path_prefix"`
}

// SwaggerUIMiddleware Swagger UI访问中间件
type SwaggerUIMiddleware struct {
	config        *SwaggerUISecurityConfig
	authService   *auth.Service
	featureToggle feature.FeatureToggle
	logger        *log.Helper
}

// NewSwaggerUIMiddleware 创建Swagger UI中间件
func NewSwaggerUIMiddleware(
	config *SwaggerUISecurityConfig,
	authService *auth.Service,
	featureToggle feature.FeatureToggle,
	logger log.Logger,
) *SwaggerUIMiddleware {
	return &SwaggerUIMiddleware{
		config:        config,
		authService:   authService,
		featureToggle: featureToggle,
		logger:        log.NewHelper(logger),
	}
}

// Handler 创建受保护的Swagger UI处理器
func (m *SwaggerUIMiddleware) Handler() http.Handler {
	baseHandler := openapiv2.NewHandler()
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		// 1. 检查功能开关
		if !m.featureToggle.IsEnabledWithContext(ctx, feature.FeatureSwaggerUI, nil) {
			m.logger.Warn("Swagger UI access denied: feature disabled")
			http.Error(w, "Swagger UI is currently disabled", http.StatusForbidden)
			return
		}
		
		// 2. IP白名单检查
		if len(m.config.AllowedIPs) > 0 {
			clientIP := getClientIP(r)
			if !m.isIPAllowed(clientIP) {
				m.logger.Warnf("Swagger UI access denied: IP %s not in whitelist", clientIP)
				http.Error(w, "Access denied: IP not allowed", http.StatusForbidden)
				return
			}
		}
		
		// 3. 认证检查
		if m.config.EnableAuth {
			if err := m.checkAuthentication(r); err != nil {
				m.logger.Warnf("Swagger UI access denied: authentication failed - %v", err)
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
		}
		
		// 4. 记录访问日志
		if m.config.EnableAuditLog {
			m.logAccess(r)
		}
		
		// 5. 调用原始处理器
		baseHandler.ServeHTTP(w, r)
	})
}

// checkAuthentication 检查用户认证
func (m *SwaggerUIMiddleware) checkAuthentication(r *http.Request) error {
	// 从Authorization头获取token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return errors.Unauthorized("MISSING_TOKEN", "Authorization header required")
	}
	
	// 检查Bearer token格式
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return errors.Unauthorized("INVALID_TOKEN_FORMAT", "Invalid authorization header format")
	}
	
	token := parts[1]
	
	// 验证token
	claims, err := m.authService.ValidateToken(r.Context(), token)
	if err != nil {
		return errors.Unauthorized("INVALID_TOKEN", err.Error())
	}
	
	// 检查用户类型权限
	if len(m.config.AllowedUserTypes) > 0 {
		userType := claims["user_type"].(string)
		if !m.isUserTypeAllowed(userType) {
			return errors.Forbidden("INSUFFICIENT_PERMISSION", "User type not allowed to access Swagger UI")
		}
	}
	
	return nil
}

// isIPAllowed 检查IP是否在白名单中
func (m *SwaggerUIMiddleware) isIPAllowed(clientIP string) bool {
	for _, allowedIP := range m.config.AllowedIPs {
		if clientIP == allowedIP || allowedIP == "*" {
			return true
		}
		// 支持CIDR格式的IP段检查
		// 这里可以添加更复杂的IP匹配逻辑
	}
	return false
}

// isUserTypeAllowed 检查用户类型是否允许访问
func (m *SwaggerUIMiddleware) isUserTypeAllowed(userType string) bool {
	for _, allowedType := range m.config.AllowedUserTypes {
		if userType == allowedType {
			return true
		}
	}
	return false
}

// logAccess 记录访问日志
func (m *SwaggerUIMiddleware) logAccess(r *http.Request) {
	clientIP := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")
	
	m.logger.Infow(
		"swagger_ui_access",
		"ip", clientIP,
		"user_agent", userAgent,
		"path", r.URL.Path,
		"method", r.Method,
		"timestamp", log.DefaultTimestamp,
	)
}

// getClientIP 获取客户端真实IP
func getClientIP(r *http.Request) string {
	// 检查X-Forwarded-For头
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// 检查X-Real-IP头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// 使用RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// RegisterSecureSwaggerUI 注册受保护的Swagger UI
func RegisterSecureSwaggerUI(
	httpSrv *kratosHttp.Server,
	config *SwaggerUISecurityConfig,
	authService *auth.Service,
	featureToggle feature.FeatureToggle,
	logger log.Logger,
) {
	if config == nil {
		config = &SwaggerUISecurityConfig{
			EnableAuth:       true,
			AllowedIPs:       []string{}, // 空表示不限制IP
			AllowedUserTypes: []string{"admin", "developer"},
			EnableAuditLog:   true,
			PathPrefix:       "/q/",
		}
	}
	
	middleware := NewSwaggerUIMiddleware(config, authService, featureToggle, logger)
	handler := middleware.Handler()
	
	// 注册处理器
	httpSrv.HandlePrefix(config.PathPrefix, handler)
	
	log.NewHelper(logger).Infof("Swagger UI registered at %s with security enabled", config.PathPrefix)
}

// 示例配置文件格式 (config.yaml)
/*
swagger_ui:
  security:
    enable_auth: true
    allowed_ips:
      - "127.0.0.1"
      - "192.168.1.0/24"
    allowed_user_types:
      - "admin"
      - "developer"
    enable_audit_log: true
    path_prefix: "/q/"
  
  # 环境特定配置
  environments:
    development:
      enable_auth: false
      allowed_ips: []
    staging:
      enable_auth: true
      allowed_ips:
        - "10.0.0.0/8"
    production:
      enable_auth: true
      allowed_ips:
        - "172.16.0.100"
        - "172.16.0.101"
*/