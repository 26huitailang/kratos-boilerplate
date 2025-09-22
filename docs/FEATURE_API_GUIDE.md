# 功能裁剪系统 API 使用指南

## 概述

本文档提供功能裁剪系统的详细API使用指南，包括所有新增的接口和功能。

## 核心接口

### 1. FeatureChecker - 基础功能检查

基础的功能开关检查接口，用于最常见的开关查询操作。

```go
type FeatureChecker interface {
    IsEnabled(ctx context.Context, flag FeatureFlag) bool
    IsEnabledWithContext(ctx context.Context, flag FeatureFlag, evalCtx *EvaluationContext) bool
}
```

#### 使用示例

```go
// 基本检查
if checker.IsEnabled(ctx, FeatureUserManagement) {
    // 功能已启用
    handleUserManagement()
}

// 带上下文检查
evalCtx := &EvaluationContext{
    UserID:      "user123",
    UserType:    "admin",
    Environment: "production",
}

if checker.IsEnabledWithContext(ctx, FeatureUserManagement, evalCtx) {
    // 基于用户上下文的功能检查
    handleAdminUserManagement()
}
```

### 2. HierarchicalChecker - 分层功能检查

支持分层功能结构的检查接口，用于A.B.C格式的功能管理。

```go
type HierarchicalChecker interface {
    IsFeatureEnabled(featurePath string) bool
    IsDomainEnabled(domain string) bool
    GetDomainFeatures(domain string) []string
    GetFeatureTree() map[string][]string
}
```

#### 使用示例

```go
// 检查分层功能
if checker.IsFeatureEnabled("user.management") {
    // 用户管理功能已启用
}

// 检查功能域
if checker.IsDomainEnabled("user") {
    // 用户域有启用的功能
}

// 获取域下的所有功能
features := checker.GetDomainFeatures("user")
// 返回: ["user.management", "user.profile", "user.security"]

// 获取功能树
tree := checker.GetFeatureTree()
// 返回: {"user": ["user.management", "user.profile"], "system": ["system.config"]}
```

### 3. CapabilityChecker - 能力检查

检查功能集合中的具体能力，支持细粒度的权限控制。

```go
type CapabilityChecker interface {
    HasPermission(featurePath, permission string) bool
    HasAPI(featurePath, api string) bool
    HasRoute(featurePath, route string) bool
    HasConfig(featurePath, config string) bool
    HasAuditLog(featurePath, logType string) bool
    GetFeatureCapabilities(featurePath string) *FeatureCapabilities
}
```

#### 使用示例

```go
// 检查权限
if checker.HasPermission("user.management", "user.create") {
    // 用户管理功能包含创建用户权限
    allowUserCreation()
}

// 检查API
if checker.HasAPI("user.management", "POST /api/v1/users") {
    // 注册API端点
    registerUserCreationAPI()
}

// 检查路由
if checker.HasRoute("user.management", "/admin/users") {
    // 注册前端路由
    registerAdminRoute()
}

// 获取完整能力
capabilities := checker.GetFeatureCapabilities("user.management")
if capabilities != nil {
    log.Printf("权限: %v", capabilities.Permissions)
    log.Printf("API: %v", capabilities.APIs)
    log.Printf("路由: %v", capabilities.Routes)
}
```

### 4. FeatureManager - 功能管理

管理功能开关的配置和状态。

```go
type FeatureManager interface {
    GetToggleConfig(flag FeatureFlag) (*ToggleConfig, error)
    UpdateToggle(flag FeatureFlag, config *ToggleConfig) error
    ListToggles() map[FeatureFlag]*ToggleConfig
    EnableFeature(flag FeatureFlag) error
    DisableFeature(flag FeatureFlag) error
    DeleteToggle(flag FeatureFlag) error
}
```

#### 使用示例

