# 认证鉴权模块 (Auth Module)

## 概述

本模块提供了完整的企业级认证鉴权解决方案，支持多种认证策略、多因素认证(MFA)、细粒度权限控制和基于角色的访问控制(RBAC)。

## 特性

### 🔐 多种认证策略
- **密码认证**: 基于用户名/密码的传统认证
- **API密钥认证**: 适用于API访问的密钥认证
- **OAuth2认证**: 支持第三方OAuth2提供商
- **扩展性**: 可轻松添加新的认证策略

### 🛡️ JWT令牌管理
- **令牌生成**: 支持访问令牌、刷新令牌、API令牌
- **令牌验证**: 完整的JWT验证机制
- **令牌撤销**: 支持令牌黑名单
- **安全性**: 可配置的签名算法和密钥

### 🔒 多因素认证 (MFA)
- **TOTP**: 基于时间的一次性密码 (Google Authenticator 兼容)
- **短信验证**: SMS验证码支持
- **邮件验证**: 邮件验证码支持
- **备用代码**: 紧急访问备用代码

### 🎯 权限管理系统
- **RBAC**: 基于角色的访问控制
- **ABAC**: 基于属性的访问控制
- **策略引擎**: 灵活的权限策略配置
- **条件访问**: 支持时间、IP、属性等条件

### 🚀 中间件支持
- **认证中间件**: 自动处理令牌验证
- **权限中间件**: 资源级权限控制
- **角色中间件**: 角色要求检查
- **MFA中间件**: 多因素认证要求

## 快速开始

### 1. 基本配置

```yaml
auth:
  jwt:
    secret: "your-secret-key"
    access_expiry: "1h"
    refresh_expiry: "24h"
    issuer: "your-app"
    signing_method: "HS256"
  
  mfa:
    enabled: true
    totp_issuer: "Your App"
    totp_period: "30s"
    backup_codes: 10
  
  oauth2:
    enabled: true
    providers:
      google:
        client_id: "your-google-client-id"
        client_secret: "your-google-client-secret"
        auth_url: "https://accounts.google.com/o/oauth2/auth"
        token_url: "https://oauth2.googleapis.com/token"
        user_info_url: "https://www.googleapis.com/oauth2/v2/userinfo"
        scopes: ["openid", "email", "profile"]
```

### 2. 初始化认证管理器

```go
package main

import (
    "github.com/go-kratos/kratos/v2/log"
    "your-project/internal/pkg/auth"
)

func setupAuth() auth.AuthManager {
    config := &auth.AuthConfig{
        // 配置信息
    }
    
    // 创建认证管理器
    authManager := auth.NewAuthManager(config, log.DefaultLogger)
    
    // 注册认证策略
    passwordStrategy := auth.NewPasswordAuthStrategy(userRepo, logger)
    authManager.RegisterStrategy(passwordStrategy)
    
    apiKeyStrategy := auth.NewAPIKeyAuthStrategy(apiKeyRepo, userRepo, logger)
    authManager.RegisterStrategy(apiKeyStrategy)
    
    return authManager
}
```

### 3. 使用认证中间件

```go
package main

import (
    "github.com/go-kratos/kratos/v2/middleware"
    "your-project/internal/pkg/auth"
)

func setupMiddleware(authManager auth.AuthManager) []middleware.Middleware {
    config := &auth.AuthMiddlewareConfig{
        AuthManager:  authManager,
        TokenManager: authManager.GetTokenManager(),
        SkipPaths:    []string{"/api/v1/auth/login", "/health"},
        HeaderName:   "Authorization",
        TokenPrefix:  "Bearer ",
    }
    
    return []middleware.Middleware{
        auth.AuthMiddleware(config),
        auth.PermissionMiddleware(permissionChecker, "posts", "read"),
        auth.RoleMiddleware("admin", "user"),
    }
}
```

## 使用示例

### 密码认证

```go
// 登录
credentials := &auth.PasswordCredentials{
    Username: "user@example.com",
    Password: "password123",
}

subject, err := authManager.Authenticate(ctx, "password", credentials)
if err != nil {
    return err
}

// 生成令牌
token, err := authManager.GetTokenManager().GenerateToken(ctx, subject, auth.TokenTypeAccess)
if err != nil {
    return err
}

fmt.Printf("Access Token: %s\n", token.Value)
```

### API密钥认证

```go
credentials := &auth.APIKeyCredentials{
    APIKey: "sk-1234567890abcdef",
}

subject, err := authManager.Authenticate(ctx, "apikey", credentials)
if err != nil {
    return err
}
```

### 设置TOTP多因素认证

