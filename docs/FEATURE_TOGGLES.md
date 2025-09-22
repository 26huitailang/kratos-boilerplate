# 功能开关系统设计文档（更新版）

> **重要更新**: 基于设计审核报告，系统已进行全面重构。建议参考以下新文档：
> - [功能裁剪系统 - 增强版设计与实现](./FEATURE_SYSTEM_ENHANCED.md)
> - [功能裁剪系统 API 使用指南](./FEATURE_API_GUIDE.md)

## 概述

本项目实现了一个完整的配置驱动功能裁剪系统，支持分层的功能集合管理。每个功能集合(Feature Set)包含权限点、审计日志、API接口、UI路由和配置点的完整组合。系统采用A.B.C的分层结构，支持多级配置策略和运行时动态控制。

**⚠️ 注意**: 本文档描述的是原有的功能开关系统。新的增强版系统提供了更完善的接口设计、错误处理、审计日志和性能优化功能。

## 核心特性

### 1. 分层功能集合
- **分层结构**: 采用A.B.C格式，如user.management、system.config
- **功能集合**: 每个功能包含权限、日志、API、UI、配置的完整组合
- **多级配置**: base基础配置 → oem定制配置 → external外部配置 → final最终配置
- **域级控制**: 支持按功能域(user、system、report等)进行批量控制

### 2. 多策略支持
- **简单开关 (Simple)**: 基本的开/关功能
- **百分比发布 (Percentage)**: 基于用户ID的一致性哈希进行百分比发布
- **用户定向 (User)**: 基于用户属性、白名单、黑名单的定向发布
- **时间控制 (Time)**: 基于时间范围、工作日、工作时间的控制
- **环境控制 (Environment)**: 基于环境、版本范围的控制

### 3. 动态配置
- 支持运行时配置变更，无需重启服务
- 文件监听自动重新加载配置
- 通过API进行实时配置管理
- CSV导出功能，便于产品和测试验证

### 4. 中间件集成
- HTTP/gRPC中间件自动检查功能开关
- 服务装饰器支持方法级别的功能控制
- 支持渐进式发布和熔断器模式

## 系统架构

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   API Layer     │    │   Middleware     │    │  Configuration  │
│                 │    │                  │    │                 │
│ - REST API      │    │ - HTTP/gRPC      │    │ - YAML/JSON     │
│ - gRPC API      │    │ - Decorators     │    │ - File Watch    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
┌─────────────────────────────────▼─────────────────────────────────┐
│                    Feature Toggle Manager                         │
│                                                                   │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐   │
│  │   Repository    │  │   Evaluators    │  │   Event Bus     │   │
│  │                 │  │                 │  │                 │   │
│  │ - File Store    │  │ - Simple        │  │ - Callbacks     │   │
│  │ - Database      │  │ - Percentage    │  │ - Notifications │   │
│  │ - Redis         │  │ - User          │  │                 │   │
│  │                 │  │ - Time          │  │                 │   │
│  │                 │  │ - Environment   │  │                 │   │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘   │
└───────────────────────────────────────────────────────────────────┘
```

## 快速开始

### 1. 基础配置

在 `configs/config.yaml` 中添加功能开关配置：

```yaml
features:
  enabled: true
  config_file: "./configs/features.yaml"
  config_format: "yaml"
  watch_config: true
  default_environment: "production"
  repository:
    type: "file"
    config_path: "./configs/features.yaml"
    format: "yaml"
```

### 2. 分层功能集合定义

#### 基础配置 (`configs/features-base.yaml`)：

```yaml
features:
  # 用户功能域
  user.management:
    enabled: true
    strategy: simple
    permissions:
      - "user.create"
      - "user.update" 
      - "user.delete"
      - "user.list"
    audit_logs:
      - "user_created"
      - "user_updated"
      - "user_deleted"
    apis:
      - "POST /api/v1/users"
      - "PUT /api/v1/users/{id}"
      - "DELETE /api/v1/users/{id}"
      - "GET /api/v1/users"
    routes:
      - "/admin/users"
      - "/admin/users/create"
    configs:
      - "user.max_count"
      - "user.password_policy"
    description: "用户管理功能集合"
    tags: ["user", "admin"]

  user.profile:
    enabled: true
    strategy: simple
    permissions:
      - "profile.view"
      - "profile.edit"
    audit_logs:
      - "profile_updated"
    apis:
      - "GET /api/v1/profile"
      - "PUT /api/v1/profile"
    routes:
      - "/profile"
      - "/profile/edit"
    configs:
      - "profile.avatar_size"
    description: "用户资料功能集合"
    tags: ["user", "profile"]

  # 系统功能域
  system.config:
    enabled: true
    strategy: environment
    rules:
      environments: ["development", "staging", "production"]
    permissions:
      - "system.config.view"
      - "system.config.edit"
    audit_logs:
      - "system_config_changed"
    apis:
      - "GET /api/v1/system/config"
      - "PUT /api/v1/system/config"
    routes:
      - "/admin/system/config"
    configs:
      - "system.debug_mode"
      - "system.log_level"
    description: "系统配置功能集合"
    tags: ["system", "admin"]

  # 报表功能域
  report.advanced:
    enabled: true
    strategy: user
    rules:
      user_types: ["admin", "premium"]
    permissions:
      - "report.advanced.view"
      - "report.advanced.export"
    audit_logs:
      - "advanced_report_generated"
      - "report_exported"
    apis:
      - "GET /api/v1/reports/advanced"
      - "POST /api/v1/reports/export"
    routes:
      - "/reports/advanced"
    configs:
      - "report.export_formats"
    description: "高级报表功能集合"
    tags: ["report", "premium"]
