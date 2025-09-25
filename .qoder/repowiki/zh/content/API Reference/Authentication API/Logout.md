# 退出登录

<cite>
**本文档引用的文件**   
- [auth.proto](file://api/auth/v1/auth.proto) - *在最近提交中更新*
- [auth.go](file://internal/biz/auth.go) - *在基础模块功能中实现*
- [auth.go](file://internal/service/auth.go) - *在服务层实现登出逻辑*
- [auth.ts](file://frontend/src/api/auth.ts) - *前端API调用实现*
</cite>

## 更新摘要
**已做更改**   
- 根据最新代码变更，全面更新了登出端点的文档内容
- 增加了对Redis令牌吊销列表的说明
- 更新了服务器端逻辑和错误处理部分以反映实际实现
- 补充了完整的客户端实现示例
- 所有技术术语和用户界面文本均已转换为中文

## 目录
1. [退出登录](#退出登录)
2. [HTTP和gRPC端点](#http和grpc端点)
3. [请求结构](#请求结构)
4. [服务器端逻辑](#服务器端逻辑)
5. [响应和错误处理](#响应和错误处理)
6. [令牌吊销列表与Redis TTL](#令牌吊销列表与redis-ttl)
7. [会话管理集成](#会话管理集成)
8. [安全考虑](#安全考虑)
9. [客户端实现示例](#客户端实现示例)

## HTTP和gRPC端点

退出登录功能通过HTTP和gRPC接口提供，允许客户端通过使当前访问令牌失效来安全终止用户会话。

**HTTP端点**  
- **方法**: POST  
- **路径**: `/api/v1/auth/logout`  

**gRPC方法**  
- **服务**: `Auth`  
- **方法**: `Logout`  
- **请求类型**: `LogoutRequest`  
- **响应类型**: `LogoutReply`  

```protobuf
rpc Logout(LogoutRequest) returns (LogoutReply) {
  option (google.api.http) = {
    post: "/api/v1/auth/logout"
    body: "*"
  };
}
```

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L35-L40)

## 请求结构

登出请求不需要请求体。相反，必须在使用Bearer方案的Authorization头中提供当前访问令牌。

**必需头信息**  
- **Authorization**: `Bearer <access_token>`

从头信息中提取访问令牌并在处理登出操作之前进行验证。可选的刷新令牌不需要在请求中，但如果存在，它将在服务器端作为会话清理的一部分被使无效。

```http
POST /api/v1/auth/logout HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

**Section sources**
- [auth.go](file://internal/service/auth.go#L121-L138)

## 服务器端逻辑

登出过程在多个层次上实现：服务、用例和数据。流程从`AuthService.Logout`方法开始，该方法从请求元数据中提取访问令牌。

```go
func (s *AuthService) Logout(ctx context.Context, req *v1.LogoutRequest) (*v1.LogoutReply, error) {
	// 从请求头中获取访问令牌
	md, ok := metadata.FromServerContext(ctx)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "未授权访问")
	}

	authorization := md.Get("Authorization")
	if authorization == "" {
		return nil, errors.Unauthorized("TOKEN_MISSING", "缺少访问令牌")
	}

	// 检查授权头格式
	if len(authorization) <= 7 || authorization[:7] != "Bearer " {
		return nil, errors.Unauthorized("INVALID_TOKEN_FORMAT", "访问令牌格式错误")
	}

	// 提取令牌
	token := authorization[7:]

	// 调用业务逻辑执行退出
	if err := s.uc.Logout(ctx, token); err != nil {
		switch err {
		case biz.ErrTokenInvalid:
			return nil, errors.Unauthorized("TOKEN_INVALID", "访问令牌无效")
		case biz.ErrTokenExpired:
			return nil, errors.Unauthorized("TOKEN_EXPIRED", "访问令牌已过期")
		default:
			return nil, errors.InternalServer("LOGOUT_ERROR", err.Error())
		}
	}

	return &v1.LogoutReply{
		Success: true,
	}, nil
}
```

业务逻辑由`authUsecase.Logout`处理，该逻辑解析访问令牌以提取声明，特别是过期时间和用户名。然后将令牌添加到具有TTL（匹配其原始过期时间）的内存吊销列表中。

```go
func (uc *authUsecase) Logout(ctx context.Context, accessToken string) error {
	// 将访问令牌加入黑名单
	claims, err := uc.parseAccessToken(accessToken)
	if err != nil {
		return err
	}

	// 获取过期时间，将令牌加入黑名单直到过期
	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return fmt.Errorf("无效的令牌过期时间")
	}
	exp := time.Unix(int64(expFloat), 0)
	uc.tokenBlacklist.Store(accessToken, exp)

	// 清理令牌黑名单中已过期的条目
	uc.cleanupTokenBlacklist()

	username := claims["username"].(string)
	// 可选：使所有刷新令牌无效
	if err := uc.repo.InvalidateAllRefreshTokens(ctx, username); err != nil {
		uc.log.Warnf("使所有刷新令牌无效失败: %v", err)
	}

	return nil
}
```

**Section sources**
- [auth.go](file://internal/service/auth.go#L121-L156)
- [auth.go](file://internal/biz/auth.go#L389-L414)

## 响应和错误处理

成功登出后，服务器返回确认响应，表示操作已完成。

**成功响应 (JSON)**  
```json
{
  "success": true
}
```

**Protobuf负载**  
```protobuf
message LogoutReply {
  bool success = 1;
}
```

**错误情况**  
- **缺少令牌**: HTTP 401, `TOKEN_MISSING`  
- **格式无效**: HTTP 401, `INVALID_TOKEN_FORMAT`  
- **令牌无效**: HTTP 401, `TOKEN_INVALID`  
- **令牌过期**: HTTP 401, `TOKEN_EXPIRED`  
- **内部错误**: HTTP 500, `LOGOUT_ERROR`  

错误响应遵循带有`error_reason`和描述性消息的标准错误格式。

```go
switch err {
case biz.ErrTokenInvalid:
	return nil, errors.Unauthorized("TOKEN_INVALID", "访问令牌无效")
case biz.ErrTokenExpired:
	return nil, errors.Unauthorized("TOKEN_EXPIRED", "访问令牌已过期")
default:
	return nil, errors.InternalServer("LOGOUT_ERROR", err.Error())
}
```

**Section sources**
- [auth.go](file://internal/service/auth.go#L148-L156)
- [auth.proto](file://api/auth/v1/auth.proto#L75-L77)

## 令牌吊销列表与Redis TTL

系统使用内存中的`sync.Map`实现令牌吊销列表，以跟踪已撤销的访问令牌直到其自然过期。这可以防止在分布式环境中重用令牌，并减轻重放攻击。

每个令牌都存储为键值对，其中：
- **键**: 访问令牌字符串  
- **值**: 过期时间 (`time.Time`)  

后台清理例程会移除已过期的条目：

```go
func (uc *authUsecase) cleanupTokenBlacklist() {
	now := time.Now()
	uc.tokenBlacklist.Range(func(key, value interface{}) bool {
		exp := value.(time.Time)
		if exp.Before(now) {
			uc.tokenBlacklist.Delete(key)
		}
		return true
	})
}
```

在生产环境中，此内存存储应替换为Redis，以确保跨分布式实例的一致性，TTL设置为匹配原始令牌过期时间。

**Section sources**
- [auth.go](file://internal/biz/auth.go#L408-L414)

## 会话管理集成

登出会通过`InvalidateAllRefreshTokens`使用户的所有并发会话失效，从而影响所有并发会话。这确保了登出后无法再颁发新的访问令牌，有效终止所有活动会话。

```go
if err := uc.repo.InvalidateAllRefreshTokens(ctx, username); err != nil {
	uc.log.Warnf("使所有刷新令牌无效失败: %v", err)
}
```

这种与会话管理的集成通过防止跨设备或浏览器会话的令牌重用来增强安全性，符合会话终止的最佳实践。

**Section sources**
- [auth.go](file://internal/biz/auth.go#L412-L414)
- [auth.go](file://internal/data/auth.go#L424-L437)

## 安全考虑

登出机制解决了几个关键的安全问题：

- **令牌吊销**: 通过吊销列表立即使访问令牌失效，防止重放攻击。
- **分布式一致性**: 使用Redis（推荐）确保令牌状态在服务实例之间同步。
- **刷新令牌失效**: 用户的所有刷新令牌都被使无效，防止静默重新认证。
- **时间安全性**: 吊销列表TTL与原始令牌过期时间匹配，避免无限期存储已撤销的令牌。
- **错误最小化**: 清晰的错误代码帮助客户端安全地处理登出失败，而不会暴露敏感细节。

此外，使用HMAC验证的JWT解析确保仅处理有效且未过期的令牌。

**Section sources**
- [auth.go](file://internal/biz/auth.go#L389-L414)
- [auth.go](file://internal/biz/auth.go#L345-L366)

## 客户端实现示例

### Go (gRPC拦截器清理)

使用gRPC时，可以使用拦截器通过清除存储的凭据自动处理登出。

```go
func LogoutClient(ctx context.Context, client AuthServiceClient, token string) error {
	md := metadata.Pairs("Authorization", "Bearer "+token)
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := client.Logout(ctx, &LogoutRequest{})
	if err == nil {
		// 清除本地令牌存储
		clearTokenStorage()
	}
	return err
}

func clearTokenStorage() {
	// 实现安全的令牌移除
	accessToken = ""
	refreshToken = ""
}
```

### TypeScript (前端：清除localStorage并调用API)

在前端，登出涉及调用API并从`localStorage`中清除存储的令牌。

```typescript
// src/api/auth.ts
export const logout = () => {
    return request.post<ApiResponse<{ success: boolean }>>('/v1/auth/logout');
};

// 组件中的用法
const handleLogout = async () => {
    try {
        await logout();
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        window.location.href = '/login';
    } catch (error) {
        console.error('登出失败:', error);
    }
};
```

Axios拦截器自动附加令牌并处理401响应，通过重定向到登录页面确保无缝的用户体验。

```typescript
request.interceptors.response.use(
    (response) => response.data,
    (error) => {
        if (error.response?.status === 401) {
            localStorage.removeItem('access_token');
            localStorage.removeItem('refresh_token');
            window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);
```

**Section sources**
- [auth.ts](file://frontend/src/api/auth.ts#L94-L96)
- [auth.ts](file://frontend/src/api/auth.ts#L20-L34)