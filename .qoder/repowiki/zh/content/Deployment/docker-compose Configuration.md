# docker-compose Configuration

<cite>
**Referenced Files in This Document**   
- [docker-compose.yml](file://docker-compose.yml)
- [docker-compose.dev.yml](file://docker-compose.dev.yml)
- [test/config/docker-compose.test.yml](file://test/config/docker-compose.test.yml)
- [Dockerfile](file://Dockerfile)
- [Dockerfile.dev](file://Dockerfile.dev)
- [configs/config.yaml](file://configs/config.yaml)
- [test/config/test.yaml](file://test/config/test.yaml)
</cite>

## Table of Contents
1. [Introduction](#introduction)
2. [Project Structure Overview](#project-structure-overview)
3. [Production Configuration (docker-compose.yml)](#production-configuration-docker-composeyml)
4. [Development Configuration (docker-compose.dev.yml)](#development-configuration-docker-composedevyml)
5. [Testing Configuration (docker-compose.test.yml)](#testing-configuration-docker-composetestyml)
6. [Service Definitions and Dependencies](#service-definitions-and-dependencies)
7. [Networking and Volume Management](#networking-and-volume-management)
8. [Environment Variables and Configuration Injection](#environment-variables-and-configuration-injection)
9. [Extending Configurations for Staging Environments](#extending-configurations-for-staging-environments)
10. [Plugin Management and Feature Toggles](#plugin-management-and-feature-toggles)
11. [Resource Limits and Performance Tuning](#resource-limits-and-performance-tuning)
12. [Local Development Overrides](#local-development-overrides)
13. [Troubleshooting Common Issues](#troubleshooting-common-issues)
14. [Conclusion](#conclusion)

## Introduction
This document provides a comprehensive analysis of the Docker Compose configurations used in the kratos-boilerplate repository. It compares and contrasts the production, development, and testing setups, detailing service definitions, networking, volume mounting, environment variable injection, and dependency ordering. The goal is to provide clear guidance for developers and DevOps engineers on how to manage, extend, and troubleshoot these configurations effectively.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)
- [test/config/docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)

## Project Structure Overview
The project follows a modular structure with distinct directories for API definitions, business logic, data access, configuration, frontend assets, and testing utilities. The root directory contains Docker-related files including multiple Docker Compose files for different environments and Dockerfiles for building container images.

Key components relevant to Docker Compose include:
- `docker-compose.yml`: Production environment configuration
- `docker-compose.dev.yml`: Development environment configuration
- `test/config/docker-compose.test.yml`: Testing environment configuration
- `Dockerfile`: Production image build instructions
- `Dockerfile.dev`: Development image build instructions
- `configs/`: Application configuration files
- `test/config/`: Test-specific configuration files

```mermaid
graph TB
subgraph "Root"
docker_compose_prod["docker-compose.yml"]
docker_compose_dev["docker-compose.dev.yml"]
Dockerfile_prod["Dockerfile"]
Dockerfile_dev["Dockerfile.dev"]
end
subgraph "Configs"
config_yaml["config.yaml"]
features_yaml["features.yaml"]
end
subgraph "Test"
test_compose["test/config/docker-compose.test.yml"]
test_yaml["test/config/test.yaml"]
end
docker_compose_prod --> |uses| Dockerfile_prod
docker_compose_dev --> |uses| Dockerfile_dev
docker_compose_prod --> |mounts| config_yaml
test_compose --> |mounts| test_yaml
```

**Diagram sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)
- [test/config/docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)
- [Dockerfile](file://Dockerfile#L1-L57)
- [Dockerfile.dev](file://Dockerfile.dev#L1-L71)

## Production Configuration (docker-compose.yml)
The production configuration (`docker-compose.yml`) defines two primary services: the main application (`app`) and PostgreSQL database (`db`). This setup prioritizes stability, security, and minimal attack surface.

### Service: app
- **Build**: Uses the standard `Dockerfile` with context set to the project root
- **Container Name**: `cross-redline-app`
- **Ports**: Maps HTTP (8000) and gRPC (9000) ports
- **Volumes**: Mounts `./configs` to `/data/conf` for external configuration
- **Dependencies**: Waits for database health check before starting
- **Environment**: Sets timezone to Asia/Shanghai
- **Network**: Connects to `cross-redline-network`

### Service: db
- **Image**: Uses official PostgreSQL 14 image
- **Container Name**: `cross-redline-db`
- **Restart Policy**: Always restart on failure
- **Environment**: Pre-configured credentials and database name
- **Ports**: Exposes PostgreSQL port 5432
- **Volumes**: Persistent storage via named volume `postgres_data`
- **Health Check**: Uses `pg_isready` to verify database readiness
- **Network**: Connects to `cross-redline-network`

The configuration also defines a local volume driver for PostgreSQL data persistence and a bridge network for inter-service communication.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)

## Development Configuration (docker-compose.dev.yml)
The development configuration (`docker-compose.dev.yml`) extends the production setup with features optimized for developer productivity and debugging.

### Key Differences from Production:
- **Service Name**: Renamed to `dev` instead of `app`
- **Dockerfile**: Uses `Dockerfile.dev` which includes development tools
- **Container Name**: `cross-redline-dev`
- **Additional Port**: Exposes SSH (port 2222) for remote debugging
- **Volume Mounts**: 
  - Full project directory mounted to `/app` for live code reloading
  - Go module cache volume for faster builds
  - SSH public key mounted for passwordless authentication
- **TTY and STDIN**: Enabled for interactive container access

### Development-Specific Features:
The `Dockerfile.dev` includes:
- SSH server for remote debugging
- Development tools (vim, curl, protobuf compiler)
- Go development utilities (wire, protoc-gen-go, kratos CLI)
- Proper user setup (`dev` user) with appropriate permissions

This configuration enables hot-reloading of code changes and provides developers with a fully-featured development environment inside the container.

```mermaid
sequenceDiagram
participant Dev as "Developer"
participant Docker as "Docker Host"
participant Container as "dev Container"
Dev->>Docker : docker-compose -f docker-compose.dev.yml up
Docker->>Container : Create container from Dockerfile.dev
Container->>Container : Start SSH daemon
Container->>Container : Mount project directory
Container->>Container : Keep container running
Dev->>Container : SSH to localhost : 2222
Container-->>Dev : Interactive shell access
Dev->>Container : Edit code in mounted volume
Container->>Container : Detect file changes (hot reload)
Container->>Dev : Serve updated application
```

**Diagram sources**
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)
- [Dockerfile.dev](file://Dockerfile.dev#L1-L71)

**Section sources**
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)
- [Dockerfile.dev](file://Dockerfile.dev#L1-L71)

## Testing Configuration (docker-compose.test.yml)
The testing configuration (`test/config/docker-compose.test.yml`) provides a comprehensive environment for running automated tests, including end-to-end, integration, and database migration testing.

### Services Overview:
#### test-db
- **Image**: PostgreSQL 15-alpine (newer than production)
- **Container Name**: `kratos-test-db`
- **Port**: Maps to host port 5433 (avoids conflict with dev DB)
- **Volumes**: 
  - Named volume `test_postgres_data` for data persistence
  - Migrations directory mounted for automatic schema setup
- **Health Check**: Verifies connection to `test_db`
- **Network**: `test-network`

#### test-redis
- **Image**: Redis 7-alpine
- **Container Name**: `kratos-test-redis`
- **Port**: Maps to host port 6380
- **Command**: Enables AOF persistence
- **Volume**: Named volume `test_redis_data`
- **Health Check**: Pings Redis server
- **Network**: `test-network`

#### test-app
- **Build**: Uses `Dockerfile.dev` from project root
- **Container Name**: `kratos-test-app`
- **Ports**: Maps to 8001 (HTTP) and 9001 (gRPC)
- **Environment**: Points to test configuration file
- **Volumes**: Mounts test config directory
- **Dependencies**: Waits for both test-db and test-redis
- **Health Check**: Probes `/health` endpoint with generous timeouts
- **Network**: `test-network`

#### test-init
- **Image**: PostgreSQL 15-alpine
- **Container Name**: `kratos-test-init`
- **Volumes**: Migrations and test data SQL file
- **Command**: Script that waits for DB, runs migrations, and loads test data
- **Dependencies**: Waits for test-db health
- **Network**: `test-network`

This configuration creates an isolated testing environment with its own database schema (`test_db`), separate ports to avoid conflicts, and automated data initialization for consistent test runs.

```mermaid
flowchart TD
Start([Start Test Environment]) --> LaunchDB["Launch test-db service"]
LaunchDB --> LaunchRedis["Launch test-redis service"]
LaunchRedis --> LaunchInit["Launch test-init service"]
LaunchInit --> WaitForDB["Wait for database readiness"]
WaitForDB --> RunMigrations["Execute migration scripts"]
RunMigrations --> LoadTestData["Load test data from SQL file"]
LoadTestData --> CompleteInit["Initialization complete"]
CompleteInit --> LaunchApp["Launch test-app service"]
LaunchApp --> HealthCheck["Perform application health check"]
HealthCheck --> Ready["Test environment ready"]
```

**Diagram sources**
- [test/config/docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)

**Section sources**
- [test/config/docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)

## Service Definitions and Dependencies
The Docker Compose configurations define several services across different environments, each with specific roles and dependencies.

### Common Services
| Service | Purpose | Environments | Key Dependencies |
|--------|--------|-------------|------------------|
| app/dev/test-app | Main application server | Production, Development, Testing | Database, Redis (indirectly) |
| db/test-db | PostgreSQL database | Production, Development, Testing | None |
| test-redis | Redis cache/server | Testing only | None |
| test-init | Data initialization | Testing only | test-db |

### Dependency Ordering
All configurations implement proper startup ordering using health checks:

```mermaid
graph LR
DB[Database] --> |healthy| APP[Application]
REDIS[Redis] --> |healthy| INIT[Init Service]
INIT --> |complete| APP
APP --> |running| HEALTH[Health Check]
```

In production and development, the application waits for the database to be healthy before starting. In testing, both the database and Redis must be healthy before the initialization service runs, and the test application waits for both database and Redis to be healthy.

The `depends_on` condition with `service_healthy` ensures that services start in the correct order and that dependent services only start when their dependencies are fully operational, not just running.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)
- [test/config/docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)

## Networking and Volume Management
The Docker Compose configurations implement isolated networking and persistent storage strategies appropriate for each environment.

### Network Configuration
Each configuration defines a dedicated bridge network:

- **Production & Development**: `cross-redline-network`
- **Testing**: `test-network`

These networks enable secure communication between services within the same environment while isolating them from other containers on the host. Service discovery is handled by Docker's built-in DNS, allowing services to communicate using their service names as hostnames (e.g., `cross-redline-db` for the database).

### Volume Management
Three types of volumes are used:

#### Named Volumes (Persistent Data)
- `postgres_data`: For PostgreSQL data in production/development
- `test_postgres_data`: For test database data
- `test_redis_data`: For Redis data in testing
- `go-cache`: For Go module cache in development

These use the local driver and persist data between container restarts.

#### Bind Mounts (Configuration and Code)
- `./configs:/data/conf`: Mounts configuration files into production containers
- `../../test/config:/app/test/config`: Mounts test configuration
- `.:app`: Full project mount for development (hot reloading)
- `~/.ssh/id_rsa.pub:/home/dev/.ssh/authorized_keys`: SSH key for development access
- `../../migrations:/docker-entrypoint-initdb.d`: Database migrations for testing

#### Anonymous Volumes
- None explicitly defined, but Docker may create anonymous volumes for certain operations.

The volume strategy ensures configuration can be externalized, code changes are reflected immediately in development, and critical data persists across container lifecycles.

```mermaid
classDiagram
class Volume {
+string name
+string driver
+map[string]string labels
+create() Volume
+remove() void
}
class NamedVolume {
+string name
+string driver = "local"
+bool persistent = true
+create() Volume
+remove() void
}
class BindMount {
+string source_path
+string container_path
+string mode = "rw"
+create() Volume
+remove() void
}
Volume <|-- NamedVolume
Volume <|-- BindMount
NamedVolume : - postgres_data
NamedVolume : - test_postgres_data
NamedVolume : - test_redis_data
NamedVolume : - go-cache
BindMount : - ./configs : /data/conf
BindMount : - . : /app
BindMount : - ../../test/config : /app/test/config
```

**Diagram sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)
- [test/config/docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)
- [test/config/docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)

## Environment Variables and Configuration Injection
The configurations use environment variables and file mounting to inject settings into containers.

### Environment Variables
Common environment variables across configurations:
- `TZ=Asia/Shanghai`: Sets timezone for all services
- Database credentials (POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB)

Environment-specific variables:
- Testing: `CONFIG_PATH=/app/test/config/test.yaml` to specify test configuration

### Configuration File Injection
Configuration is primarily injected through volume mounts:

#### Production & Development
```yaml
volumes:
  - ./configs:/data/conf
```
The application is configured (via Dockerfile) to look for configuration in `/data/conf`, which is mounted from the local `configs` directory.

#### Testing
```yaml
volumes:
  - ../../test/config:/app/test/config
```
The test application uses a specific configuration file path set by the `CONFIG_PATH` environment variable.

The `config.yaml` file in the repository defines connection strings using service names that resolve via Docker's internal DNS:
```yaml
data:
  database:
    source: postgresql://postgres:postgres@cross-redline-db:5432/test?sslmode=disable
  redis:
    addr: 127.0.0.1:6379
```

Note that in testing, the configuration changes to use localhost since the test-app container accesses services via host port mappings rather than Docker network.

The test configuration (`test.yaml`) overrides various settings appropriate for testing, including JWT secrets, captcha settings, and test-specific timeouts and concurrency limits.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)
- [test/config/docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)
- [configs/config.yaml](file://configs/config.yaml#L1-L38)
- [test/config/test.yaml](file://test/config/test.yaml#L1-L41)

## Extending Configurations for Staging Environments
While not explicitly provided, the existing configurations can be extended to create a staging environment that bridges the gap between development and production.

### Recommended Approach: Docker Compose Extension
Create `docker-compose.staging.yml` that extends the production configuration:

```yaml
# docker-compose.staging.yml
version: "3.8"

services:
  app:
    # Use production image but with staging-specific settings
    environment:
      - TZ=Asia/Shanghai
      - STAGING_MODE=true
    # Additional monitoring ports if needed
    ports:
      - "8000:8000"
      - "9000:9000"
      - "6060:6060" # Go pprof for performance monitoring
    # Slightly more resources than production
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 1G

  db:
    # Use same version as production
    image: postgres:14
    environment:
      POSTGRES_DB: staging
      # Use secrets for credentials in real implementation
      POSTGRES_USER: staging_user
      POSTGRES_PASSWORD: staging_password
    # Larger volume for staging data
    volumes:
      - staging_postgres_data:/var/lib/postgresql/data
    # Same health check as production
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U staging_user -d staging"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  staging_postgres_data:
    driver: local

networks:
  cross-redline-network:
    external: true
```

### Usage
```bash
# Start staging environment
docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d

# Stop staging environment
docker-compose -f docker-compose.yml -f docker-compose.staging.yml down
```

### Key Staging Considerations
1. **Database**: Use a dedicated database name (`staging`) to avoid data contamination
2. **Credentials**: Use staging-specific credentials, preferably managed via secrets
3. **Monitoring**: Expose additional ports for profiling and monitoring tools
4. **Resource Limits**: Set realistic limits closer to production
5. **External Network**: Reuse the production network if appropriate, or create a dedicated one
6. **Configuration**: Mount staging-specific configuration files
7. **Backups**: Implement regular backup procedures for staging data

This approach maintains consistency with production while allowing for staging-specific customizations.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)

## Plugin Management and Feature Toggles
The application supports dynamic plugin loading and feature toggles, which are configured through the Docker Compose files and configuration files.

### Plugin Configuration
From `configs/config.yaml`:
```yaml
plugins:
  enabled: true
  directory: "./plugins"
  config_directory: "./configs/plugins"
  auto_load: true
  security:
    sandbox_enabled: true
    max_memory: "100MB"
    max_cpu_percent: 10
```

The Docker Compose configurations ensure plugins are accessible:
- In production: The entire `configs` directory is mounted, which includes `configs/plugins`
- The `plugins` directory containing Go plugin files is included in the Docker image build

### Feature Toggles
The configuration supports feature toggles via:
```yaml
features:
  enabled: true
  config_file: "./configs/features.yaml"
  config_format: "yaml"
  watch_config: true
  default_environment: "production"
  repository:
    type: "file"
    config_path: "./configs/features.yaml"
    format: "yaml"
```

Different feature configurations can be injected per environment:
- `features.yaml`: Default features
- `features.dev.yaml`: Development-specific feature overrides
- `features.prod.yaml`: Production-specific feature settings

To enable/disable plugins or features in different environments:
1. **Development**: Mount development-specific feature files
2. **Testing**: Use test configuration to enable test-specific features
3. **Production**: Ensure only approved features are enabled

Example override in `docker-compose.dev.yml`:
```yaml
services:
  dev:
    volumes:
      - ./configs/features.dev.yaml:/data/conf/features.yaml
```

This allows developers to enable experimental features without affecting other environments.

**Section sources**
- [configs/config.yaml](file://configs/config.yaml#L1-L38)
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)

## Resource Limits and Performance Tuning
While the current configurations don't explicitly define resource limits, they can be enhanced for production use.

### Current Resource Usage
The configurations rely on default Docker resource allocation, which allows containers to use available host resources without limits.

### Recommended Production Resource Limits
Add resource constraints to the production configuration:

```yaml
services:
  app:
    # ... existing configuration
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 1024M
        reservations:
          cpus: '0.2'
          memory: 512M
    
  db:
    # ... existing configuration
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 2048M
        reservations:
          cpus: '0.5'
          memory: 1024M
```

### Performance Considerations
1. **Database**: PostgreSQL benefits from sufficient memory for caching
2. **Application**: The Go application is memory-efficient but may need CPU limits to prevent overutilization
3. **Redis**: Not used in production configuration but would need appropriate memory allocation

### Monitoring and Tuning
Implement monitoring to determine optimal resource allocation:
- Use Docker stats or monitoring tools to observe CPU and memory usage
- Adjust limits based on observed usage patterns
- Consider horizontal scaling for high-load scenarios

For development, resource limits are typically not needed as the focus is on developer productivity rather than resource efficiency.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)

## Local Development Overrides
Developers can customize the development environment using Docker Compose override mechanisms.

### Method 1: docker-compose.override.yml
Create `docker-compose.override.yml` in the project root:

```yaml
version: '3.8'

services:
  dev:
    environment:
      - CUSTOM_DEBUG=true
      - LOG_LEVEL=debug
    volumes:
      - ./local-configs:/data/conf/local
    ports:
      - "8080:8080" # Additional service port
```

This file is automatically picked up by `docker-compose` when running commands.

### Method 2: Custom Override File
Create `docker-compose.local.yml` for local customizations:

```yaml
version: '3.8'

services:
  dev:
    # Enable additional debugging tools
    cap_add:
      - SYS_PTRACE
    security_opt:
      - seccomp:unconfined
    environment:
      - GODEBUG=schedtrace=1000
      - GOTRACEBACK=crash
```

Use with:
```bash
docker-compose -f docker-compose.dev.yml -f docker-compose.local.yml up
```

### Method 3: Environment Variables
Override specific settings using environment variables:

```bash
# Override database password
POSTGRES_PASSWORD=mypassword docker-compose -f docker-compose.dev.yml up

# Override timezone
TZ=America/New_York docker-compose -f docker-compose.dev.yml up
```

### Useful Development Overrides
- **Increased Logging**: Set log levels to debug
- **Additional Ports**: Expose ports for debugging tools (pprof, trace)
- **Custom Configuration**: Mount local configuration files
- **Debug Capabilities**: Add ptrace capability for debugging
- **Resource Limits**: Add limits to simulate production conditions

These overrides allow developers to customize their environment without modifying the base configuration files.

**Section sources**
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)

## Troubleshooting Common Issues
This section addresses common Docker Compose issues and their solutions.

### Container Startup Failures
**Symptom**: Container exits immediately after startup
**Diagnosis**:
```bash
docker-compose -f docker-compose.dev.yml logs dev
```
**Common Causes**:
1. Missing dependencies (database not ready)
2. Configuration file errors
3. Permission issues with mounted volumes

**Solution**: Check logs and verify:
- Database is healthy before application starts
- Configuration files are valid YAML
- Mounted directories have correct permissions

### Network Connectivity Problems
**Symptom**: Application cannot connect to database
**Diagnosis**:
```bash
docker-compose -f docker-compose.dev.yml exec dev ping cross-redline-db
docker-compose -f docker-compose.dev.yml exec dev nc -zv cross-redline-db 5432
```
**Common Causes**:
1. Incorrect service name in connection string
2. Database not fully initialized
3. Network configuration issues

**Solution**: Verify:
- Service names match in `docker-compose.yml` and connection strings
- Use health checks for proper startup ordering
- Services are on the same network

### Volume Mounting Issues
**Symptom**: Code changes not reflected in container
**Diagnosis**:
```bash
docker-compose -f docker-compose.dev.yml exec dev ls -la /app
```
**Common Causes**:
1. Incorrect mount path
2. File permission issues
3. Docker Desktop file sharing not enabled (on macOS/Windows)

**Solution**: Verify:
- Mount path is correct in `docker-compose.dev.yml`
- File permissions allow container user to read files
- Docker Desktop settings include project directory in file sharing

### Health Check Failures
**Symptom**: Service reports unhealthy status
**Diagnosis**:
```bash
docker-compose -f docker-compose.yml ps
docker-compose -f docker-compose.yml inspect <container_name>
```
**Common Causes**:
1. Health check command fails
2. Service takes longer to start than health check timeout
3. Dependencies not available

**Solution**: Adjust health check parameters:
```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U postgres"]
  interval: 10s
  timeout: 10s
  retries: 10
  start_period: 30s
```

### Port Conflicts
**Symptom**: "Port already allocated" error
**Diagnosis**:
```bash
lsof -i :5432  # Check what's using port 5432
```
**Common Causes**:
1. Multiple compose files using same ports
2. Host services using the same ports
3. Previous containers not properly cleaned up

**Solution**: 
- Use different ports for different environments
- Stop conflicting services
- Clean up old containers: `docker-compose -f docker-compose.dev.yml down`

### Build Failures
**Symptom**: Image build fails during `docker-compose up`
**Diagnosis**:
```bash
docker-compose -f docker-compose.yml build --no-cache
```
**Common Causes**:
1. Network issues downloading packages
2. Missing build dependencies
3. Code compilation errors

**Solution**: Verify:
- Internet connection for package downloads
- Required tools are in Dockerfile
- Code compiles locally

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)
- [test/config/docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)

## Conclusion
The kratos-boilerplate repository provides a well-structured set of Docker Compose configurations for production, development, and testing environments. Each configuration is tailored to its specific use case:

- **Production** (`docker-compose.yml`): Focuses on stability and security with minimal attack surface
- **Development** (`docker-compose.dev.yml`): Optimized for developer productivity with hot reloading and debugging capabilities
- **Testing** (`docker-compose.test.yml`): Comprehensive environment for automated testing with isolated services and data initialization

Key strengths include proper dependency ordering using health checks, consistent networking patterns, and flexible configuration injection. The configurations can be extended for staging environments and customized for local development needs.

To further improve the setup, consider:
1. Adding explicit resource limits for production
2. Implementing secrets management for sensitive data
3. Adding monitoring and logging infrastructure
4. Creating a staging configuration that bridges development and production
5. Documenting the complete development workflow

These configurations provide a solid foundation for deploying and developing the application across different environments.