# 主配置

<cite>
**本文档引用的文件**   
- [conf.proto](file://internal/conf/conf.proto#L1-L69)
- [config.yaml](file://configs/config.yaml#L1-L47)
- [main.go](file://cmd/kratos-boilerplate/main.go#L53-L90)
- [features.yaml](file://configs/features.yaml#L1-L340)
- [hierarchical_features_demo.go](file://examples/hierarchical_features_demo.go#L13-L97)
</cite>

## 更新摘要
**已做更改**   
- 更新了“功能开关和动态配置”章节，以反映分层功能开关系统的完整结构和行为
- 添加了新的“分层功能开关系统”章节，详细说明功能域、功能集合及其能力
- 更新了“配置结构概述”和“Bootstrap消息与配置层次结构”以包含`features`配置项
- 修正了“认证与安全设置”章节中缺失的`auth`配置示例
- 增强了“常见配置问题与故障排除”中的调试建议

## 目录
1. [简介](#简介)
2. [配置结构概述](#配置结构概述)
3. [Bootstrap消息与配置层次结构](#bootstrap消息与配置层次结构)
4. [服务器配置](#服务器配置)
5. [数据库与Redis配置](#数据库与redis配置)
6. [认证与安全设置](#认证与安全设置)
7. [功能开关和动态配置](#功能开关和动态配置)
8. [分层功能开关系统](#分层功能开关系统)
9. [配置加载与验证过程](#配置加载与验证过程)
10. [扩展配置模式](#扩展配置模式)
11. [常见配置问题与故障排除](#常见配置问题与故障排除)

## 简介
kratos-boilerplate中的主配置系统基于protobuf定义的模式（`conf.proto`）构建，确保类型安全、验证和配置数据的结构化解析。主配置文件`config.yaml`在应用程序启动时被解析为该模式，从而在不同环境中实现健壮且可预测的行为。本文档详细说明了配置系统的结构、使用方法和扩展方式，重点关注服务器设置、数据库连接、认证参数和功能开关管理。

**章节来源**
- [conf.proto](file://internal/conf/conf.proto#L1-L69)
- [config.yaml](file://configs/config.yaml#L1-L47)

## 配置结构概述
配置系统使用Protocol Buffers（protobuf）为所有应用程序设置定义严格的模式。根消息是`Bootstrap`，其中包含不同子系统的嵌套消息：`Server`、`Data`、`Auth`和`Features`。这些消息在启动时通过Kratos的配置加载器从YAML文件映射而来。

`config.yaml`文件使用与protobuf字段名称直接对应的YAML键来镜像此结构。这种设计在保持人类可读的配置文件的同时，实现了强类型化。

```yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 1s
data:
  database:
    driver: postgres
    source: postgresql://postgres:postgres@cross-redline-db:5432/test?sslmode=disable
  redis:
    addr: 127.0.0.1:6379
    read_timeout: 0.2s
    write_timeout: 0.2s
auth:
  jwt_secret_key: "your-super-secret-jwt-key-here"
  access_token_expiration: "24h"
  refresh_token_expiration: "7d"
  captcha_enabled: true
  captcha_expiration: "5m"
  max_login_attempts: 5
  lock_duration: "30m"
  totp_enabled: false
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

**章节来源**
- [conf.proto](file://internal/conf/conf.proto#L1-L69)
- [config.yaml](file://configs/config.yaml#L1-L47)

## Bootstrap消息与配置层次结构
`Bootstrap`消息作为所有配置数据的顶级容器。它定义了配置加载器的入口点，并将设置组织成逻辑组。

```protobuf
message Bootstrap {
  Server server = 1;
  Data data = 2;
  Auth auth = 3;
  Features features = 4;
}
```

每个字段对应YAML文件中的一个部分。这种层次结构确保了模块化和关注点分离，允许开发人员在不影响其他部分的情况下修改特定子系统。

**图表来源**
- [conf.proto](file://internal/conf/conf.proto#L1-L69)

```mermaid
classDiagram
class Bootstrap {
+Server server
+Data data
+Auth auth
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
+Duration access_token_expiration
+Duration refresh_token_expiration
+bool captcha_enabled
+Duration captcha_expiration
+int32 max_login_attempts
+Duration lock_duration
+bool totp_enabled
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
Bootstrap --> Server
Bootstrap --> Data
Bootstrap --> Auth
Bootstrap --> Features
Features --> FeatureRepository
```

**章节来源**
- [conf.proto](file://internal/conf/conf.proto#L1-L69)

## 服务器配置
`Server`消息配置HTTP和gRPC端点。它包括两种协议的网络绑定、地址和超时设置。

```protobuf
message Server {
  message HTTP {
    string network = 1;
    string addr = 2;
    google.protobuf.Duration timeout = 3;
  }
  message GRPC {
    string network = 1;
    string addr = 2;
    google.protobuf.Duration timeout = 3;
  }
  HTTP http = 1;
  GRPC grpc = 2;
}
```

在`config.yaml`中，这转换为：

```yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 1s
```

- **addr**: 绑定服务器的IP和端口（例如，`0.0.0.0:8000`）
- **timeout**: Go持续时间格式的请求超时（例如，`1s`，`300ms`）
- **network**: 可选的网络类型（默认为`"tcp"`）

这些设置控制应用程序如何暴露其API以及处理传入请求。

**章节来源**
- [conf.proto](file://internal/conf/conf.proto#L1-L69)
- [config.yaml](file://configs/config.yaml#L1-L47)

## 数据库与Redis配置
`Data`消息封装了数据库和Redis设置，实现与持久性和内存存储系统的连接。

```protobuf
message Data {
  message Database {
    string driver = 1;
    string source = 2;
  }
  message Redis {
    string network = 1;
    string addr = 2;
    google.protobuf.Duration read_timeout = 3;
    google.protobuf.Duration write_timeout = 4;
  }
  Database database = 1;
  Redis redis = 2;
}
```

示例YAML：
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

- **driver**: 支持的值包括`"postgres"`、`"mysql"`、`"sqlite"`
- **source**: 带凭证和选项的连接字符串
- **addr**: Redis服务器地址
- **read_timeout/write_timeout**: I/O操作超时

此配置由数据层用于初始化数据库和Redis客户端。

**章节来源**
- [conf.proto](file://internal/conf/conf.proto#L1-L69)
- [config.yaml](file://configs/config.yaml#L1-L47)

## 认证与安全设置
`Auth`消息定义了基于JWT的认证、验证码和账户锁定策略的参数。

```protobuf
message Auth {
  string jwt_secret_key = 1;
  google.protobuf.Duration access_token_expiration = 2;
  google.protobuf.Duration refresh_token_expiration = 3;
  bool captcha_enabled = 4;
  google.protobuf.Duration captcha_expiration = 5;
  int32 max_login_attempts = 6;
  google.protobuf.Duration lock_duration = 7;
  bool totp_enabled = 8;
}
```

尽管在当前的`config.yaml`中不存在，但这些字段将出现在`auth:`部分下。示例：

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

这些设置管理：
- 令牌生成和过期
- 通过验证码和账户锁定防止暴力破解
- 多因素认证（TOTP）

**章节来源**
- [conf.proto](file://internal/conf/conf.proto#L1-L69)
- [config.yaml](file://configs/config.yaml#L1-L47)

## 功能开关和动态配置
`Features`和`FeatureRepository`消息启用动态功能切换和外部配置源。

```protobuf
message Features {
  bool enabled = 1;
  string config_file = 2;
  string config_format = 3;
  bool watch_config = 4;
  string default_environment = 5;
  FeatureRepository repository = 6;
}

message FeatureRepository {
  string type = 1;  // "file", "redis", "database"
  string config_path = 2;
  string format = 3;  // "yaml", "json"
}
```

当前配置：
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

- **enabled**: 功能开关系统的全局切换
- **watch_config**: 启用功能配置的实时重载
- **repository.type**: 功能数据源（`file`、`redis`、`database`）
- **default_environment**: 功能解析的回退环境

此系统允许在不重启的情况下修改应用程序行为。

**章节来源**
- [conf.proto](file://internal/conf/conf.proto#L1-L69)
- [config.yaml](file://configs/config.yaml#L1-L47)

## 分层功能开关系统
新的分层功能开关系统将功能组织为功能域和功能集合，提供更精细的控制和丰富的元数据。

### 功能域与功能集合
功能域（如`user`、`system`、`report`）将相关功能分组。每个功能域包含多个功能集合，每个集合代表一组相关的功能。

```yaml
# 用户功能域
user.management:
  enabled: true
  strategy: simple
  permissions:
    - "user.create"
    - "user.update"
  apis:
    - "POST /api/v1/users"
  description: "用户管理功能集合"
  tags: ["user", "admin"]

# 系统功能域
system.config:
  enabled: true
  strategy: environment
  rules:
    environments: ["development", "staging", "production"]
  description: "系统配置功能集合"
  tags: ["system", "admin"]
```

### 功能集合能力
每个功能集合可以定义多种能力，用于细粒度的访问控制和行为管理：

- **permissions**: 定义该功能所需的权限点
- **apis**: 列出该功能涉及的API端点
- **routes**: 指定该功能关联的前端路由
- **configs**: 包含该功能相关的配置项
- **audit_logs**: 记录该功能触发的审计日志事件

### 功能策略
支持多种启用策略：
- **simple**: 简单启用/禁用
- **environment**: 基于环境启用
- **user**: 基于用户类型或属性
- **percentage**: 百分比渐进式发布
- **time**: 基于时间窗口启用

### 使用示例
```go
// 检查功能是否启用
manager.IsFeatureEnabled("user.management")

// 检查权限
manager.HasPermission("user.management", "user.create")

// 检查API访问
manager.HasAPI("user.management", "POST /api/v1/users")

// 获取功能树
tree := manager.GetFeatureTree()
```

**章节来源**
- [features.yaml](file://configs/features.yaml#L1-L340)
- [hierarchical_features_demo.go](file://examples/hierarchical_features_demo.go#L13-L97)

## 配置加载与验证过程
配置在`main.go`中的应用程序启动期间加载和验证。该过程遵循以下步骤：

1. 使用文件源初始化配置加载器
2. 加载YAML内容
3. 根据protobuf模式解析和验证
4. 扫描到`Bootstrap`结构体
5. 将验证后的配置传递给依赖注入（Wire）

```go
func main() {
	flag.Parse()
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	app, cleanup, err := wireApp(bc.Server, bc.Data, bc.Auth, &bc, logger)
	// ...
}
```

关键行为：
- **验证**: Protobuf强制执行字段类型和存在性
- **持续时间解析**: `"1s"`等字符串转换为`google.protobuf.Duration`
- **错误处理**: 无效的YAML或缺少必需字段会导致启动时panic
- **可扩展性**: 可通过`config.WithSource`添加额外源（如etcd、vault）

**章节来源**
- [main.go](file://cmd/kratos-boilerplate/main.go#L53-L90)
- [conf.proto](file://internal/conf/conf.proto#L1-L69)

## 扩展配置模式
要为新功能添加自定义配置，请修改`conf.proto`并重新生成代码：

1. 在`conf.proto`中添加新字段或消息
2. 使用`protoc`重新生成Go代码
3. 在`config.yaml`中使用新部分更新
4. 通过`Bootstrap`结构体在代码中访问这些值

示例：添加日志配置
```protobuf
message Logging {
  string level = 1; // "debug", "info", "warn", "error"
  string format = 2; // "json", "text"
  bool enable_file_output = 3;
}

message Bootstrap {
  // 现有字段...
  Logging logging = 5;
}
```

然后在`config.yaml`中：
```yaml
logging:
  level: info
  format: json
  enable_file_output: true
```

这种方法保持了类型安全和集中式配置管理。

**章节来源**
- [conf.proto](file://internal/conf/conf.proto#L1-L69)

## 常见配置问题与故障排除
### 格式错误的YAML
症状：`c.Load()`因解析错误而失败  
修复：使用在线工具或`yamllint`验证YAML语法

示例无效：
```yaml
server:
  http:
    addr: 0.0.0.0:8000
  timeout: 1s  # 错误的缩进
```

### 无效值
症状：`c.Scan()`因类型错误而失败  
修复：确保持续时间使用有效格式（`1s`、`300ms`、`2m`），数字不应加引号

示例无效：
```yaml
timeout: "1"  # 应为1s
max_login_attempts: "5"  # 应为5
```

### 缺少必需字段
症状：由于空指针导致运行时panic  
修复：确保YAML中存在所有必需的protobuf字段

在可选时在代码中使用默认值：
```go
if bc.Auth == nil {
	bc.Auth = &conf.Auth{
		JwtSecretKey: "default-key-for-dev",
		AccessTokenExpiration: durationpb.New(15 * time.Minute),
	}
}
```

### 调试技巧
- 打印加载的配置：`log.Printf("Config: %+v", bc)`
- 使用`protoc`验证`.proto`语法
- 检查`flagconf`和`config_file`中的文件路径
- 使用`hierarchical_features_demo.go`验证功能开关行为

**章节来源**
- [main.go](file://cmd/kratos-boilerplate/main.go#L53-L90)
- [conf.proto](file://internal/conf/conf.proto#L1-L69)
- [config.yaml](file://configs/config.yaml#L1-L47)
- [features.yaml](file://configs/features.yaml#L1-L340)