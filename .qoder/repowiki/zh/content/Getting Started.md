# Getting Started

<cite>
**Referenced Files in This Document**   
- [README.md](file://README.md)
- [Makefile](file://Makefile)
- [docker-compose.yml](file://docker-compose.yml)
- [docker-compose.dev.yml](file://docker-compose.dev.yml)
- [test/scripts/run_all.sh](file://test/scripts/run_all.sh)
- [configs/config.yaml](file://configs/config.yaml)
- [cmd/kratos-boilerplate/main.go](file://cmd/kratos-boilerplate/main.go)
- [frontend/package.json](file://frontend/package.json)
</cite>

## Table of Contents
1. [Introduction](#introduction)
2. [Prerequisites and Environment Setup](#prerequisites-and-environment-setup)
3. [Quick Start Guide](#quick-start-guide)
4. [Running the Application](#running-the-application)
5. [Frontend and API Access](#frontend-and-api-access)
6. [Makefile Commands Overview](#makefile-commands-overview)
7. [Troubleshooting Common Issues](#troubleshooting-common-issues)
8. [Conclusion](#conclusion)

## Introduction
This guide provides a comprehensive onboarding process for developers new to the **kratos-boilerplate** project. It covers all necessary steps to set up a fully functional development environment, including prerequisites, installation, configuration, and execution workflows using both direct execution and Docker-based methods. The goal is to enable any developer to have a working instance of the application within 10 minutes.

The project is built using Go with the Kratos framework, features a Vue 3 frontend, and uses PostgreSQL as the primary database. It follows a modular architecture with Protocol Buffers for API definitions and supports plugin-based extensibility.

**Section sources**
- [README.md](file://README.md#L1-L92)

## Prerequisites and Environment Setup
Before starting with the project, ensure that the following tools are installed on your system:

- **Go 1.22 or higher**: The project is written in Go and requires version 1.22+.
- **Node.js (with npm)**: Required for building and running the frontend application.
- **Docker and Docker Compose**: Used for managing the PostgreSQL database and optional containerized execution.
- **Protocol Buffers Compiler (protoc)**: Needed for generating API code from `.proto` files.
- **Kratos CLI**: The official command-line tool for Go-Kratos projects.

To install the required tools and dependencies, run the following command:

```bash
make init
```

This command installs essential Go tools such as:
- `protoc-gen-go`
- `protoc-gen-go-grpc`
- `kratos` CLI
- `protoc-gen-go-http`
- `protoc-gen-openapi`
- `wire`

Additionally, the frontend uses **Vue 3**, **Vite**, **Pinia**, and **Element Plus**, which are managed via `npm`. These will be installed automatically when you run frontend-specific commands.

**Section sources**
- [README.md](file://README.md#L10-L15)
- [Makefile](file://Makefile#L10-L20)

## Quick Start Guide
Follow these steps to get the project up and running quickly.

### Step 1: Clone the Repository
```bash
git clone https://github.com/26huitailang/kratos-boilerplate.git
cd kratos-boilerplate
```

### Step 2: Start the Database Using Docker
The project uses PostgreSQL as its database. Start the database container using Docker Compose:

```bash
docker compose up db -d
```

This command starts the PostgreSQL service defined in `docker-compose.yml`, exposing it on port `5432`. The database container is named `cross-redline-db` and initializes with the username `postgres`, password `postgres`, and database name `test`.

```mermaid
flowchart TD
A["Start Docker Compose"] --> B["Launch PostgreSQL Container"]
B --> C["Expose Port 5432"]
C --> D["Initialize Database: test"]
D --> E["Ready for Application Connection"]
```

**Diagram sources**
- [docker-compose.yml](file://docker-compose.yml#L15-L34)

### Step 3: Generate API and Configuration Files
Generate the necessary Go and API files from the `.proto` definitions:

```bash
make api
```

This command runs `protoc` to generate:
- Go structs (`*.pb.go`)
- HTTP and gRPC server bindings
- OpenAPI/Swagger specifications

You can also generate internal configuration files:

```bash
make config
```

And generate all code (API + config + wire DI):

```bash
make all
```

**Section sources**
- [README.md](file://README.md#L17-L25)
- [Makefile](file://Makefile#L22-L41)

## Running the Application
After setting up the environment and generating the necessary files, you can build and run the application.

### Build the Application
```bash
make build
```

This compiles the Go application into a binary located at `./bin/kratos-boilerplate`.

### Run the Application
```bash
./bin/kratos-boilerplate -conf ./configs
```

The `-conf` flag specifies the configuration directory. The application reads `config.yaml` from this directory to configure servers, database connections, plugins, and feature flags.

Alternatively, you can use Docker to run the entire stack:

```bash
docker compose up --build
```

This builds the application image using `Dockerfile` and starts both the app and database containers as defined in `docker-compose.yml`.

```mermaid
sequenceDiagram
participant User as Developer
participant Make as Makefile
participant App as Application
participant DB as Database
User->>Make : make build
Make->>App : Compile Go code
App-->>Make : Output binary to ./bin/
User->>App : ./bin/kratos-boilerplate -conf ./configs
App->>DB : Connect via PostgreSQL URL
DB-->>App : Connection established
App-->>User : HTTP server running on : 8000
```

**Diagram sources**
- [Makefile](file://Makefile#L43-L50)
- [configs/config.yaml](file://configs/config.yaml#L1-L38)
- [cmd/kratos-boilerplate/main.go](file://cmd/kratos-boilerplate/main.go#L1-L91)

## Frontend and API Access
Once the application is running, you can access the following endpoints:

### Frontend Application
The frontend is served through the backend on port `8000`. Open your browser and navigate to:

```
http://localhost:8000
```

The frontend is built with Vue 3 and uses Vite for development. You can also run the frontend independently using:

```bash
cd frontend
npm install
npm run dev
```

This starts the Vite development server on `http://localhost:5173`.

### API Documentation
Swagger UI is available at:

```
http://localhost:8000/q/swagger-ui/
```

This endpoint is automatically registered in the `main.go` file using the `openapiv2.NewHandler()` middleware.

### Health Check Endpoint
A basic health check is available at:

```
GET http://localhost:8000/health
```

## Makefile Commands Overview
The `Makefile` provides several convenient commands for development:

| Command | Description |
|--------|-------------|
| `make init` | Install all required Go tools |
| `make api` | Generate API files from `.proto` definitions |
| `make config` | Generate internal configuration Go files |
| `make all` | Run `api`, `config`, and `generate` |
| `make build` | Build the application binary |
| `make test` | Run all Go tests |
| `make test-coverage` | Run tests and generate HTML coverage report |
| `make logcheck` | Check log usage compliance using custom tool |

For testing, the script `scripts/run_tests.sh` can be used to run all tests including BDD and integration tests:

```bash
scripts/run_tests.sh
```

This script handles environment setup, database initialization, test execution, and cleanup.

**Section sources**
- [Makefile](file://Makefile#L1-L140)
- [README.md](file://README.md#L70-L92)
- [test/scripts/run_all.sh](file://test/scripts/run_all.sh#L183-L436)

## Troubleshooting Common Issues
Here are solutions to common setup problems:

### Port Conflicts
If you encounter errors like `port is already allocated`, ensure no other services are using ports `8000`, `9000`, or `5432`:

```bash
lsof -i :8000
lsof -i :5432
```

Stop conflicting processes or modify the ports in `docker-compose.yml`.

### Dependency Installation Failures
If `make init` fails due to network issues (e.g., in China), configure Go proxy:

```bash
go env -w GOPROXY=https://goproxy.io,direct
```

Then retry `make init`.

### Database Initialization Problems
If the application fails to connect to the database:
1. Ensure the database container is running:
   ```bash
   docker compose ps
   ```
2. Check logs:
   ```bash
   docker compose logs db
   ```
3. Verify the connection string in `configs/config.yaml` matches the service name `cross-redline-db`.

### Missing Migration Files
Ensure migration files in the `migrations/` directory are correctly formatted and applied:

```bash
# Manually apply migrations if needed
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/test?sslmode=disable" up
```

### Code Generation Errors
If `make api` fails:
- Ensure `protoc` is installed and in your PATH.
- Verify all `.proto` files are syntactically correct.
- Confirm `third_party` dependencies are present.

**Section sources**
- [configs/config.yaml](file://configs/config.yaml#L15-L20)
- [docker-compose.yml](file://docker-compose.yml#L15-L34)
- [test/scripts/run_all.sh](file://test/scripts/run_all.sh#L237-L269)

## Conclusion
This guide has walked you through setting up the **kratos-boilerplate** project from scratch. You've learned how to:
- Install prerequisites using `make init`
- Start the PostgreSQL database with Docker
- Generate API and configuration files
- Build and run the application
- Access the frontend and API documentation
- Use Makefile commands for efficient development
- Troubleshoot common issues

With this foundation, you're ready to begin developing features, writing tests, and extending the application using its plugin architecture and modular design.