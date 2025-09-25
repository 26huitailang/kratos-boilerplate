# User Register

<cite>
**Referenced Files in This Document**   
- [auth.proto](file://api/auth/v1/auth.proto)
- [auth.go](file://internal/biz/auth.go)
- [auth.go](file://internal/service/auth.go)
- [validate.proto](file://third_party/validate/validate.proto)
</cite>

## Table of Contents
1. [User Register](#user-register)
2. [HTTP and gRPC Endpoints](#http-and-grpc-endpoints)
3. [Request and Response Structure](#request-and-response-structure)
4. [Validation Rules](#validation-rules)
5. [Registration Flow](#registration-flow)
6. [Error Handling](#error-handling)
7. [Security Practices](#security-practices)
8. [Client Examples](#client-examples)

## HTTP and gRPC Endpoints

The User Register endpoint is accessible via both HTTP and gRPC protocols. The gRPC method is defined in the `auth.proto` file and mapped to a RESTful endpoint using the `google.api.http` annotation.

```protobuf
rpc Register(RegisterRequest) returns (RegisterReply) {
  option (google.api.http) = {
    post: "/api/v1/auth/register"
    body: "*"
  };
}
```

- **HTTP Method**: POST
- **RESTful Path**: `/api/v1/auth/register`
- **gRPC Method**: `auth.v1.Auth.Register`

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L25-L30)

## Request and Response Structure

### Request Fields

The `RegisterRequest` message contains the following fields:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `username` | string | Yes | Unique user identifier |
| `password` | string | Yes | User password (will be hashed) |
| `email` | string | Yes | User email address |
| `phone` | string | No | User phone number |
| `captcha_id` | string | Yes | Identifier of the captcha |
| `captcha_code` | string | Yes | User-entered captcha code |

### Response Structure

The `RegisterReply` message contains a simple success message:

```protobuf
message RegisterReply {
  string message = 1;
}
```

Upon successful registration, the response will contain:
```json
{
  "message": "Registration successful"
}
```

**Section sources**
- [auth.proto](file://api/auth/v1/auth.proto#L78-L83)
- [auth.proto](file://api/auth/v1/auth.proto#L85-L88)

## Validation Rules

Validation is enforced through both protobuf validation rules and custom business logic.

### Protobuf Validation

The `validate.proto` file provides field-level validation rules that can be applied to protobuf messages. For the registration endpoint, the following validations are relevant:

```protobuf
message StringRules {
  optional uint64 min_len = 2;
  optional uint64 max_len = 3;
  optional string pattern = 6;
  optional bool email = 12;
}
```

These rules can be applied to ensure:
- Username length constraints
- Email format validation
- Password complexity patterns

### Business Logic Validation

The actual validation is implemented in the `authUsecase.Register` method in `internal/biz/auth.go`. The validation rules include:

1. **Captcha Validation**: Must be enabled and provided
2. **Username Uniqueness**: Username must not already exist
3. **Email Uniqueness**: Email must not be used by another account
4. **Phone Uniqueness**: Phone number must not be used by another account
5. **Password Strength**: Minimum 8 characters

```go
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	// Additional password strength checks would be implemented here
	return nil
}
```

**Section sources**
- [validate.proto](file://third_party/validate/validate.proto#L470-L480)
- [auth.go](file://internal/biz/auth.go#L653-L657)

## Registration Flow

The user registration process follows a structured flow across multiple layers of the application architecture.

```mermaid
sequenceDiagram
participant Client as "Client Application"
participant HTTP as "HTTP Server"
participant Service as "AuthService"
participant Biz as "AuthUsecase"
participant Data as "User Repository"
participant Captcha as "Captcha Service"
Client->>HTTP : POST /api/v1/auth/register
HTTP->>Service : Unmarshal request
Service->>Biz : uc.Register()
Biz->>Captcha : Verify captcha
Captcha-->>Biz : Validation result
alt Captcha invalid
Biz-->>Service : ErrCaptchaInvalid
Service-->>Client : 400 Bad Request
return
end
Biz->>Data : GetUser(username)
Data-->>Biz : User existence check
alt User exists
Biz-->>Service : ErrUserExists
Service-->>Client : 409 Conflict
return
end
Biz->>Data : GetUserByEmail(email)
Data-->>Biz : Email uniqueness check
alt Email in use
Biz-->>Service : Email already registered error
Service-->>Client : 409 Conflict
return
end
Biz->>Data : GetUserByPhone(phone)
Data-->>Biz : Phone uniqueness check
alt Phone in use
Biz-->>Service : Phone already registered error
Service-->>Client : 409 Conflict
return
end
Biz->>Biz : validatePassword()
alt Password weak
Biz-->>Service : Password validation error
Service-->>Client : 400 Bad Request
return
end
Biz->>Biz : bcrypt.GenerateFromPassword()
Biz->>Data : CreateUser()
Data-->>Biz : Creation result
Biz-->>Service : Success
Service-->>Client : 200 OK with success message
```

**Diagram sources**
- [auth.go](file://internal/biz/auth.go#L249-L317)
- [auth.go](file://internal/service/auth.go#L68-L85)

## Error Handling

The registration endpoint returns specific error codes and HTTP status codes for different failure scenarios.

### Error Cases and Status Codes

| Error Case | Error Reason Code | HTTP Status | gRPC Code |
|------------|-------------------|-------------|-----------|
| Duplicate email | "EMAIL_EXISTS" | 409 Conflict | Already Exists |
| Weak password | "WEAK_PASSWORD" | 400 Bad Request | Invalid Argument |
| Invalid captcha | "CAPTCHA_INVALID" | 400 Bad Request | Invalid Argument |
| Missing captcha | "CAPTCHA_REQUIRED" | 400 Bad Request | Invalid Argument |
| Internal server error | "REGISTER_ERROR" | 500 Internal Server Error | Internal |

### Error Response Format

```json
{
  "error": {
    "code": 400,
    "message": "Bad Request",
    "details": [
      {
        "type": "type.googleapis.com/google.rpc.BadRequest",
        "field_violations": [
          {
            "field": "password",
            "description": "Password must be at least 8 characters long"
          }
        ]
      }
    ]
  }
}
```

In the implementation, errors are handled with specific error types that are mapped to appropriate HTTP responses:

```go
func (s *AuthService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterReply, error) {
	err := s.uc.Register(ctx, req.Username, req.Password, req.Email, req.Phone, req.CaptchaId, req.CaptchaCode)
	if err != nil {
		switch err {
		case biz.ErrUserExists:
			return nil, errors.BadRequest("USER_EXISTS", "Username already exists")
		case biz.ErrCaptchaRequired:
			return nil, errors.BadRequest("CAPTCHA_REQUIRED", "Captcha is required")
		case biz.ErrCaptchaInvalid:
			return nil, errors.BadRequest("CAPTCHA_INVALID", "Invalid captcha")
		case biz.ErrCaptchaExpired:
			return nil, errors.BadRequest("CAPTCHA_EXPIRED", "Captcha has expired")
		default:
			return nil, errors.InternalServer("REGISTER_ERROR", err.Error())
		}
	}
	return &v1.RegisterReply{Message: "Registration successful"}, nil
}
```

**Section sources**
- [auth.go](file://internal/service/auth.go#L68-L85)
- [auth.go](file://internal/biz/auth.go#L249-L317)

## Security Practices

The registration endpoint implements several security measures to protect user data and prevent abuse.

### Password Hashing

Passwords are securely hashed using the bcrypt algorithm before storage:

```go
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
if err != nil {
    return fmt.Errorf("failed to hash password: %v", err)
}
```

The hashed password is then stored in the database, ensuring that even if the database is compromised, user passwords remain protected.

### Captcha Verification

Captcha verification is integrated to prevent automated registration attempts:

```go
valid, err := uc.captchaService.Verify(ctx, captchaID, captchaCode)
if err != nil {
    return err
}
if !valid {
    return ErrCaptchaInvalid
}
```

The captcha system helps prevent bot attacks and brute force registration attempts.

### Rate Limiting

Although not explicitly shown in the code, rate limiting should be implemented at the HTTP server level to prevent abuse of the registration endpoint. This could be achieved through:

- IP-based rate limiting
- Account creation limits per time period
- CAPTCHA challenges for suspicious activity

### Protection Against Enumeration Attacks

The system avoids revealing whether a username or email already exists through generic error messages. However, the current implementation does distinguish between username and email conflicts in the error messages, which could be improved for better security.

### Secure Error Messaging

Error messages are designed to be informative for legitimate users while not revealing sensitive system information:

```go
return nil, errors.BadRequest("USER_EXISTS", "Username already exists")
```

Generic error messages prevent attackers from gathering information about existing accounts.

**Section sources**
- [auth.go](file://internal/biz/auth.go#L249-L317)
- [auth.go](file://internal/service/auth.go#L68-L85)

## Client Examples

### Go Client Example (gRPC)

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
	// Connect to the gRPC server
	conn, err := grpc.Dial("localhost:8000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create a client
	client := v1.NewAuthClient(conn)

	// Create a registration request
	req := &v1.RegisterRequest{
		Username:    "newuser",
		Password:    "SecurePassword123",
		Email:       "newuser@example.com",
		Phone:       "13800138000",
		CaptchaId:   "captcha123",
		CaptchaCode: "1234",
	}

	// Call the Register method
	resp, err := client.Register(context.Background(), req)
	if err != nil {
		log.Fatalf("Registration failed: %v", err)
	}

	log.Printf("Registration successful: %s", resp.Message)
}
```

### TypeScript Fetch Example (HTTP)

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
        message: 'Internal Server Error',
        details: [],
      },
    };
  }
}

// Usage example
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
      console.log('Registration successful:', response.message);
    } else {
      console.error('Registration failed:', response.error);
    }
  });
```

**Section sources**
- [auth.go](file://internal/service/auth.go#L68-L85)
- [auth.proto](file://api/auth/v1/auth.proto#L25-L30)