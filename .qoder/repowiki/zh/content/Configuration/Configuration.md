# 配置

<cite>
**本文档引用的文件**   
- [config.yaml](file://configs/config.yaml)
- [conf.proto](file://internal/conf/conf.proto)
- [main.go](file://cmd/kratos-boilerplate/main.go)
- [audit_logger.yaml](file://configs/plugins/audit_logger.yaml)
- [auth_enhancer.yaml](file://configs/plugins/auth_enhancer.yaml)
- [wire.go](file://internal/pkg/feature/wire.go)
- [config.go](file://internal/pkg/config/config.go) - *在最近提交中更新*
- [validator.go](file://internal/pkg/config/validator.go) - *在最近提交中新增*
</cite>

## 更新摘要
**已做更改**   
- 根据最新代码变更全面更新了“主配置文件结构”部分
- 新增了关于配置热更新和多源加载机制的内容
- 增强了配置验证机制的详细说明
- 更新了配置加载过程的序列图以反映实际实现
- 所有引用文件路径均已更新并标注最新提交状态

## 目录
1. [简介](#简介)
2. [主配置文件结构](#主配置文件结构)
3. [配置加载与验证](#配置加载与验证)
4. [插件特定配置](#插件特定配置)
5. [环境变量覆盖](#环境变量覆盖)
6. [不同环境的配置](#不同环境的配置)
7. [敏感值的安全最佳实践](#敏感值的安全最佳实践)
8. [跨环境的配置管理](#跨环境的配置管理)
9. [配置变更的验证](#配置变更的验证)
10. [常见配置场景](#常见配置场景)
11. [配置问题排查](#配置问题排查)

## 简介
kratos-boilerplate 项目实现了一个基于 Protobuf 定义和 Kratos 框架的模块化、可扩展的配置系统。本文档全面概述了该配置架构，详细说明了配置如何在不同环境中进行结构化、加载、验证和应用。该系统支持分层配置合并、插件特定设置、基于环境的覆盖以及动态功能开关，从而实现灵活部署于开发、测试和生产环境。

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L47)
- [conf.proto](file://internal/conf/conf.proto#L1-L70)

## 主配置文件结构
主配置文件 `config.yaml` 位于 `configs/` 目录中，使用直接映射到 `conf.proto` 中 Protobuf 消息定义的分层结构定义核心应用程序设置。配置按逻辑部分组织，对应于不同的系统组件。

### 服务器配置
`server` 部分配置 HTTP 和 gRPC 服务的网络端点。

**Labels:**
- `server.http.addr`: 指定 HTTP 服务器的绑定地址和端口
- `server.http.timeout`: 设置请求超时时间
- `server.grpc.addr`: 指定 gRPC 服务器的绑定地址和端口
- `server.grpc.timeout`: 设置请求超时时间

```yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 1s
```

### 数据配置
`data` 部分配置数据库和 Redis 连接。

**Labels:**
- `data.database.driver`: 指定数据库驱动（例如 postgres）
- `data.database.source`: 数据库连接字符串
- `data.redis.addr`: Redis 服务器地址
- `data.redis.read_timeout`: Redis 操作的读取超时
- `data.redis.write_timeout`: Redis 操作的写入超时

```yaml
data:
  database:
    driver: postgres
    source: postgresql://postgres:postgres@cross-redline-db:5432/test?sslmode=disable
  redis:
    addr: 127.0.0.1:6379
    read_timeout: 0.2s
    write_timeout: 0.2s
```

### 认证配置
`auth` 部分配置认证相关参数。

**Labels:**
- `auth.jwt_secret_key`: JWT 密钥，必须至少16个字符
- `auth.access_token_expiration`: 访问令牌过期时间
- `auth.refresh_token_expiration`: 刷新令牌过期时间
- `auth.captcha_enabled`: 是否启用验证码
- `auth.captcha_expiration`: 验证码过期时间
- `auth.max_login_attempts`: 最大登录尝试次数
- `auth.lock_duration`: 账户锁定持续时间
- `auth.totp_enabled`: 是否启用 TOTP 双因素认证

```yaml
auth:
  jwt_secret_key: "your-super-secret-jwt-key-here"
  access_token_expiration: "24h"
  refresh_token_expiration: "7d"
  captcha_enabled: true
  captcha_expiration: "5m"
  max_login_attempts: 5
  lock_duration: "30m"
  totp_enabled: false
```

### 日志配置
`log` 部分配置日志记录行为。

**Labels:**
- `log.level`: 日志级别（debug, info, warn, error, fatal）
- `log.format`: 输出格式（json, text）
- `log.output`: 输出目标（stdout, file）
- `log.file.path`: 日志文件路径
- `log.file.max_size`: 单个日志文件最大大小（MB）
- `log.file.max_backups`: 保留的旧日志文件最大数量
- `log.file.max_age`: 保留旧日志文件的最大天数
- `log.file.compress`: 是否压缩归档日志

```yaml
log:
  level: info
  format: json
  output: stdout
  file:
    path: /var/log/app.log
    max_size: 100
    max_backups: 3
    max_age: 28
    compress: true
```

### 链路追踪配置
`tracing` 部分配置分布式追踪。

**Labels:**
- `tracing.enabled`: 是否启用链路追踪
- `tracing.jaeger.endpoint`: Jaeger 收集器端点
- `tracing.jaeger.sample_rate`: 采样率（0.0-1.0）

```yaml
tracing:
  enabled: true
  jaeger:
    endpoint: "http://jaeger-collector:14268/api/traces"
    sample_rate: 0.1
```

### 插件系统配置
`plugins` 部分控制插件加载机制。

**Labels:**
- `plugins.enabled`: 启用或禁用插件系统
- `plugins.directory`: 存放插件二进制文件的文件系统路径
- `plugins.config_directory`: 包含插件配置文件的目录
- `plugins.auto_load`: 在启动时自动加载插件
- `plugins.security.sandbox_enabled`: 为插件启用安全沙箱
- `plugins.security.max_memory`: 插件的最大内存分配
- `plugins.security.max_cpu_percent`: 插件的 CPU 使用率限制

```yaml
plugins:
  enabled: true
  directory: "./plugins"
  config_directory: "./configs/plugins"
  auto_load: true
  security:
    sandbox_enabled: true
    max_memory: "100MB"
    max_cpu_percent: 10
```

### 功能开关配置
`features` 部分管理功能标志和动态配置。

**Labels:**
- `features.enabled`: 启用功能开关系统
- `features.config_file`: 功能配置文件的路径
- `features.config_format`: 功能配置的格式（yaml/json）
- `features.watch_config`: 启用功能配置的实时重载
- `features.default_environment`: 默认部署环境
- `features.repository.type`: 功能配置的存储后端
- `features.repository.config_path`: 存储库配置路径
- `features.repository.format`: 存储库配置的格式

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

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L47)
- [config.go](file://internal/pkg/config/config.go#L32-L87)

## 配置加载与验证
配置系统遵循结构化的初始化过程，通过 Protobuf 消息定义确保类型安全和验证。

### 配置加载过程
应用程序通过增强的配置管理器加载配置，该管理器支持多种源、环境特定配置和热更新。

```mermaid
sequenceDiagram
participant Main as main.go
participant ConfigManager as NewManager()
participant FileSource as file.NewSource()
participant EnvSource as env.NewSource()
participant Watcher as fsnotify.Watcher
Main->>ConfigManager : 创建配置管理器
ConfigManager->>FileSource : 添加主配置文件源
ConfigManager->>FileSource : 添加环境特定配置文件源(如果存在)
ConfigManager->>EnvSource : 添加环境变量源(KRATOS_前缀)
ConfigManager->>Watcher : 启动文件监听器
Main->>ConfigManager : Load()
FileSource->>ConfigManager : 加载YAML文件
EnvSource->>ConfigManager : 覆盖环境变量值
ConfigManager->>Main : 返回配置实例
```

**Diagram sources**
- [main.go](file://cmd/kratos-boilerplate/main.go#L53-L90)
- [config.go](file://internal/pkg/config/config.go#L112-L146)

### 通过 Protobuf 和结构体标签进行验证
配置验证通过 `conf.proto` 中的 Protobuf 模式和 Go 结构体标签双重强制执行，确保数据完整性和安全性。

```mermaid
classDiagram
class Bootstrap {
+Server server
+Data data
+Auth auth
+Log log
+Tracing tracing
+Features features
}
class Server {
+HTTP http
+GRPC grpc
}
class Data {
+Database database
+Redis redis
}
class Auth {
+string jwt_secret_key
+time.Duration access_token_expiration
+int max_login_attempts
}
class Log {
+string level
+string format
+string output
+LogFile file
}
class Tracing {
+bool enabled
+Jaeger jaeger
}
class Features {
+bool enabled
+string config_file
+string config_format
+bool watch_config
+string default_environment
+FeatureRepository repository
}
class FeatureRepository {
+string type
+string config_path
+string format
}
class LogFile {
+string path
+int max_size
+int max_backups
+int max_age
+bool compress
}
class Jaeger {
+string endpoint
+float64 sample_rate
}
Bootstrap --> Server
Bootstrap --> Data
Bootstrap --> Auth
Bootstrap --> Log
Bootstrap --> Tracing
Bootstrap --> Features
Features --> FeatureRepository
Log --> LogFile
Tracing --> Jaeger
```

**Diagram sources**
- [conf.proto](file://internal/conf/conf.proto#L1-L70)
- [config.go](file://internal/pkg/config/config.go#L32-L87)

验证机制包括：
- **必填字段验证**: 使用 `validate:"required"` 标签确保关键字段存在
- **值范围验证**: 使用 `min`, `max` 标签限制数值范围
- **枚举验证**: 使用 `oneof` 标签限制字符串值
- **自定义验证**: 实现端口、持续时间等特殊格式验证
- **默认值设置**: 在验证后应用合理的默认值

**Section sources**
- [config.go](file://internal/pkg/config/config.go#L32-L87)
- [validator.go](file://internal/pkg/config/validator.go#L1-L321)

## 插件特定配置
插件系统支持每个插件的独立配置文件，从而实现模块化和隔离的设置管理。

### 插件配置目录结构
插件配置存储在 `configs/plugins/` 中，每个插件一个 YAML 文件：

```
configs/plugins/
├── audit_logger.yaml
└── auth_enhancer.yaml
```

### 审计日志插件配置
`audit_logger.yaml` 文件使用全面的日志设置配置审计日志插件。

**Labels:**
- `enabled`: 启用插件
- `priority`: 执行优先级（数值越低优先级越高）
- `timeout`: 最大执行时间
- `retry_count`: 重试次数
- `settings.log_level`: 日志详细级别
- `settings.output_format`: 日志输出格式
- `settings.max_log_size`: 旋转前的最大大小
- `settings.retention_days`: 保留日志的天数
- `settings.sensitive_data_masking`: 启用敏感数据掩码

```yaml
enabled: true
priority: 20
timeout: 30s
retry_count: 3
settings:
  log_level: "info"
  output_format: "json"
  max_log_size: "100MB"
  retention_days: 30
  sensitive_data_masking: true
metadata:
  category: "logging"
  author: "system"
  version: "1.0.0"
```

### 认证增强插件配置
`auth_enhancer.yaml` 文件配置与认证相关的功能。

**Labels:**
- `settings.jwt_validation`: 启用 JWT 令牌验证
- `settings.rate_limiting`: 启用速率限制
- `settings.max_requests_per_minute`: 速率限制阈值
- `settings.token_expiry`: 令牌过期时间（秒）
- `settings.audit_logging`: 启用认证事件的审计日志

```yaml
enabled: true
priority: 10
timeout: 30s
retry_count: 3
settings:
  jwt_validation: true
  rate_limiting: true
  max_requests_per_minute: 100
  token_expiry: 3600
  audit_logging: true
metadata:
  category: "authentication"
  author: "system"
  version: "1.0.0"
```

插件配置系统允许：
- 独立配置插件行为
- 基于优先级的执行顺序
- 每个插件的超时和重试策略
- 模块化功能的启用/禁用
- 插件管理的元数据跟踪

**Section sources**
- [audit_logger.yaml](file://configs/plugins/audit_logger.yaml#L1-L14)
- [auth_enhancer.yaml](file://configs/plugins/auth_enhancer.yaml#L1-L14)

## 环境变量覆盖
配置系统支持环境变量覆盖，允许在不修改配置文件的情况下进行运行时自定义。

### 覆盖机制
可以使用点分隔路径表示法转换为下划线的环境变量来覆盖任何配置值，并带有 `KRATOS_` 前缀：

- `server.http.addr` → `KRATOS_SERVER_HTTP_ADDR`
- `data.database.source` → `KRATOS_DATA_DATABASE_SOURCE`
- `features.enabled` → `KRATOS_FEATURES_ENABLED`

例如：
```bash
export KRATOS_SERVER_HTTP_ADDR="0.0.0.0:8080"
export KRATOS_DATA_DATABASE_SOURCE="postgresql://user:pass@prod-db:5432/prod"
```

### 命令行配置路径
可以通过命令行标志指定配置文件路径：

```bash
./kratos-boilerplate -conf ./configs/config.prod.yaml
```

默认值在 `main.go` 中设置：
```go
flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
```

这使得：
- 选择特定于环境的配置文件
- 使用替代配置源进行测试
- 开发覆盖而无需提交更改

**Section sources**
- [main.go](file://cmd/kratos-boilerplate/main.go#L48-L50)
- [config.go](file://internal/pkg/config/config.go#L112-L146)

## 不同环境的配置
该系统通过专用配置文件和功能开关管理支持多种部署环境。

### 特定于环境的配置文件
仓库包含特定于环境的功能配置文件：

```
configs/
├── features.dev.yaml
├── features.prod.yaml
└── features.yaml
```

配置管理器会自动查找并加载环境特定配置文件（如 `config.dev.yaml`），如果存在则将其与主配置合并。

### 功能开关环境策略
功能系统通过 `EnvironmentStrategy` 支持基于环境的激活：

```go
func (s *EnvironmentStrategy) Evaluate(ctx context.Context, config *ToggleConfig, evalCtx *EvaluationContext) bool {
    if environments, ok := config.Rules["environments"].([]interface{}); ok {
        for _, env := range environments {
            if env.(string) == evalCtx.Environment {
                return true
            }
        }
    }
    return false
}
```

带环境规则的功能配置示例：
```yaml
user_profile_enhancement:
  enabled: true
  rules:
    environments:
      - "development"
      - "staging"
    version_range:
      min: "1.0.0"
      max: "2.0.0"
```

### 默认环境配置
系统在主配置中指定默认环境：

```yaml
features:
  default_environment: "production"
```

当未提供显式环境时，功能系统使用此值，确保在所有部署场景中行为可预测。

**Section sources**
- [wire.go](file://internal/pkg/feature/wire.go#L82-L119)
- [strategies_test.go](file://internal/pkg/feature/strategies_test.go#L230-L279)
- [config.go](file://internal/pkg/config/config.go#L112-L146)

## 敏感值的安全最佳实践
配置系统实施了多项安全措施来保护敏感信息。

### 敏感数据保护
系统包含处理敏感数据的内置机制：

**Labels:**
- `auth.jwt_secret_key`: 应通过安全方式提供
- 包含凭据的数据库连接字符串
- API 密钥和身份验证令牌

### 推荐的安全实践
1. **环境变量用于密钥**：将敏感值存储在环境变量中，而不是配置文件中
2. **文件权限**：限制配置文件权限（600）以防止未授权访问
3. **Git 忽略**：将特定于环境的配置文件添加到 `.gitignore`
4. **密钥管理**：集成密钥管理系统（Vault、AWS Secrets Manager）
5. **敏感数据掩码**：在审计日志中启用掩码

安全部署模式示例：
```bash
export KRATOS_DATA_DATABASE_SOURCE="postgresql://app-user:$(cat /secrets/db-password)@db:5432/app"
export KRATOS_AUTH_JWT_SECRET_KEY="$(cat /secrets/jwt-key)"
./kratos-boilerplate -conf ./configs/config.yaml
```

系统的插件安全沙箱进一步增强了保护：
```yaml
plugins:
  security:
    sandbox_enabled: true
    max_memory: "100MB"
    max_cpu_percent: 10
```

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L47)
- [conf.proto](file://internal/conf/conf.proto#L1-L70)

## 跨环境的配置管理
有效的跨环境配置管理需要结构化的方法来避免配置漂移。

### 配置层次结构
系统遵循分层配置模式：

1. **基础配置**: `config.yaml` - 所有环境的通用设置
2. **环境覆盖**: `config.{env}.yaml` - 环境特定的覆盖
3. **功能配置**: `features.yaml` - 功能开关状态
4. **环境变量**: 运行时覆盖，最高优先级
5. **插件配置**: 插件特定设置

### 环境提升策略
推荐的配置提升工作流程：

```mermaid
flowchart LR
A["开发环境"] --> |测试和验证| B["预生产环境"]
B --> |审批和测试| C["生产环境"]
style A fill:#f9f,stroke:#333
style B fill:#ff9,stroke:#333
style C fill:#f99,stroke:#333
subgraph "配置管理"
D["config.yaml - 基础设置"]
E["config.dev.yaml - 开发覆盖"]
F["config.staging.yaml - 预生产覆盖"]
G["config.prod.yaml - 生产覆盖"]
end
```

**Diagram sources**
- [config.yaml](file://configs/config.yaml#L1-L47)
- [config.dev.yaml](file://configs/config.dev.yaml)
- [config.prod.yaml](file://configs/config.prod.yaml)

关键实践：
- 维护一致的基础配置
- 使用功能开关控制功能
- 首先在较低环境中测试配置更改
- 自动化配置部署
- 实施配置版本控制

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L47)
- [config.go](file://internal/pkg/config/config.go#L112-L146)

## 配置变更的验证
正确验证配置变更可以防止运行时错误和服务中断。

### 静态验证
基于 Protobuf 和结构体标签的配置提供编译时和加载时验证：

```go
var bc conf.Bootstrap
if err := c.Scan(&bc); err != nil {
    panic(err)
}
```

这确保了：
- 正确的数据类型
- 必填字段存在
- 持续时间字符串格式正确
- 无未知字段

### 动态验证
配置管理器执行运行时验证和默认值设置：

```go
func (m *Manager) Validate() error {
    var cfg Config
    if err := m.config.Scan(&cfg); err != nil {
        return fmt.Errorf("failed to scan config: %w", err)
    }
    
    return m.validator.Validate(&cfg)
}
```

### 配置测试
推荐的验证工作流程：
1. 验证 YAML 语法
2. 使用 `c.Load()` 和 `c.Scan()` 测试加载
3. 验证插件配置兼容性
4. 测试功能开关评估
5. 执行集成测试

**Section sources**
- [main.go](file://cmd/kratos-boilerplate/main.go#L68-L71)
- [wire.go](file://internal/pkg/feature/wire.go#L42-L80)
- [validator.go](file://internal/pkg/config/validator.go#L1-L321)

## 常见配置场景

### 开发环境设置
```yaml
server:
  http:
    addr: 0.0.0.0:8000
  grpc:
    addr: 0.0.0.0:9000
data:
  database:
    source: postgresql://postgres:postgres@localhost:5432/dev
features:
  default_environment: "development"
  watch_config: true
plugins:
  security:
    sandbox_enabled: false  # 更容易调试
```

### 生产环境设置
```yaml
server:
  http:
    addr: 0.0.0.0:80
  grpc:
    addr: 0.0.0.0:443
data:
  database:
    source: ${DATA_SOURCE}  # 来自环境变量
features:
  default_environment: "production"
  watch_config: false  # 防止运行时更改
plugins:
  security:
    sandbox_enabled: true
    max_memory: "100MB"
```

### 用于 A/B 测试的功能开关
```yaml
# features.yaml
new_search_algorithm:
  enabled: true
  rules:
    percentage:
      value: 10
      identifier: "user_id"
```

### 用于监控的插件配置
```yaml
# configs/plugins/audit_logger.yaml
enabled: true
settings:
  log_level: "warn"
  output_format: "json"
  sensitive_data_masking: true
```

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L47)
- [features.yaml](file://configs/features.yaml)

## 配置问题排查

### 常见问题及解决方案

**问题：找不到配置文件**
- **症状**: `panic: config file not found`
- **解决方案**: 验证提供给 `-conf` 标志的路径是否存在
- **检查**: `ls -la configs/` 以确认文件存在

**问题：无效的 YAML 语法**
- **症状**: `panic: yaml: line X: could not find expected ':'`
- **解决方案**: 使用在线解析器或 `yamllint` 验证 YAML
- **检查**: 确保正确的缩进和引号

**问题：缺少必填字段**
- **症状**: `configuration validation failed: field 'jwt_secret_key' is required`
- **解决方案**: 与 `config.go` 结构进行比较
- **检查**: 必填字段如 `server.http.addr`, `auth.jwt_secret_key`

**问题：环境变量未应用**
- **症状**: 配置使用默认值而不是环境覆盖
- **解决方案**: 验证环境变量命名约定（需 `KRATOS_` 前缀）
- **检查**: `printenv | grep KRATOS` 以确认变量已设置

### 诊断命令
```bash
# 检查当前环境变量
printenv | grep KRATOS

# 测试配置加载
./kratos-boilerplate -conf ./configs/config.yaml

# 验证文件权限
ls -la configs/

# 检查语法错误
yamllint configs/config.yaml
```

### 调试技巧
1. 启用详细日志以查看配置加载过程
2. 使用 `c.Load()` 返回值检测加载问题
3. 验证配置结构是否符合 `config.go` 定义
4. 单独测试环境变量覆盖
5. 检查文件路径是否相对于工作目录

**Section sources**
- [main.go](file://cmd/kratos-boilerplate/main.go#L65-L68)
- [config.yaml](file://configs/config.yaml#L1-L47)
- [config.go](file://internal/pkg/config/config.go#L112-L146)