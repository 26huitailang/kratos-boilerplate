# API 参考

<cite>
**本文档引用的文件**   
- [auth.proto](file://api/auth/v1/auth.proto)
- [greeter.proto](file://api/helloworld/v1/greeter.proto)
- [error_reason.proto](file://api/helloworld/v1/error_reason.proto)
- [validate.proto](file://third_party/validate/validate.proto)
- [http.proto](file://third_party/google/api/http.proto)
- [feature.proto](file://api/feature/v1/feature.proto) - *新增功能开关服务*
</cite>

## 更新摘要
**已做更改**   
- 在 API 组概述中添加了新的功能开关 API 组
- 新增了功能开关 API 详细文档，包括所有端点、请求/响应模式和示例
- 更新了 gRPC 到 REST 映射部分以包含新服务
- 更新了客户端实现指南以包含功能开关客户端示例
- 添加了新的架构图展示功能开关系统

## 目录
1. [简介](#简介)
2. [API 组概述](#api-组概述)
3. [认证 API](#认证-api)
4. [问候 API](#问候-api)
5. [功能开关 API](#功能开关-api)
6. [gRPC 到 REST 映射](#grpc-到-rest-映射)
7. [错误处理和状态码](#错误处理和状态码)
8. [输入验证](#输入验证)
9. [安全考虑](#安全考虑)
10. [客户端实现指南](#客户端实现指南)

## 简介

本文档提供了 kratos-boilerplate 项目的全面 API 参考文档。该系统暴露了三个主要的 API 组：认证、问候和功能开关服务，使用 gRPC 实现并带有 HTTP/JSON 转码。所有端点都可以通过 gRPC 和 RESTful HTTP 接口访问，为各种客户端类型提供灵活的集成。

文档涵盖了所有公共 API 端点，包括请求/响应模式、认证方法、错误处理策略和客户端实现指南。特别关注 gRPC 服务与 RESTful 端点之间的映射、使用 protobuf 扩展的输入验证机制以及生产部署的安全考虑。

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L1-L155)
- [greeter.proto](file://api/helloworld/v1/greeter.proto#L1-L30)
- [feature.proto](file://api/feature/v1/feature.proto#L1-L181)

## API 组概述

kratos-boilerplate 暴露了三个不同的 API 组：

1. **认证 API** (`auth.v1`): 处理用户认证、注册、会话管理和安全功能，包括验证码验证和账户锁定状态。
2. **问候 API** (`helloworld.v1`): 一个简单的问候服务，演示基本的 RPC 功能和错误处理模式。
3. **功能开关 API** (`feature.v1`): 一个全面的分层功能开关系统，支持基于用户、环境、时间和百分比的策略。

这些 API 在 `api/` 目录中的 Protocol Buffer (protobuf) 文件中定义，服务定义在 `.proto` 文件中，共享错误定义在 `error_reason.proto` 中。系统使用 gRPC 作为主要的 RPC 框架，通过 Google 的 HTTP 注解启用 HTTP/JSON 转码，允许通过 gRPC 和 RESTful HTTP 接口消费相同的服务。

```mermaid
graph TB
subgraph "API 层"
AuthAPI[认证 API<br/>auth.v1]
GreeterAPI[问候 API<br/>helloworld.v1]
FeatureAPI[功能开关 API<br/>feature.v1]
end
subgraph "Protobuf 定义"
AuthProto[auth.proto]
GreeterProto[greeter.proto]
FeatureProto[feature.proto]
ErrorProto[error_reason.proto]
end
AuthProto --> AuthAPI
GreeterProto --> GreeterAPI
FeatureProto --> FeatureAPI
ErrorProto --> AuthAPI
ErrorProto --> GreeterAPI
ErrorProto --> FeatureAPI
style AuthAPI fill:#f9f,stroke:#333
style GreeterAPI fill:#bbf,stroke:#333
style FeatureAPI fill:#f90,stroke:#333
```

**Diagram sources**
- [auth.proto](file://api/auth/v1/auth.proto#L1-L10)
- [greeter.proto](file://api/helloworld/v1/greeter.proto#L1-L10)
- [feature.proto](file://api/feature/v1/feature.proto#L1-L10)

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L1-L20)
- [greeter.proto](file://api/helloworld/v1/greeter.proto#L1-L20)
- [feature.proto](file://api/feature/v1/feature.proto#L1-L20)

## 认证 API

认证 API 提供了全面的用户认证功能，包括注册、登录、会话管理和安全功能。所有端点都在 `api/auth/v1/auth.proto` 中定义，并可通过 gRPC 和 HTTP/JSON 接口访问。

### 获取验证码
检索用于安全验证的验证码。

**HTTP 方法**: GET  
**URL 模式**: `/api/v1/auth/captcha`  
**认证**: 无  
**gRPC 方法**: `Auth.GetCaptcha`

#### 请求模式
```protobuf
message GetCaptchaRequest {
  string captcha_type = 1; // sms, email, image
  string target = 2;       // 电话号码或电子邮件地址
}
```

#### 响应模式
```protobuf
message GetCaptchaReply {
  string captcha_id = 1;
  string image_data = 2; // base64 编码的图像数据（用于图像验证码）
}
```

#### 示例请求
```http
GET /api/v1/auth/captcha?captcha_type=image HTTP/1.1
Host: api.example.com
Content-Type: application/json
```

#### 示例响应
```json
{
  "captcha_id": "cap_123456",
  "image_data": "iVBORw0KGgoAAAANSUhEUgAAAKAAAAB..."
}
```

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L15-L20)

### 验证验证码
验证用户提供的验证码代码。

**HTTP 方法**: POST  
**URL 模式**: `/api/v1/auth/captcha/verify`  
**认证**: 无  
**gRPC 方法**: `Auth.VerifyCaptcha`

#### 请求模式
```protobuf
message VerifyCaptchaRequest {
  string captcha_id = 1;
  string captcha_code = 2;
}
```

#### 响应模式
```protobuf
message VerifyCaptchaReply {
  bool success = 1;
}
```

#### 示例请求
```http
POST /api/v1/auth/captcha/verify HTTP/1.1
Host: api.example.com
Content-Type: application/json

{
  "captcha_id": "cap_123456",
  "captcha_code": "abcd1234"
}
```

#### 示例响应
```json
{
  "success": true
}
```

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L22-L28)

### 注册
注册新用户账户。

**HTTP 方法**: POST  
**URL 模式**: `/api/v1/auth/register`  
**认证**: 无  
**gRPC 方法**: `Auth.Register`

#### 请求模式
```protobuf
message RegisterRequest {
  string username = 1;
  string password = 2;
  string email = 3;
  string phone = 4;
  string captcha_id = 5;
  string captcha_code = 6;
}
```

#### 响应模式
```protobuf
message RegisterReply {
  string message = 1;
}
```

#### 示例请求
```http
POST /api/v1/auth/register HTTP/1.1
Host: api.example.com
Content-Type: application/json

{
  "username": "john_doe",
  "password": "securePassword123",
  "email": "john@example.com",
  "phone": "+1234567890",
  "captcha_id": "cap_123456",
  "captcha_code": "abcd1234"
}
```

#### 示例响应
```json
{
  "message": "用户注册成功"
}
```

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L30-L36)

### 登录
认证用户并返回认证令牌。

**HTTP 方法**: POST  
**URL 模式**: `/api/v1/auth/login`  
**认证**: 无  
**gRPC 方法**: `Auth.Login`

#### 请求模式
```protobuf
message LoginRequest {
  string username = 1;
  string password = 2;
  string captcha_id = 3;
  string captcha_code = 4;
  string totp_code = 5; // 用于基于 TOTP 的双因素认证
}
```

#### 响应模式
```protobuf
message LoginReply {
  string access_token = 1;
  string refresh_token = 2;
  int64 expires_in = 3; // 令牌过期时间（秒）
}
```

#### 示例请求
```http
POST /api/v1/auth/login HTTP/1.1
Host: api.example.com
Content-Type: application/json

{
  "username": "john_doe",
  "password": "securePassword123",
  "captcha_id": "cap_123456",
  "captcha_code": "abcd1234"
}
```

#### 示例响应
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...",
  "refresh_token": "def456...",
  "expires_in": 3600
}
```

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L38-L44)

### 注销
终止用户的当前会话。

**HTTP 方法**: POST  
**URL 模式**: `/api/v1/auth/logout`  
**认证**: Bearer 令牌（访问令牌）  
**gRPC 方法**: `Auth.Logout`

#### 请求模式
```protobuf
message LogoutRequest {}
```

#### 响应模式
```protobuf
message LogoutReply {
  bool success = 1;
}
```

#### 示例请求
```http
POST /api/v1/auth/logout HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...
Content-Type: application/json
```

#### 示例响应
```json
{
  "success": true
}
```

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L46-L52)

### 刷新令牌
使用刷新令牌获取新的访问令牌。

**HTTP 方法**: POST  
**URL 模式**: `/api/v1/auth/refresh`  
**认证**: 无（在请求体中使用刷新令牌）  
**gRPC 方法**: `Auth.RefreshToken`

#### 请求模式
```protobuf
message RefreshTokenRequest {
  string refresh_token = 1;
}
```

#### 响应模式
```protobuf
message RefreshTokenReply {
  string access_token = 1;
  string refresh_token = 2;
  int64 expires_in = 3; // 新令牌过期时间（秒）
}
```

#### 示例请求
```http
POST /api/v1/auth/refresh HTTP/1.1
Host: api.example.com
Content-Type: application/json

{
  "refresh_token": "def456..."
}
```

#### 示例响应
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.yK...",
  "refresh_token": "ghi789...",
  "expires_in": 3600
}
```

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L54-L60)

### 锁定状态
检索给定用户名的账户锁定状态。

**HTTP 方法**: GET  
**URL 模式**: `/api/v1/auth/lock-status/{username}`  
**认证**: 无  
**gRPC 方法**: `Auth.LockStatus`

#### 请求模式
```protobuf
message LockStatusRequest {
  string username = 1;
}
```

#### 响应模式
```protobuf
message LockStatusReply {
  bool locked = 1;
  int64 unlock_time = 2; // 解锁时间戳（秒）
  int32 failed_attempts = 3; // 失败尝试次数
  int32 max_attempts = 4; // 最大允许尝试次数
}
```

#### 示例请求
```http
GET /api/v1/auth/lock-status/john_doe HTTP/1.1
Host: api.example.com
```

#### 示例响应
```json
{
  "locked": false,
  "unlock_time": 0,
  "failed_attempts": 2,
  "max_attempts": 5
}
```

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L62-L68)

## 问候 API

问候 API 提供了一个简单的问候服务，演示了基本的 RPC 功能和错误处理模式。

### 说你好
向指定用户发送问候消息。

**HTTP 方法**: GET  
**URL 模式**: `/helloworld/{name}`  
**认证**: Bearer 令牌（访问令牌）  
**gRPC 方法**: `Greeter.SayHello`

#### 请求模式
```protobuf
message HelloRequest {
  string name = 1;
}
```

#### 响应模式
```protobuf
message HelloReply {
  string message = 1;
}
```

#### 示例请求
```http
GET /helloworld/John HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...
```

#### 示例响应
```json
{
  "message": "Hello John"
}
```

**Section sources**
- [greeter.proto](file://api/helloworld/v1/greeter.proto#L15-L20)

## 功能开关 API

功能开关 API 提供了一个全面的分层功能开关系统，支持多种策略类型和复杂的评估逻辑。所有端点都在 `api/feature/v1/feature.proto` 中定义，并可通过 gRPC 和 HTTP/JSON 接口访问。

### 列出开关
获取所有功能开关的列表。

**HTTP 方法**: GET  
**URL 模式**: `/api/v1/features`  
**认证**: Bearer 令牌（访问令牌）  
**gRPC 方法**: `FeatureToggle.ListToggles`

#### 请求模式
```protobuf
message ListTogglesRequest {
  int32 page = 1;
  int32 page_size = 2;
  repeated string tags = 3;
  optional bool enabled_only = 4;
}
```

#### 响应模式
```protobuf
message ListTogglesReply {
  repeated FeatureToggleInfo toggles = 1;
  int32 total = 2;
  int32 page = 3;
  int32 page_size = 4;
}
```

#### 示例请求
```http
GET /api/v1/features?page=1&page_size=10&tags=security HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...
```

#### 示例响应
```json
{
  "toggles": [
    {
      "flag": "user.management",
      "config": {
        "enabled": true,
        "strategy": "SIMPLE",
        "rules": {},
        "description": "用户管理功能",
        "tags": ["security", "user"],
        "created_at": "2023-01-01T00:00:00Z",
        "updated_at": "2023-01-01T00:00:00Z"
      }
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 10
}
```

**Section sources**
- [feature.proto](file://api/feature/v1/feature.proto#L15-L25)

### 获取开关
获取单个功能开关的详细信息。

**HTTP 方法**: GET  
**URL 模式**: `/api/v1/features/{flag}`  
**认证**: Bearer 令牌（访问令牌）  
**gRPC 方法**: `FeatureToggle.GetToggle`

#### 请求模式
```protobuf
message GetToggleRequest {
  string flag = 1;
}
```

#### 响应模式
```protobuf
message GetToggleReply {
  FeatureToggleInfo toggle = 1;
}
```

#### 示例请求
```http
GET /api/v1/features/user.management HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...
```

#### 示例响应
```json
{
  "toggle": {
    "flag": "user.management",
    "config": {
      "enabled": true,
      "strategy": "SIMPLE",
      "rules": {},
      "description": "用户管理功能",
      "tags": ["security", "user"],
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-01T00:00:00Z"
    }
  }
}
```

**Section sources**
- [feature.proto](file://api/feature/v1/feature.proto#L27-L33)

### 更新开关
更新功能开关的配置。

**HTTP 方法**: PUT  
**URL 模式**: `/api/v1/features/{flag}`  
**认证**: Bearer 令牌（访问令牌）  
**gRPC 方法**: `FeatureToggle.UpdateToggle`

#### 请求模式
```protobuf
message UpdateToggleRequest {
  string flag = 1;
  ToggleConfig config = 2;
}
```

#### 响应模式
```protobuf
message UpdateToggleReply {
  FeatureToggleInfo toggle = 1;
}
```

#### 示例请求
```http
PUT /api/v1/features/user.management HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...
Content-Type: application/json

{
  "config": {
    "enabled": true,
    "strategy": "USER",
    "rules": {
      "user_types": "admin,moderator"
    },
    "description": "更新后的用户管理功能",
    "tags": ["security", "user", "updated"]
  }
}
```

#### 示例响应
```json
{
  "toggle": {
    "flag": "user.management",
    "config": {
      "enabled": true,
      "strategy": "USER",
      "rules": {
        "user_types": "admin,moderator"
      },
      "description": "更新后的用户管理功能",
      "tags": ["security", "user", "updated"],
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-02T00:00:00Z"
    }
  }
}
```

**Section sources**
- [feature.proto](file://api/feature/v1/feature.proto#L35-L41)

### 启用开关
启用功能开关。

**HTTP 方法**: POST  
**URL 模式**: `/api/v1/features/{flag}/enable`  
**认证**: Bearer 令牌（访问令牌）  
**gRPC 方法**: `FeatureToggle.EnableToggle`

#### 请求模式
```protobuf
message EnableToggleRequest {
  string flag = 1;
}
```

#### 响应模式
```protobuf
message EnableToggleReply {
  FeatureToggleInfo toggle = 1;
}
```

#### 示例请求
```http
POST /api/v1/features/user.management/enable HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...
Content-Type: application/json
```

#### 示例响应
```json
{
  "toggle": {
    "flag": "user.management",
    "config": {
      "enabled": true,
      "strategy": "USER",
      "rules": {
        "user_types": "admin,moderator"
      },
      "description": "更新后的用户管理功能",
      "tags": ["security", "user", "updated"],
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-02T00:00:00Z"
    }
  }
}
```

**Section sources**
- [feature.proto](file://api/feature/v1/feature.proto#L43-L49)

### 禁用开关
禁用功能开关。

**HTTP 方法**: POST  
**URL 模式**: `/api/v1/features/{flag}/disable`  
**认证**: Bearer 令牌（访问令牌）  
**gRPC 方法**: `FeatureToggle.DisableToggle`

#### 请求模式
```protobuf
message DisableToggleRequest {
  string flag = 1;
}
```

#### 响应模式
```protobuf
message DisableToggleReply {
  FeatureToggleInfo toggle = 1;
}
```

#### 示例请求
```http
POST /api/v1/features/user.management/disable HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...
Content-Type: application/json
```

#### 示例响应
```json
{
  "toggle": {
    "flag": "user.management",
    "config": {
      "enabled": false,
      "strategy": "USER",
      "rules": {
        "user_types": "admin,moderator"
      },
      "description": "更新后的用户管理功能",
      "tags": ["security", "user", "updated"],
      "created_at": "2023-01-01T00:00:00Z",
      "updated_at": "2023-01-03T00:00:00Z"
    }
  }
}
```

**Section sources**
- [feature.proto](file://api/feature/v1/feature.proto#L51-L57)

### 删除开关
删除功能开关。

**HTTP 方法**: DELETE  
**URL 模式**: `/api/v1/features/{flag}`  
**认证**: Bearer 令牌（访问令牌）  
**gRPC 方法**: `FeatureToggle.DeleteToggle`

#### 请求模式
```protobuf
message DeleteToggleRequest {
  string flag = 1;
}
```

#### 响应模式
```protobuf
message DeleteToggleReply {
  bool success = 1;
}
```

#### 示例请求
```http
DELETE /api/v1/features/user.management HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...
```

#### 示例响应
```json
{
  "success": true
}
```

**Section sources**
- [feature.proto](file://api/feature/v1/feature.proto#L59-L65)

### 评估开关
评估功能开关在特定上下文中的状态。

**HTTP 方法**: POST  
**URL 模式**: `/api/v1/features/{flag}/evaluate`  
**认证**: Bearer 令牌（访问令牌）  
**gRPC 方法**: `FeatureToggle.EvaluateToggle`

#### 请求模式
```protobuf
message EvaluateToggleRequest {
  string flag = 1;
  EvaluationContext context = 2;
}

message EvaluationContext {
  string user_id = 1;
  string user_type = 2;
  string environment = 3;
  string version = 4;
  map<string, string> attributes = 5;
}
```

#### 响应模式
```protobuf
message EvaluateToggleReply {
  bool enabled = 1;
  string reason = 2;
}
```

#### 示例请求
```http
POST /api/v1/features/user.management/evaluate HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...
Content-Type: application/json

{
  "context": {
    "user_id": "user123",
    "user_type": "admin",
    "environment": "production",
    "version": "1.0.0",
    "attributes": {
      "department": "engineering"
    }
  }
}
```

#### 示例响应
```json
{
  "enabled": true,
  "reason": "Feature is enabled"
}
```

**Section sources**
- [feature.proto](file://api/feature/v1/feature.proto#L67-L73)

### 获取统计
获取功能开关系统的统计信息。

**HTTP 方法**: GET  
**URL 模式**: `/api/v1/features/stats`  
**认证**: Bearer 令牌（访问令牌）  
**gRPC 方法**: `FeatureToggle.GetStats`

#### 请求模式
```protobuf
message GetStatsRequest {}
```

#### 响应模式
```protobuf
message GetStatsReply {
  int32 total_toggles = 1;
  int32 enabled_count = 2;
  int32 disabled_count = 3;
  map<string, int32> strategies = 4;
}
```

#### 示例请求
```http
GET /api/v1/features/stats HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xJ...
```

#### 示例响应
```json
{
  "total_toggles": 15,
  "enabled_count": 8,
  "disabled_count": 7,
  "strategies": {
    "SIMPLE": 10,
    "USER": 3,
    "PERCENTAGE": 2
  }
}
```

**Section sources**
- [feature.proto](file://api/feature/v1/feature.proto#L75-L81)

## gRPC 到 REST 映射

kratos-boilerplate 使用 Google 的 HTTP 注解将 gRPC 服务映射到 RESTful 端点。这使得相同的服务可以通过 gRPC 和 HTTP/JSON 接口消费，为不同的客户端需求提供灵活性。

### 映射机制

映射在 `.proto` 文件中使用 `google.api.http` 注解定义，该注解指定了如何将 gRPC 方法暴露为 HTTP 端点。系统使用 `google/api/annotations.proto` 和 `google/api/http.proto` 文件定义这些映射。

```mermaid
sequenceDiagram
participant Client as "HTTP 客户端"
participant Gateway as "gRPC-Gateway"
participant Service as "gRPC 服务"
Client->>Gateway : HTTP 请求<br>GET /api/v1/auth/captcha
Gateway->>Service : gRPC 调用<br>Auth.GetCaptcha()
Service-->>Gateway : gRPC 响应
Gateway-->>Client : HTTP 响应 (JSON)
Note over Client,Gateway : HTTP/JSON 接口
Client->>Service : gRPC 调用<br>Auth.GetCaptcha()
Service-->>Client : gRPC 响应
Note over Client,Service : 原生 gRPC 接口
```

**Diagram sources**
- [auth.proto](file://api/auth/v1/auth.proto#L15-L20)
- [feature.proto](file://api/feature/v1/feature.proto#L15-L25)
- [http.proto](file://third_party/google/api/http.proto#L1-L50)

### HTTP 注解语法

HTTP 注解在 `.proto` 文件中使用以下语法：

```protobuf
rpc GetCaptcha(GetCaptchaRequest) returns (GetCaptchaReply) {
  option (google.api.http) = {
    get: "/api/v1/auth/captcha"
  };
}
```

该注解支持多种 HTTP 方法：
- `get`: 映射到 HTTP GET 方法
- `post`: 映射到 HTTP POST 方法
- `put`: 映射到 HTTP PUT 方法
- `delete`: 映射到 HTTP DELETE 方法
- `patch`: 映射到 HTTP PATCH 方法

`body` 字段指定哪个请求字段映射到 HTTP 请求体。当设置为 `"*"` 时，所有未被路径模板捕获的字段都映射到请求体。

### 路径参数映射

URL 模板中的路径参数使用花括号表示法映射到请求消息字段：

```protobuf
rpc LockStatus(LockStatusRequest) returns (LockStatusReply) {
  option (google.api.http) = {
    get: "/api/v1/auth/lock-status/{username}"
  };
}
```

在此示例中，`{username}` 路径参数映射到 `LockStatusRequest` 消息中的 `username` 字段。

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L15-L68)
- [feature.proto](file://api/feature/v1/feature.proto#L15-L81)
- [http.proto](file://third_party/google/api/http.proto#L1-L100)

## 错误处理和状态码

API 使用基于 gRPC 状态码和在 `error_reason.proto` 中定义的自定义错误原因的一致错误处理策略。

### 标准 gRPC 状态码

API 返回标准 gRPC 状态码，这些状态码由 gRPC-Gateway 自动转换为 HTTP 状态码：

| gRPC 状态 | HTTP 状态 | 描述 |
|-------------|-------------|-------------|
| OK | 200 | 请求成功 |
| INVALID_ARGUMENT | 400 | 客户端指定了无效参数 |
| UNAUTHENTICATED | 401 | 请求未认证 |
| PERMISSION_DENIED | 403 | 请求未授权 |
| NOT_FOUND | 404 | 资源未找到 |
| ALREADY_EXISTS | 409 | 资源已存在 |
| FAILED_PRECONDITION | 412 | 请求失败前提条件 |
| INTERNAL | 500 | 内部服务器错误 |
| UNAVAILABLE | 503 | 服务不可用 |

### 自定义错误原因

`error_reason.proto` 文件定义了域特定错误的自定义错误原因：

```protobuf
enum ErrorReason {
  GREETER_UNSPECIFIED = 0;
  USER_NOT_FOUND = 1;
}
```

发生错误时，API 返回包含错误原因、消息和详细信息的结构化错误响应：

```json
{
  "error": {
    "code": 5,
    "message": "用户未找到",
    "details": [
      {
        "@type": "type.googleapis.com/helloworld.v1.ErrorReason",
        "reason": "USER_NOT_FOUND"
      }
    ]
  }
}
```

### 错误响应结构

错误响应遵循标准 gRPC 错误格式，包括：
- `code`: gRPC 状态码
- `message`: 人类可读的错误消息
- `details`: 附加的结构化错误信息，包括自定义错误原因

```mermaid
flowchart TD
Request["客户端请求"] --> Validation["输入验证"]
Validation --> Valid{"有效?"}
Valid --> |否| ValidationError["返回 400 错误请求"]
Valid --> |是| Auth["认证检查"]
Auth --> Authenticated{"已认证?"}
Authenticated --> |否| AuthError["返回 401 未授权"]
Authenticated --> |是| Authz["授权检查"]
Authz --> Authorized{"已授权?"}
Authorized --> |否| AuthzError["返回 403 禁止"]
Authorized --> |是| Processing["业务逻辑处理"]
Processing --> Success{"成功?"}
Success --> |否| BusinessError["返回适当的错误"]
Success --> |是| SuccessResponse["返回 200 OK"]
style ValidationError fill:#f99,stroke:#333
style AuthError fill:#f99,stroke:#333
style AuthzError fill:#f99,stroke:#333
style BusinessError fill:#f99,stroke:#333
style SuccessResponse fill:#9f9,stroke:#333
```

**Diagram sources**
- [error_reason.proto](file://api/helloworld/v1/error_reason.proto#L1-L12)
- [auth.proto](file://api/auth/v1/auth.proto#L1-L155)
- [feature.proto](file://api/feature/v1/feature.proto#L1-L181)

**Section sources**
- [error_reason.proto](file://api/helloworld/v1/error_reason.proto#L1-L12)
- [auth.proto](file://api/auth/v1/auth.proto#L1-L155)
- [feature.proto](file://api/feature/v1/feature.proto#L1-L181)

## 输入验证

API 使用 `validate.proto` 扩展实现全面的输入验证，为 protobuf 消息提供字段级验证规则。

### 验证规则

`validate.proto` 文件定义了各种字段类型的验证规则：

#### 字符串验证
```protobuf
message StringRules {
  optional uint64 min_len = 2;
  optional uint64 max_len = 3;
  optional string pattern = 6;
  optional string prefix = 7;
  optional string suffix = 8;
  repeated string in = 10;
  bool email = 12;
  bool hostname = 13;
  bool ip = 14;
  bool ipv4 = 15;
  bool ipv6 = 16;
  bool uri = 17;
  bool uri_ref = 18;
  bool uuid = 22;
}
```

#### 示例：注册请求验证
```protobuf
message RegisterRequest {
  string username = 1 [(validate.rules).string = {
    min_len: 3,
    max_len: 32,
    pattern: "^[a-zA-Z0-9_-]+$"
  }];
  string password = 2 [(validate.rules).string = {
    min_len: 8,
    max_len: 128,
    pattern: "^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d)(?=.*[@$!%*?&])[A-Za-z\\d@$!%*?&]+$"
  }];
  string email = 3 [(validate.rules).string = {
    email: true
  }];
  string phone = 4 [(validate.rules).string = {
    pattern: "^\\+?[1-9]\\d{1,14}$"
  }];
  string captcha_id = 5 [(validate.rules).string = {
    min_len: 1
  }];
  string captcha_code = 6 [(validate.rules).string = {
    min_len: 4,
    max_len: 8
  }];
}
```

### 验证执行

验证在收到请求时自动发生，在请求被业务逻辑处理之前。如果验证失败，API 返回 `400 错误请求` 响应，并附带验证错误的详细信息。

```mermaid
sequenceDiagram
participant Client as "客户端"
participant Gateway as "gRPC-Gateway"
participant Validator as "验证器"
participant Service as "服务"
Client->>Gateway : 发送请求
Gateway->>Validator : 验证请求
alt 有效
Validator-->>Gateway : 验证成功
Gateway->>Service : 转发请求
Service-->>Gateway : 处理请求
Gateway-->>Client : 返回响应
else 无效
Validator-->>Gateway : 验证错误
Gateway-->>Client : 返回 400 错误请求
end
```

**Diagram sources**
- [validate.proto](file://third_party/validate/validate.proto#L1-L50)
- [auth.proto](file://api/auth/v1/auth.proto#L1-L155)
- [feature.proto](file://api/feature/v1/feature.proto#L1-L181)

**Section sources**
- [validate.proto](file://third_party/validate/validate.proto#L1-L100)
- [auth.proto](file://api/auth/v1/auth.proto#L1-L155)
- [feature.proto](file://api/feature/v1/feature.proto#L1-L181)

## 安全考虑

API 实施了多项安全措施以防止常见威胁并确保安全运行。

### 认证和授权

API 使用 JWT (JSON Web 令牌) 进行认证和授权：

- **访问令牌**: 短寿命令牌（通常 1 小时），用于认证 API 请求
- **刷新令牌**: 长寿命令牌（通常 7 天），用于在无需用户凭据的情况下获取新的访问令牌
- **令牌存储**: 令牌应在客户端安全存储（Web 的 HTTP-only cookie，移动设备的安全存储）

### 速率限制

API 应实施速率限制以防止滥用和拒绝服务攻击：

- **验证码端点**: 限制以防止垃圾邮件
- **登录端点**: 限制以防止暴力攻击
- **注册端点**: 限制以防止账户枚举

### 输入验证

通过 protobuf 验证规则实施全面的输入验证，以防止：
- 注入攻击（SQL、NoSQL、命令等）
- 跨站脚本（XSS）
- 路径遍历
- 缓冲区溢出

### 敏感数据保护

API 应通过以下方式保护敏感数据：
- **加密**: 所有敏感数据在静态和传输中加密
- **掩码**: 敏感数据在日志和响应中掩码
- **访问控制**: 对敏感操作的严格访问控制策略

```mermaid
graph TD
A[客户端请求] --> B{安全检查}
B --> C[认证]
B --> D[授权]
B --> E[速率限制]
B --> F[输入验证]
C --> G[有效令牌?]
D --> H[有权限?]
E --> I[在限制内?]
F --> J[有效输入?]
G --> |否| K[返回 401]
H --> |否| L[返回 403]
I --> |否| M[返回 429]
J --> |否| N[返回 400]
G --> |是| O[处理请求]
H --> |是| O
I --> |是| O
J --> |是| O
O --> P[返回响应]
style K fill:#f99,stroke:#333
style L fill:#f99,stroke:#333
style M fill:#f99,stroke:#333
style N fill:#f99,stroke:#333
style O fill:#9f9,stroke:#333
style P fill:#9f9,stroke:#333
```

**Diagram sources**
- [auth.proto](file://api/auth/v1/auth.proto#L1-L155)
- [validate.proto](file://third_party/validate/validate.proto#L1-L100)
- [feature.proto](file://api/feature/v1/feature.proto#L1-L181)

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L1-L155)
- [validate.proto](file://third_party/validate/validate.proto#L1-L100)
- [feature.proto](file://api/feature/v1/feature.proto#L1-L181)

## 客户端实现指南

本节提供指南，用于实现通过 HTTP 和 gRPC 接口消费 API 的客户端。

### HTTP 客户端实现

HTTP 客户端可以使用标准 HTTP 库消费 API。API 返回与任何 JSON 能力客户端兼容的 JSON 响应。

#### JavaScript 示例
```javascript
// 使用 fetch API 的登录请求
async function login(username, password, captchaId, captchaCode) {
  const response = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      username,
      password,
      captcha_id: captchaId,
      captcha_code: captchaCode
    })
  });
  
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  
  return await response.json();
}

// 使用访问令牌进行认证请求
async function sayHello(name, accessToken) {
  const response = await fetch(`/helloworld/${name}`, {
    headers: {
      'Authorization': `Bearer ${accessToken}`
    }
  });
  
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  
  return await response.json();
}

// 功能开关评估请求
async function evaluateFeature(flag, context, accessToken) {
  const response = await fetch(`/api/v1/features/${flag}/evaluate`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${accessToken}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ context })
  });
  
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  
  return await response.json();
}
```

#### Python 示例
```python
import requests

# 登录请求
def login(username, password, captcha_id, captcha_code):
    url = "http://api.example.com/api/v1/auth/login"
    data = {
        "username": username,
        "password": password,
        "captcha_id": captcha_id,
        "captcha_code": captcha_code
    }
    response = requests.post(url, json=data)
    response.raise_for_status()
    return response.json()

# 认证请求
def say_hello(name, access_token):
    url = f"http://api.example.com/helloworld/{name}"
    headers = {"Authorization": f"Bearer {access_token}"}
    response = requests.get(url, headers=headers)
    response.raise_for_status()
    return response.json()

# 功能开关评估请求
def evaluate_feature(flag, context, access_token):
    url = f"http://api.example.com/api/v1/features/{flag}/evaluate"
    headers = {"Authorization": f"Bearer {access_token}"}
    data = {"context": context}
    response = requests.post(url, json=data, headers=headers)
    response.raise_for_status()
    return response.json()
```

### gRPC 客户端实现

gRPC 客户端提供比 HTTP 客户端更好的性能和类型安全性。

#### Go 示例
```go
package main

import (
    "context"
    "log"
    
    pb "kratos-boilerplate/api/auth/v1"
    featurepb "kratos-boilerplate/api/feature/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    // 连接到服务器
    conn, err := grpc.Dial("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()
    
    // 认证客户端
    authClient := pb.NewAuthClient(conn)
    
    // 登录请求
    loginReq := &pb.LoginRequest{
        Username:     "john_doe",
        Password:     "securePassword123",
        CaptchaId:    "cap_123456",
        CaptchaCode:  "abcd1234",
    }
    
    loginResp, err := authClient.Login(context.Background(), loginReq)
    if err != nil {
        log.Fatalf("login failed: %v", err)
    }
    
    log.Printf("Access token: %s", loginResp.AccessToken)
    
    // 功能开关客户端
    featureClient := featurepb.NewFeatureToggleClient(conn)
    
    // 评估功能开关
    evalReq := &featurepb.EvaluateToggleRequest{
        Flag: "user.management",
        Context: &featurepb.EvaluationContext{
            UserId:      "user123",
            UserType:    "admin",
            Environment: "production",
            Version:     "1.0.0",
            Attributes:  map[string]string{"department": "engineering"},
        },
    }
    
    evalResp, err := featureClient.EvaluateToggle(context.Background(), evalReq)
    if err != nil {
        log.Fatalf("evaluate toggle failed: %v", err)
    }
    
    log.Printf("Feature enabled: %v, reason: %s", evalResp.Enabled, evalResp.Reason)
}
```

#### JavaScript/TypeScript 示例
```typescript
import { AuthClient } from './gen/auth_grpc_web_pb';
import { FeatureToggleClient } from './gen/feature_grpc_web_pb';
import { LoginRequest, LoginReply } from './gen/auth_pb';
import { EvaluateToggleRequest, EvaluateToggleReply } from './gen/feature_pb';

// 创建客户端
const authClient = new AuthClient('http://localhost:8080');
const featureClient = new FeatureToggleClient('http://localhost:8080');

// 登录请求
const loginRequest = new LoginRequest();
loginRequest.setUsername('john_doe');
loginRequest.setPassword('securePassword123');
loginRequest.setCaptchaId('cap_123456');
loginRequest.setCaptchaCode('abcd1234');

authClient.login(loginRequest, {}, (err: grpcWeb.Error, response: LoginReply) => {
  if (err) {
    console.error('Login error:', err.message);
    return;
  }
  
  console.log('Access token:', response.getAccessToken());
  
  // 评估功能开关
  const evalRequest = new EvaluateToggleRequest();
  evalRequest.setFlag('user.management');
  
  const context = new EvaluationContext();
  context.setUserId('user123');
  context.setUserType('admin');
  context.setEnvironment('production');
  context.setVersion('1.0.0');
  context.putAllAttributes({'department': 'engineering'});
  
  evalRequest.setContext(context);
  
  featureClient.evaluateToggle(evalRequest, {}, (err: grpcWeb.Error, response: EvaluateToggleReply) => {
    if (err) {
      console.error('Evaluate toggle error:', err.message);
      return;
    }
    
    console.log('Feature enabled:', response.getEnabled());
    console.log('Reason:', response.getReason());
  });
});
```

### 客户端中的错误处理

客户端应实现健壮的错误处理以处理各种错误情况：

```javascript
// 通用错误处理包装器
async function apiCall(callFn) {
  try {
    const result = await callFn();
    return { success: true, data: result };
  } catch (error) {
    if (error.status === 401) {
      // 处理未授权 - 重定向到登录
      window.location.href = '/login';
      return { success: false, error: '未授权' };
    } else if (error.status === 429) {
      // 处理速率限制
      return { success: false, error: '请求过多。请稍后再试。' };
    } else if (error.status >= 400 && error.status < 500) {
      // 客户端错误
      return { success: false, error: `客户端错误: ${error.message}` };
    } else {
      // 服务器错误
      return { success: false, error: '服务器错误。请稍后再试。' };
    }
  }
}
```

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L1-L155)
- [greeter.proto](file://api/helloworld/v1/greeter.proto#L1-L30)
- [feature.proto](file://api/feature/v1/feature.proto#L1-L181)