```

#### OEM定制配置 (`configs/features-oem.yaml`)：

```yaml
features:
  # 禁用高级报表功能
  report.advanced:
    enabled: false
  
  # 限制系统配置功能
  system.config:
    enabled: true
    apis:
      - "GET /api/v1/system/config"  # 只保留查看，移除编辑
    routes:
      - "/admin/system/config"
```

#### 外部控制配置 (`configs/features-external.yaml`)：

```yaml
features:
  # 临时关闭用户管理
  user.management:
    enabled: false
  
  # 临时开启高级报表（覆盖OEM设置）
  report.advanced:
    enabled: true
```

### 3. 代码集成

#### 分层功能检查

```go
import "kratos-boilerplate/internal/pkg/feature"

func (s *YourService) SomeMethod(ctx context.Context, req *YourRequest) (*YourResponse, error) {
    // 检查功能域是否有任何启用的功能
    if !s.featureToggle.IsDomainEnabled("user") {
        return nil, errors.Forbidden("DOMAIN_DISABLED", "用户相关功能已禁用")
    }
    
    // 检查具体的分层feature
    if !s.featureToggle.IsFeatureEnabled("user.management") {
        return nil, errors.Forbidden("FEATURE_DISABLED", "用户管理功能已禁用")
    }
    
    // 检查具体能力
    evalCtx := &feature.EvaluationContext{
        UserID:   getUserID(ctx),
        UserType: getUserType(ctx),
        Environment: "production",
    }
    
    // 检查API权限
    if !s.featureToggle.HasAPI("user.management", "POST /api/v1/users") {
        return nil, errors.Forbidden("API_DISABLED", "用户创建API已禁用")
    }
    
    // 检查权限点
    if !s.featureToggle.HasPermission("user.management", "user.create") {
        return nil, errors.Forbidden("PERMISSION_DENIED", "无创建用户权限")
    }
    
    // 检查配置点
    if !s.featureToggle.HasConfig("user.management", "user.max_count") {
        return nil, errors.Forbidden("CONFIG_DISABLED", "用户数量限制配置已禁用")
    }
    
    return s.handleUserManagement(ctx, req)
}
```

#### 功能集合查询

```go
// 获取用户域下的all功能
userFeatures := featureManager.GetDomainFeatures("user")
// 返回: ["user.management", "user.profile", "user.security"]

// 检查多个相关feature
if !featureManager.IsAnyFeatureEnabled("user.management", "user.profile") {
    return errors.Forbidden("NO_USER_FEATURES", "用户功能不可用")
}

// 获取功能的完整能力
capabilities := featureManager.GetFeatureCapabilities("user.management")
if capabilities != nil {
    log.Printf("用户管理功能包含 %d 个权限点", len(capabilities.Permissions))
    log.Printf("用户管理功能包含 %d 个API", len(capabilities.APIs))
}
```

#### 使用装饰器

```go
// 功能集合检查装饰器
decorator := feature.NewServiceDecorator(featureToggle, logger)

wrappedMethod := decorator.WithFeatureCheck("user.management")()
    func(ctx context.Context, req interface{}) (interface{}, error) {
        // 您的业务逻辑
        return yourBusinessLogic(ctx, req)
    },
)

// 条件装饰器 - 基于功能集合选择不同实现
conditionalDecorator := feature.NewConditionalDecorator(featureToggle, logger)

wrappedMethod := conditionalDecorator.WithAlternative(
    "report.advanced",
    advancedReportImpl,  // 高级报表实现
    basicReportImpl,     // 基础报表实现
)

// 渐进式发布装饰器
rolloutDecorator := feature.NewGradualRolloutDecorator(featureToggle, logger)

wrappedMethod := rolloutDecorator.WithRollout(
    "system.config",
    newConfigImpl,  // 新版本实现
    oldConfigImpl,  // 旧版本实现
)
```

#### 使用中间件

```go
import "kratos-boilerplate/internal/pkg/feature"

