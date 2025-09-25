# Authentication Flow Security

<cite>
**Referenced Files in This Document**   
- [auth.go](file://internal/biz/auth.go#L0-L695)
- [auth.go](file://internal/data/auth.go#L0-L438)
- [auth.go](file://internal/service/auth.go#L0-L235)
- [auth.proto](file://api/auth/v1/auth.proto#L0-L156)
- [config.yaml](file://configs/config.yaml#L0-L38)
- [conf.proto](file://internal/conf/conf.proto#L0-L68)
</cite>

## Table of Contents
1. [Introduction](#introduction)
2. [Project Structure](#project-structure)
3. [Core Components](#core-components)
4. [Architecture Overview](#architecture-overview)
5. [Detailed Component Analysis](#detailed-component-analysis)
6. [Authentication Flow Security](#authentication-flow-security)
7. [Configuration and Security Policies](#configuration-and-security-policies)
8. [Vulnerability Prevention Mechanisms](#vulnerability-prevention-mechanisms)
9. [Token Management and Validation](#token-management-and-validation)
10. [Integration and Data Flow](#integration-and-data-flow)
11. [Best Practices for Production](#best-practices-for-production)
12. [Conclusion](#conclusion)

## Introduction
This document provides a comprehensive analysis of the authentication security mechanisms implemented in the kratos-boilerplate repository. The system employs a JWT-based authentication flow with robust security measures to protect user credentials and session state. The documentation details the complete authentication lifecycle including token generation, validation, expiration policies, and secure refresh token handling. It also examines how the system prevents common security vulnerabilities such as brute force attacks, session fixation, and token leakage. The integration between the authentication service, business logic, and data layer is thoroughly documented, along with configuration options that affect authentication security.

## Project Structure
The kratos-boilerplate project follows a clean architecture pattern with clear separation of concerns between layers. The authentication functionality is distributed across multiple directories, each serving a specific purpose in the overall system architecture.

```mermaid
graph TB
subgraph "API Layer"
A[auth.proto]
B[error_reason.proto]
end
subgraph "Service Layer"
C[auth.go]
D[provider.go]
end
subgraph "Business Logic Layer"
E[auth.go]
F[kms.go]
G[operation_log.go]
end
subgraph "Data Layer"
H[auth.go]
I[captcha.go]
J[kms.go]
end
subgraph "Configuration"
K[config.yaml]
L[conf.proto]
end
A --> C
C --> E
E --> H
K --> E
L --> K
style A fill:#f9f,stroke:#333
style C fill:#f9f,stroke:#333
style E fill:#f9f,stroke:#333
style H fill:#f9f,stroke:#333
style K fill:#bbf,stroke:#333
style L fill:#bbf,stroke:#333
```

**Diagram sources**
- [auth.proto](file://api/auth/v1/auth.proto#L0-L156)
- [auth.go](file://internal/service/auth.go#L0-L235)
- [auth.go](file://internal/biz/auth.go#L0-L695)
- [auth.go](file://internal/data/auth.go#L0-L438)
- [config.yaml](file://configs/config.yaml#L0-L38)
- [conf.proto](file://internal/conf/conf.proto#L0-L68)

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L0-L156)
- [auth.go](file://internal/service/auth.go#L0-L235)
- [auth.go](file://internal/biz/auth.go#L0-L695)
- [auth.go](file://internal/data/auth.go#L0-L438)
- [config.yaml](file://configs/config.yaml#L0-L38)

## Core Components
The authentication system in kratos-boilerplate consists of several core components that work together to provide secure user authentication and authorization. These components are organized according to the clean architecture pattern, with clear boundaries between layers.

The primary components include:
- **API Definitions**: Protocol buffer files that define the authentication service interface and message structures
- **Service Layer**: gRPC service implementations that handle incoming requests and coordinate with business logic
- **Business Logic Layer**: Use cases that implement authentication workflows and security policies
- **Data Layer**: Repository implementations that handle data persistence and retrieval
- **Configuration System**: YAML and protobuf configuration files that control authentication behavior

These components follow a dependency rule where higher-level layers depend on lower-level layers, but not vice versa, ensuring loose coupling and high cohesion.

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L0-L156)
- [auth.go](file://internal/service/auth.go#L0-L235)
- [auth.go](file://internal/biz/auth.go#L0-L695)
- [auth.go](file://internal/data/auth.go#L0-L438)

## Architecture Overview
The authentication architecture in kratos-boilerplate follows a layered approach with clear separation between presentation, business logic, and data access concerns. This design enables maintainability, testability, and security by isolating different aspects of the authentication system.

```mermaid
graph TD
Client[Client Application] --> |HTTP/gRPC| API[API Gateway]
API --> |Request| AuthService[AuthService]
AuthService --> |Business Logic| AuthUsecase[AuthUsecase]
AuthUsecase --> |Data Operations| UserRepo[UserRepo]
UserRepo --> |Database| DB[(PostgreSQL)]
UserRepo --> |Cache| Redis[(Redis)]
AuthUsecase --> |Captcha Service| CaptchaService[CaptchaService]
AuthUsecase --> |KMS| KMS[KMS Manager]
Config[config.yaml] --> |Configuration| AuthUsecase
Plugins[Plugins] --> |Enhancements| AuthService
style Client fill:#f96,stroke:#333
style API fill:#69f,stroke:#333
style AuthService fill:#6f9,stroke:#333
style AuthUsecase fill:#6f9,stroke:#333
style UserRepo fill:#6f9,stroke:#333
style DB fill:#9f6,stroke:#333
style Redis fill:#9f6,stroke:#333
style CaptchaService fill:#9f6,stroke:#333
style KMS fill:#9f6,stroke:#333
style Config fill:#bbf,stroke:#333
style Plugins fill:#f96,stroke:#333
```

**Diagram sources**
- [auth.go](file://internal/service/auth.go#L0-L235)
- [auth.go](file://internal/biz/auth.go#L0-L695)
- [auth.go](file://internal/data/auth.go#L0-L438)
- [config.yaml](file://configs/config.yaml#L0-L38)

## Detailed Component Analysis

### Authentication Service Analysis
The AuthService component serves as the entry point for all authentication-related operations. It implements the gRPC service interface defined in auth.proto and acts as a bridge between the API layer and the business logic layer.

```mermaid
classDiagram
class AuthService {
-uc : AuthUsecase
-log : *log.Helper
+GetCaptcha(ctx, req) : (*GetCaptchaReply, error)
+VerifyCaptcha(ctx, req) : (*VerifyCaptchaReply, error)
+Register(ctx, req) : (*RegisterReply, error)
+Login(ctx, req) : (*LoginReply, error)
+Logout(ctx, req) : (*LogoutReply, error)
+RefreshToken(ctx, req) : (*RefreshTokenReply, error)
+LockStatus(ctx, req) : (*LockStatusReply, error)
}
class AuthUsecase {
<<interface>>
+Register(ctx, username, password, email, phone, captchaID, captchaCode) : error
+Login(ctx, username, password, captchaID, captchaCode, totpCode) : (*TokenPair, error)
+Logout(ctx, accessToken) : error
+RefreshToken(ctx, refreshToken) : (*TokenPair, error)
+GetCaptcha(ctx, captchaType, target) : (*Captcha, error)
+VerifyCaptcha(ctx, captchaID, captchaCode) : (bool, error)
+GetLockStatus(ctx, username) : (*AccountLock, error)
+Now() : time.Time
+GetMaxLoginAttempts() : int32
}
class TokenPair {
+AccessToken : string
+RefreshToken : string
+ExpiresIn : int64
}
class Captcha {
+ID : string
+Code : string
+Type : string
+Target : string
+ExpireAt : time.Time
+Used : bool
}
class AccountLock {
+Username : string
+FailedAttempts : int32
+LockUntil : time.Time
+LastAttempt : time.Time
}
AuthService --> AuthUsecase : "depends on"
AuthUsecase <.. TokenPair : "returns"
AuthUsecase <.. Captcha : "returns"
AuthUsecase <.. AccountLock : "returns"
```

**Diagram sources**
- [auth.go](file://internal/service/auth.go#L0-L235)
- [auth.go](file://internal/biz/auth.go#L0-L695)

**Section sources**
- [auth.go](file://internal/service/auth.go#L0-L235)
- [auth.go](file://internal/biz/auth.go#L0-L695)

### Business Logic Analysis
The AuthUsecase implementation contains the core business logic for authentication operations. It orchestrates the various steps required for secure authentication while enforcing security policies and handling errors appropriately.

```mermaid
flowchart TD
Start([Login Request]) --> ValidateAccountLock["Check Account Lock Status"]
ValidateAccountLock --> AccountLocked{"Account Locked?"}
AccountLocked --> |Yes| ReturnLocked["Return Account Locked Error"]
AccountLocked --> |No| ValidateCaptcha["Validate Captcha"]
ValidateCaptcha --> CaptchaValid{"Captcha Valid?"}
CaptchaValid --> |No| ReturnCaptchaError["Return Captcha Error"]
CaptchaValid --> |Yes| GetUser["Get User from Repository"]
GetUser --> UserFound{"User Found?"}
UserFound --> |No| ReturnUserNotFound["Return User Not Found"]
UserFound --> |Yes| VerifyPassword["Verify Password"]
VerifyPassword --> PasswordCorrect{"Password Correct?"}
PasswordCorrect --> |No| RecordFailedAttempt["Record Failed Attempt"]
PasswordCorrect --> |Yes| ClearLock["Clear Account Lock"]
RecordFailedAttempt --> ReturnPasswordError["Return Password Incorrect"]
ClearLock --> GenerateTokens["Generate JWT Tokens"]
GenerateTokens --> ReturnTokens["Return Access and Refresh Tokens"]
style Start fill:#f9f,stroke:#333
style ReturnLocked fill:#f66,stroke:#333
style ReturnCaptchaError fill:#f66,stroke:#333
style ReturnUserNotFound fill:#f66,stroke:#333
style ReturnPasswordError fill:#f66,stroke:#333
style ReturnTokens fill:#6f9,stroke:#333
```

**Diagram sources**
- [auth.go](file://internal/biz/auth.go#L0-L695)

**Section sources**
- [auth.go](file://internal/biz/auth.go#L0-L695)

### Data Access Analysis
The UserRepo implementation handles all data persistence operations for the authentication system. It interacts with the database to store and retrieve user information, while also managing in-memory storage for temporary data like captchas and refresh tokens.

```mermaid
classDiagram
class UserRepo {
<<interface>>
+CreateUser(ctx, user) : error
+GetUser(ctx, username) : (*User, error)
+GetUserByEmail(ctx, email) : (*User, error)
+GetUserByPhone(ctx, phone) : (*User, error)
+GetUserByName(ctx, name) : (*User, error)
+UpdateUser(ctx, user) : error
+GetLock(ctx, username) : (*AccountLock, error)
+SaveLock(ctx, lock) : error
+RemoveLock(ctx, username) : error
+SaveRefreshToken(ctx, username, tokenID, expiresAt) : error
+GetRefreshToken(ctx, tokenID) : (string, bool, error)
+InvalidateRefreshToken(ctx, tokenID) : error
+InvalidateAllRefreshTokens(ctx, username) : error
+SaveCaptcha(ctx, captcha) : error
+GetCaptcha(ctx, captchaID) : (*Captcha, error)
+MarkCaptchaUsed(ctx, captchaID) : error
}
class userRepo {
-data : *Data
-log : *log.Helper
-enc : crypto.Encryptor
-kms : kms.KMSManager
-captchas : sync.Map
-accountLocks : sync.Map
-refreshTokens : sync.Map
+CreateUser(ctx, u) : error
+GetUser(ctx, username) : (*User, error)
+GetUserByEmail(ctx, email) : (*User, error)
+GetUserByPhone(ctx, phone) : (*User, error)
+GetUserByName(ctx, name) : (*User, error)
+UpdateUser(ctx, u) : error
+GetLock(ctx, username) : (*AccountLock, error)
+SaveLock(ctx, lock) : error
+RemoveLock(ctx, username) : error
+SaveRefreshToken(ctx, username, tokenID, expiresAt) : error
+GetRefreshToken(ctx, tokenID) : (string, bool, error)
+InvalidateRefreshToken(ctx, tokenID) : error
+InvalidateAllRefreshTokens(ctx, username) : error
+SaveCaptcha(ctx, captcha) : error
+GetCaptcha(ctx, captchaID) : (*Captcha, error)
+MarkCaptchaUsed(ctx, captchaID) : error
}
class User {
+ID : int64
+Username : string
+Password : string
+Email : string
+Phone : string
+Name : string
+TotpSecret : string
+CreatedAt : time.Time
+UpdatedAt : time.Time
}
UserRepo <|.. userRepo : "implementation"
userRepo --> User : "manipulates"
userRepo --> sync.Map : "uses"
userRepo --> crypto.Encryptor : "uses"
userRepo --> kms.KMSManager : "uses"
```

**Diagram sources**
- [auth.go](file://internal/data/auth.go#L0-L438)

**Section sources**
- [auth.go](file://internal/data/auth.go#L0-L438)

## Authentication Flow Security
The authentication flow in kratos-boilerplate is designed with security as a primary concern. The system implements a comprehensive set of security measures to protect user accounts and prevent unauthorized access.

### JWT-Based Authentication Flow
The system uses JSON Web Tokens (JWT) for stateless authentication, allowing clients to maintain authenticated sessions without server-side session storage. The authentication flow consists of several key steps:

1. **User Registration**: New users provide their credentials and contact information, which are validated and securely stored
2. **User Login**: Existing users authenticate with their credentials, triggering a multi-step verification process
3. **Token Generation**: Upon successful authentication, the system generates an access token and refresh token pair
4. **Token Usage**: Clients use the access token to authenticate subsequent requests
5. **Token Refresh**: When the access token expires, clients can use the refresh token to obtain a new token pair
6. **Logout**: Users can explicitly terminate their session, invalidating their tokens

```mermaid
sequenceDiagram
participant Client as "Client App"
participant AuthService as "AuthService"
participant AuthUsecase as "AuthUsecase"
participant UserRepo as "UserRepo"
participant DB as "Database"
Client->>AuthService : Login Request
AuthService->>AuthUsecase : Login(username, password, captcha)
AuthUsecase->>UserRepo : GetLock(username)
UserRepo->>DB : Query account lock status
DB-->>UserRepo : Lock status
UserRepo-->>AuthUsecase : AccountLock
AuthUsecase->>AuthUsecase : Check if account locked
AuthUsecase->>AuthUsecase : Validate captcha
AuthUsecase->>UserRepo : GetUser(username)
UserRepo->>DB : Query user data
DB-->>UserRepo : User data
UserRepo-->>AuthUsecase : User
AuthUsecase->>AuthUsecase : Verify password
AuthUsecase->>AuthUsecase : Generate JWT tokens
AuthUsecase->>UserRepo : SaveRefreshToken()
UserRepo->>DB : Store refresh token
DB-->>UserRepo : Success
UserRepo-->>AuthUsecase : Success
AuthUsecase-->>AuthService : TokenPair
AuthService-->>Client : Access and Refresh Tokens
Note over Client,DB : Secure authentication with multiple validation steps
```

**Diagram sources**
- [auth.go](file://internal/service/auth.go#L0-L235)
- [auth.go](file://internal/biz/auth.go#L0-L695)
- [auth.go](file://internal/data/auth.go#L0-L438)

**Section sources**
- [auth.go](file://internal/service/auth.go#L0-L235)
- [auth.go](file://internal/biz/auth.go#L0-L695)
- [auth.go](file://internal/data/auth.go#L0-L438)

### Token Generation and Signature Validation
The system generates JWT tokens using the HMAC-SHA256 algorithm for signature validation. Each token contains essential claims that enable secure authentication while preventing tampering.

**Access Token Structure:**
- **user_id**: User's unique identifier
- **username**: User's username
- **exp**: Expiration time (Unix timestamp)
- **iat**: Issued at time (Unix timestamp)
- **type**: Token type ("access")

**Refresh Token Structure:**
- **user_id**: User's unique identifier
- **username**: User's username
- **exp**: Expiration time (Unix timestamp)
- **iat**: Issued at time (Unix timestamp)
- **type**: Token type ("refresh")
- **jti**: JWT ID (unique identifier for the token)

The tokens are signed using a secret key configured in the system, ensuring that any modification to the token payload can be detected during validation.

```go
// Token generation example from auth.go
func (uc *authUsecase) generateTokens(ctx context.Context, user *User) (*TokenPair, error) {
    now := time.Now()

    // Generate access token
    accessExp := now.Add(uc.config.AccessTokenExpiration)
    accessClaims := jwt.MapClaims{
        "user_id":  user.ID,
        "username": user.Username,
        "exp":      accessExp.Unix(),
        "iat":      now.Unix(),
        "type":     "access",
    }
    accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
    signedAccessToken, err := accessToken.SignedString([]byte(uc.config.JWTSecretKey))
    
    // Generate refresh token
    refreshExp := now.Add(uc.config.RefreshTokenExpiration)
    tokenID := generateRandomString(32)
    refreshClaims := jwt.MapClaims{
        "user_id":  user.ID,
        "username": user.Username,
        "exp":      refreshExp.Unix(),
        "iat":      now.Unix(),
        "type":     "refresh",
        "jti":      tokenID,
    }
    refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
    signedRefreshToken, err := refreshToken.SignedString([]byte(uc.config.JWTSecretKey))
    
    // Store refresh token in repository
    if err := uc.repo.SaveRefreshToken(ctx, user.Username, tokenID, refreshExp); err != nil {
        return nil, fmt.Errorf("保存刷新令牌失败: %v", err)
    }

    return &TokenPair{
        AccessToken:  signedAccessToken,
        RefreshToken: signedRefreshToken,
        ExpiresIn:    int64(uc.config.AccessTokenExpiration.Seconds()),
    }, nil
}
```

**Section sources**
- [auth.go](file://internal/biz/auth.go#L0-L695)

### Token Expiration Policies
The system implements strict token expiration policies to minimize the window of opportunity for token misuse. Two types of tokens with different expiration characteristics are used:

**Access Tokens:**
- Short-lived with a default expiration of 15 minutes
- Used for authenticating API requests
- Stored in memory with a cleanup mechanism for expired entries
- Added to a blacklist upon logout to prevent reuse

**Refresh Tokens:**
- Long-lived with a default expiration of 7 days
- Used to obtain new access tokens when they expire
- Stored in the repository with a unique identifier (jti)
- Marked as used after being refreshed to prevent replay attacks
- Automatically invalidated when suspicious activity is detected

The expiration times are configurable through the system configuration, allowing administrators to adjust the security vs. usability trade-off based on their requirements.

**Section sources**
- [auth.go](file://internal/biz/auth.go#L0-L695)
- [config.yaml](file://configs/config.yaml#L0-L38)

### Secure Refresh Token Handling
The system implements several security measures to protect refresh tokens and prevent common attacks:

1. **Token Binding**: Each refresh token is bound to a specific user and contains a unique identifier (jti)
2. **One-Time Use**: Refresh tokens are marked as used after they are consumed, preventing replay attacks
3. **Revocation on Suspicious Activity**: If a refresh token is used multiple times, all tokens for that user are invalidated
4. **Secure Storage**: Refresh tokens are stored in the repository with their expiration time and usage status

```go
// Refresh token validation from auth.go
func (uc *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
    // Parse refresh token
    claims, err := uc.parseRefreshToken(refreshToken)
    if err != nil {
        return nil, err
    }

    tokenID := claims["jti"].(string)
    username := claims["username"].(string)

    // Check if token has been used
    _, used, err := uc.repo.GetRefreshToken(ctx, tokenID)
    if err != nil {
        return nil, fmt.Errorf("验证刷新令牌失败: %v", err)
    }

    if used {
        // Token reuse detected - possible token theft
        if err := uc.repo.InvalidateAllRefreshTokens(ctx, username); err != nil {
            uc.log.Warnf("令牌被重用，使所有令牌无效失败: %v", err)
        }
        return nil, ErrRefreshTokenReused
    }

    // Mark current refresh token as used
    if err := uc.repo.InvalidateRefreshToken(ctx, tokenID); err != nil {
        uc.log.Warnf("将刷新令牌标记为已使用失败: %v", err)
    }

    // Generate new token pair
    user, err := uc.repo.GetUser(ctx, username)
    if err != nil {
        return nil, fmt.Errorf("获取用户失败: %v", err)
    }

    tokenPair, err := uc.generateTokens(ctx, user)
    if err != nil {
        return nil, fmt.Errorf("生成新令牌失败: %v", err)
    }

    return tokenPair, nil
}
```

**Section sources**
- [auth.go](file://internal/biz/auth.go#L0-L695)

## Configuration and Security Policies
The authentication system's behavior is controlled by configuration settings that can be adjusted to meet different security requirements and operational needs.

### Configuration Options
The system's authentication security is configured through the config.yaml file and the Auth message in conf.proto. Key configuration options include:

**JWT Configuration:**
- **JWTSecretKey**: Secret key used for signing JWT tokens
- **AccessTokenExpiration**: Duration for which access tokens remain valid
- **RefreshTokenExpiration**: Duration for which refresh tokens remain valid

**Captcha Configuration:**
- **CaptchaEnabled**: Whether captcha verification is required
- **CaptchaExpiration**: Duration for which captcha codes remain valid

**Account Security Configuration:**
- **MaxLoginAttempts**: Maximum number of failed login attempts before account lockout
- **LockDuration**: Duration for which an account remains locked after too many failed attempts
- **TOTPEnabled**: Whether Time-based One-Time Password (TOTP) authentication is enabled

```yaml
# Example from config.yaml
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
```

**Section sources**
- [config.yaml](file://configs/config.yaml#L0-L38)
- [conf.proto](file://internal/conf/conf.proto#L0-L68)

### Default Security Configuration
The system defines a default authentication configuration that provides a reasonable balance between security and usability:

```go
// DefaultAuthConfig from auth.go
var DefaultAuthConfig = AuthConfig{
    JWTSecretKey:           "your-secret-key",
    AccessTokenExpiration:  15 * time.Minute,
    RefreshTokenExpiration: 7 * 24 * time.Hour,
    CaptchaEnabled:         true,
    CaptchaExpiration:      5 * time.Minute,
    MaxLoginAttempts:       5,
    LockDuration:           30 * time.Minute,
    TOTPEnabled:            false,
}
```

These defaults can be overridden by the application configuration, allowing administrators to customize the security settings based on their specific requirements.

**Section sources**
- [auth.go](file://internal/biz/auth.go#L0-L695)

## Vulnerability Prevention Mechanisms
The authentication system implements several mechanisms to prevent common security vulnerabilities and protect against various attack vectors.

### Brute Force Attack Prevention
The system protects against brute force attacks through account lockout mechanisms and captcha requirements:

1. **Failed Attempt Tracking**: Each failed login attempt is recorded in the AccountLock structure
2. **Account Lockout**: After a configurable number of failed attempts (default: 5), the account is locked
3. **Lock Duration**: Locked accounts remain inaccessible for a configurable duration (default: 30 minutes)
4. **Captcha Requirement**: Captcha verification is required for login attempts, making automated attacks more difficult

```go
// Failed attempt tracking from auth.go
func (uc *authUsecase) recordFailedAttempt(ctx context.Context, username string) {
    lock, err := uc.repo.GetLock(ctx, username)
    if err != nil {
        if err != ErrUserNotFound {
            uc.log.Errorf("获取账户锁定状态失败: %v", err)
            return
        }
        lock = &AccountLock{
            Username:       username,
            FailedAttempts: 0,
            LastAttempt:    time.Now(),
        }
    }

    now := time.Now()
    lock.FailedAttempts++
    lock.LastAttempt = now

    // Lock account if maximum attempts reached
    if lock.FailedAttempts >= uc.config.MaxLoginAttempts {
        lock.LockUntil = now.Add(uc.config.LockDuration)
    }

    if err := uc.repo.SaveLock(ctx, lock); err != nil {
        uc.log.Errorf("保存账户锁定状态失败: %v", err)
    }
}
```

**Section sources**
- [auth.go](file://internal/biz/auth.go#L0-L695)

### Session Fixation Prevention
The system prevents session fixation attacks through several mechanisms:

1. **Stateless JWT Tokens**: The system uses stateless JWT tokens rather than server-side sessions, eliminating the session identifier that could be fixed
2. **Token Rotation**: Each successful authentication generates new tokens, invalidating any previous tokens
3. **Token Blacklisting**: When a user logs out, their access token is added to a blacklist to prevent reuse
4. **Refresh Token Invalidation**: All refresh tokens are invalidated when suspicious activity is detected

### Token Leakage Prevention
The system implements several measures to prevent token leakage and mitigate the impact if tokens are compromised:

1. **Short-Lived Access Tokens**: Access tokens have a short expiration time (15 minutes by default), limiting the window of opportunity for misuse
2. **Token Blacklist**: Logged-out tokens are added to a blacklist and rejected if presented
3. **Refresh Token Revocation**: Suspicious refresh token usage triggers revocation of all tokens for that user
4. **Secure Transmission**: Tokens should be transmitted over HTTPS in production environments

## Token Management and Validation
The system implements robust token management and validation processes to ensure the integrity and security of the authentication system.

### Token Validation Process
When a token is presented for authentication, the system performs the following validation steps:

1. **Format Validation**: Check that the token is in the correct JWT format
2. **Signature Verification**: Verify the token's signature using the configured secret key
3. **Expiration Check**: Ensure the token has not expired
4. **Token Type Verification**: Confirm the token is of the expected type (access or refresh)
5. **Blacklist Check**: For access tokens, check if the token has been blacklisted (logged out)
6. **Usage Status Check**: For refresh tokens, verify the token has not been used

```go
// Access token parsing from auth.go
func (uc *authUsecase) parseAccessToken(tokenStr string) (jwt.MapClaims, error) {
    // Check if token is in blacklist
    if _, ok := uc.tokenBlacklist.Load(tokenStr); ok {
        return nil, ErrTokenInvalid
    }

    token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(uc.config.JWTSecretKey), nil
    })

    if err != nil {
        if ve, ok := err.(*jwt.ValidationError); ok {
            if ve.Errors&jwt.ValidationErrorExpired != 0 {
                return nil, ErrTokenExpired
            }
        }
        return nil, ErrTokenInvalid
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid || claims["type"] != "access" {
        return nil, ErrTokenInvalid
    }

    return claims, nil
}
```

**Section sources**
- [auth.go](file://internal/biz/auth.go#L0-L695)

### Token Blacklist Management
The system maintains an in-memory blacklist of invalidated access tokens to prevent their reuse after logout:

1. **Blacklist Storage**: Uses a sync.Map to store blacklisted tokens with their expiration times
2. **Automatic Cleanup**: Periodically removes expired entries from the blacklist
3. **Time-Based Expiration**: Tokens remain in the blacklist only until their natural expiration time

```go
// Token blacklist cleanup from auth.go
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

**Section sources**
- [auth.go](file://internal/biz/auth.go#L0-L695)

## Integration and Data Flow
The authentication system integrates with various components to provide a complete security solution.

### Service Integration
The AuthService integrates with the business logic layer through dependency injection, receiving an AuthUsecase instance that implements the required interface:

```go
// Service creation from auth.go
func NewAuthService(uc biz.AuthUsecase, logger log.Logger) *AuthService {
    return &AuthService{
        uc:  uc,
        log: log.NewHelper(logger),
    }
}
```

This design allows for easy testing and replacement of the business logic implementation without affecting the service layer.

**Section sources**
- [auth.go](file://internal/service/auth.go#L0-L235)

### Data Layer Integration
The UserRepo implementation integrates with the database to persist user data and authentication state:

1. **User Data Encryption**: Sensitive user information (email, phone, name) is encrypted before storage
2. **Hash Indexing**: Hashes of sensitive data are stored to enable lookup without decrypting
3. **In-Memory Storage**: Temporary data (captchas, locks, refresh tokens) is stored in memory using sync.Map

```go
// User creation with encryption from auth.go
func (r *userRepo) CreateUser(ctx context.Context, u *biz.User) error {
    // Encrypt sensitive information
    emailEnc, err := r.enc.Encrypt([]byte(u.Email))
    if err != nil {
        return err
    }
    emailHash := r.enc.Hash([]byte(u.Email))

    phoneEnc, err := r.enc.Encrypt([]byte(u.Phone))
    if err != nil {
        return err
    }
    phoneHash := r.enc.Hash([]byte(u.Phone))

    nameEnc, err := r.enc.Encrypt([]byte(u.Name))
    if err != nil {
        return err
    }
    nameHash := r.enc.Hash([]byte(u.Name))
    
    // Insert into database
    query := `
        INSERT INTO users (
            username, password, 
            email_encrypted, email_hash,
            phone_encrypted, phone_hash,
            name_encrypted, name_hash,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        RETURNING id
    `
    err = r.data.db.QueryRowContext(ctx, query,
        u.Username, u.Password,
        emailEnc, emailHash,
        phoneEnc, phoneHash,
        nameEnc, nameHash,
        time.Now(), time.Now(),
    ).Scan(&u.ID)
    return err
}
```

**Section sources**
- [auth.go](file://internal/data/auth.go#L0-L438)

## Best Practices for Production
To ensure the highest level of security in production deployments, the following best practices should be followed:

### HTTPS Enforcement
All authentication endpoints should be served over HTTPS to prevent token interception and man-in-the-middle attacks. This can be achieved through:

1. **TLS Termination**: Configure a reverse proxy or load balancer to handle TLS termination
2. **HSTS Headers**: Implement HTTP Strict Transport Security to ensure browsers always use HTTPS
3. **Certificate Management**: Use valid certificates from trusted certificate authorities

### Secure Cookie Settings
When using cookies to store tokens (if applicable), the following settings should be used:

1. **Secure Flag**: Ensure cookies are only transmitted over HTTPS
2. **HttpOnly Flag**: Prevent client-side JavaScript access to cookies
3. **SameSite Attribute**: Set to "Strict" or "Lax" to prevent CSRF attacks

### CORS Policies
Implement strict CORS policies to prevent unauthorized cross-origin requests:

1. **Whitelist Origins**: Only allow requests from trusted domains
2. **Limit Methods**: Restrict allowed HTTP methods to only those required
3. **Credentials Handling**: Carefully manage when credentials are allowed in cross-origin requests

### Security Monitoring
Implement comprehensive monitoring to detect and respond to security incidents:

1. **Audit Logging**: Log all authentication attempts, including successes and failures
2. **Anomaly Detection**: Monitor for unusual patterns of activity that may indicate attacks
3. **Alerting**: Set up alerts for suspicious activities such as multiple failed login attempts

## Conclusion
The authentication system in kratos-boilerplate provides a comprehensive and secure solution for user authentication and authorization. By implementing JWT-based authentication with robust security measures, the system protects against common vulnerabilities while providing a flexible and maintainable architecture.

Key security features include:
- JWT-based stateless authentication with short-lived access tokens
- Secure refresh token handling with one-time use and revocation mechanisms
- Protection against brute force attacks through account lockout and captcha requirements
- Prevention of session fixation through token rotation and blacklisting
- Secure storage of sensitive user data with encryption and hashing

The system's modular design allows for easy customization and extension, while the comprehensive configuration options enable administrators to adjust security settings based on their specific requirements. By following the best practices outlined in this document, organizations can deploy the system with confidence in its ability to protect user accounts and sensitive data.