```go
// 1. 设置TOTP
setupInfo, err := mfaManager.SetupTOTP(ctx, "user123")
if err != nil {
    return err
}

fmt.Printf("Secret: %s\n", setupInfo.Secret)
fmt.Printf("QR Code: %s\n", setupInfo.QRCodeURL)
fmt.Printf("Backup Codes: %v\n", setupInfo.BackupCodes)

// 2. 用户扫描二维码后，验证并启用
err = mfaManager.EnableMFA(ctx, "user123", auth.MFAMethodTOTP, "123456")
if err != nil {
    return err
}
```

### 权限检查

```go
// 直接权限检查
subject := &auth.Subject{
    ID:          "user123",
    Permissions: []string{"posts:read", "posts:write"},
    Roles:       []string{"editor"},
}

err := authManager.CheckPermission(ctx, subject, "posts", "read")
if err != nil {
    // 权限不足
    return err
}
```

### 从上下文获取认证信息

```go
func handleRequest(ctx context.Context) error {
    // 获取当前认证主体
    subject := auth.GetSubjectFromContext(ctx)
    if subject == nil {
        return errors.New("authentication required")
    }
    
    fmt.Printf("User ID: %s\n", subject.ID)
    fmt.Printf("Roles: %v\n", subject.Roles)
    fmt.Printf("Permissions: %v\n", subject.Permissions)
    
    return nil
}
```

## 高级功能

### 自定义认证策略

```go
type CustomAuthStrategy struct {
    // 自定义字段
}

func (s *CustomAuthStrategy) GetName() string {
    return "custom"
}

func (s *CustomAuthStrategy) Authenticate(ctx context.Context, credentials interface{}) (*auth.Subject, error) {
    // 实现自定义认证逻辑
    return &auth.Subject{
        ID:   "custom-user",
        Type: "custom",
    }, nil
}

// 注册自定义策略
authManager.RegisterStrategy(&CustomAuthStrategy{})
```

### 自定义权限检查器

```go
type CustomPermissionChecker struct {
    // 自定义字段
}

func (c *CustomPermissionChecker) Check(ctx context.Context, subject *auth.Subject, resource string, action string) error {
    // 实现自定义权限检查逻辑
    return nil
}

// 设置自定义权限检查器
authManager.SetPermissionChecker(&CustomPermissionChecker{})
```

### 令牌黑名单

```go
// 使用Redis作为令牌黑名单
blacklist := auth.NewRedisTokenBlacklist(redisClient, logger)
tokenManager.SetBlacklist(blacklist)

// 撤销令牌
err := tokenManager.RevokeToken(ctx, "jwt-token-string")
if err != nil {
    return err
}
```

## API文档

### 核心接口

#### AuthManager
- `RegisterStrategy(strategy AuthStrategy) error`: 注册认证策略
- `Authenticate(ctx context.Context, strategyName string, credentials interface{}) (*Subject, error)`: 执行认证
- `GetTokenManager() TokenManager`: 获取令牌管理器
- `CheckPermission(ctx context.Context, subject *Subject, resource string, action string) error`: 检查权限

#### TokenManager
- `GenerateToken(ctx context.Context, subject *Subject, tokenType TokenType) (*Token, error)`: 生成令牌
- `VerifyToken(ctx context.Context, tokenValue string) (*Subject, error)`: 验证令牌
- `RevokeToken(ctx context.Context, tokenValue string) error`: 撤销令牌
- `RefreshToken(ctx context.Context, refreshToken string) (*Token, error)`: 刷新令牌

#### MFAManager
- `SetupTOTP(ctx context.Context, userID string) (*TOTPSetupInfo, error)`: 设置TOTP
- `EnableMFA(ctx context.Context, userID string, method MFAMethod, code string) error`: 启用MFA
- `CreateChallenge(ctx context.Context, userID string, method MFAMethod) (*MFAChallenge, error)`: 创建MFA挑战
- `VerifyChallenge(ctx context.Context, challengeID string, code string) error`: 验证MFA挑战

## 测试

运行单元测试：
```bash
go test -v ./internal/pkg/auth/...
```

运行性能测试：
```bash
go test -bench=. ./internal/pkg/auth/...
```

## 安全注意事项

1. **密钥管理**: 确保JWT密钥的安全性，定期轮换
2. **令牌过期**: 设置合理的令牌过期时间
3. **传输安全**: 在生产环境中使用HTTPS
4. **日志记录**: 记录认证失败和可疑活动
5. **速率限制**: 实施登录尝试速率限制
6. **会话管理**: 实施适当的会话超时和并发控制

## 性能优化

1. **令牌验证**: JWT验证是无状态的，性能较好
2. **权限缓存**: 缓存用户权限信息以减少数据库查询
3. **连接池**: 使用数据库连接池
4. **异步处理**: 异步处理MFA通知发送

## 监控和日志

模块内置了详细的日志记录，包括：
- 认证成功/失败事件
- 权限检查结果
- MFA操作记录
- 令牌操作日志

建议配置适当的日志级别和输出格式以便于监控和调试。