// 在HTTP服务器中使用
srv := http.NewServer(
    http.Middleware(
        recovery.Recovery(),
        feature.Middleware(featureToggle, logger),
    ),
)
```

### 4. API管理

系统提供了完整的REST API来管理分层功能集合：

```bash
# 获取所有功能集合
curl http://localhost:8000/api/v1/features

# 获取用户域下的所有功能
curl http://localhost:8000/api/v1/features/domain/user

# 获取特定功能集合
curl http://localhost:8000/api/v1/features/user.management

# 获取功能集合的能力详情
curl http://localhost:8000/api/v1/features/user.management/capabilities

# 启用功能集合
curl -X POST http://localhost:8000/api/v1/features/user.management/enable

# 禁用功能集合
curl -X POST http://localhost:8000/api/v1/features/user.management/disable

# 检查具体能力
curl http://localhost:8000/api/v1/features/user.management/permissions/user.create
curl http://localhost:8000/api/v1/features/user.management/apis/POST%20%2Fapi%2Fv1%2Fusers
curl http://localhost:8000/api/v1/features/user.management/routes/%2Fadmin%2Fusers

# 更新功能集合配置
curl -X PUT http://localhost:8000/api/v1/features/user.management \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "enabled": true,
      "strategy": "USER",
      "rules": {"user_types": ["admin"]},
      "description": "Updated user management config"
    }
  }'

# 评估功能集合
curl -X POST http://localhost:8000/api/v1/features/user.management/evaluate \
  -H "Content-Type: application/json" \
  -d '{
    "context": {
      "user_id": "user123",
      "user_type": "admin",
      "environment": "production"
    }
  }'

# 获取功能树结构
curl http://localhost:8000/api/v1/features/tree

# 导出CSV
curl http://localhost:8000/api/v1/features/export/csv

# 获取统计信息
curl http://localhost:8000/api/v1/features/stats
```

## 功能集合策略详解

### 1. 功能集合结构
每个功能集合包含以下组件：

```yaml
feature_name:
  enabled: true              # 总开关
  strategy: simple           # 评估策略
  rules: {}                  # 策略规则
  
  # 功能集合组件
  permissions:               # 权限点列表
    - "permission.name"
  audit_logs:               # 审计日志类型
    - "log_type"
  apis:                     # API接口列表
    - "METHOD /path"
  routes:                   # 前端路由列表
    - "/route/path"
  configs:                  # 配置点列表
    - "config.key"
  
  # 元数据
  description: "功能描述"
  tags: ["tag1", "tag2"]
```

### 2. 多级配置合并

配置优先级：external > oem > base

```yaml
# 最终配置 = base + oem + external
# 如果 external 中 enabled=false，则整个功能集合禁用
# 如果 oem 中只定义了部分apis，则只保留这些apis
```

### 3. 检查逻辑

```go
// 检查顺序
1. 检查功能集合是否启用 (IsFeatureEnabled)
2. 检查具体能力是否在集合中 (HasPermission/HasAPI/etc)
3. 执行业务逻辑
```

## 最佳实践

### 1. 功能集合命名
- 使用A.B.C分层结构，如 `user.management`、`system.config`
- 采用统一的命名约定：`domain.module`
- 示例：`user.profile`、`system.monitoring`、`report.advanced`

### 2. 功能集合组织
- **用户域 (user.*)**: 用户管理、资料、安全相关功能
- **系统域 (system.*)**: 系统配置、监控、维护相关功能
- **报表域 (report.*)**: 各类报表和统计功能
- **业务域**: 根据具体业务需要定义

### 3. 多级配置管理
- **开发环境**: base配置，启用所有调试功能
- **测试环境**: base + minimal oem，接近生产环境
- **生产环境**: base + oem + external，保守的配置

### 4. 功能集合设计原则
- 每个功能集合应该是一个完整的业务功能
- 包含该功能所需的所有能力组件
- 避免功能集合之间的强耦合
- 支持独立启用/禁用

### 5. CSV导出使用
```bash
# 导出所有功能集合
curl http://localhost:8000/api/v1/features/export/csv > features.csv

# 按域过滤导出
curl "http://localhost:8000/api/v1/features/export/csv?domain=user" > user_features.csv

# 按状态过滤导出
curl "http://localhost:8000/api/v1/features/export/csv?enabled=true" > enabled_features.csv
```

### 6. 清理策略
- 定期清理不再需要的功能集合
- 在功能稳定后移除相关的功能开关代码
- 保持配置文件的整洁
- 记录功能集合的变更历史

## 扩展开发

### 1. 自定义功能集合策略
实现 `StrategyEvaluator` 接口来创建自定义策略：

```go
type CustomStrategy struct{}

func (c *CustomStrategy) Evaluate(ctx context.Context, config *ToggleConfig, evalCtx *EvaluationContext) bool {
    // 实现自定义评估逻辑
    // 可以访问 config.Capabilities 来检查具体能力
    return true
}

