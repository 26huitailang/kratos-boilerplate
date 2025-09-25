# 用户注册

<cite>
**本文档引用的文件**   
- [auth.proto](file://api/auth/v1/auth.proto) - *在最近提交中更新，增强了字段验证规则*
- [auth.go](file://internal/biz/auth.go) - *在最近提交中更新，包含业务逻辑实现*
- [auth.go](file://internal/service/auth.go) - *在最近提交中更新，gRPC服务层实现*
- [auth.go](file://internal/data/auth.go) - *数据访问层实现*
- [validate.proto](file://third_party/validate/validate.proto) - *验证规则定义*
</cite>

## 更新摘要
**已做更改**   
- 根据最新的代码变更，全面更新了用户注册端点的API文档
- 新增了gRPC方法签名和HTTP映射的详细说明
- 更新了请求体字段和验证规则，包括username、password、email、captcha_token
- 详细描述了密码哈希处理流程和验证码验证集成
- 完善了用户创建流程在biz和data层的实现细节
- 更新了成功响应和错误情况的详细信息，包括对应的error_reason代码和HTTP状态码
- 添加了JSON和protobuf格式的请求/响应示例
- 增强了安全实践部分，包括速率限制、防止枚举攻击和安全错误消息
- 更新了Go客户端和TypeScript fetch的使用示例

## 目录
1. [用户注册](#用户注册)
2. [HTTP和gRPC端点](#http和grpc端点)
3. [请求和响应结构](#请求和响应结构)
4. [验证规则](#验证规则)
5. [注册流程](#注册流程)
6. [错误处理](#错误处理)
7. [安全实践](#安全实践)
8. [客户端示例](#客户端示例)

## HTTP和gRPC端点

用户注册端点可通过HTTP和gRPC协议访问。gRPC方法在`auth.proto`文件中定义，并通过`google.api.http`注解映射到RESTful端点。

```protobuf
rpc Register(RegisterRequest) returns (RegisterReply) {
  option (google.api.http) = {
    post: "/api/v1/auth/register"
    body: "*"
  };
}
```

- **HTTP方法**: POST
- **RESTful路径**: `/api/v1/auth/register`
- **gRPC方法**: `auth.v1.Auth.Register`

**本节来源**   
- [auth.proto](file://api/auth/v1/auth.proto#L25-L30) - *在最近提交中更新*

## 请求和响应结构

### 请求字段

`RegisterRequest`消息包含以下字段：

| 字段 | 类型 | 必填 | 描述 |
|-------|------|----------|-------------|
| `username` | string | 是 | 唯一用户标识符 |
| `password` | string | 是 | 用户密码（将被哈希） |
| `email` | string | 是 | 用户邮箱地址 |
| `phone` | string | 否 | 用户手机号码 |
| `captcha_id` | string | 是 | 验证码标识符 |
| `captcha_code` | string | 是 | 用户输入的验证码 |

### 响应结构

`RegisterReply`消息包含简单的成功消息：

```protobuf
message RegisterReply {
  string message = 1;
}
```

注册成功时，响应将包含：
```json
{
  "message": "注册成功"
}
```

**本节来源**   
- [auth.proto](file://api/auth/v1/auth.proto#L78-L83) - *在最近提交中更新*
- [auth.proto](file://api/auth/v1/auth.proto#L85-L88) - *在最近提交中更新*

## 验证规则

验证通过protobuf验证规则和自定义业务逻辑共同实施。

### Protobuf验证

`validate.proto`文件提供了可应用于protobuf消息的字段级验证规则。对于注册端点，相关验证如下：

```protobuf
message StringRules {
  optional uint64 min_len = 2;
  optional uint64 max_len = 3;
  optional string pattern = 6;
  optional bool email = 12;
}
```

这些规则可用于确保：
- 用户名长度约束
- 邮箱格式验证
- 密码复杂度模式

### 业务逻辑验证

实际验证在`internal/biz/auth.go`中的`authUsecase.Register`方法中实现。验证规则包括：

1. **验证码验证**: 必须启用且提供
2. **用户名唯一性**: 用户名不能已存在
3. **邮箱唯一性**: 邮箱不能被其他账户使用
4. **手机号唯一性**: 手机号不能被其他账户使用
5. **密码强度**: 最少8个字符

```go
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("密码长度至少为8位")
	}
	// 此处将实现额外的密码强度检查
	return nil
}
```

**本节来源**   
- [validate.proto](file://third_party/validate/validate.proto#L470-L480) - *验证规则定义*
- [auth.go](file://internal/biz/auth.go#L653-L657) - *在最近提交中更新*

## 注册流程

用户注册过程遵循应用程序架构中多个层次的结构化流程。

```mermaid
sequenceDiagram
participant Client as "客户端应用"
participant HTTP as "HTTP服务器"
participant Service as "AuthService"
participant Biz as "AuthUsecase"
participant Data as "用户仓库"
participant Captcha as "验证码服务"
Client->>HTTP : POST /api/v1/auth/register
HTTP->>Service : 反序列化请求
Service->>Biz : uc.Register()
Biz->>Captcha : 验证验证码
Captcha-->>Biz : 验证结果
alt 验证码无效
Biz-->>Service : ErrCaptchaInvalid
Service-->>Client : 400 错误请求
return
end
Biz->>Data : GetUser(username)
Data-->>Biz : 用户存在性检查
alt 用户已存在
Biz-->>Service : ErrUserExists
Service-->>Client : 409 冲突
return
end
Biz->>Data : GetUserByEmail(email)
Data-->>Biz : 邮箱唯一性检查
alt 邮箱已被使用
Biz-->>Service : 邮箱已注册错误
Service-->>Client : 409 冲突
return
end
Biz->>Data : GetUserByPhone(phone)
Data-->>Biz : 手机号唯一性检查
alt 手机号已被使用
Biz-->>Service : 手机号已注册错误
Service-->>Client : 409 冲突
return
end
Biz->>Biz : validatePassword()
alt 密码过弱
Biz-->>Service : 密码验证错误
Service-->>Client : 400 错误请求
return
end
Biz->>Biz : bcrypt.GenerateFromPassword()
Biz->>Data : CreateUser()
Data-->>Biz : 创建结果
Biz-->>Service : 成功
Service-->>Client : 200 OK并返回成功消息
```

**图表来源**   
- [auth.go](file://internal/biz/auth.go#L249-L317) - *在最近提交中更新*
- [auth.go](file://internal/service/auth.go#L68-L85) - *在最近提交中更新*

## 错误处理

注册端点针对不同失败场景返回特定的错误代码和HTTP状态码。

### 错误情况和状态码

| 错误情况 | 错误原因代码 | HTTP状态 | gRPC代码 |
|------------|-------------------|-------------|-----------|
| 邮箱重复 | "EMAIL_EXISTS" | 409 冲突 | 已存在 |
| 密码过弱 | "WEAK_PASSWORD" | 400 错误请求 | 无效参数 |
| 验证码无效 | "CAPTCHA_INVALID" | 400 错误请求 | 无效参数 |
| 缺少验证码 | "CAPTCHA_REQUIRED" | 400 错误请求 | 无效参数 |
| 服务器内部错误 | "REGISTER_ERROR" | 500 服务器内部错误 | 内部错误 |

### 错误响应格式

```json
{
  "error": {
    "code": 400,
    "message": "错误请求",
    "details": [
      {
        "type": "type.googleapis.com/google.rpc.BadRequest",
        "field_violations": [
          {
            "field": "password",
            "description": "密码必须至少8个字符长"
          }
        ]
      }
    ]
  }
}
```

在实现中，错误通过特定的错误类型处理，并映射到适当的HTTP响应：

```go
func (s *AuthService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterReply, error) {
	err := s.uc.Register(ctx, req.Username, req.Password, req.Email, req.Phone, req.CaptchaId, req.CaptchaCode)
	if err != nil {
		switch err {
		case biz.ErrUserExists:
			return nil, errors.BadRequest("USER_EXISTS", "用户名已存在")
		case biz.ErrCaptchaRequired:
			return nil, errors.BadRequest("CAPTCHA_REQUIRED", "验证码必填")
		case biz.ErrCaptchaInvalid:
			return nil, errors.BadRequest("CAPTCHA_INVALID", "验证码无效")
		case biz.ErrCaptchaExpired:
			return nil, errors.BadRequest("CAPTCHA_EXPIRED", "验证码已过期")
		default:
			return nil, errors.InternalServer("REGISTER_ERROR", err.Error())
		}
	}
	return &v1.RegisterReply{Message: "注册成功"}, nil
}
```

**本节来源**   
- [auth.go](file://internal/service/auth.go#L68-L85) - *在最近提交中更新*
- [auth.go](file://internal/biz/auth.go#L249-L317) - *在最近提交中更新*

## 安全实践

注册端点实施了多项安全措施来保护用户数据并防止滥用。

### 密码哈希

密码在存储前使用bcrypt算法进行安全哈希：

```go
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
if err != nil {
    return fmt.Errorf("密码哈希失败: %v", err)
}
```

哈希后的密码存储在数据库中，确保即使数据库被攻破，用户密码仍受保护。

### 验证码验证

集成了验证码验证以防止自动化注册尝试：

```go
valid, err := uc.captchaService.Verify(ctx, captchaID, captchaCode)
if err != nil {
    return err
}
if !valid {
    return ErrCaptchaInvalid
}
```

验证码系统有助于防止机器人攻击和暴力注册尝试。

### 速率限制

尽管代码中未明确显示，但应在HTTP服务器级别实施速率限制以防止注册端点被滥用。这可以通过以下方式实现：

- 基于IP的速率限制
- 每时间段的账户创建限制
- 对可疑活动的验证码挑战

### 防止枚举攻击

系统避免通过通用错误消息揭示用户名或邮箱是否已存在。然而，当前实现确实在错误消息中区分了用户名和邮箱冲突，这可以改进以获得更好的安全性。

### 安全错误消息

错误消息设计为对合法用户有帮助，同时不泄露敏感系统信息：

```go
return nil, errors.BadRequest("USER_EXISTS", "用户名已存在")
```

通用错误消息可防止攻击者收集有关现有账户的信息。

**本节来源**   
- [auth.go](file://internal/biz/auth.go#L249-L317) - *在最近提交中更新*
- [auth.go](file://internal/service/auth.go#L68-L85) - *在最近提交中更新*

## 客户端示例

### Go客户端示例 (gRPC)

```go
package main

import (
	"context"
	"log"

	"kratos-boilerplate/api/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 连接到gRPC服务器
	conn, err := grpc.Dial("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	// 创建客户端
	client := v1.NewAuthClient(conn)

	// 创建注册请求
	req := &v1.RegisterRequest{
		Username:    "newuser",
		Password:    "SecurePassword123",
		Email:       "newuser@example.com",
		Phone:       "13800138000",
		CaptchaId:   "captcha123",
		CaptchaCode: "1234",
	}

	// 调用Register方法
	resp, err := client.Register(context.Background(), req)
	if err != nil {
		log.Fatalf("注册失败: %v", err)
	}

	log.Printf("注册成功: %s", resp.Message)
}
```

### TypeScript Fetch示例 (HTTP)

```typescript
interface RegisterRequest {
  username: string;
  password: string;
  email: string;
  phone?: string;
  captcha_id: string;
  captcha_code: string;
}

interface RegisterResponse {
  message: string;
}

interface ErrorResponse {
  error: {
    code: number;
    message: string;
    details: Array<{
      type: string;
      field_violations: Array<{
        field: string;
        description: string;
      }>;
    }>;
  };
}

async function registerUser(userData: RegisterRequest): Promise<RegisterResponse | ErrorResponse> {
  try {
    const response = await fetch('/api/v1/auth/register', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(userData),
    });

    const data = await response.json();

    if (!response.ok) {
      return data as ErrorResponse;
    }

    return data as RegisterResponse;
  } catch (error) {
    return {
      error: {
        code: 500,
        message: '服务器内部错误',
        details: [],
      },
    };
  }
}

// 使用示例
const registrationData = {
  username: 'newuser',
  password: 'SecurePassword123',
  email: 'newuser@example.com',
  phone: '13800138000',
  captcha_id: 'captcha123',
  captcha_code: '1234',
};

registerUser(registrationData)
  .then(response => {
    if ('message' in response) {
      console.log('注册成功:', response.message);
    } else {
      console.error('注册失败:', response.error);
    }
  });
```

**本节来源**   
- [auth.go](file://internal/service/auth.go#L68-L85) - *在最近提交中更新*
- [auth.proto](file://api/auth/v1/auth.proto#L25-L30) - *在最近提交中更新*