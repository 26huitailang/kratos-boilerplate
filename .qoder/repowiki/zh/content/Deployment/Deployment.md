# Deployment

<cite>
**Referenced Files in This Document**   
- [Dockerfile](file://Dockerfile)
- [docker-compose.yml](file://docker-compose.yml)
- [docker-compose.dev.yml](file://docker-compose.dev.yml)
- [Makefile](file://Makefile)
- [config.yaml](file://configs/config.yaml)
- [features.dev.yaml](file://configs/features.dev.yaml)
- [features.prod.yaml](file://configs/features.prod.yaml)
</cite>

## Table of Contents
1. [Introduction](#introduction)
2. [Project Structure Overview](#project-structure-overview)
3. [Docker-Based Deployment Process](#docker-based-deployment-process)
4. [Production vs Development Configuration](#production-vs-development-configuration)
5. [Build Process and Makefile Commands](#build-process-and-makefile-commands)
6. [Container Orchestration with Docker Compose](#container-orchestration-with-docker-compose)
7. [Environment Variables and Secret Management](#environment-variables-and-secret-management)
8. [Logging, Monitoring, and Health Checks](#logging-monitoring-and-health-checks)
9. [Deployment to Different Environments](#deployment-to-different-environments)
10. [Zero-Downtime Deployment Strategies](#zero-downtime-deployment-strategies)
11. [Performance Tuning and Resource Requirements](#performance-tuning-and-resource-requirements)
12. [Rollback Procedures](#rollback-procedures)

## Introduction
This document provides comprehensive deployment guidance for the **kratos-boilerplate** project, a Go-based microservices template built using the Kratos framework. The system supports both production and development deployments via Docker and Docker Compose, with distinct configurations for each environment. This guide details the full deployment lifecycle, including build processes, container orchestration, environment configuration, scaling strategies, and operational best practices.

The deployment model leverages multi-stage Docker builds to produce lean production images while providing a rich development container with SSH access and hot-reloading capabilities. The architecture integrates PostgreSQL for persistent storage and supports feature toggling through external configuration files.

**Section sources**
- [README.md](file://README.md#L1-L92)

## Project Structure Overview
The kratos-boilerplate follows a modular structure aligned with Kratos framework conventions:
- **api/**: Protocol Buffer definitions for gRPC and HTTP APIs
- **cmd/kratos-boilerplate/**: Application entry point and dependency injection (Wire)
- **configs/**: Configuration files for different environments
- **internal/**: Core business logic, data access, services, and middleware
- **frontend/**: Vue.js-based web interface
- **migrations/**: Database schema migration scripts
- **plugins/**: Custom plugin implementations
- **test/**: BDD and integration tests
- **Dockerfile**: Production container build definition
- **Dockerfile.dev**: Development container with tooling
- **docker-compose.yml**: Production orchestration
- **docker-compose.dev.yml**: Development orchestration
- **Makefile**: Build automation and task orchestration

This separation enables independent development and deployment of backend and frontend components while maintaining clear boundaries between configuration, code, and infrastructure.

```mermaid
graph TB
subgraph "Source Code"
API[api/*.proto]
Internal[internal/...]
Frontend[frontend/...]
Configs[configs/...]
end
subgraph "Build Process"
Dockerfile[Dockerfile]
Makefile[Makefile]
DockerfileDev[Dockerfile.dev]
end
subgraph "Deployment"
docker-compose["docker-compose.yml"]
docker-compose-dev["docker-compose.dev.yml"]
end
API --> Dockerfile
Internal --> Dockerfile
Frontend --> Dockerfile
Configs --> docker-compose
Makefile --> Dockerfile
DockerfileDev --> docker-compose-dev
Dockerfile --> docker-compose
```

**Diagram sources**
- [Dockerfile](file://Dockerfile#L1-L57)
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)

## Docker-Based Deployment Process
The deployment process uses Docker to containerize the application, ensuring consistency across environments. The production image is built in two stages: a builder stage that compiles the Go binary and frontend assets, and a minimal runtime stage that includes only the necessary artifacts.

The Docker build process:
1. Uses `openeuler/openeuler:24.03-lts` as base image
2. Installs Go 1.24.1 and Node.js/npm
3. Builds the Go backend using `make build`
4. Builds the Vue.js frontend using `npm run build`
5. Copies binaries and static assets to a clean runtime image

To build and deploy:
```bash
docker build -t kratos-boilerplate:latest .
docker run --rm -p 8000:8000 -p 9000:9000 -v ./configs:/data/conf kratos-boilerplate:latest
```

The container exposes HTTP (8000) and gRPC (9000) endpoints and mounts configuration files from the host system.

**Section sources**
- [Dockerfile](file://Dockerfile#L1-L57)

## Production vs Development Configuration
The system provides separate Docker Compose configurations for production and development environments, reflecting different operational requirements.

### Production Configuration (docker-compose.yml)
Designed for stable, secure, and scalable deployment:
- Single service `app` running the compiled binary
- Configuration mounted from `./configs` to `/data/conf`
- Health check dependency on database
- Network isolation via custom bridge network
- No SSH access or source code mounting

### Development Configuration (docker-compose.dev.yml)
Optimized for developer productivity:
- Service `dev` with full development environment
- Source code mounted at `/app` for live reloading
- SSH access on port 2222 for remote debugging
- Go module cache volume for faster builds
- Development user with SSH key authentication

Key differences:
| Aspect | Production | Development |
|------|------------|-------------|
| Image | Multi-stage build | Full dev tools |
| Code Mount | Binaries only | Full source code |
| SSH Access | No | Yes (port 2222) |
| Build Tools | Not included | Installed |
| Caching | None | Go module cache |

```mermaid
graph TD
A[Development Environment] --> B[Source Code Mounted]
A --> C[SSH Access Enabled]
A --> D[Hot Reload Support]
A --> E[Full Toolchain]
F[Production Environment] --> G[Static Binaries]
F --> H[No SSH Access]
F --> I[Configuration Mount]
F --> J[Health Checks]
B --> K[Live Code Updates]
C --> L[Remote Debugging]
D --> M[Fast Iteration]
E --> N[protoc, wire, etc.]
G --> O[Smaller Attack Surface]
H --> P[Improved Security]
I --> Q[External Config Management]
J --> R[Orchestration Readiness]
```

**Diagram sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [docker-compose.dev.yml](file://docker-compose.dev.yml#L1-L56)

## Build Process and Makefile Commands
The Makefile orchestrates the entire build and test lifecycle, providing standardized commands for development and deployment.

### Key Make Targets
| Target | Purpose | Dependencies |
|--------|-------|--------------|
| `init` | Install development tools | protoc-gen-go, kratos CLI |
| `api` | Generate API code from .proto files | protoc plugins |
| `config` | Generate internal proto bindings | internal/*.proto |
| `build` | Compile Go binary | go build |
| `generate` | Run code generation | go generate |
| `all` | Complete code generation | api, config, generate |
| `test` | Run unit tests | go test |
| `test-coverage` | Generate coverage report | coverage.out |
| `logcheck` | Validate logging compliance | tools/logchecker |

### Build Workflow
```mermaid
flowchart TD
Start([Make all]) --> API["make api\nGenerate API code"]
API --> Config["make config\nGenerate config bindings"]
Config --> Generate["make generate\nRun go generate"]
Generate --> Build["make build\nCompile binary"]
Build --> End([Ready for deployment])
style Start fill:#4CAF50,stroke:#388E3C
style End fill:#4CAF50,stroke:#388E3C
```

**Diagram sources**
- [Makefile](file://Makefile#L1-L141)

**Section sources**
- [Makefile](file://Makefile#L1-L141)

## Container Orchestration with Docker Compose
The system uses Docker Compose to manage multi-container deployments, coordinating the application and database services.

### Production Orchestration (docker-compose.yml)
- **app service**: Main application container
  - Built from Dockerfile
  - Exposes ports 8000 (HTTP) and 9000 (gRPC)
  - Mounts configs directory
  - Depends on healthy database
- **db service**: PostgreSQL 14 database
  - Persistent volume for data
  - Health check using pg_isready
  - Environment variables for credentials
- **Network**: Custom bridge network for isolation
- **Volumes**: Named volume for database persistence

### Service Dependencies
```mermaid
graph LR
App[App Service] --> DB[Database Service]
DB -.-> HealthCheck["Health Check: pg_isready"]
App --> Config["Config Volume"]
App --> Logs["Log Output"]
Client --> App
DB --> Persistence["Postgres Data Volume"]
style App fill:#2196F3,stroke:#1976D2
style DB fill:#4CAF50,stroke:#388E3C
style Persistence fill:#FF9800,stroke:#F57C00
```

**Diagram sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)

## Environment Variables and Secret Management
The deployment uses environment variables for configuration that varies between environments, with secrets managed through configuration files and Docker volume mounts.

### Environment Configuration
- **TZ**: Timezone setting (Asia/Shanghai)
- **POSTGRES_USER/PASSWORD/DB**: Database credentials
- **GOHOSTOS**: Build-time OS detection

### Secret Management Strategy
The current implementation uses:
- Plain text credentials in docker-compose.yml (suitable for development)
- Configuration files mounted as volumes
- External configuration for production secrets

Recommended production enhancements:
1. Use Docker secrets or Hashicorp Vault
2. Environment-specific .env files not committed to version control
3. Kubernetes Secrets when deployed to K8s
4. AWS Parameter Store or similar managed services

The config.yaml file contains database credentials in plain text:
```yaml
data:
  database:
    source: postgresql://postgres:postgres@cross-redline-db:5432/test?sslmode=disable
```

This should be parameterized in production using environment variable substitution.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [config.yaml](file://configs/config.yaml#L1-L38)

## Logging, Monitoring, and Health Checks
The system includes built-in monitoring capabilities through health checks and structured logging.

### Health Checks
The PostgreSQL service includes a health check:
```yaml
healthcheck:
  test: ["CMD-SHELL", "pg_isready -U postgres"]
  interval: 5s
  timeout: 5s
  retries: 5
```

The application service depends on database health:
```yaml
depends_on:
  db:
    condition: service_healthy
```

### Logging Strategy
- Standard output for containerized logging
- Structured JSON logging recommended for production
- Log compliance checking via `make logcheck`
- HTML and JSON report generation available

### Monitoring Recommendations
For production deployment:
1. Integrate with Prometheus for metrics collection
2. Use Grafana for visualization
3. Implement centralized logging with ELK or Loki
4. Set up alerting for error rates and latency
5. Monitor database connection pool usage

The logchecker tool validates logging practices:
```bash
make logcheck
make logcheck-html
```

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)
- [Makefile](file://Makefile#L1-L141)

## Deployment to Different Environments
The system supports deployment to local, staging, and production environments through configuration overrides.

### Environment Configuration Files
- **configs/features.yaml**: Default feature toggles
- **configs/features.dev.yaml**: Development overrides
- **configs/features.prod.yaml**: Production overrides
- **configs/config.yaml**: Base configuration

### Deployment Scenarios
| Environment | Command | Configuration |
|-----------|---------|---------------|
| Local Development | `docker compose -f docker-compose.dev.yml up` | features.dev.yaml |
| Staging | `docker compose -f docker-compose.yml up` | features.yaml |
| Production | `docker compose -f docker-compose.yml up` | features.prod.yaml |

### Cloud Deployment Guidance
**AWS**:
- Use ECS with Fargate for container orchestration
- RDS for managed PostgreSQL
- S3 for configuration storage
- CloudWatch for logging and monitoring

**GCP**:
- Cloud Run or GKE for container deployment
- Cloud SQL for database
- Cloud Storage for configs
- Stackdriver for observability

**Azure**:
- Azure Container Instances or AKS
- Azure Database for PostgreSQL
- Blob Storage for configs
- Azure Monitor

**Section sources**
- [configs/features.dev.yaml](file://configs/features.dev.yaml)
- [configs/features.prod.yaml](file://configs/features.prod.yaml)
- [configs/config.yaml](file://configs/config.yaml#L1-L38)

## Zero-Downtime Deployment Strategies
While the current configuration doesn't include built-in rolling updates, the architecture supports zero-downtime deployments with minor modifications.

### Recommended Approach
1. **Blue-Green Deployment**:
   - Run two identical environments
   - Switch traffic using load balancer
   - Validate new version
   - Decommission old environment

2. **Rolling Updates** (with orchestration):
   ```yaml
   # Add to docker-compose.yml for swarm mode
   deploy:
     replicas: 2
     update_config:
       parallelism: 1
       delay: 10s
       order: start-first
     rollback_config:
       parallelism: 1
       order: stop-first
   ```

3. **Canary Releases**:
   - Gradually shift traffic to new version
   - Monitor error rates and performance
   - Full rollout or rollback based on metrics

### Prerequisites
- Multiple application instances
- Load balancer (nginx, HAProxy, or cloud LB)
- Proper health checks
- Session persistence or external session store

The current single-container setup should be scaled to multiple replicas in production.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)

## Performance Tuning and Resource Requirements
Optimize the deployment for performance and resource efficiency.

### Resource Allocation
**Minimum Requirements**:
- CPU: 1 vCPU
- Memory: 1GB RAM
- Storage: 1GB (plus database requirements)

**Recommended Production**:
- CPU: 2-4 vCPUs
- Memory: 4-8GB RAM
- Storage: SSD with 10GB+

### Performance Tuning
**Go Binary**:
- Set GOMAXPROCS to match CPU count
- Tune GC with GOGC environment variable
- Use pprof for profiling

**Database**:
- Connection pooling configuration
- Index optimization
- Regular vacuuming (PostgreSQL)

**HTTP Server**:
- Increase timeout values in config.yaml
- Enable compression
- Implement caching

### Configuration Tuning Example
```yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 30s  # Increased from 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 30s  # Increased from 1s
data:
  database:
    max_open_conns: 100
    max_idle_conns: 10
    conn_max_lifetime: 30m
```

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L38)

## Rollback Procedures
Establish reliable rollback mechanisms for failed deployments.

### Manual Rollback
1. Stop current containers:
   ```bash
   docker compose -f docker-compose.yml down
   ```
2. Revert to previous image tag:
   ```bash
   docker compose -f docker-compose.yml up
   ```
3. Verify service health

### Automated Rollback
Using Docker Swarm or Kubernetes:
- Configure rollback policies
- Set health check thresholds
- Automate based on error rates

### Configuration Rollback
Since configs are volume-mounted:
1. Revert config files to previous version
2. Restart application container
3. Verify functionality

Maintain version-controlled configuration files to enable reliable rollbacks.

**Section sources**
- [docker-compose.yml](file://docker-compose.yml#L1-L50)