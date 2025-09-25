# Logout

<cite>
**Referenced Files in This Document**   
- [auth.proto](file://api/auth/v1/auth.proto)
- [auth.go](file://internal/biz/auth.go)
- [auth.go](file://internal/service/auth.go)
- [auth.ts](file://frontend/src/api/auth.ts)
</cite>

## Table of Contents
1. [Logout](#logout)
2. [HTTP and gRPC Endpoints](#http-and-grpc-endpoints)
3. [Request Structure](#request-structure)
4. [Server-Side Logic](#server-side-logic)
5. [Response and Error Handling](#response-and-error-handling)
6. [Token Denylist with Redis TTL](#token-denylist-with-redis-ttl)
7. [Session Management Integration](#session-management-integration)
8. [Security Considerations](#security-considerations)
9. [Client Implementation Examples](#client-implementation-examples)

## HTTP and gRPC Endpoints

The Logout functionality is exposed through both HTTP and gRPC interfaces, allowing clients to securely terminate user sessions by invalidating the current access token and optionally associated refresh tokens.

**HTTP Endpoint**  
- **Method**: POST  
- **Path**: `/api/v1/auth/logout`  

**gRPC Method**  
- **Service**: `Auth`  
- **Method**: `Logout`  
- **Request Type**: `LogoutRequest`  
- **Response Type**: `LogoutReply`  

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

## Request Structure

The logout request does not require a request body. Instead, the current access token must be provided in the Authorization header using the Bearer scheme.

**Required Header**  
- **Authorization**: `Bearer <access_token>`

The access token is extracted from the header and validated before processing the logout operation. The optional refresh token is not required in the request, but if present, it will be invalidated server-side as part of the session cleanup.

```http
POST /api/v1/auth/logout HTTP/1.1
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

**Section sources**
- [auth.go](file://internal/service/auth.go#L121-L138)

## Server-Side Logic

The logout process is implemented across multiple layers: service, use case, and data. The flow begins in the `AuthService.Logout` method, which extracts the access token from the request metadata.

```go
func (s *AuthService) Logout(ctx context.Context, req *v1.LogoutRequest) (*v1.LogoutReply, error) {
	md, ok := metadata.FromServerContext(ctx)
	if !ok {
		return nil, errors.Unauthorized("UNAUTHORIZED", "unauthorized access")
	}

	authorization := md.Get("Authorization")
	if authorization == "" {
		return nil, errors.Unauthorized("TOKEN_MISSING", "access token missing")
	}

	if len(authorization) <= 7 || authorization[:7] != "Bearer " {
		return nil, errors.Unauthorized("INVALID_TOKEN_FORMAT", "invalid access token format")
	}

	token := authorization[7:]
	if err := s.uc.Logout(ctx, token); err != nil {
		// error handling
	}

	return &v1.LogoutReply{Success: true}, nil
}
```

The business logic is handled by `authUsecase.Logout`, which parses the access token to extract claims, particularly the expiration time and username. The token is then added to an in-memory denylist with a TTL matching its original expiration.

```go
func (uc *authUsecase) Logout(ctx context.Context, accessToken string) error {
	claims, err := uc.parseAccessToken(accessToken)
	if err != nil {
		return err
	}

	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return fmt.Errorf("invalid token expiration time")
	}
	exp := time.Unix(int64(expFloat), 0)
	uc.tokenBlacklist.Store(accessToken, exp)

	uc.cleanupTokenBlacklist()

	username := claims["username"].(string)
	if err := uc.repo.InvalidateAllRefreshTokens(ctx, username); err != nil {
		uc.log.Warnf("failed to invalidate all refresh tokens: %v", err)
	}

	return nil
}
```

**Section sources**
- [auth.go](file://internal/service/auth.go#L121-L156)
- [auth.go](file://internal/biz/auth.go#L389-L414)

## Response and Error Handling

Upon successful logout, the server returns a confirmation response indicating the operation was completed.

**Successful Response (JSON)**  
```json
{
  "success": true
}
```

**Protobuf Payload**  
```protobuf
message LogoutReply {
  bool success = 1;
}
```

**Error Cases**  
- **Missing Token**: HTTP 401, `TOKEN_MISSING`  
- **Invalid Format**: HTTP 401, `INVALID_TOKEN_FORMAT`  
- **Invalid Token**: HTTP 401, `TOKEN_INVALID`  
- **Expired Token**: HTTP 401, `TOKEN_EXPIRED`  
- **Internal Error**: HTTP 500, `LOGOUT_ERROR`  

Error responses follow the standard error format with `error_reason` and descriptive messages.

```go
switch err {
case biz.ErrTokenInvalid:
	return nil, errors.Unauthorized("TOKEN_INVALID", "access token is invalid")
case biz.ErrTokenExpired:
	return nil, errors.Unauthorized("TOKEN_EXPIRED", "access token has expired")
default:
	return nil, errors.InternalServer("LOGOUT_ERROR", err.Error())
}
```

**Section sources**
- [auth.go](file://internal/service/auth.go#L148-L156)
- [auth.proto](file://api/auth/v1/auth.proto#L75-L77)

## Token Denylist with Redis TTL

The system implements a token denylist using an in-memory `sync.Map` to track invalidated access tokens until their natural expiration. This prevents reuse of tokens in distributed environments and mitigates replay attacks.

Each token is stored with a key-value pair where:
- **Key**: Access token string  
- **Value**: Expiration time (`time.Time`)  

A background cleanup routine removes expired entries:

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

In production, this in-memory store should be replaced with Redis to ensure consistency across distributed instances, with the TTL set to match the original token expiration.

**Section sources**
- [auth.go](file://internal/biz/auth.go#L408-L414)

## Session Management Integration

Logout affects all concurrent sessions for the user by invalidating all associated refresh tokens via `InvalidateAllRefreshTokens`. This ensures that no new access tokens can be issued after logout, effectively terminating all active sessions.

```go
if err := uc.repo.InvalidateAllRefreshTokens(ctx, username); err != nil {
	uc.log.Warnf("failed to invalidate all refresh tokens: %v", err)
}
```

This integration with session management enhances security by preventing token reuse across devices or browser sessions, aligning with best practices for session termination.

**Section sources**
- [auth.go](file://internal/biz/auth.go#L412-L414)
- [auth.go](file://internal/data/auth.go#L424-L437)

## Security Considerations

The logout mechanism addresses several critical security concerns:

- **Token Revocation**: Immediate invalidation of access tokens via denylist prevents replay attacks.
- **Distributed Consistency**: Using Redis (recommended) ensures token state is synchronized across service instances.
- **Refresh Token Invalidation**: All refresh tokens for the user are invalidated, preventing silent re-authentication.
- **Timing Safety**: The denylist TTL matches the original token expiration, avoiding indefinite storage of revoked tokens.
- **Error Minimization**: Clear error codes help clients handle logout failures securely without exposing sensitive details.

Additionally, the use of JWT parsing with HMAC validation ensures only valid, unexpired tokens are processed during logout.

**Section sources**
- [auth.go](file://internal/biz/auth.go#L389-L414)
- [auth.go](file://internal/biz/auth.go#L345-L366)

## Client Implementation Examples

### Go (gRPC Interceptor Cleanup)

When using gRPC, interceptors can be used to automatically handle logout by clearing stored credentials.

```go
func LogoutClient(ctx context.Context, client AuthServiceClient, token string) error {
	md := metadata.Pairs("Authorization", "Bearer "+token)
	ctx = metadata.NewOutgoingContext(ctx, md)

	_, err := client.Logout(ctx, &LogoutRequest{})
	if err == nil {
		// Clear local token storage
		clearTokenStorage()
	}
	return err
}

func clearTokenStorage() {
	// Implement secure token removal
	accessToken = ""
	refreshToken = ""
}
```

### TypeScript (Frontend: Clear localStorage and Call API)

In the frontend, logout involves calling the API and clearing stored tokens from `localStorage`.

```typescript
// src/api/auth.ts
export const logout = () => {
    return request.post<ApiResponse<{ success: boolean }>>('/v1/auth/logout');
};

// Usage in component
const handleLogout = async () => {
    try {
        await logout();
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        window.location.href = '/login';
    } catch (error) {
        console.error('Logout failed:', error);
    }
};
```

The Axios interceptor automatically attaches the token and handles 401 responses by redirecting to login, ensuring a seamless user experience.

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