# Configuration Best Practices

<cite>
**Referenced Files in This Document**   
- [config.yaml](file://configs/config.yaml)
- [features.yaml](file://configs/features.yaml)
- [features.dev.yaml](file://configs/features.dev.yaml)
- [features.prod.yaml](file://configs/features.prod.yaml)
- [conf.proto](file://internal/conf/conf.proto)
- [main.go](file://cmd/kratos-boilerplate/main.go)
- [wire.go](file://cmd/kratos-boilerplate/wire.go)
- [toggle_manager.go](file://internal/pkg/feature/toggle_manager.go)
- [file_repository.go](file://internal/pkg/feature/file_repository.go)
- [wire.go](file://internal/pkg/feature/wire.go)
</cite>

## Table of Contents
1. [Introduction](#introduction)
2. [Project Structure](#project-structure)
3. [Core Configuration Components](#core-configuration-components)
4. [Multi-Environment Configuration Strategy](#multi-environment-configuration-strategy)
5. [Feature Toggle System Architecture](#feature-toggle-system-architecture)
6. [Configuration Validation and Schema Management](#configuration-validation-and-schema-management)
7. [Secure Handling of Sensitive Data](#secure-handling-of-sensitive-data)
8. [Configuration Loading and Caching Performance](#configuration-loading-and-caching-performance)
9. [Monitoring and Auditing Configuration Changes](#monitoring-and-auditing-configuration-changes)
10. [Rollback and Pre-Deployment Validation](#rollback-and-pre-deployment-validation)

## Introduction
This document provides comprehensive configuration best practices for the kratos-boilerplate project, a Go-based microservices framework built on Kratos. The system implements a robust configuration management strategy with support for multi-environment deployments, feature toggles, and dynamic configuration updates. The architecture separates core service configuration from feature-specific settings, enabling safe deployment across development, staging, and production environments. This guide details the configuration patterns, validation mechanisms, and operational procedures that ensure reliable and secure application behavior across all deployment scenarios.

## Project Structure
The kratos-boilerplate project follows a clean, layered architecture with clear separation of concerns. Configuration files are centralized in the `configs/` directory, while configuration logic is implemented in the `internal/conf/` and `internal/pkg/feature/` packages. The structure supports both static service configuration and dynamic feature toggles, with environment-specific overrides.

```mermaid
graph TB
subgraph "Configuration Files"
A[config.yaml]
B[features.yaml]
C[features.dev.yaml]
D[features.prod.yaml]
E[plugins/*.yaml]
end
subgraph "Application Code"
F[main.go]
G[conf.proto]
H[toggle_manager.go]
I[file_repository.go]
J[wire.go]
end
A --> F
B --> H
C --> H
D --> H
E --> F
G --> F
H --> F
I --> H
J --> F
style A fill:#f9f,stroke:#333
style B fill:#f9f,stroke:#333
style C fill:#f9f,stroke:#333
style D fill:#f9f,stroke:#333
style E fill:#f9f,stroke:#333
```

**Diagram sources**
- [config.yaml](file://configs/config.yaml)
- [features.yaml](file://configs/features.yaml)
- [features.dev.yaml](file://configs/features.dev.yaml)
- [features.prod.yaml](file://configs/features.prod.yaml)
- [conf.proto](file://internal/conf/conf.proto)
- [main.go](file://cmd/kratos-boilerplate/main.go)

**Section sources**
- [config.yaml](file://configs/config.yaml)
- [features.yaml](file://configs/features.yaml)
- [main.go](file://cmd/kratos-boilerplate/main.go)

## Core Configuration Components
The configuration system is built around Protocol Buffers (Proto) definitions that provide strong typing and validation. The `Bootstrap` message in `conf.proto` defines the root configuration structure, which is loaded from YAML files and scanned into the Proto structure. This approach ensures type safety and provides clear documentation of all configuration options.

```go
// Bootstrap configuration structure
message Bootstrap {
  Server server = 1;
  Data data = 2;
  Auth auth = 3;
  Features features = 4;
}
```

The main application entry point in `main.go` uses the Kratos config package to load configuration from a file source, then scans it into the Bootstrap structure:

```go
c := config.New(
    config.WithSource(
        file.NewSource(flagconf),
    ),
)
defer c.Close()

if err := c.Load(); err != nil {
    panic(err)
}

var bc conf.Bootstrap
if err := c.Scan(&bc); err != nil {
    panic(err)
}
```

This pattern ensures that configuration loading failures are caught early during application startup, preventing runtime errors due to invalid configuration.

**Section sources**
- [conf.proto](file://internal/conf/conf.proto#L5-L9)
- [main.go](file://cmd/kratos-boilerplate/main.go#L53-L70)

## Multi-Environment Configuration Strategy
The project implements a sophisticated multi-environment configuration strategy using a base configuration file (`features.yaml`) with environment-specific overrides (`features.dev.yaml`, `features.prod.yaml`). This approach allows teams to maintain consistent feature definitions while customizing behavior for different deployment environments.

The configuration hierarchy follows these rules:
1. Base configuration in `features.yaml` provides default values
2. Environment-specific files override base configuration
3. Runtime environment variables can override file configuration
4. Feature toggle system allows dynamic changes without restart

For example, the Swagger UI is enabled in development but disabled in production:

```yaml
# features.dev.yaml
swagger_ui:
  enabled: true
  description: "Development environment enables Swagger UI"

# features.prod.yaml  
swagger_ui:
  enabled: false
  description: "Production environment disables Swagger UI"
```

Similarly, rate limiting is disabled in development for testing convenience but enabled in production for protection:

```yaml
# features.dev.yaml
rate_limit:
  enabled: false
  description: "Development disables rate limiting for testing"

# features.prod.yaml
rate_limit:
  enabled: true
  description: "Production enables rate limiting"
```

This strategy enables safe feature development and testing while maintaining production stability and security.

**Section sources**
- [features.yaml](file://configs/features.yaml)
- [features.dev.yaml](file://configs/features.dev.yaml)
- [features.prod.yaml](file://configs/features.prod.yaml)

## Feature Toggle System Architecture
The feature toggle system is implemented as a modular component with clear separation between configuration storage, evaluation logic, and application integration. The architecture follows the dependency injection pattern using Wire, with components defined in the `internal/pkg/feature/` package.

```mermaid
classDiagram
class FeatureToggle {
<<interface>>
+IsEnabled(flag FeatureFlag) bool
+IsEnabledWithContext(ctx context.Context, flag FeatureFlag, context *EvaluationContext) bool
+EnableFeature(flag FeatureFlag) error
+DisableFeature(flag FeatureFlag) error
+GetToggleConfig(flag FeatureFlag) (*ToggleConfig, error)
+Subscribe(callback ToggleChangeCallback) error
}
class ToggleManager {
-toggles map[FeatureFlag]*ToggleConfig
-repository FeatureRepository
-strategyEvaluator StrategyEvaluator
-callbacks []ToggleChangeCallback
+Initialize(ctx context.Context) error
+watchConfigChanges(ctx context.Context)
+GetStats() map[string]interface{}
}
class FeatureRepository {
<<interface>>
+LoadToggles(ctx context.Context) (map[FeatureFlag]*ToggleConfig, error)
+SaveToggle(ctx context.Context, flag FeatureFlag, config *ToggleConfig) error
+DeleteToggle(ctx context.Context, flag FeatureFlag) error
+Watch(ctx context.Context, callback func()) error
}
class FileRepository {
-configPath string
-format string
+LoadToggles(ctx context.Context) (map[FeatureFlag]*ToggleConfig, error)
+Watch(ctx context.Context, callback func()) error
}
class StrategyEvaluator {
<<interface>>
+Evaluate(flag FeatureFlag, config *ToggleConfig, context *EvaluationContext) bool
+SupportsStrategy(strategy FeatureStrategy) bool
}
FeatureToggle <|-- ToggleManager
FeatureRepository <|-- FileRepository
ToggleManager --> FeatureRepository : "uses"
ToggleManager --> StrategyEvaluator : "uses"
ToggleManager --> "Logger" : "uses"
```

**Diagram sources**
- [toggle_manager.go](file://internal/pkg/feature/toggle_manager.go#L15-L35)
- [file_repository.go](file://internal/pkg/feature/file_repository.go#L10-L15)
- [interfaces.go](file://internal/pkg/feature/interfaces.go#L5-L20)

**Section sources**
- [toggle_manager.go](file://internal/pkg/feature/toggle_manager.go)
- [file_repository.go](file://internal/pkg/feature/file_repository.go)
- [interfaces.go](file://internal/pkg/feature/interfaces.go)

## Configuration Validation and Schema Management
The configuration system implements multiple layers of validation to ensure correctness and backward compatibility. The Proto-based configuration schema provides compile-time type safety, while runtime validation occurs during configuration loading.

The `Features` message in `conf.proto` defines the schema for feature toggle configuration:

```protobuf
message Features {
  bool enabled = 1;
  string config_file = 2;
  string config_format = 3;
  bool watch_config = 4;
  string default_environment = 5;
  FeatureRepository repository = 6;
}

message FeatureRepository {
  string type = 1;  // "file", "redis", "database"
  string config_path = 2;
  string format = 3;  // "yaml", "json"
}
```

The feature toggle system includes default value handling to ensure backward compatibility:

```go
// Fill default values
if config.ConfigFile == "" {
    config.ConfigFile = "./configs/features.yaml"
}
if config.ConfigFormat == "" {
    config.ConfigFormat = "yaml"
}
if config.DefaultEnvironment == "" {
    config.DefaultEnvironment = "production"
}
```

Each feature configuration includes metadata fields like `created_at`, `updated_at`, and `tags` to support auditing and discovery. The system also supports configuration watching, enabling automatic reload when configuration files change:

```go
// Start configuration watcher
go tm.watchConfigChanges(ctx)
```

This approach allows teams to implement automated configuration validation in CI/CD pipelines using schema validation tools and linting rules.

**Section sources**
- [conf.proto](file://internal/conf/conf.proto#L45-L68)
- [wire.go](file://internal/pkg/feature/wire.go#L81-L100)
- [toggle_manager.go](file://internal/pkg/feature/toggle_manager.go#L50-L55)

## Secure Handling of Sensitive Data
The configuration system follows security best practices by separating sensitive data from code and implementing secure defaults. Database credentials and JWT secrets are stored in configuration files that should be protected through access controls and encryption at rest.

The main configuration file contains database connection details:

```yaml
data:
  database:
    driver: postgres
    source: postgresql://postgres:postgres@cross-redline-db:5432/test?sslmode=disable
  redis:
    addr: 127.0.0.1:6379
```

For production deployments, these values should be overridden using environment variables or secret management systems. The architecture supports this through the Kratos config package's ability to merge multiple configuration sources.

The feature toggle system also includes security-related features like plugin sandboxing:

```yaml
plugins:
  security:
    sandbox_enabled: true
    max_memory: "100MB"
    max_cpu_percent: 10
```

These settings help prevent malicious or poorly written plugins from compromising system stability. Teams should implement additional security measures including:
- Encrypting configuration files at rest
- Using secret management systems (e.g., Hashicorp Vault)
- Implementing strict access controls
- Regular security audits of configuration changes
- Automated scanning for secrets in configuration files

**Section sources**
- [config.yaml](file://configs/config.yaml#L10-L15)
- [config.yaml](file://configs/config.yaml#L25-L30)
- [config.yaml](file://configs/config.yaml#L20-L24)

## Configuration Loading and Caching Performance
The configuration system is designed for high performance with efficient loading and caching strategies. Configuration is loaded once at startup and cached in memory, eliminating the need for repeated file I/O during request processing.

The `ToggleManager` maintains an in-memory cache of feature configurations:

```go
type ToggleManager struct {
    mu                sync.RWMutex
    toggles           map[FeatureFlag]*ToggleConfig
    // ... other fields
}
```

The use of `sync.RWMutex` allows multiple concurrent readers (feature checks) while ensuring exclusive access during configuration updates. This design supports high-throughput services with minimal performance impact.

Configuration loading follows an optimized sequence:
1. Load configuration from file source
2. Scan into Proto structure
3. Initialize feature toggle system
4. Start configuration watcher (optional)

The system also supports configuration watching for dynamic updates without restart:

```go
// Start configuration watcher
go tm.watchConfigChanges(ctx)
```

For high-throughput services, teams should consider additional caching strategies:
- Implement TTL-based cache invalidation
- Use distributed caching for multi-instance deployments
- Pre-load configurations during application startup
- Monitor configuration access patterns for optimization

The current implementation provides excellent performance for most use cases, with feature checks executing in O(1) time due to the in-memory map storage.

**Section sources**
- [toggle_manager.go](file://internal/pkg/feature/toggle_manager.go#L15-L20)
- [main.go](file://cmd/kratos-boilerplate/main.go#L53-L70)
- [toggle_manager.go](file://internal/pkg/feature/toggle_manager.go#L50-L55)

## Monitoring and Auditing Configuration Changes
The feature toggle system includes comprehensive monitoring and auditing capabilities to track configuration changes and ensure accountability. The system emits events for all configuration modifications, enabling real-time monitoring and alerting.

The `ToggleManager` supports event subscription for configuration changes:

```go
// Subscribe to configuration change events
func (tm *ToggleManager) Subscribe(callback ToggleChangeCallback) error {
    if callback == nil {
        return errors.New("callback cannot be nil")
    }
    tm.mu.Lock()
    defer tm.mu.Unlock()
    tm.callbacks = append(tm.callbacks, callback)
    return nil
}
```

Clients can register callbacks to receive notifications when features are enabled, disabled, or modified:

```go
callback := func(flag FeatureFlag, oldConfig, newConfig *ToggleConfig) {
    // Log configuration change
    tm.logger.Infof("Feature %s changed: %v -> %v", flag, oldConfig, newConfig)
}
```

The system also provides operational metrics through the `GetStats()` method:

```go
func (tm *ToggleManager) GetStats() map[string]interface{} {
    tm.mu.RLock()
    defer tm.mu.RUnlock()
    
    stats := map[string]interface{}{
        "total_toggles":   len(tm.toggles),
        "enabled_count":   0,
        "disabled_count":  0,
        "strategies":      make(map[FeatureStrategy]int),
    }
    // ... populate stats
    return stats
}
```

These capabilities enable teams to implement comprehensive auditing solutions, including:
- Integration with centralized logging systems
- Alerting on critical configuration changes
- Regular audit reports
- Compliance tracking
- Change impact analysis

**Section sources**
- [toggle_manager.go](file://internal/pkg/feature/toggle_manager.go#L130-L145)
- [toggle_manager.go](file://internal/pkg/feature/toggle_manager.go#L300-L320)
- [toggle_manager_test.go](file://internal/pkg/feature/toggle_manager_test.go#L360-L371)

## Rollback and Pre-Deployment Validation
The configuration system supports safe deployment practices through feature toggles and pre-deployment validation. Teams can implement blue-green deployments, canary releases, and instant rollback using the dynamic feature management capabilities.

The feature toggle strategies enable various deployment patterns:
- **Simple**: Binary on/off control
- **Percentage**: Gradual rollout to percentage of users
- **User**: Target specific user types or attributes
- **Time**: Schedule feature activation/deactivation
- **Environment**: Control by deployment environment

Pre-deployment validation should include:
1. Schema validation against `conf.proto`
2. Linting for configuration consistency
3. Security scanning for exposed secrets
4. Integration testing with target environment configuration
5. Performance testing with production-like configuration

The system supports automated pre-deployment checks through the configuration loading process:

```go
// Load and validate configuration
if err := c.Load(); err != nil {
    panic(err) // Fail fast on invalid configuration
}

var bc conf.Bootstrap
if err := c.Scan(&bc); err != nil {
    panic(err) // Fail on type mismatch
}
```

For rollback procedures, teams should:
1. Maintain versioned configuration files in source control
2. Implement automated rollback scripts
3. Test rollback procedures regularly
4. Monitor system behavior after rollback
5. Document rollback decision criteria

The combination of feature toggles and environment-specific configuration enables instant rollback by simply disabling a feature or reverting to a previous configuration file version.

**Section sources**
- [main.go](file://cmd/kratos-boilerplate/main.go#L60-L70)
- [features.yaml](file://configs/features.yaml#L150-L165)
- [wire.go](file://internal/pkg/feature/wire.go#L81-L100)