# API接口参考文档

<cite>
**本文档引用的文件**
- [api/auth/v1/auth.proto](file://api/auth/v1/auth.proto)
- [api/helloworld/v1/greeter.proto](file://api/helloworld/v1/greeter.proto)
- [api/common/v1/error.proto](file://api/common/v1/error.proto)
- [frontend/src/api/auth.ts](file://frontend/src/api/auth.ts)
- [frontend/src/api/request.ts](file://frontend/src/api/request.ts)
- [frontend/src/types/api.d.ts](file://frontend/src/types/api.d.ts)
- [internal/service/auth.go](file://internal/service/auth.go)
- [internal/biz/auth.go](file://internal/biz/auth.go)
- [cmd/kratos-boilerplate/main.go](file://cmd/kratos-boilerplate/main.go)
- [configs/swagger-security-example.yaml](file://configs/swagger-security-example.yaml)
</cite>

## 目录
1. [简介](#简介)
2. [项目架构概览](#项目架构概览)
3. [认证服务API](#认证服务api)
4. [功能开关服务API](#功能开关服务api)
5. [问候服务API](#问候服务api)
6. [统一错误处理](#统一错误处理)
7. [API调用示例](#api调用示例)
8. [安全与认证](#安全与认证)
9. [性能考虑](#性能考虑)
10. [故障排除指南](#故障排除指南)
11. [总结](#总结)

## 简介

本文档提供了Kratos Boilerplate项目的完整API接口参考。该项目采用Go-Kratos框架构建，支持gRPC和HTTP RESTful API。通过Protocol Buffers定义API接口，并使用google.api.http注解将gRPC方法映射到RESTful端点。

项目包含三个主要的API服务组：
- **认证服务** (`auth.v1`) - 用户认证、注册、令牌管理
- **问候服务** (`helloworld.v1`) - 基础的gRPC/HTTP示例

## 项目架构概览

```mermaid
graph TB
subgraph "客户端层"
Web[Web浏览器]
Mobile[移动应用]
CLI[命令行工具]
end
subgraph "API网关层"
HTTP[HTTP服务器]
GRPC[gRPC服务器]
Swagger[Swagger UI]
end
subgraph "业务逻辑层"
AuthService[认证服务]
GreeterService[问候服务]
end
subgraph "数据层"
Database[(数据库)]
Cache[(缓存)]
KMS[(密钥管理)]
end
Web --> HTTP
Mobile --> HTTP
CLI --> HTTP
HTTP --> AuthService
HTTP --> GreeterService
GRPC --> AuthService
GRPC --> GreeterService
AuthService --> Database
AuthService --> Cache
AuthService --> KMS
GreeterService --> Database
Swagger --> HTTP
```

**图表来源**
- [cmd/kratos-boilerplate/main.go](file://cmd/kratos-boilerplate/main.go#L1-L92)
- [internal/server/http.go](file://internal/server/http.go)
- [internal/server/grpc.go](file://internal/server/grpc.go)

## 认证服务API

认证服务提供了完整的用户身份验证和授权功能，支持多种认证方式和安全特性。

### API端点概览

| 方法          | HTTP方法 | URL路径                               | 描述             |
| ------------- | -------- | ------------------------------------- | ---------------- |
| GetCaptcha    | GET      | `/api/v1/auth/captcha`                | 获取验证码       |
| VerifyCaptcha | POST     | `/api/v1/auth/captcha/verify`         | 验证验证码       |
| Register      | POST     | `/api/v1/auth/register`               | 用户注册         |
| Login         | POST     | `/api/v1/auth/login`                  | 用户登录         |
| Logout        | POST     | `/api/v1/auth/logout`                 | 退出登录         |
| RefreshToken  | POST     | `/api/v1/auth/refresh`                | 刷新令牌         |
| LockStatus    | GET      | `/api/v1/auth/lock-status/{username}` | 查询账户锁定状态 |

### GetCaptcha - 获取验证码

**HTTP端点**: `GET /api/v1/auth/captcha`

**请求参数**:
```typescript
interface GetCaptchaRequest {
  captcha_type: string;  // 验证码类型: "image", "sms", "email"
  target?: string;       // 目标地址（短信/邮箱）
}
```

**响应数据**:
```typescript
interface CaptchaResponse {
  captcha_id: string;    // 验证码ID
  image_data?: string;   // 图片验证码的Base64数据
}
```

**cURL示例**:
```bash
curl -X GET "http://localhost:8000/api/v1/auth/captcha?captcha_type=image" \
  -H "Content-Type: application/json"
```

**JavaScript示例**:
```javascript
const getCaptcha = async (type, target) => {
  const response = await fetch(`/api/v1/auth/captcha?captcha_type=${type}&target=${target}`);
  return response.json();
};
```

### VerifyCaptcha - 验证验证码

**HTTP端点**: `POST /api/v1/auth/captcha/verify`

**请求体**:
```typescript
interface VerifyCaptchaRequest {
  captcha_id: string;
  captcha_code: string;
}
```

**响应数据**:
```typescript
interface ApiResponse<boolean> {
  code: number;
  message: string;
  data: boolean;
}
```

**cURL示例**:
```bash
curl -X POST "http://localhost:8000/api/v1/auth/captcha/verify" \
  -H "Content-Type: application/json" \
  -d '{"captcha_id": "captcha_123", "captcha_code": "ABCD"}'
```

### Register - 用户注册

**HTTP端点**: `POST /api/v1/auth/register`

**请求体**:
```typescript
interface RegisterRequest {
  username: string;      // 用户名，3-32位，字母数字下划线
  password: string;      // 密码，8-32位，需包含字母数字
  email: string;         // 邮箱地址
  phone?: string;        // 手机号码（国际格式）
  captcha_id: string;    // 验证码ID
  captcha_code: string;  // 验证码内容
}
```

**响应数据**:
```typescript
interface RegisterResponse {
  message: string;       // 注册成功消息
}
```

**cURL示例**:
```bash
curl -X POST "http://localhost:8000/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "password": "SecurePass123!",
    "email": "john@example.com",
    "captcha_id": "captcha_123",
    "captcha_code": "ABCD"
  }'
```

### Login - 用户登录

**HTTP端点**: `POST /api/v1/auth/login`

**请求体**:
```typescript
interface LoginRequest {
  username: string;      // 用户名/邮箱/手机号
  password: string;      // 用户密码
  captcha_id?: string;   // 验证码ID（可选）
  captcha_code?: string; // 验证码内容（可选）
  totp_code?: string;    // TOTP双因子认证码（6位数字）
}
```

**响应数据**:
```typescript
interface LoginResponse {
  access_token: string;  // JWT访问令牌
  refresh_token: string; // 刷新令牌
  expires_in: number;    // 令牌过期时间（秒）
}
```

**cURL示例**:
```bash
curl -X POST "http://localhost:8000/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "password": "SecurePass123!",
    "captcha_id": "captcha_123",
    "captcha_code": "ABCD"
  }'
```

### Logout - 退出登录

**HTTP端点**: `POST /api/v1/auth/logout`

**请求头**:
```typescript
interface LogoutRequest {
  Authorization: string; // Bearer token
}
```

**响应数据**:
```typescript
interface ApiResponse<boolean> {
  code: number;
  message: string;
  data: boolean;
}
```

**cURL示例**:
```bash
curl -X POST "http://localhost:8000/api/v1/auth/logout" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json"
```

### RefreshToken - 刷新令牌

**HTTP端点**: `POST /api/v1/auth/refresh`

**请求体**:
```typescript
interface RefreshTokenRequest {
  refresh_token: string; // 刷新令牌
}
```

**响应数据**:
```typescript
interface LoginResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}
```

### LockStatus - 查询账户锁定状态

**HTTP端点**: `GET /api/v1/auth/lock-status/{username}`

**路径参数**:
```typescript
interface LockStatusRequest {
  username: string;      // 用户名
}
```

**响应数据**:
```typescript
interface LockStatusResponse {
  locked: boolean;       // 是否锁定
  unlock_time: number;   // 解锁时间戳（秒）
  failed_attempts: number; // 已失败次数
  max_attempts: number;  // 最大允许失败次数
}
```

**cURL示例**:
```bash
curl -X GET "http://localhost:8000/api/v1/auth/lock-status/john_doe" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**章节来源**
- [api/auth/v1/auth.proto](file://api/auth/v1/auth.proto#L1-L235)
- [frontend/src/api/auth.ts](file://frontend/src/api/auth.ts#L1-L99)

## 问候服务API

基础的gRPC/HTTP示例服务，演示了最简单的API设计模式。

### SayHello - 发送问候

**HTTP端点**: `GET /helloworld/{name}`

**路径参数**:
```typescript
interface HelloRequest {
  name: string;          // 用户名
}
```

**响应数据**:
```typescript
interface HelloReply {
  message: string;       // 问候消息
}
```

**cURL示例**:
```bash
curl -X GET "http://localhost:8000/helloworld/John" \
  -H "Content-Type: application/json"
```

**JavaScript示例**:
```javascript
const sayHello = async (name) => {
  const response = await fetch(`/helloworld/${name}`);
  const data = await response.json();
  return data.message;
};
```

**章节来源**
- [api/helloworld/v1/greeter.proto](file://api/helloworld/v1/greeter.proto#L1-L31)

## 统一错误处理

项目采用统一的错误响应格式，便于客户端程序化处理各种错误情况。

### 错误响应结构

```mermaid
classDiagram
class ErrorResponse {
+int32 status
+string code
+string message
+string detail
+string trace_id
+Timestamp timestamp
+FieldError[] field_errors
+string suggestion
+string help_url
}
class FieldError {
+string field
+string code
+string message
+string value
}
class ErrorCode {
<<enumeration>>
UNKNOWN
INVALID_REQUEST
VALIDATION_ERROR
UNAUTHORIZED
FORBIDDEN
NOT_FOUND
BUSINESS_ERROR
INTERNAL_ERROR
}
ErrorResponse --> FieldError : "包含"
ErrorResponse --> ErrorCode : "使用"
```

**图表来源**
- [api/common/v1/error.proto](file://api/common/v1/error.proto#L1-L154)

### 错误码分类

| 错误码类别   | HTTP状态码 | 描述                             |
| ------------ | ---------- | -------------------------------- |
| 请求参数错误 | 400        | 参数验证失败、缺失必要参数等     |
| 认证授权错误 | 401/403    | 未授权访问、令牌无效、权限不足等 |
| 资源错误     | 404/409    | 资源不存在、冲突等               |
| 业务逻辑错误 | 422        | 账户锁定、功能禁用、配额超限等   |
| 系统错误     | 500        | 内部服务器错误、服务不可用等     |

### 错误处理示例

**JavaScript错误处理**:
```javascript
const handleError = (error) => {
  if (error.response) {
    const { status, data } = error.response;
    
    switch (status) {
      case 400:
        console.error('请求参数错误:', data.message);
        break;
      case 401:
        console.error('认证失败:', data.message);
        localStorage.removeItem('access_token');
        window.location.href = '/login';
        break;
      case 403:
        console.error('权限不足:', data.message);
        break;
      case 404:
        console.error('资源不存在:', data.message);
        break;
      case 422:
        console.error('业务逻辑错误:', data.message);
        break;
      default:
        console.error('服务器错误:', data.message);
    }
  }
};
```

**章节来源**
- [api/common/v1/error.proto](file://api/common/v1/error.proto#L1-L154)

## API调用示例

### 完整的认证流程示例

```mermaid
sequenceDiagram
participant Client as 客户端
participant Auth as 认证服务
participant Captcha as 验证码服务
participant DB as 数据库
Note over Client,DB : 1. 获取验证码
Client->>Auth : GET /api/v1/auth/captcha
Auth->>Captcha : 生成验证码
Captcha-->>Auth : 验证码信息
Auth-->>Client : {captcha_id, image_data}
Note over Client,DB : 2. 验证验证码
Client->>Auth : POST /api/v1/auth/captcha/verify
Auth->>Captcha : 验证验证码
Captcha-->>Auth : 验证结果
Auth-->>Client : {success : true/false}
Note over Client,DB : 3. 用户注册
Client->>Auth : POST /api/v1/auth/register
Auth->>DB : 保存用户信息
DB-->>Auth : 注册成功
Auth-->>Client : {message : "注册成功"}
Note over Client,DB : 4. 用户登录
Client->>Auth : POST /api/v1/auth/login
Auth->>DB : 验证用户信息
DB-->>Auth : 用户信息
Auth->>Auth : 生成JWT令牌
Auth-->>Client : {access_token, refresh_token, expires_in}
Note over Client,DB : 5. 使用访问令牌
Auth->>Auth : 验证JWT令牌
Auth-->>Client : 功能开关列表
Note over Client,DB : 6. 刷新令牌
Client->>Auth : POST /api/v1/auth/refresh
Auth->>Auth : 验证刷新令牌
Auth->>Auth : 生成新的JWT令牌
Auth-->>Client : {access_token, refresh_token, expires_in}
```

**图表来源**
- [frontend/src/api/auth.ts](file://frontend/src/api/auth.ts#L1-L99)
- [internal/service/auth.go](file://internal/service/auth.go#L1-L235)

### Vue组件集成示例

```vue
<template>
  <div>
    <!-- 登录表单 -->
    <form @submit.prevent="handleSubmit">
      <input v-model="username" placeholder="用户名/邮箱/手机号" required />
      <input v-model="password" type="password" placeholder="密码" required />
      <button type="submit">登录</button>
    </form>
    
    <!-- 功能开关列表 -->
    <div v-if="features.length">
      <h3>功能开关列表</h3>
      <ul>
        <li v-for="feature in features" :key="feature.flag">
          {{ feature.flag }} - {{ feature.config.enabled ? '启用' : '禁用' }}
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { login, getCaptcha, getFeatures } from '@/api/auth'

const username = ref('')
const password = ref('')
const features = ref([])

const handleSubmit = async () => {
  try {
    // 1. 获取验证码
    const captcha = await getCaptcha('image')
    
    // 2. 验证验证码（此处省略用户输入）
    await verifyCaptcha(captcha.captcha_id, 'user_input_captcha')
    
    // 3. 用户登录
    const loginResult = await login({
      username: username.value,
      password: password.value,
      captchaId: captcha.captcha_id,
      captchaCode: 'user_input_captcha'
    })
    
    // 4. 获取功能开关列表
    features.value = await getFeatures()
    
  } catch (error) {
    console.error('登录失败:', error)
  }
}
</script>
```

**章节来源**
- [frontend/src/api/auth.ts](file://frontend/src/api/auth.ts#L1-L99)
- [frontend/src/api/request.ts](file://frontend/src/api/request.ts#L1-L29)

## 安全与认证

### JWT令牌机制

系统采用JSON Web Token (JWT)进行身份验证和授权：

```mermaid
flowchart TD
Login[用户登录] --> GenJWT[生成JWT令牌]
GenJWT --> AccessTok[访问令牌<br/>15分钟过期]
GenJWT --> RefreshTok[刷新令牌<br/>7天过期]
AccessTok --> APIReq[API请求]
APIReq --> ValTok[验证令牌]
ValTok --> Valid{令牌有效?}
Valid --> |是| ProcessReq[处理请求]
Valid --> |否| CheckType{令牌类型}
CheckType --> |访问令牌| TokenErr[令牌无效/过期]
CheckType --> |刷新令牌| RefreshFlow[刷新流程]
RefreshFlow --> ValRefresh[验证刷新令牌]
ValRefresh --> RefreshValid{刷新令牌有效?}
RefreshValid --> |是| GenNewJWT[生成新的JWT令牌]
RefreshValid --> |否| RefreshErr[刷新令牌无效]
GenNewJWT --> NewAccessTok[新的访问令牌]
NewAccessTok --> APIReq
```

**图表来源**
- [internal/biz/auth.go](file://internal/biz/auth.go#L400-L450)

### 认证中间件

```typescript
// 请求拦截器 - 添加认证信息
request.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('access_token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

// 响应拦截器 - 处理认证错误
request.interceptors.response.use(
    (response) => {
        return response.data;
    },
    (error) => {
        if (error.response?.status === 401) {
            // 处理token过期
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);
```

### 安全配置

系统支持多环境的安全配置：

```yaml
# 开发环境 - 最宽松的安全策略
development:
  swagger_ui:
    security:
      enable_auth: false
      allowed_ips: []
      allowed_user_types: []

# 生产环境 - 最严格的安全策略
production:
  swagger_ui:
    security:
      enable_auth: true
      allowed_ips:
        - "172.16.0.100"     # 运维跳板机
        - "172.16.0.101"     # 备用跳板机
      allowed_user_types:
        - "admin"            # 仅管理员
      enable_audit_log: true
```

**章节来源**
- [internal/biz/auth.go](file://internal/biz/auth.go#L1-L695)
- [configs/swagger-security-example.yaml](file://configs/swagger-security-example.yaml#L1-L189)

## 性能考虑

### 缓存策略

- **验证码缓存**: 验证码生成后缓存5分钟
- **令牌黑名单**: 访问令牌过期后立即加入黑名单
- **功能开关缓存**: 功能开关配置定期缓存

### 并发控制

- **账户锁定**: 登录失败达到阈值后锁定账户
- **速率限制**: Swagger UI访问频率限制
- **会话管理**: 最大并发用户数限制

### 数据库优化

- **索引优化**: 用户名、邮箱、手机号建立唯一索引
- **连接池**: 数据库连接池配置
- **事务管理**: 合理使用数据库事务

## 故障排除指南

### 常见错误及解决方案

| 错误类型   | HTTP状态码 | 错误码          | 解决方案       |
| ---------- | ---------- | --------------- | -------------- |
| 验证码错误 | 400        | CAPTCHA_INVALID | 重新获取验证码 |
| 令牌过期   | 401        | TOKEN_EXPIRED   | 使用刷新令牌   |
| 用户不存在 | 404        | USER_NOT_FOUND  | 检查用户名     |
| 权限不足   | 403        | FORBIDDEN       | 检查用户权限   |

### 调试技巧

1. **启用调试日志**:
```bash
export LOG_LEVEL=debug
```

2. **检查Swagger UI**:
```bash
curl http://localhost:8000/q/swagger/index.html
```

3. **验证JWT令牌**:
```javascript
// 使用jwt.io验证令牌
const token = 'your_jwt_token_here';
console.log(atob(token.split('.')[1]));
```

**章节来源**
- [internal/service/auth.go](file://internal/service/auth.go#L50-L150)

## 总结

本文档详细介绍了Kratos Boilerplate项目的API接口参考，涵盖了：

- **完整的API端点**: 认证、功能开关、问候服务的所有接口
- **统一的错误处理**: 标准化的错误响应格式和错误码
- **安全机制**: JWT令牌、认证中间件、多环境安全配置
- **实际示例**: cURL命令、JavaScript调用、Vue组件集成
- **最佳实践**: 性能优化、故障排除、安全考虑

项目采用现代化的微服务架构，支持gRPC和HTTP RESTful API，具有良好的扩展性和维护性。通过Protocol Buffers定义API接口，确保了前后端的一致性和类型安全。

开发者可以基于本文档快速理解和使用这些API接口，构建稳定可靠的客户端应用程序。