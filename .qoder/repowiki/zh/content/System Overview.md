# System Overview

<cite>
**Referenced Files in This Document**   
- [main.go](file://cmd/kratos-boilerplate/main.go)
- [auth.go](file://internal/biz/auth.go)
- [auth.go](file://internal/data/auth.go)
- [auth.go](file://internal/service/auth.go)
- [auth.proto](file://api/auth/v1/auth.proto)
- [auth.ts](file://frontend/src/api/auth.ts)
- [crypto_service.go](file://internal/pkg/kms/crypto_service.go)
- [manager.go](file://internal/pkg/kms/manager.go)
</cite>

## Table of Contents
1. [System Overview](#system-overview)
2. [Core Components](#core-components)
3. [Architecture Overview](#architecture-overview)
4. [Detailed Component Analysis](#detailed-component-analysis)
5. [Authentication Flow](#authentication-flow)
6. [Security and Data Protection](#security-and-data-protection)
7. [Frontend-Backend Integration](#frontend-backend-integration)

## Core Components

The kratos-boilerplate system is a production-ready microservice template built with Go and the Kratos framework, featuring comprehensive authentication, security, and extensibility capabilities. The system follows a clean architecture pattern with clear separation between layers.

The core components include:
- **API Layer**: Defined in Protocol Buffer files, providing gRPC and HTTP interfaces
- **Service Layer**: Implements business logic and coordinates between use cases and data access
- **Business Logic (Usecase)**: Contains the core business rules and workflows
- **Data Access Layer**: Handles database operations and data persistence
- **Frontend Application**: Vue.js-based user interface for interacting with the backend

The system leverages the Kratos framework for service creation, dependency injection via Wire, and standardized project structure. It includes built-in support for authentication, authorization, and secure data handling.

**Section sources**
- [main.go](file://cmd/kratos-boilerplate/main.go#L1-L92)
- [auth.go](file://internal/biz/auth.go#L1-L695)

## Architecture Overview

The kratos-boilerplate follows a microservices architecture with a clear separation between frontend and backend components. The system is designed to be modular, extensible, and production-ready with comprehensive security features.

```mermaid
graph TB
subgraph "Frontend"
UI[Vue.js Application]
Router[Vue Router]
Store[Pinia Store]
end
subgraph "Backend"
API[HTTP/gRPC API]
Service[Service Layer]
Biz[Business Logic]
Data[Data Access]
DB[(PostgreSQL)]
Redis[(Redis)]
end
UI --> API
API --> Service
Service --> Biz
Biz --> Data
Data --> DB
Data --> Redis
subgraph "Security"
KMS[KMS System]
Crypto[Crypto Service]
Sensitive[Data Anonymization]
end
Biz --> KMS
KMS --> Crypto
Biz --> Sensitive
subgraph "Plugins"
Audit[Audit Logger Plugin]
AuthEnhancer[Auth Enhancer Plugin]
end
Service --> Audit
Service --> AuthEnhancer
```

**Diagram sources**
- [main.go](file://cmd/kratos-boilerplate/main.go#L1-L92)
- [auth.go](file://internal/biz/auth.go#L1-L695)
- [auth.go](file://internal/data/auth.go#L1-L438)
- [auth.go](file://internal/service/auth.go#L1-L235)

## Detailed Component Analysis

### Authentication Business Logic

The authentication business logic is implemented in the `authUsecase` struct, which follows the use case pattern to encapsulate business rules. The implementation provides comprehensive user management functionality including registration, login, logout, and token refresh operations.

```mermaid
classDiagram
class AuthUsecase {
<<interface>>
+Register(ctx, username, password, email, phone, captchaID, captchaCode) error
+Login(ctx, username, password, captchaID, captchaCode, totpCode) (*TokenPair, error)
+Logout(ctx, accessToken) error
+RefreshToken(ctx, refreshToken) (*TokenPair, error)
+GetCaptcha(ctx, captchaType, target) (*Captcha, error)
+VerifyCaptcha(ctx, captchaID, captchaCode) (bool, error)
+GetLockStatus(ctx, username) (*AccountLock, error)
}
class authUsecase {
-repo UserRepo
-captchaService CaptchaService
-config AuthConfig
-log *log.Helper
-tokenBlacklist sync.Map
+Register(ctx, username, password, email, phone, captchaID, captchaCode) error
+Login(ctx, username, password, captchaID, captchaCode, totpCode) (*TokenPair, error)
+Logout(ctx, accessToken) error
+RefreshToken(ctx, refreshToken) (*TokenPair, error)
+GetCaptcha(ctx, captchaType, target) (*Captcha, error)
+VerifyCaptcha(ctx, captchaID, captchaCode) (bool, error)
+GetLockStatus(ctx, username) (*AccountLock, error)
+generateTokens(ctx, user *User) (*TokenPair, error)
+parseAccessToken(tokenStr string) (jwt.MapClaims, error)
+parseRefreshToken(tokenStr string) (jwt.MapClaims, error)
+recordFailedAttempt(ctx, username string)
+cleanupTokenBlacklist()
+verifyTOTP(secret, code string) bool
}
class UserRepo {
<<interface>>
+CreateUser(ctx, user *User) error
+GetUser(ctx, username string) (*User, error)
+GetUserByEmail(ctx, email string) (*User, error)
+GetUserByPhone(ctx, phone string) (*User, error)
+UpdateUser(ctx, user *User) error
+GetLock(ctx, username string) (*AccountLock, error)
+SaveLock(ctx, lock *AccountLock) error
+RemoveLock(ctx, username string) error
+SaveRefreshToken(ctx, username, tokenID string, expiresAt time.Time) error
+GetRefreshToken(ctx, tokenID string) (string, bool, error)
+InvalidateRefreshToken(ctx, tokenID string) error
+InvalidateAllRefreshTokens(ctx, username string) error
+SaveCaptcha(ctx, captcha *Captcha) error
+GetCaptcha(ctx, captchaID string) (*Captcha, error)
+MarkCaptchaUsed(ctx, captchaID string) error
}
class CaptchaService {
<<interface>>
+Generate(ctx, captchaType, target string) (*Captcha, error)
+Verify(ctx, captchaID, captchaCode string) (bool, error)
}
AuthUsecase <|.. authUsecase : implements
authUsecase --> UserRepo : "uses"
authUsecase --> CaptchaService : "uses"
```

**Diagram sources**
- [auth.go](file://internal/biz/auth.go#L1-L695)

**Section sources**
- [auth.go](file://internal/biz/auth.go#L1-L695)

### Data Access Implementation

The data access layer implements the UserRepo interface with a concrete implementation that handles database operations for user management. The implementation includes encryption and hashing of sensitive data using the KMS system.

```mermaid
classDiagram
class userRepo {
-data *Data
-log *log.Helper
-enc crypto.Encryptor
-kms kms.KMSManager
-captchas sync.Map
-accountLocks sync.Map
-refreshTokens sync.Map
+CreateUser(ctx, u *biz.User) error
+GetUser(ctx, username string) (*biz.User, error)
+GetUserByEmail(ctx, email string) (*biz.User, error)
+GetUserByPhone(ctx, phone string) (*biz.User, error)
+GetUserByName(ctx, name string) (*biz.User, error)
+UpdateUser(ctx, u *biz.User) error
+SaveCaptcha(ctx, captcha *biz.Captcha) error
+GetCaptcha(ctx, captchaID string) (*biz.Captcha, error)
+MarkCaptchaUsed(ctx, captchaID string) error
+GetLock(ctx, username string) (*biz.AccountLock, error)
+SaveLock(ctx, lock *biz.AccountLock) error
+RemoveLock(ctx, username string) error
+SaveRefreshToken(ctx, username, tokenID string, expiresAt time.Time) error
+GetRefreshToken(ctx, tokenID string) (string, bool, error)
+InvalidateRefreshToken(ctx, tokenID string) error
+InvalidateAllRefreshTokens(ctx, username string) error
}
class kmsEncryptorWrapper {
-cryptoService kms.CryptoService
+Encrypt(data []byte) ([]byte, error)
+Decrypt(data []byte) ([]byte, error)
+Hash(data []byte) string
}
class Data {
-db *sql.DB
-conn *gorm.DB
}
userRepo --> Data : "uses"
userRepo --> kmsEncryptorWrapper : "uses"
userRepo --> biz.User : "persists"
userRepo --> biz.Captcha : "stores"
userRepo --> biz.AccountLock : "manages"
userRepo --> biz.EncryptedField : "handles"
```

**Diagram sources**
- [auth.go](file://internal/data/auth.go#L1-L438)

**Section sources**
- [auth.go](file://internal/data/auth.go#L1-L438)

### Service Layer Implementation

The service layer exposes the business logic through gRPC and HTTP endpoints. The AuthService implements the AuthServer interface defined in the protobuf specification, translating API requests into business logic calls.

```mermaid
classDiagram
class AuthService {
-uc biz.AuthUsecase
-log *log.Helper
+GetCaptcha(ctx, req *v1.GetCaptchaRequest) (*v1.GetCaptchaReply, error)
+VerifyCaptcha(ctx, req *v1.VerifyCaptchaRequest) (*v1.VerifyCaptchaReply, error)
+Register(ctx, req *v1.RegisterRequest) (*v1.RegisterReply, error)
+Login(ctx, req *v1.LoginRequest) (*v1.LoginReply, error)
+Logout(ctx, req *v1.LogoutRequest) (*v1.LogoutReply, error)
+RefreshToken(ctx, req *v1.RefreshTokenRequest) (*v1.RefreshTokenReply, error)
+LockStatus(ctx, req *v1.LockStatusRequest) (*v1.LockStatusReply, error)
}
class AuthServer {
<<interface>>
+GetCaptcha(context.Context, *GetCaptchaRequest) (*GetCaptchaReply, error)
+VerifyCaptcha(context.Context, *VerifyCaptchaRequest) (*VerifyCaptchaReply, error)
+Register(context.Context, *RegisterRequest) (*RegisterReply, error)
+Login(context.Context, *LoginRequest) (*LoginReply, error)
+Logout(context.Context, *LogoutRequest) (*LogoutReply, error)
+RefreshToken(context.Context, *RefreshTokenRequest) (*RefreshTokenReply, error)
+LockStatus(context.Context, *LockStatusRequest) (*LockStatusReply, error)
}
class AuthUsecase {
<<interface>>
+Register(ctx, username, password, email, phone, captchaID, captchaCode) error
+Login(ctx, username, password, captchaID, captchaCode, totpCode) (*TokenPair, error)
+Logout(ctx, accessToken) error
+RefreshToken(ctx, refreshToken) (*TokenPair, error)
+GetCaptcha(ctx, captchaType, target) (*Captcha, error)
+VerifyCaptcha(ctx, captchaID, captchaCode) (bool, error)
+GetLockStatus(ctx, username) (*AccountLock, error)
}
AuthServer <|.. AuthService : implements
AuthService --> AuthUsecase : "delegates"
AuthService --> v1.GetCaptchaRequest : "consumes"
AuthService --> v1.GetCaptchaReply : "produces"
AuthService --> v1.RegisterRequest : "consumes"
AuthService --> v1.RegisterReply : "produces"
AuthService --> v1.LoginRequest : "consumes"
AuthService --> v1.LoginReply : "produces"
```

**Diagram sources**
- [auth.go](file://internal/service/auth.go#L1-L235)

**Section sources**
- [auth.go](file://internal/service/auth.go#L1-L235)

## Authentication Flow

The authentication flow in kratos-boilerplate follows a secure, multi-step process that includes captcha verification, credential validation, and JWT token generation. The system implements account locking after multiple failed attempts to prevent brute force attacks.

```mermaid
sequenceDiagram
participant Client as "Client App"
participant AuthService as "AuthService"
participant AuthUsecase as "AuthUsecase"
participant UserRepo as "UserRepo"
participant CaptchaService as "CaptchaService"
participant KMS as "KMS System"
Client->>AuthService : Login Request (username, password, captcha)
AuthService->>AuthUsecase : Login(username, password, captchaID, captchaCode)
AuthUsecase->>CaptchaService : Verify(captchaID, captchaCode)
CaptchaService-->>AuthUsecase : Validation Result
alt Captcha Invalid
AuthUsecase-->>AuthService : Error (Captcha Invalid)
AuthService-->>Client : 400 Bad Request
else Captcha Valid
AuthUsecase->>UserRepo : GetLock(username)
UserRepo-->>AuthUsecase : Lock Status
alt Account Locked
AuthUsecase-->>AuthService : Error (Account Locked)
AuthService-->>Client : 403 Forbidden
else Account Not Locked
AuthUsecase->>UserRepo : GetUser(username)
UserRepo-->>AuthUsecase : User Data
AuthUsecase->>AuthUsecase : Verify Password
alt Password Incorrect
AuthUsecase->>AuthUsecase : recordFailedAttempt()
AuthUsecase-->>AuthService : Error (Password Incorrect)
AuthService-->>Client : 401 Unauthorized
else Password Correct
AuthUsecase->>AuthUsecase : generateTokens(user)
AuthUsecase->>KMS : Encrypt Sensitive Data
KMS-->>AuthUsecase : Encrypted Data
AuthUsecase->>UserRepo : SaveRefreshToken()
UserRepo-->>AuthUsecase : Success
AuthUsecase-->>AuthService : TokenPair
AuthService-->>Client : 200 OK (access_token, refresh_token)
end
end
end
Note over Client,AuthService : Secure authentication with<br/>captcha protection and<br/>account lockout mechanism
```

**Diagram sources**
- [auth.go](file://internal/biz/auth.go#L1-L695)
- [auth.go](file://internal/service/auth.go#L1-L235)
- [auth.proto](file://api/auth/v1/auth.proto#L1-L156)

**Section sources**
- [auth.go](file://internal/biz/auth.go#L1-L695)
- [auth.go](file://internal/service/auth.go#L1-L235)

## Security and Data Protection

The kratos-boilerplate implements a comprehensive security model with multiple layers of protection for sensitive data. The Key Management System (KMS) provides centralized encryption key management and data protection.

```mermaid
graph TD
A[Application Data] --> B[Encryption Request]
B --> C{KMS Manager}
C --> D[Root Key Generator]
C --> E[Data Key Manager]
C --> F[Crypto Service]
E --> G[Key Storage]
F --> H[AES-GCM Encryption]
H --> I[Encrypted Data]
I --> J[Database Storage]
K[Data Retrieval] --> L[Decryption Request]
L --> C
C --> F
F --> M[AES-GCM Decryption]
M --> N[Plaintext Data]
N --> O[Application]
P[Security Features]
P --> Q[Field-Level Encryption]
P --> R[Data Key Rotation]
P --> S[Secure Key Storage]
P --> T[Hashing for Indexing]
P --> U[Memory Protection]
P --> V[Access Control]
style C fill:#f9f,stroke:#333
style F fill:#bbf,stroke:#333
style H fill:#bbf,stroke:#333
style M fill:#bbf,stroke:#333
Note over C: KMS Manager coordinates<br/>all encryption operations
Note over F: Crypto Service performs<br/>actual encryption/decryption
Note over H,M: AES-GCM provides authenticated<br/>encryption with integrity
```

**Diagram sources**
- [crypto_service.go](file://internal/pkg/kms/crypto_service.go#L1-L321)
- [manager.go](file://internal/pkg/kms/manager.go#L1-L308)

**Section sources**
- [crypto_service.go](file://internal/pkg/kms/crypto_service.go#L1-L321)
- [manager.go](file://internal/pkg/kms/manager.go#L1-L308)

## Frontend-Backend Integration

The frontend application is built with Vue.js and communicates with the backend through well-defined API endpoints. The integration includes proper error handling, authentication token management, and request/response typing.

```mermaid
sequenceDiagram
participant Browser as "Browser"
participant VueApp as "Vue.js Application"
participant AuthAPI as "Auth API Module"
participant Backend as "Backend Service"
Browser->>VueApp : User navigates to login page
VueApp->>VueApp : Render login form
VueApp->>VueApp : User enters credentials
VueApp->>AuthAPI : login(data)
AuthAPI->>Backend : POST /api/v1/auth/login
Backend->>Backend : Process authentication
alt Authentication Successful
Backend-->>AuthAPI : 200 OK + tokens
AuthAPI->>VueApp : Store tokens in localStorage
VueApp->>VueApp : Redirect to dashboard
VueApp->>AuthAPI : Set Authorization header
else Authentication Failed
Backend-->>AuthAPI : 401 Unauthorized
AuthAPI-->>VueApp : Show error message
VueApp->>VueApp : Display error to user
end
VueApp->>Backend : Subsequent API requests
Backend->>Backend : Validate token
alt Token Valid
Backend-->>VueApp : Requested data
else Token Invalid/Expired
Backend-->>VueApp : 401 Unauthorized
VueApp->>VueApp : Clear tokens
VueApp->>VueApp : Redirect to login
end
Note over VueApp,Backend : Axios interceptors handle<br/>authentication headers and<br/>token refresh automatically
```

**Diagram sources**
- [auth.ts](file://frontend/src/api/auth.ts#L1-L99)
- [auth.go](file://internal/service/auth.go#L1-L235)

**Section sources**
- [auth.ts](file://frontend/src/api/auth.ts#L1-L99)
- [auth.go](file://internal/service/auth.go#L1-L235)