func (c *CustomStrategy) SupportsStrategy(strategy FeatureStrategy) bool {
    return strategy == "custom"
}

// 注册自定义策略
compositeStrategy.RegisterEvaluator("custom", &CustomStrategy{})
```

### 2. 自定义存储
实现 `FeatureRepository` 接口来支持不同的存储后端：

```go
type DatabaseRepository struct {
    db *sql.DB
}

func (r *DatabaseRepository) LoadToggles(ctx context.Context) (map[FeatureFlag]*ToggleConfig, error) {
    // 从数据库加载功能集合配置
    // 包括 capabilities 信息
}

func (r *DatabaseRepository) SaveToggle(ctx context.Context, flag FeatureFlag, config *ToggleConfig) error {
    // 保存功能集合到数据库
    // 包括所有 permissions、apis、routes 等
}

// 其他方法实现...
```

### 3. 功能集合事件处理
订阅功能集合变更事件：

```go
err := featureToggle.Subscribe(func(flag FeatureFlag, oldConfig, newConfig *ToggleConfig) {
    log.Printf("Feature set %s changed", flag)
    
    // 检查功能集合能力变化
    if newConfig != nil && newConfig.Capabilities != nil {
        log.Printf("New capabilities: %+v", newConfig.Capabilities)
    }
    
    // 可以在这里实现：
    // - 清理相关缓存
    // - 通知前端更新
    // - 记录审计日志
})
```

### 4. 自定义功能集合检查器

```go
type CustomCapabilityChecker struct {
    featureManager HierarchicalFeatureManager
}

func (c *CustomCapabilityChecker) CheckBusinessRule(ctx context.Context, featurePath string, rule string) bool {
    // 检查功能集合是否启用
    if !c.featureManager.IsFeatureEnabled(featurePath) {
        return false
    }
    
    // 检查具体业务规则
    capabilities := c.featureManager.GetFeatureCapabilities(featurePath)
    if capabilities == nil {
        return false
    }
    
    // 自定义检查逻辑
    return c.validateBusinessRule(capabilities, rule)
}
```

## 故障排查

### 1. 常见问题

**功能集合配置不生效**
- 检查配置文件路径是否正确
- 确认YAML语法是否正确
- 检查功能集合结构是否完整
- 查看日志中的错误信息

**功能集合返回意外结果**
- 检查多级配置合并是否正确
- 确认评估上下文是否正确传递
- 检查功能集合中具体能力配置
- 使用API测试功能评估结果

**性能问题**
- 检查是否有过多的文件监听
- 考虑使用缓存策略
- 优化功能集合评估逻辑
- 检查功能集合的复杂度

### 2. 调试工具

使用API获取调试信息：

```bash
# 获取功能集合统计
curl http://localhost:8000/api/v1/features/stats

# 获取功能树结构
curl http://localhost:8000/api/v1/features/tree

# 测试功能集合评估
curl -X POST http://localhost:8000/api/v1/features/user.management/evaluate \
  -d '{"context": {"user_id": "test_user", "user_type": "admin"}}'

# 检查具体能力
curl http://localhost:8000/api/v1/features/user.management/capabilities

# 检查配置合并结果
curl http://localhost:8000/api/v1/features/config/final
```

### 3. 日志分析

功能集合相关的日志关键词：
- `feature set loaded`: 功能集合加载
- `capability check`: 能力检查
- `config merged`: 配置合并
- `evaluation result`: 评估结果

## 安全考虑

### 1. 访问控制
- 功能开关管理API应该有适当的认证和授权
- 敏感功能的开关变更应该有审计日志
- 考虑实现变更审批流程

### 2. 配置保护
- 功能开关配置文件应该有适当的文件权限
- 考虑加密敏感的配置信息
- 实施配置变更的版本控制

### 3. 降级策略
- 当功能开关系统不可用时，应该有合理的默认行为
- 实施熔断器模式防止级联故障
- 确保核心功能不依赖于功能开关系统

## 总结

本功能集合系统为 kratos-boilerplate 项目提供了强大而灵活的功能裁剪能力。通过分层的功能集合设计和多级配置策略，开发团队可以：

- **精细化功能控制**: 每个功能集合包含权限、日志、API、UI、配置的完整组合
- **分层管理**: A.B.C结构实现功能域和模块的层次化管理
- **多级配置**: base + oem + external 多层配置支持不同部署场景
- **运行时控制**: 支持动态配置和热更新
- **便捷验证**: CSV导出功能便于产品和测试团队验证
- **安全可靠**: 完整的访问控制和降级策略

建议在使用过程中结合具体的业务场景，逐步完善和优化功能集合的配置和策略。通过合理使用分层功能集合，可以实现精细化的功能控制和高效的产品迭代。