```go
// 获取配置
config, err := manager.GetToggleConfig(FeatureUserManagement)
if err != nil {
    // 处理错误
    if featureErr, ok := err.(*FeatureError); ok {
        if featureErr.Code == ErrCodeFeatureNotFound {
            // 功能不存在
        }
    }
}

// 更新配置
newConfig := &ToggleConfig{
    Enabled:  true,
    Strategy: StrategyPercentage,
    Rules: map[string]interface{}{
        "percentage": 50,
    },
    Description: "用户管理功能 - 50%灰度发布",
}

err = manager.UpdateToggle(FeatureUserManagement, newConfig)
if err != nil {
    log.Printf("更新失败: %v", err)
}

// 快速启用/禁用
err = manager.EnableFeature(FeatureUserManagement)
err = manager.DisableFeature(FeatureUserManagement)

// 列出所有功能
toggles := manager.ListToggles()
for feature, config := range toggles {
    log.Printf("功能 %s: 启用=%t, 策略=%s", feature, config.Enabled, config.Strategy)
}
```

## 错误处理

### FeatureError 类型

系统提供了结构化的错误类型，便于错误处理和调试。

```go
type FeatureError struct {
    Code      string      `json:"code"`
    Message   string      `json:"message"`
    Feature   FeatureFlag `json:"feature,omitempty"`
    Operation string      `json:"operation,omitempty"`
    Cause     error       `json:"cause,omitempty"`
    Details   map[string]interface{} `json:"details,omitempty"`
}
```

#### 错误处理示例

```go
err := manager.UpdateToggle(feature, config)
if err != nil {
    if featureErr, ok := err.(*FeatureError); ok {
        switch featureErr.Code {
        case ErrCodeFeatureNotFound:
            // 功能不存在
            log.Printf("功能 %s 不存在", featureErr.Feature)
            
        case ErrCodeInvalidConfig:
            // 配置无效
            log.Printf("配置无效: %s", featureErr.Message)
            
        case ErrCodeRepositoryError:
            // 存储错误，可能需要重试
            if featureErr.IsRetryable() {
                time.Sleep(time.Second)
                // 重试逻辑
            }
            
        default:
            log.Printf("未知错误: %s", featureErr.Error())
        }
    }
}

// 错误链处理
if cause := featureErr.Unwrap(); cause != nil {
    log.Printf("根本原因: %v", cause)
}

// 检查错误类型
if featureErr.IsClientError() {
    // 客户端错误，返回400
    http.Error(w, featureErr.Error(), http.StatusBadRequest)
} else if featureErr.IsServerError() {
    // 服务端错误，返回500
    http.Error(w, "内部服务器错误", http.StatusInternalServerError)
}
```

### 预定义错误创建函数

```go
// 功能不存在
err := ErrFeatureNotFound(FeatureUserManagement)

// 配置无效
err := ErrInvalidConfig(FeatureUserManagement, "策略不支持")

// 策略不支持
err := ErrStrategyNotSupported("unknown_strategy")

// 存储错误
err := ErrRepositoryError("save", cause)

// 带详细信息的错误
err := NewFeatureError(ErrCodeValidation, "validate", "验证失败")
    .WithFeature(FeatureUserManagement)
    .WithDetail("field", "strategy")
    .WithDetail("value", "invalid")
```

## 上下文提取器

### ConfigurableExtractor

可配置的上下文提取器，支持自定义键名和默认值。

```go
// 配置提取器
config := &ExtractorConfig{
    Keys: map[string]string{
        "user_id":     "x-user-id",
        "user_type":   "x-user-type",
        "environment": "x-environment",
    },
    Defaults: map[string]string{
        "environment": "production",
        "version":     "1.0.0",
    },
    Attributes: map[string]string{
        "tenant": "x-tenant-id",
        "region": "x-region",
    },
}

extractor := NewConfigurableExtractor(config)

// 从HTTP头提取
headerExtractor := NewHTTPHeaderExtractor(config)
headerExtractor.SetHeaderMapping("X-Custom-Header", "custom_attr")

// 链式提取器
chain := NewChainExtractor(extractor, headerExtractor)
```

#### 使用示例

```go
// 在HTTP中间件中使用
func FeatureMiddleware(extractor EvaluationContextExtractor) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 设置上下文值
            ctx := r.Context()
            ctx = context.WithValue(ctx, "x-user-id", r.Header.Get("X-User-ID"))
            ctx = context.WithValue(ctx, "x-tenant-id", r.Header.Get("X-Tenant-ID"))
            
            // 提取评估上下文
            evalCtx := extractor.Extract(ctx)
            
            // 检查功能
            if checker.IsEnabledWithContext(ctx, FeatureUserManagement, evalCtx) {
                // 功能启用，继续处理
                next.ServeHTTP(w, r.WithContext(ctx))
            } else {
                // 功能未启用
                http.Error(w, "功能未启用", http.StatusForbidden)
            }
        })
    }
}
```

