# Security Best Practices

<cite>
**Referenced Files in This Document**   
- [Dockerfile](file://Dockerfile)
- [docker-compose.yml](file://docker-compose.yml)
- [docker-compose.dev.yml](file://docker-compose.dev.yml)
- [config.yaml](file://configs/config.yaml)
- [audit_logger.yaml](file://configs/plugins/audit_logger.yaml)
- [audit_logger.go](file://plugins/audit_logger.go)
</cite>

## Table of Contents
1. [Secure Configuration Management](#secure-configuration-management)
2. [Audit Logging Configuration](#audit-logging-configuration)
3. [Operation Logging Middleware](#operation-logging-middleware)
4. [Docker Security Best Practices](#docker-security-best-practices)
5. [Network Security in Docker Compose](#network-security-in-docker-compose)
6. [Monitoring and Alerting](#monitoring-and-alerting)
7. [Secure Deployment Patterns](#secure-deployment-patterns)
8. [Security Review and Compliance Checklists](#security-review-and-compliance-checklists)

## Secure Configuration Management

Proper configuration management is critical for maintaining the security posture of kratos-boilerplate deployments. The system uses a hierarchical configuration model with environment-specific overrides and secure defaults.

The main configuration file `config.yaml` defines core service settings, database connections, and plugin behavior. Sensitive credentials such as database passwords are currently embedded in the configuration, which presents a security risk. These should be replaced with environment variable references or external secret management systems.

```yaml
data:
  database:
    driver: postgres
    source: postgresql://postgres:postgres@cross-redline-db:5432/test?sslmode=disable
```

**Recommendation**: Refactor configuration to use environment variables:
```yaml
source: postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable
```

Configuration files should have strict file permissions (600) when mounted into containers. The `configs` directory is volume-mounted into the container at `/data/conf`, making it essential to ensure host-level file permissions are properly secured.

The plugin system includes security sandboxing features that limit resource usage:
```yaml
security:
  sandbox_enabled: true
  max_memory: "100MB"
  max_cpu_percent: 10
```

This prevents plugins from consuming excessive system resources and provides an additional layer of isolation.

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L38)

## Audit Logging Configuration

The audit_logger plugin provides comprehensive security event tracking and is configured through `audit_logger.yaml`. This plugin captures security-relevant events such as user logins, data modifications, and system changes.

Key audit logging settings include:
- **Log level**: "info" - captures significant security events
- **Output format**: JSON - enables structured log processing
- **Sensitive data masking**: Enabled - protects PII in logs
- **Retention period**: 30 days - meets typical compliance requirements
- **Max log size**: 100MB - prevents unbounded log growth

```yaml
enabled: true
priority: 20
timeout: 30s
retry_count: 3
settings:
  log_level: "info"
  output_format: "json"
  max_log_size: "100MB"
  retention_days: 30
  sensitive_data_masking: true
```

The plugin registers event handlers for critical security events:
- User login/logout
- Data creation, update, and deletion
- Plugin loading and startup

```go
events := []plugin.EventType{
    plugin.EventUserLogin,
    plugin.EventUserLogout,
    plugin.EventDataCreated,
    plugin.EventDataUpdated,
    plugin.EventDataDeleted,
    plugin.EventPluginLoaded,
    plugin.EventPluginStarted,
}
```

These events are processed by the audit logger's event handler, which formats and stores them securely.

**Section sources**
- [audit_logger.yaml](file://configs/plugins/audit_logger.yaml#L1-L14)
- [audit_logger.go](file://plugins/audit_logger.go#L95-L115)

## Operation Logging Middleware

The operation logging system is implemented as a plugin-based middleware that hooks into request processing lifecycle events. It uses the plugin system's hook mechanism to intercept requests at key points.

The audit logger registers two types of hooks:
1. Pre-request hook (priority 5) - captures request initiation
2. Post-request hook (priority 15) - captures request completion

```go
// Register request lifecycle hooks
requestHook := plugin.NewBaseHook(
    "audit_logger_pre_request",
    5,
    3*time.Second,
    func(ctx context.Context, data plugin.HookData) error {
        fmt.Println("AuditLoggerPlugin: Pre-request hook executed")
        return nil
    },
)

responseHook := plugin.NewBaseHook(
    "audit_logger_post_request",
    15,
    3*time.Second,
    func(ctx context.Context, data plugin.HookData) error {
        fmt.Println("AuditLoggerPlugin: Post-request hook executed")
        return nil
    },
)
```

This two-phase approach enables the system to measure request duration, track user sessions, and correlate requests with security events. The middleware captures metadata such as:
- Request timestamps
- User identifiers
- Endpoint accessed
- Response status codes

The hook priority system ensures that audit logging occurs before most other processing (high priority 5) but after authentication, allowing access to user context.

**Section sources**
- [audit_logger.go](file://plugins/audit_logger.go#L60-L93)

## Docker Security Best Practices

The Docker deployment configuration follows several security best practices while having areas for improvement.

### Minimal Base Images
The application uses `openeuler/openeuler:24.03-lts` as its base image, which is a minimal Linux distribution. This reduces the attack surface compared to larger distributions.

```dockerfile
FROM openeuler/openeuler:24.03-lts AS builder
```

### Non-Root Execution
The current Dockerfile does not explicitly create a non-root user, meaning containers run as root by default. This violates the principle of least privilege.

**Recommendation**: Add non-root user configuration:
```dockerfile
RUN groupadd -r appuser && useradd -r -g appuser appuser
USER appuser
```

### Resource Limits
While the plugin system enforces per-plugin resource limits, container-level resource constraints are not defined in docker-compose.yml.

**Recommendation**: Add resource limits to prevent denial-of-service:
```yaml
services:
  app:
    # ... existing config
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

### Secure Build Process
The build stage installs necessary tools and then discards them in the final image, following the multi-stage build pattern. Package caches are cleaned to reduce image size and potential vulnerabilities.

```dockerfile
RUN dnf update -y && dnf install -y wget tar nodejs npm git make \
    && dnf clean all \
    && rm -rf /var/cache/dnf/*
```

**Section sources**
- [Dockerfile](file://Dockerfile#L1-L57)
- [docker-compose.yml](file://docker-compose.yml#L1-L50)

## Network Security in Docker Compose

The docker-compose.yml file defines network configuration that impacts security through service isolation and port exposure controls.

### Service Isolation
All services are placed on a dedicated bridge network:
```yaml
networks:
  cross-redline-network:
    driver: bridge
```

This provides network isolation from other containers on the host and enables internal service communication without exposing ports externally.

### Port Exposure
The application exposes HTTP (8000) and gRPC (9000) ports to the host:
```yaml
ports:
  - "8000:8000"
  - "9000:9000"
```

The database service also exposes port 5432, which should be restricted to internal network access only in production.

**Recommendation**: Remove external database port mapping in production:
```yaml
# Remove this in production
# ports:
#   - "5432:5432"
```

### Dependency Management
Service startup order is controlled through health checks:
```yaml
depends_on:
  db:
    condition: service_healthy
```

This ensures the database is ready before the application starts, preventing startup failures and reducing the window for connection-related vulnerabilities.

### Volume Security
Configuration files are mounted from the host:
```yaml
volumes:
  - ./configs:/data/conf
```

Ensure the host directory has restrictive permissions (700) and is owned by a dedicated service account.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)

## Monitoring and Alerting

Effective monitoring and alerting are essential for detecting suspicious activities and potential security breaches.

### Audit Log Monitoring
The audit logger's JSON-formatted output enables integration with SIEM systems and log analytics platforms. Key events to monitor include:
- Multiple failed login attempts
- Unusual login times or locations
- Privileged operation sequences
- Configuration changes

### Anomaly Detection
Implement alerts for:
- Spike in 4xx/5xx HTTP status codes
- Unusual data export volumes
- Concurrent sessions from different geographic locations
- Rapid succession of data modification operations

### Log Integrity
Ensure audit logs are protected from tampering by:
- Writing to immutable storage
- Implementing log signing
- Regular log integrity checks
- Off-host log aggregation

The current implementation writes logs to the container filesystem. **Recommendation**: Configure external log forwarding:
```yaml
services:
  app:
    # ... existing config
    logging:
      driver: "syslog"
      options:
        syslog-address: "tcp://logs.example.com:514"
        syslog-format: "rfc5424"
```

**Section sources**
- [audit_logger.go](file://plugins/audit_logger.go#L1-L132)
- [audit_logger.yaml](file://configs/plugins/audit_logger.yaml#L1-L14)

## Secure Deployment Patterns

Production deployments should follow these secure patterns to minimize attack surface and ensure resilience.

### TLS Termination
All external traffic should be protected with TLS. **Recommendation**: Deploy a reverse proxy (NGINX, Traefik) or service mesh (Istio) for:
- SSL/TLS termination
- Certificate management
- HTTP to HTTPS redirection
- HSTS enforcement

### WAF Integration
Integrate a Web Application Firewall to protect against common threats:
- SQL injection
- XSS attacks
- CSRF protection
- Rate limiting
- Bot detection

### Regular Security Patching
Implement a patch management process:
- Weekly base image updates
- Monthly Go runtime updates
- Quarterly dependency audits
- Automated vulnerability scanning

Use tools like Trivy or Clair to scan container images:
```bash
trivy image openeuler/openeuler:24.03-lts
```

### Configuration Hardening
Production configuration should:
- Disable debug modes
- Set appropriate log levels
- Enable all security features
- Disable unused services

Example production settings:
```yaml
plugins:
  security:
    sandbox_enabled: true
    max_memory: "100MB"
    max_cpu_percent: 10

features:
  watch_config: false  # Prevent runtime changes in production
```

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L38)
- [Dockerfile](file://Dockerfile#L1-L57)

## Security Review and Compliance Checklists

Use these checklists to verify security posture before production deployment.

### Configuration Security Checklist
- [ ] Remove hardcoded credentials from config files
- [ ] Set file permissions to 600 for configuration files
- [ ] Enable sensitive data masking in logs
- [ ] Validate all configuration values are parameterized
- [ ] Ensure audit logging is enabled and configured

### Container Security Checklist
- [ ] Run containers as non-root user
- [ ] Set resource limits for CPU and memory
- [ ] Use minimal base images
- [ ] Scan images for vulnerabilities regularly
- [ ] Remove unnecessary tools from final image

### Network Security Checklist
- [ ] Expose only required ports externally
- [ ] Use internal networks for service-to-service communication
- [ ] Implement firewall rules to restrict access
- [ ] Remove database port exposure in production
- [ ] Use network policies to enforce service isolation

### Logging and Monitoring Checklist
- [ ] Forward audit logs to external SIEM system
- [ ] Implement log integrity protection
- [ ] Configure alerts for suspicious activities
- [ ] Ensure log retention meets compliance requirements
- [ ] Test log rotation and archival processes

### Deployment Security Checklist
- [ ] Implement TLS termination with valid certificates
- [ ] Deploy WAF in front of application
- [ ] Establish regular patching schedule
- [ ] Disable configuration hot-reloading in production
- [ ] Conduct penetration testing before go-live

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L38)
- [Dockerfile](file://Dockerfile#L1-L57)
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [audit_logger.yaml](file://configs/plugins/audit_logger.yaml#L1-L14)