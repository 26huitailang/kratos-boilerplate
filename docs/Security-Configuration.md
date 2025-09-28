# Security Configuration Guide

This document explains the security features and configuration options available in the project.

## Security Features

### 1. Security Headers

The application automatically adds the following security headers to HTTP responses:

#### HSTS (HTTP Strict Transport Security)
- **Purpose**: Forces HTTPS connections and prevents protocol downgrade attacks
- **Configuration**: Enabled by default for HTTPS connections
- **Header**: `Strict-Transport-Security: max-age=31536000; includeSubDomains; preload`

#### Content Security Policy (CSP)
- **Purpose**: Prevents XSS attacks by controlling resource loading
- **Default Policy**: `default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';`
- **Header**: `Content-Security-Policy`

#### X-Frame-Options
- **Purpose**: Prevents clickjacking attacks
- **Default**: `DENY`
- **Header**: `X-Frame-Options: DENY`

#### X-Content-Type-Options
- **Purpose**: Prevents MIME type sniffing
- **Default**: `nosniff`
- **Header**: `X-Content-Type-Options: nosniff`

#### X-XSS-Protection
- **Purpose**: Enables XSS filtering in browsers
- **Default**: `1; mode=block`
- **Header**: `X-XSS-Protection: 1; mode=block`

#### Additional Headers
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: camera=(), microphone=(), geolocation=(), payment=()`
- `X-Permitted-Cross-Domain-Policies: none`
- `Cross-Origin-Embedder-Policy: require-corp`
- `Cross-Origin-Opener-Policy: same-origin`
- `Cross-Origin-Resource-Policy: same-origin`

### 2. CORS (Cross-Origin Resource Sharing)

#### Default Configuration
```yaml
allowed_origins: ["*"]
allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH"]
allowed_headers: ["Accept", "Accept-Language", "Content-Language", "Content-Type", "Authorization", "X-Requested-With", "X-Request-ID"]
allow_credentials: false
cors_max_age: 86400  # 24 hours
```

#### Environment Configuration
```bash
# CORS configuration via environment variables
CORS_ALLOWED_ORIGINS="https://yourdomain.com,https://api.yourdomain.com"
CORS_ALLOWED_METHODS="GET,POST,PUT,DELETE,OPTIONS"
CORS_ALLOWED_HEADERS="Content-Type,Authorization,X-Request-ID"
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=86400
```

### 3. HTTPS/TLS Configuration

#### Environment Variables
```bash
# Enable HTTPS
TLS_ENABLED=true

# Certificate files
TLS_CERT_FILE=/path/to/certificate.pem
TLS_KEY_FILE=/path/to/private-key.pem

# TLS version (1.0, 1.1, 1.2, 1.3)
TLS_MIN_VERSION=1.2
```

#### Supported Cipher Suites
- `TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384`
- `TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305`
- `TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256`
- `TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384`
- `TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305`
- `TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256`

### 4. Configuration Validation

The application validates security configuration at startup:

#### Production Environment Checks
- JWT secret key strength (minimum 32 characters)
- Database SSL connections (sslmode=require)
- Redis password authentication
- TLS certificate file existence

#### Security Checklist
- [ ] JWT secret key is strong and randomly generated
- [ ] Database connections use SSL in production
- [ ] Redis authentication is enabled
- [ ] HTTPS is enabled with valid certificates
- [ ] CORS origins are restricted (not using "*")
- [ ] Security headers are properly configured

## Configuration Examples

### Development Environment
```yaml
# configs/config.yaml
security:
  enable_hsts: false  # HTTP in development
  content_security_policy: "default-src 'self' 'unsafe-inline' 'unsafe-eval'"  # Relaxed for development
  allowed_origins: ["*"]
  allow_credentials: false
```

### Production Environment
```yaml
# configs/config.prod.yaml
security:
  enable_hsts: true
  hsts_max_age: 31536000
  hsts_include_subdomains: true
  content_security_policy: "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';"
  frame_options: "DENY"
  content_type_options: true
  xss_protection: "1; mode=block"
  referrer_policy: "strict-origin-when-cross-origin"
  permissions_policy: "camera=(), microphone=(), geolocation=(), payment=()"
  allowed_origins: ["https://yourdomain.com", "https://app.yourdomain.com"]
  allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
  allowed_headers: ["Content-Type", "Authorization", "X-Request-ID"]
  allow_credentials: true
  cors_max_age: 86400
```

## Security Best Practices

### 1. HTTPS Configuration
- Always use HTTPS in production
- Use strong TLS versions (1.2 or higher)
- Regularly update certificates
- Enable HSTS with reasonable max-age

### 2. CORS Configuration
- Restrict allowed origins to specific domains
- Limit allowed methods to what's actually needed
- Be cautious with `allow_credentials: true`
- Set appropriate max-age for preflight caching

### 3. Content Security Policy
- Start with restrictive policy and gradually relax as needed
- Avoid `unsafe-inline` and `unsafe-eval` in production
- Use nonces or hashes for inline scripts/styles
- Regularly audit and update CSP rules

### 4. JWT Security
- Use strong, randomly generated secret keys
- Keep access tokens short-lived (1 hour or less)
- Implement token refresh mechanism
- Consider using asymmetric keys for better security

### 5. Database Security
- Always use SSL/TLS for database connections
- Use connection pooling with reasonable limits
- Implement proper authentication and authorization
- Regular security updates and patches

### 6. Monitoring and Logging
- Log security-related events
- Monitor for unusual patterns
- Set up alerts for security violations
- Regular security audits

## Troubleshooting

### Common Issues

#### CORS Errors
```
Access to fetch at 'https://api.example.com' from origin 'https://app.example.com' has been blocked by CORS policy
```
**Solution**: Add the origin to `allowed_origins` configuration.

#### Mixed Content Warnings
```
Mixed Content: The page at 'https://example.com' was loaded over HTTPS, but requested an insecure resource
```
**Solution**: Ensure all resources are loaded over HTTPS when using HTTPS.

#### CSP Violations
```
Content Security Policy: The page's settings blocked the loading of a resource
```
**Solution**: Update CSP policy to allow the resource or use nonces/hashes.

### Security Headers Testing

Use online tools to test security headers:
- [Mozilla Observatory](https://observatory.mozilla.org/)
- [Security Headers](https://securityheaders.com/)
- [SSL Labs](https://www.ssllabs.com/ssltest/)

### Command Line Testing
```bash
# Test security headers
curl -I https://yourdomain.com/api/health

# Test CORS
curl -H "Origin: https://example.com" \
     -H "Access-Control-Request-Method: POST" \
     -X OPTIONS \
     https://yourdomain.com/api/auth/login
```