## 审计日志

### AuditLogger 接口

提供完整的审计日志功能。

```go
type AuditLogger interface {
    LogFeatureChange(ctx context.Context, eventType AuditEventType, feature FeatureFlag, oldConfig, newConfig *ToggleConfig) error
    LogFeatureAccess(ctx context.Context, feature FeatureFlag, operation string, result bool, evalCtx *EvaluationContext, duration time.Duration) error
    LogPermissionCheck(ctx context.Context, feature FeatureFlag, permission string, result bool, evalCtx *EvaluationContext) error
    LogError(ctx context.Context, operation string, err error, metadata map[string]interface{}) error
    LogSystemEvent(ctx context.Context, eventType AuditEventType, message string, metadata map[string]interface{}) error
    GetAuditLogs(ctx context.Context, filter AuditFilter) ([]AuditLog, error)
}
```

#### 使用示例

```go
// 配置审计日志
auditConfig := &AuditLoggerConfig{
    Enabled:       true,
    ExcludeEvents: []AuditEventType{EventFeatureAccessed},
    MaskSensitive: true,
}

auditLogger := NewDefaultAuditLogger(repository, extractor, logger, auditConfig)

// 手动记录审计日志
err := auditLogger.LogFeatureAccess(
    ctx,
    FeatureUserManagement,
    "check_permission",
    true,
    evalCtx,
    time.Millisecond*100,
)

// 查询审计日志
filter := AuditFilter{
    StartTime:  &startTime,
    EndTime:    &endTime,
    Features:   []FeatureFlag{FeatureUserManagement},
    EventTypes: []AuditEventType{EventFeatureUpdated},
    Limit:      100,
}

logs, err := auditLogger.GetAuditLogs(ctx, filter)
for _, log := range logs {
    fmt.Printf("时间: %s, 事件: %s, 功能: %s, 用户: %s\n",
        log.Timestamp, log.EventType, log.Feature, log.UserID)
}
```

## 配置验证

### ConfigValidator 接口

提供配置验证和一致性检查。

```go
// 创建验证器
settings := &ValidationSettings{
    Strict:                 true,
    RequiredFields:         []string{"enabled", "strategy", "metadata.owner"},
    AllowedStrategies:      []string{"simple", "percentage", "user"},
    AllowedEnvironments:    []string{"development", "staging", "production"},
    FeatureNamePattern:     `^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*$`,
    MaxFeatures:            1000,
    MaxRulesPerFeature:     10,
    RequireOwnership:       true,
    RequireDocumentation:   true,
}

validator := NewDefaultConfigValidator(settings)

// 验证配置
result := validator.ValidateConfig(config)
if !result.Valid {
    for _, err := range result.Errors {
        log.Printf("错误: %s - %s (功能: %s)", err.Code, err.Message, err.Feature)
    }
    
    for _, warning := range result.Warnings {
        log.Printf("警告: %s - %s", warning.Code, warning.Message)
    }
}

// 验证单个功能
toggleResult := validator.ValidateToggle(FeatureUserManagement, toggleConfig)

// 添加自定义验证器
customValidator := &MyCustomValidator{}
validator.AddCustomValidator(customValidator)
```

## 性能优化

### 缓存使用

```go
// 配置缓存
cacheConfig := &CacheConfig{
    Enabled:         true,
    MaxSize:         1000,
    DefaultTTL:      5 * time.Minute,
    CleanupInterval: time.Minute,
}

// 创建缓存管理器
cachedManager := NewCachedToggleManager(baseManager, &CachedManagerConfig{
    CacheConfig:      cacheConfig,
    EnableCacheWrite: true,
}, logger)

// 使用缓存
result := cachedManager.IsEnabledWithContext(ctx, feature, evalCtx)

// 缓存统计
stats := cachedManager.GetCacheStats()
log.Printf("缓存命中率: %.2f%%", stats["hit_rate"].(float64)*100)

// 手动清理缓存
cachedManager.SetCacheEnabled(false) // 禁用缓存
```

