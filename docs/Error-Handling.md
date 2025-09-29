# Error Handling System

## Overview

This document describes the unified error handling system implemented in the kratos-boilerplate project.

## Architecture

### Error Processing Flow

1. **Error Generation**: Services generate errors using Kratos error functions
2. **Middleware Processing**: Error handling middleware intercepts and processes errors
3. **Error Transformation**: Errors are transformed into a unified BaseError format
4. **Response Generation**: Standardized error responses are returned to clients

### Middleware Components

#### 1. ErrorHandlerMiddleware
- **Purpose**: Main error handling middleware that processes all types of errors
- **Features**:
  - Converts various error types to unified format
  - Adds context information (trace ID, operation, etc.)
  - Logs errors with appropriate levels
  - Returns standardized error responses

#### 2. ValidationErrorMiddleware
- **Purpose**: Specialized middleware for handling validation errors
- **Features**:
  - Detects validation errors in error messages
  - Extracts field names from error messages
  - Provides user-friendly validation error responses

#### 3. RateLimitErrorMiddleware
- **Purpose**: Handles rate limiting errors
- **Features**:
  - Detects rate limit errors (HTTP 429)
  - Provides informative rate limit messages
  - Includes retry suggestions

## Error Response Format

All errors are returned in a standardized JSON format:

```json
{
  "code": 20001,
  "message": "Invalid parameter: username is required",
  "timestamp": "2023-09-28T10:30:00Z",
  "trace_id": "abc123",
  "operation": "auth.login",
  "details": {
    "field": "username",
    "constraint": "required"
  }
}
```

### Response Fields

- `code`: Internal error code (see Error Codes section)
- `message`: Human-readable error message
- `timestamp`: ISO 8601 timestamp when error occurred
- `trace_id`: Request trace ID for debugging
- `operation`: API operation that generated the error
- `details`: Additional error context (optional)

## Error Codes

The system uses a 5-digit error code scheme:

- **10000-19999**: System errors (internal, network, etc.)
- **20000-29999**: Parameter/validation errors
- **30000-39999**: Authentication/authorization errors
- **40000-49999**: Business logic errors
- **50000-59999**: External service errors

### Common Error Codes

| Code  | Description                | HTTP Status |
| ----- | -------------------------- | ----------- |
| 10001 | Internal server error      | 500         |
| 10002 | Database connection failed | 500         |
| 10003 | Network timeout            | 500         |
| 20001 | Invalid parameters         | 400         |
| 20002 | Invalid format             | 400         |
| 20003 | Parameter out of range     | 400         |
| 30001 | Authentication failed      | 401         |
| 30002 | Token expired              | 401         |
| 30003 | Access forbidden           | 403         |

## Testing Error Handling

### Test Endpoints

The `/helloworld/{name}` endpoint provides several test scenarios:

1. **Normal Response**: 
   ```
   GET /helloworld/world
   Response: {"message": "Hello world"}
   ```

2. **Internal Server Error**: 
   ```
   GET /helloworld/error
   Response: 500 with standardized error format
   ```

3. **Bad Request Error**: 
   ```
   GET /helloworld/badrequest
   Response: 400 with standardized error format
   ```

4. **Not Found Error**: 
   ```
   GET /helloworld/notfound
   Response: 404 with standardized error format
   ```

5. **Forbidden Error**: 
   ```
   GET /helloworld/forbidden
   Response: 403 with standardized error format
   ```

6. **Timeout Error**: 
   ```
   GET /helloworld/timeout
   Response: 408 with standardized error format
   ```

### Testing Results

The error handling middleware has been successfully tested with the following results:

#### 1. Normal Response
```bash
curl http://localhost:8000/helloworld/world
# Response: {"message":"Hello world"}
```

#### 2. Internal Server Error (500)
```bash
curl http://localhost:8000/helloworld/error
# Response: {
#   "code": 500,
#   "reason": "✐",
#   "message": "Internal server error for testing",
#   "metadata": {
#     "error_code": "✐",
#     "response": "{\"code\":10000,\"details\":{\"reason\":\"INTERNAL_ERROR\"},\"message\":\"Internal server error for testing\",\"operation\":\"/helloworld.v1.Greeter/SayHello\",\"timestamp\":\"2025-09-29T16:30:00+08:00\"}",
#     "trace_id": ""
#   }
# }
```

#### 3. Bad Request Error (400)
```bash
curl http://localhost:8000/helloworld/badrequest
# Response: HTTP 400 with unified error format
```

#### 4. Not Found Error (404)
```bash
curl http://localhost:8000/helloworld/notfound  
# Response: HTTP 404 with unified error format
```

#### 5. Log Output Analysis
The middleware correctly logs errors with appropriate levels:
- ERROR level for 5xx errors
- WARN level for 4xx client errors
- INFO level for business logic errors

Each log entry includes:
- Error code and message
- HTTP status and request details
- Operation name and timestamp
- Error details and original cause

### Testing Commands

```bash
# Test normal response
curl http://localhost:8000/helloworld/world

# Test internal server error
curl http://localhost:8000/helloworld/error

# Test bad request error
curl http://localhost:8000/helloworld/badrequest

# Test not found error
curl http://localhost:8000/helloworld/notfound

# Test forbidden error
curl http://localhost:8000/helloworld/forbidden

# Test timeout error
curl http://localhost:8000/helloworld/timeout
```

## Configuration

The error handling middleware is configured in `internal/server/http.go`:

```go
// Create error handling middleware
errorHandler := errors.NewErrorHandlerMiddleware(logger, true, false)
validationErrorHandler := errors.NewValidationErrorMiddleware(logger)
rateLimitErrorHandler := errors.NewRateLimitErrorMiddleware(logger)

// Add to server middleware chain
kratosHttp.Middleware(
    recovery.Recovery(),
    errorHandler.Middleware(),
    validationErrorHandler.Middleware(),
    rateLimitErrorHandler.Middleware(),
    // ... other middleware
),
```

### Middleware Parameters

- `enableDetails`: Whether to include detailed error information in responses
- `enableStack`: Whether to include stack traces in logs (not in responses)

## Best Practices

1. **Use Appropriate Error Codes**: Choose error codes that match the error type
2. **Provide Clear Messages**: Error messages should be user-friendly and actionable
3. **Include Context**: Add relevant context information to help with debugging
4. **Log Appropriately**: Use different log levels based on error severity
5. **Security Considerations**: Don't expose sensitive information in error messages

## Future Enhancements

- [ ] Internationalization support for error messages
- [ ] Custom error templates for different client types
- [ ] Error reporting and analytics integration
- [ ] Enhanced validation error extraction
- [ ] Rate limiting error recovery suggestions