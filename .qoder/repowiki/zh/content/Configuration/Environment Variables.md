# Environment Variables

<cite>
**Referenced Files in This Document**   
- [config.yaml](file://configs/config.yaml)
- [conf.proto](file://internal/conf/conf.proto)
- [main.go](file://cmd/kratos-boilerplate/main.go)
</cite>

## Table of Contents
1. [Introduction](#introduction)
2. [Configuration Structure Overview](#configuration-structure-overview)
3. [Environment Variable Mapping Convention](#environment-variable-mapping-convention)
4. [Practical Usage Examples](#practical-usage-examples)
5. [Security Considerations](#security-considerations)

## Introduction
This document explains how environment variables are used for configuration in the kratos-boilerplate project. The system allows environment variables to override settings defined in `config.yaml` and those specified in the `conf.proto` schema. This mechanism is particularly useful for deployment-specific values such as database connection strings, JWT secrets, and service endpoints. Environment variables provide a flexible way to manage configurations across different environments (development, staging, production) without modifying configuration files.

The configuration system is initialized in the `main.go` file using the Kratos framework's config module, which supports multiple sources including file and environment variables. However, in the current implementation, only file-based configuration is explicitly registered, meaning environment variable support must be inferred or extended through the underlying Kratos configuration loader behavior.

## Configuration Structure Overview
The configuration structure is defined using Protocol Buffers in `conf.proto`, which describes the schema for the `Bootstrap` message containing nested messages for `Server`, `Data`, `Auth`, and `Features`. These correspond directly to sections in the `config.yaml` file.

```yaml
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
auth:
  jwt_secret_key: "your-jwt-secret-key"
  access_token_expiration: 3600s
  refresh_token_expiration: 86400s
  captcha_enabled: true
  captcha_expiration: 300s
  max_login_attempts: 5
  lock_duration: 900s
  totp_enabled: false
```

The `conf.proto` file defines the structure that this YAML file is unmarshaled into:

```protobuf
message Bootstrap {
  Server server = 1;
  Data data = 2;
  Auth auth = 3;
  Features features = 4;
}

message Auth {
  string jwt_secret_key = 1;
  google.protobuf.Duration access_token_expiration = 2;
  google.protobuf.Duration refresh_token_expiration = 3;
  bool captcha_enabled = 4;
  google.protobuf.Duration captcha_expiration = 5;
  int32 max_login_attempts = 6;
  google.protobuf.Duration lock_duration = 7;
  bool totp_enabled = 8;
}
```

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L37)
- [conf.proto](file://internal/conf/conf.proto#L1-L69)

## Environment Variable Mapping Convention
Although the current `main.go` only registers a file source for configuration, the Kratos config system supports environment variables by default when using the standard loader. When environment variables are used, they follow a specific naming convention to map to nested configuration fields.

The mapping convention is:
- Convert the field path to uppercase
- Replace dots and nested structure with underscores
- Use the exact field names as defined in the proto message

For example:
- `data.database.source` → `DATA_DATABASE_SOURCE`
- `auth.jwt_secret_key` → `AUTH_JWT_SECRET_KEY`
- `server.http.addr` → `SERVER_HTTP_ADDR`

This allows overriding any configuration value via environment variables without changing the `config.yaml` file.

The following table shows key configuration fields and their corresponding environment variable names:

| Configuration Field | Environment Variable Name | Example Value |
|---------------------|---------------------------|---------------|
| data.database.source | DATA_DATABASE_SOURCE | postgresql://user:pass@host:5432/db |
| auth.jwt_secret_key | AUTH_JWT_SECRET_KEY | my-super-secret-jwt-key |
| server.http.addr | SERVER_HTTP_ADDR | 0.0.0.0:8080 |
| server.grpc.addr | SERVER_GRPC_ADDR | 0.0.0.0:9090 |
| data.redis.addr | DATA_REDIS_ADDR | redis-host:6379 |
| auth.captcha_enabled | AUTH_CAPTCHA_ENABLED | true |

**Section sources**
- [conf.proto](file://internal/conf/conf.proto#L1-L69)
- [config.yaml](file://configs/config.yaml#L1-L37)

## Practical Usage Examples
### Docker
When running the application in Docker, environment variables can be passed using the `-e` flag:

```bash
docker run -d \
  -e DATA_DATABASE_SOURCE="postgresql://produser:prodpass@prod-db:5432/proddb?sslmode=require" \
  -e AUTH_JWT_SECRET_KEY="production-jwt-secret-32chars-or-more" \
  -e SERVER_HTTP_ADDR="0.0.0.0:80" \
  -p 80:80 \
  kratos-boilerplate:latest
```

### docker-compose
In `docker-compose.yml`, environment variables can be defined under the `environment` section:

```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8000:8000"
    environment:
      - DATA_DATABASE_SOURCE=postgresql://user:pass@postgres:5432/appdb?sslmode=disable
      - AUTH_JWT_SECRET_KEY=my-production-jwt-secret-key
      - SERVER_HTTP_ADDR=0.0.0.0:8000
      - DATA_REDIS_ADDR=redis:6379
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:13
    environment:
      POSTGRES_DB: appdb
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass

  redis:
    image: redis:6-alpine
```

For development, you can use a `.env` file:

```env
DATA_DATABASE_SOURCE=postgresql://postgres:postgres@localhost:5432/test?sslmode=disable
AUTH_JWT_SECRET_KEY=dev-jwt-secret-key
SERVER_HTTP_ADDR=0.0.0.0:8000
DATA_REDIS_ADDR=localhost:6379
```

Then reference it in `docker-compose.dev.yml`:

```yaml
version: '3.8'
services:
  app:
    build: .
    env_file: .env
    ports:
      - "8000:8000"
    volumes:
      - ./configs:/app/configs
```

### Kubernetes
In Kubernetes, use Secrets to store sensitive values and environment variables to inject them:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-config
type: Opaque
data:
  jwt-secret-key: bXktcHJvZHVjdGlvbi1qd3Qtc2VjcmV0LWtleQ== # base64 encoded
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kratos-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: kratos-app
  template:
    metadata:
      labels:
        app: kratos-app
    spec:
      containers:
      - name: app
        image: kratos-boilerplate:latest
        ports:
        - containerPort: 8000
        env:
        - name: DATA_DATABASE_SOURCE
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: connection-string
        - name: AUTH_JWT_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: app-config
              key: jwt-secret-key
        - name: SERVER_HTTP_ADDR
          value: "0.0.0.0:8000"
```

**Section sources**
- [main.go](file://cmd/kratos-boilerplate/main.go#L53-L90)
- [config.yaml](file://configs/config.yaml#L1-L37)

## Security Considerations
When using environment variables for configuration, especially in production environments, several security best practices should be followed:

### Secrets Management
- **Never commit secrets to version control**: Keep `.env` files in `.gitignore`
- **Use secret managers in production**: Prefer Kubernetes Secrets, Hashicorp Vault, AWS Secrets Manager, or similar over plain environment variables
- **Limit environment variable exposure**: Only expose necessary variables to containers/pods

### Development vs Production
For development, `.env` files provide convenience:

```env
# .env.development
DATA_DATABASE_SOURCE=postgresql://dev:dev@localhost:5432/devdb?sslmode=disable
AUTH_JWT_SECRET_KEY=development-secret-key-change-in-production
AUTH_CAPTCHA_ENABLED=false
```

For production, use encrypted secrets:

```bash
# Example: Using Kubernetes Secrets
kubectl create secret generic app-secrets \
  --from-literal=jwt-secret-key=$(openssl rand -base64 32) \
  --from-literal=database-password=$(openssl rand -base64 24)
```

### Sensitive Data Protection
The project includes a sensitive data handling package (`internal/pkg/sensitive`) that can be configured differently per environment:

```go
// Example from test code - different logging behavior per environment
devLogger.Infow("development log", "password", "plaintext") // No masking in dev
prodLogger.Infow("production log", "password", "masked-value") // Masked in prod
```

Always ensure that:
- JWT secrets are at least 32 characters long
- Database passwords are complex and rotated regularly
- Environment variables containing secrets are not logged or exposed in error messages
- Use `readonly` filesystems where possible to prevent runtime modification

**Section sources**
- [main.go](file://cmd/kratos-boilerplate/main.go#L53-L90)
- [conf.proto](file://internal/conf/conf.proto#L1-L69)
- [config.yaml](file://configs/config.yaml#L1-L37)
- [examples_test.go](file://internal/pkg/sensitive/examples_test.go#L200-L230)