### 对象池使用

```go
// 使用预定义对象池
config := ToggleConfigPool.Get()
defer ToggleConfigPool.Put(config)

// 使用配置
config.Enabled = true
config.Strategy = StrategySimple
config.Rules["percentage"] = 50

// 评估上下文池
evalCtx := EvaluationContextPool.Get()
defer EvaluationContextPool.Put(evalCtx)

evalCtx.UserID = "user123"
evalCtx.Environment = "production"

// 验证结果池
result := ValidationResultPool.Get()
defer ValidationResultPool.Put(result)
```

## 配置管理

### 标准化配置

```go
// 创建配置管理器
parser := NewYAMLConfigurationParser()
migrator := NewDefaultConfigurationMigrator()
validator := NewDefaultConfigValidator(nil)

configManager := NewConfigurationManager(parser, migrator, validator)

// 加载配置
yamlData := []byte(`
apiVersion: v1
kind: FeatureConfiguration
metadata:
  name: my-features
spec:
  features:
    user.management:
      enabled: true
      strategy: simple
`)

config, err := configManager.LoadConfiguration(yamlData)
if err != nil {
    log.Fatal(err)
}

// 创建默认配置
defaultConfig := configManager.CreateDefaultConfiguration("my-app-features")

// 保存配置
data, err := configManager.SaveConfiguration(config)
```

## 事件订阅

### 功能变更事件

```go
// 订阅功能变更
callback := func(flag FeatureFlag, oldConfig, newConfig *ToggleConfig) {
    if oldConfig == nil {
        log.Printf("功能 %s 已创建", flag)
    } else if newConfig == nil {
        log.Printf("功能 %s 已删除", flag)
    } else {
        log.Printf("功能 %s 已更新: %t -> %t", flag, oldConfig.Enabled, newConfig.Enabled)
    }
}

err := manager.Subscribe(callback)
if err != nil {
    log.Printf("订阅失败: %v", err)
}

// 取消订阅
err = manager.Unsubscribe(callback)
```

## 中间件集成

### HTTP中间件

```go
func FeatureToggleMiddleware(checker FeatureChecker) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 从URL路径确定需要检查的功能
            feature := extractFeatureFromPath(r.URL.Path)
            if feature != "" {
                if !checker.IsEnabled(r.Context(), FeatureFlag(feature)) {
                    http.Error(w, "功能未启用", http.StatusForbidden)
                    return
                }
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### gRPC拦截器

```go
func FeatureToggleInterceptor(checker FeatureChecker) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // 从方法名确定功能
        feature := extractFeatureFromMethod(info.FullMethod)
        if feature != "" {
            if !checker.IsEnabled(ctx, FeatureFlag(feature)) {
                return nil, status.Error(codes.PermissionDenied, "功能未启用")
            }
        }
        
        return handler(ctx, req)
    }
}
```

## 最佳实践

### 1. 错误处理

```go
// 始终检查具体的错误类型
if err != nil {
    if featureErr, ok := err.(*FeatureError); ok {
        // 根据错误代码采取不同的处理策略
        switch featureErr.Code {
        case ErrCodeFeatureNotFound:
            // 创建默认配置
        case ErrCodeRepositoryError:
            // 重试或降级
        }
    }
}
```

### 2. 上下文管理

```go
// 在HTTP请求开始时提取上下文
evalCtx := extractor.Extract(ctx)

// 在整个请求生命周期中重用
if checker.IsEnabledWithContext(ctx, feature1, evalCtx) {
    // ...
}
if checker.IsEnabledWithContext(ctx, feature2, evalCtx) {
    // ...
}
```

### 3. 缓存策略

```go
// 为频繁访问的功能启用缓存
if isHighTrafficFeature(feature) {
    result := cachedChecker.IsEnabled(ctx, feature)
} else {
    result := directChecker.IsEnabled(ctx, feature)
}
```

### 4. 配置验证

```go
// 在部署前验证配置
result := validator.ValidateConfig(config)
if !result.Valid {
    // 阻止部署
    return fmt.Errorf("配置验证失败: %d错误", len(result.Errors))
}
```

这个API使用指南涵盖了功能裁剪系统的所有主要接口和功能，提供了详细的使用示例和最佳实践建议。