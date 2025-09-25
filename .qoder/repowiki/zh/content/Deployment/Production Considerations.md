# Production Considerations

<cite>
**Referenced Files in This Document**   
- [Dockerfile](file://Dockerfile)
- [docker-compose.yml](file://docker-compose.yml)
- [docker-compose.dev.yml](file://docker-compose.dev.yml)
- [config.yaml](file://configs/config.yaml)
- [features.yaml](file://configs/features.yaml)
- [features.prod.yaml](file://configs/features.prod.yaml)
- [auth.go](file://internal/biz/auth.go)
- [data.go](file://internal/data/data.go)
- [kms.go](file://internal/biz/kms.go)
- [operation_log.go](file://internal/biz/operation_log.go)
- [server.go](file://internal/server/server.go)
- [http.go](file://internal/server/http.go)
- [grpc.go](file://internal/server/grpc.go)
</cite>

## Table of Contents
1. [Introduction](#introduction)
2. [Environment Variable and Configuration Management](#environment-variable-and-configuration-management)
3. [Secrets and Credential Handling](#secrets-and-credential-handling)
4. [Logging and Monitoring Configuration](#logging-and-monitoring-configuration)
5. [Security Hardening Measures](#security-hardening-measures)
6. [High Availability and Connection Pooling](#high-availability-and-connection-pooling)
7. [Graceful Shutdown and Health Probes](#graceful-shutdown-and-health-probes)
8. [Scaling Strategies](#scaling-strategies)
9. [Backup, Disaster Recovery, and Incident Response](#backup-disaster-recovery-and-incident-response)

## Introduction
The kratos-boilerplate project is a modular Go-based microservices framework built on the Kratos ecosystem, designed for extensibility, security, and observability. This document outlines production deployment best practices, focusing on critical operational aspects such as configuration, security, monitoring, scalability, and resilience. The system leverages Docker and Docker Compose for containerization, supports feature toggles, plugin architecture, and integrates with PostgreSQL and Redis. The analysis is based on actual code and configuration files to ensure accuracy and relevance.

## Environment Variable and Configuration Management
The application uses a hierarchical configuration system with YAML files and environment-based overrides. The primary configuration is defined in `config.yaml`, which supports environment-specific feature flags via `features.yaml`, `features.dev.yaml`, and `features.prod.yaml`.

Key configuration elements include:
- **Server settings**: HTTP and gRPC endpoints with timeouts
- **Database and Redis connections**: Connection strings and timeouts
- **Plugin system**: Dynamic loading, sandboxing, and security limits
- **Feature flags**: Environment-aware toggles for progressive rollouts

The configuration is mounted into the container via volume mapping in `docker-compose.yml`, allowing runtime changes without rebuilding the image.

```yaml
# configs/config.yaml
data:
  database:
    driver: postgres
    source: postgresql://postgres:postgres@cross-redline-db:5432/test?sslmode=disable
  redis:
    addr: 127.0.0.1:6379
    read_timeout: 0.2s
    write_timeout: 0.2s

features:
  enabled: true
  config_file: "./configs/features.yaml"
  watch_config: true
  default_environment: "production"
```

The `watch_config: true` setting enables dynamic reloading of feature flags, supporting zero-downtime configuration updates.

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L38)
- [features.yaml](file://configs/features.yaml#L1-L180)
- [features.prod.yaml](file://configs/features.prod.yaml)

## Secrets and Credential Handling
The current implementation uses hardcoded credentials in `docker-compose.yml` for PostgreSQL:

```yaml
# docker-compose.yml
environment:
  POSTGRES_USER: postgres
  POSTGRES_PASSWORD: postgres
  POSTGRES_DB: test
```

While suitable for development, this approach is insecure for production. Recommended improvements:

1. **Use environment variables or Docker secrets**:
```yaml
environment:
  POSTGRES_USER: ${DB_USER}
  POSTGRES_PASSWORD: ${DB_PASSWORD}
```

2. **Integrate with a secrets manager** (e.g., Hashicorp Vault, AWS Secrets Manager) via the KMS package (`internal/pkg/kms`), which already supports cryptographic operations and key management.

3. **Encrypt sensitive configuration files** at rest using the `data_encryption` feature flag and KMS integration.

The `internal/biz/kms.go` and `internal/data/kms.go` files indicate the presence of a Key Management System, which should be leveraged for secure secret storage and retrieval.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L20-L35)
- [config.yaml](file://configs/config.yaml#L15-L18)
- [kms.go](file://internal/biz/kms.go)
- [kms.go](file://internal/data/kms.go)

## Logging and Monitoring Configuration
The system includes comprehensive observability features controlled by feature flags:

```yaml
# configs/features.yaml
metrics:
  enabled: true
tracing:
  enabled: true
audit_log:
  enabled: true
health_check:
  enabled: true
```

These flags enable:
- **Metrics collection**: For integration with Prometheus
- **Distributed tracing**: Likely using OpenTelemetry or Jaeger
- **Audit logging**: Tracks critical operations via `operation_log.go`
- **Health checks**: Standard endpoints for liveness and readiness

The `internal/biz/operation_log.go` implements structured logging of user actions, which should be secured and retained according to compliance requirements.

Log output should be directed to stdout/stderr for container orchestration platforms (e.g., Kubernetes) to collect and forward to centralized logging systems (e.g., ELK, Splunk).

**Section sources**
- [features.yaml](file://configs/features.yaml#L130-L150)
- [operation_log.go](file://internal/biz/operation_log.go)
- [server.go](file://internal/server/server.go)

## Security Hardening Measures
### Non-Root Containers
The Dockerfile does not specify a user, meaning the application runs as root. This should be remediated:

```dockerfile
# Add to Dockerfile
RUN groupadd -r appuser && useradd -r -g appuser appuser
USER appuser
```

### Read-Only Filesystem
The container should run with a read-only filesystem where possible:

```yaml
# In docker-compose.yml
services:
  app:
    read_only: true
    tmpfs:
      - /tmp
      - /run
```

### Network Policies
The current `docker-compose.yml` exposes ports 8000 and 9000. In production, restrict access using:
- Internal service mesh (e.g., Istio)
- Firewall rules
- API gateway for external access

### Plugin Sandbox
The configuration enables a plugin sandbox with resource limits:

```yaml
plugins:
  security:
    sandbox_enabled: true
    max_memory: "100MB"
    max_cpu_percent: 10
```

This limits potential damage from malicious or faulty plugins.

### Sensitive Data Handling
The `sensitive_data` feature enables data sanitization, likely using the `internal/pkg/sensitive` package to detect and anonymize PII in logs and responses.

**Section sources**
- [Dockerfile](file://Dockerfile#L1-L57)
- [docker-compose.yml](file://docker-compose.yml#L5-L50)
- [config.yaml](file://configs/config.yaml#L25-L31)
- [sensitive](file://internal/pkg/sensitive)

## High Availability and Connection Pooling
The application connects to PostgreSQL and Redis. To ensure high availability:

1. **Database**: Use PostgreSQL with streaming replication and a connection pooler (e.g., PgBouncer). The current configuration does not specify connection pooling.

2. **Redis**: Deploy Redis in sentinel or cluster mode for failover.

3. **Application**: Scale horizontally using multiple replicas behind a load balancer.

The `data` package (`internal/data/data.go`) should be reviewed to ensure proper connection pooling configuration for both database and Redis clients.

```go
// Example: Ensure DB connection pooling is configured
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)
```

**Section sources**
- [data.go](file://internal/data/data.go)
- [config.yaml](file://configs/config.yaml#L10-L18)

## Graceful Shutdown and Health Probes
The application exposes HTTP (8000) and gRPC (9000) ports. The `internal/server/server.go` implements server lifecycle management.

### Health Check Endpoints
The `health_check` feature enables standard endpoints:
- `/healthz` for liveness
- `/readyz` for readiness

These are critical for orchestration platforms to manage container lifecycle.

### Graceful Shutdown
The server should handle SIGTERM by:
1. Stopping accepting new requests
2. Completing in-flight requests
3. Closing database and Redis connections
4. Exiting cleanly

This is typically implemented using context cancellation and wait groups.

```mermaid
flowchart TD
A["SIGTERM Received"] --> B["Stop Accepting New Requests"]
B --> C["Wait for In-Flight Requests to Complete"]
C --> D["Close Database Connections"]
D --> E["Close Redis Connections"]
E --> F["Exit Process"]
```

**Diagram sources**
- [server.go](file://internal/server/server.go)
- [http.go](file://internal/server/http.go)
- [grpc.go](file://internal/server/grpc.go)

**Section sources**
- [server.go](file://internal/server/server.go)
- [http.go](file://internal/server/http.go)
- [grpc.go](file://internal/server/grpc.go)

## Scaling Strategies
The stateless nature of the application allows for horizontal scaling:

1. **Kubernetes Deployment**: Use deployments with multiple replicas and a service load balancer.
2. **Auto-scaling**: Based on CPU, memory, or request rate metrics.
3. **Session Management**: Since the app uses JWT, no shared session storage is needed.

Use the feature flags to enable rate limiting (`rate_limit`) in production to prevent abuse:

```yaml
rate_limit:
  enabled: true
  strategy: environment
  rules:
    environments: ["production", "staging"]
```

**Section sources**
- [features.yaml](file://configs/features.yaml#L50-L58)
- [auth.go](file://internal/biz/auth.go)

## Backup, Disaster Recovery, and Incident Response
### Backup Procedures
1. **Database**: Regular PostgreSQL dumps with `pg_dump` and WAL archiving.
2. **Configuration**: Version control for `configs/` directory.
3. **Plugins**: Backup `plugins/` directory if custom plugins are used.

### Disaster Recovery
1. **Multi-region deployment**: For high availability.
2. **Infrastructure as Code**: Use Docker Compose or Kubernetes manifests.
3. **Automated Recovery**: Scripts to restore from backups.

### Incident Response
1. **Monitoring Alerts**: On error rates, latency, or system health.
2. **Log Retention**: Keep audit logs for forensic analysis.
3. **Rollback Mechanism**: Disable features via feature flags or redeploy previous version.

The `internal/pkg/feature` package supports dynamic feature toggling, enabling quick mitigation of issues without redeployment.

**Section sources**
- [features.yaml](file://configs/features.yaml)
- [feature](file://internal/pkg/feature)
- [operation_log.go](file://internal/biz/operation_log.go)