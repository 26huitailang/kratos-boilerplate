# API Documentation Guide

## Overview

This document provides comprehensive information about the Kratos Boilerplate API documentation system, including how to use, maintain, and extend the API documentation.

## Documentation Files

The API documentation consists of several files:

- `openapi.yaml` - Primary OpenAPI 3.0 specification in YAML format
- `openapi.json` - OpenAPI specification in JSON format (for tools compatibility)
- `api/` directory - Protocol Buffer definitions and generated code

## Accessing API Documentation

### 1. Swagger UI (Recommended)

Access the interactive Swagger UI interface at:
```
http://localhost:8000/q/swagger-ui/
```

Features:
- Interactive API testing
- Request/response examples
- Authentication support
- Real-time API exploration

### 2. Static Documentation Files

- **YAML Format**: `openapi.yaml` - Human-readable, version control friendly
- **JSON Format**: `openapi.json` - Tool-compatible, for Postman/Insomnia import

## Importing to External Tools

### Postman

1. Open Postman
2. Click "Import" button
3. Choose "File" tab
4. Select `openapi.yaml` (recommended) or `openapi.json`
5. Click "Import"
6. Postman will automatically create a collection with all endpoints

### Insomnia

1. Open Insomnia
2. Go to "Application" → "Preferences" → "Data"
3. Click "Import Data"
4. Select `openapi.yaml` or `openapi.json` file
5. Choose import options and confirm

### Other Tools

Most API tools support OpenAPI 3.0 format. Use either:
- `openapi.yaml` for tools that prefer YAML (recommended)
- `openapi.json` for tools that require JSON

## API Authentication

### JWT Token Authentication

Most API endpoints require authentication via JWT tokens:

```http
Authorization: Bearer <your-jwt-token>
```

### Getting Authentication Token

1. Register a new account via `/api/v1/auth/register`
2. Login via `/api/v1/auth/login`
3. Use the returned `access_token` in subsequent requests

Example login request:
```json
{
  "username": "your_username",
  "password": "your_password",
  "captcha_id": "captcha_id_from_captcha_endpoint",
  "captcha_code": "ABCD"
}
```

## API Endpoints Overview

### Authentication Endpoints

| Method | Endpoint                              | Description               |
| ------ | ------------------------------------- | ------------------------- |
| GET    | `/api/v1/auth/captcha`                | Get verification captcha  |
| POST   | `/api/v1/auth/captcha/verify`         | Verify captcha code       |
| POST   | `/api/v1/auth/register`               | User registration         |
| POST   | `/api/v1/auth/login`                  | User login                |
| POST   | `/api/v1/auth/logout`                 | User logout               |
| POST   | `/api/v1/auth/refresh`                | Refresh JWT token         |
| GET    | `/api/v1/auth/lock-status/{username}` | Check account lock status |

### Demo Endpoints

| Method | Endpoint             | Description              |
| ------ | -------------------- | ------------------------ |
| GET    | `/helloworld/{name}` | Sample greeting endpoint |

## Request/Response Examples

### User Registration

**Request:**
```bash
curl -X POST "http://localhost:8000/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "password": "MyPassword123!",
    "email": "john.doe@example.com",
    "phone": "+8613812345678",
    "captcha_id": "captcha_12345",
    "captcha_code": "ABCD"
  }'
```

**Response:**
```json
{
  "message": "Registration successful"
}
```

### User Login

**Request:**
```bash
curl -X POST "http://localhost:8000/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john_doe",
    "password": "MyPassword123!",
    "captcha_id": "captcha_12345",
    "captcha_code": "ABCD"
  }'
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "refresh_token_abc123...",
  "expires_in": 3600
}
```

## Error Handling

The API uses standard HTTP status codes and returns error details in JSON format:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input parameters",
    "details": [
      {
        "field": "username",
        "message": "Username is required"
      }
    ]
  }
}
```

### Common Error Codes

- `400` - Bad Request (validation errors)
- `401` - Unauthorized (authentication required)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found (resource not found)
- `429` - Too Many Requests (rate limit exceeded)
- `500` - Internal Server Error

## Rate Limiting

API calls are rate limited to prevent abuse:

- **Authentication endpoints**: 10 requests per minute per IP
- **General endpoints**: 100 requests per minute per authenticated user
- **Public endpoints**: 50 requests per minute per IP

Rate limit headers are included in responses:
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1640995200
```

## Versioning

The API uses URL versioning:
- Current version: `v1`
- Base path: `/api/v1/`

When breaking changes are introduced, a new version will be created (e.g., `v2`).

## CORS Policy

The API supports Cross-Origin Resource Sharing (CORS):

- **Allowed Origins**: Configurable per environment
- **Allowed Methods**: GET, POST, PUT, DELETE, OPTIONS
- **Allowed Headers**: Authorization, Content-Type, X-Requested-With
- **Max Age**: 86400 seconds (24 hours)

## Development and Testing

### Regenerating Documentation

When API definitions change, regenerate the documentation:

```bash
# Regenerate proto files and OpenAPI spec
make api

# Convert YAML to JSON (if needed)
python3 -c "
import yaml, json
with open('openapi.yaml', 'r') as f: data = yaml.safe_load(f)
with open('openapi.json', 'w') as f: json.dump(data, f, indent=2)
"
```

### Testing API Endpoints

Use the Swagger UI for interactive testing, or use curl/Postman with the provided examples.

### API Documentation Maintenance

1. **Update Proto Files**: Modify `.proto` files in the `api/` directory
2. **Add Comments**: Include detailed comments in proto files for better documentation
3. **Regenerate**: Run `make api` to update generated files
4. **Review**: Check the updated Swagger UI for accuracy
5. **Test**: Verify all endpoints work as documented

## Security Considerations

### Authentication
- Always use HTTPS in production
- JWT tokens should have reasonable expiration times
- Implement proper token refresh mechanisms

### Rate Limiting
- Monitor rate limit usage
- Implement backoff strategies in client applications
- Consider implementing different limits for different user tiers

### Input Validation
- All input is validated server-side
- Captcha verification for critical operations
- SQL injection and XSS protection

## Support and Contributing

### Getting Help
- Check this documentation first
- Review the Swagger UI for endpoint details
- Check the repository issues for known problems

### Contributing
- Follow the API design guidelines
- Update documentation when adding new endpoints
- Ensure all new endpoints include proper examples
- Test changes thoroughly before submitting

### Contact
- Development Team: dev@your-company.com
- Repository: https://github.com/your-org/kratos-boilerplate