# Integration Testing

<cite>
**Referenced Files in This Document**   
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L1-L477)
- [docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)
- [config.yaml](file://configs/config.yaml#L1-L38)
- [run_all.sh](file://test/scripts/run_all.sh#L183-L436)
</cite>

## Table of Contents
1. [Integration Testing Overview](#integration-testing-overview)
2. [Test Environment Configuration](#test-environment-configuration)
3. [Database and Redis Integration](#database-and-redis-integration)
4. [Integration Test Structure and Lifecycle](#integration-test-structure-and-lifecycle)
5. [Test Case Implementation](#test-case-implementation)
6. [Containerized Test Environment with Docker Compose](#containerized-test-environment-with-docker-compose)
7. [Test Setup and Teardown Procedures](#test-setup-and-teardown-procedures)
8. [Configuration Management in Tests](#configuration-management-in-tests)
9. [Concurrency and Edge Case Testing](#concurrency-and-edge-case-testing)
10. [Execution Commands and Test Orchestration](#execution-commands-and-test-orchestration)
11. [Performance and Best Practices](#performance-and-best-practices)

## Integration Testing Overview

Integration testing in kratos-boilerplate validates the interactions between service, biz, and data layers with real database connectivity. The framework ensures that components work together as expected, particularly focusing on data persistence, retrieval, and business logic execution across service boundaries. The primary integration test file, `auth_integration_test.go`, demonstrates full-stack test execution from API call to database persistence, using real PostgreSQL and Redis instances in containerized environments.

The integration tests are designed to validate the complete workflow of authentication features, including user creation, account locking, refresh token management, and captcha handling. These tests run against actual database instances rather than mocks, ensuring that the data access layer functions correctly with real SQL operations and transaction handling.

**Section sources**
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L1-L477)

## Test Environment Configuration

The integration tests rely on environment variables to configure database and Redis connections. The `auth_integration_test.go` file defines default values for local development but allows overriding via environment variables `TEST_DATABASE_URL` and `TEST_REDIS_URL`. This flexibility enables the tests to run in different environments, including local development and CI/CD pipelines.

The configuration loading mechanism first checks for environment variables and falls back to default values if they are not set. This approach ensures that developers can run tests locally without additional setup while allowing CI systems to provide custom configurations. The test suite logs the selected database and Redis URLs for debugging purposes.

```mermaid
flowchart TD
Start([Test Execution]) --> CheckEnv["Check Environment Variables"]
CheckEnv --> HasCustomDB{"TEST_DATABASE_URL Set?"}
HasCustomDB --> |Yes| UseCustomDB["Use Custom Database URL"]
HasCustomDB --> |No| UseDefaultDB["Use Default Database URL"]
UseCustomDB --> ConfigureDB
UseDefaultDB --> ConfigureDB
ConfigureDB --> HasCustomRedis{"TEST_REDIS_URL Set?"}
HasCustomRedis --> |Yes| UseCustomRedis["Use Custom Redis URL"]
HasCustomRedis --> |No| UseDefaultRedis["Use Default Redis URL"]
UseCustomRedis --> ConfigureRedis
UseDefaultRedis --> ConfigureRedis
ConfigureRedis --> InitializeComponents["Initialize Data Layer Components"]
InitializeComponents --> RunTests["Execute Test Cases"]
RunTests --> End([Test Completion])
```

**Diagram sources**
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L50-L85)

**Section sources**
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L50-L85)
- [config.yaml](file://configs/config.yaml#L1-L38)

## Database and Redis Integration

The integration tests utilize PostgreSQL and Redis services defined in `docker-compose.test.yml`. The PostgreSQL container runs on port 5433 (mapped from 5432) with a database named `test_db`, while Redis runs on port 6380 (mapped from 6379). Both services include health checks to ensure they are ready before tests begin.

The data layer connects to these services using the configuration provided in the test setup. The PostgreSQL instance is initialized with migration scripts from the `migrations` directory, ensuring the database schema is up-to-date before tests execute. Redis is configured with append-only file persistence to maintain data consistency across test runs.

```mermaid
graph TB
subgraph "Test Environment"
TestDB[(PostgreSQL<br/>test_db:5433)]
TestRedis[(Redis<br/>localhost:6380)]
TestApp["Test Application"]
end
TestApp --> TestDB
TestApp --> TestRedis
TestDB --> Migrations["Database Migrations"]
TestRedis --> Persistence["AOF Persistence"]
style TestDB fill:#f9f,stroke:#333
style TestRedis fill:#f9f,stroke:#333
style TestApp fill:#bbf,stroke:#333
```

**Diagram sources**
- [docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)

**Section sources**
- [docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L70-L75)

## Integration Test Structure and Lifecycle

The integration tests follow the testify suite pattern, using `suite.Suite` from the testify library to manage test lifecycle hooks. The `AuthIntegrationTestSuite` struct contains the test suite state, including the data layer instance, cleanup function, user repository, and context.

The test lifecycle consists of four main phases:
1. `SetupSuite`: Executed once before all tests in the suite
2. `TearDownSuite`: Executed once after all tests in the suite
3. `SetupTest`: Executed before each individual test
4. `TearDownTest`: Executed after each individual test

This structure ensures proper resource management and test isolation. The `SetupSuite` method initializes the database connection and creates the user repository, while `TearDownSuite` ensures all resources are properly cleaned up. The per-test setup and teardown methods handle test-specific cleanup to prevent test interference.

```mermaid
sequenceDiagram
participant TestRunner
participant Suite as AuthIntegrationTestSuite
participant DataLayer as Data Layer
participant DB as Database
TestRunner->>Suite : Run Suite
Suite->>Suite : SetupSuite()
Suite->>DataLayer : NewData(config)
DataLayer->>DB : Connect
DB-->>DataLayer : Connection
DataLayer-->>Suite : Data instance
Suite->>Suite : Create UserRepo
loop Each Test Case
Suite->>Suite : SetupTest()
Suite->>Suite : cleanupTestData()
Suite->>TestRunner : Execute Test
TestRunner->>Suite : TestCreateUser()
Suite->>Suite : TearDownTest()
Suite->>Suite : cleanupTestData()
end
Suite->>Suite : TearDownSuite()
Suite->>DataLayer : cleanup()
```

**Diagram sources**
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L30-L150)

**Section sources**
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L30-L150)

## Test Case Implementation

The integration test suite includes several test cases that validate different aspects of the authentication system:

- `TestCreateUser`: Verifies that users can be created and retrieved from the database
- `TestGetUser`: Tests user retrieval by username, including error handling for non-existent users
- `TestGetUserByEmail`: Validates user lookup by email address
- `TestAccountLock`: Tests account locking functionality for failed login attempts
- `TestRefreshToken`: Validates refresh token creation, retrieval, and invalidation
- `TestCaptcha`: Tests captcha creation, retrieval, and usage tracking
- `TestConcurrentOperations`: Validates system behavior under concurrent access

Each test case follows the Arrange-Act-Assert pattern, setting up test data, executing the operation, and verifying the results. The tests use testify's assertion methods (`assert` and `require`) to validate outcomes, with `require` used for critical failures that should stop test execution.

```mermaid
flowchart TD
TestCreateUser([TestCreateUser]) --> Arrange["Arrange: Create User Object"]
Arrange --> Act["Act: Call CreateUser"]
Act --> Assert["Assert: Verify No Error"]
Assert --> VerifyID["Verify User ID Set"]
VerifyID --> Retrieve["Retrieve User by Username"]
Retrieve --> VerifyData["Verify Retrieved Data Matches"]
VerifyData --> End([Test Complete])
style TestCreateUser fill:#f96,stroke:#333
```

**Diagram sources**
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L152-L200)

**Section sources**
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L152-L320)

## Containerized Test Environment with Docker Compose

The test environment is defined in `docker-compose.test.yml`, which specifies four services:

1. `test-db`: PostgreSQL 15 container with persistent volume and automatic migration execution
2. `test-redis`: Redis 7 container with AOF persistence enabled
3. `test-app`: Application container for end-to-end testing
4. `test-init`: Initialization service that runs database migrations and loads test data

The Docker Compose configuration uses health checks to ensure services are ready before dependent services start. The `test-init` service waits for the database to be ready before applying migrations, ensuring the schema is properly set up before tests begin.

The containerized environment provides isolation and consistency across different development and CI environments. The use of named volumes (`test_postgres_data` and `test_redis_data`) ensures data persistence between test runs, while the bridge network (`test-network`) enables service communication.

```mermaid
graph TB
subgraph "Docker Compose Services"
TestDB[(PostgreSQL)]
TestRedis[(Redis)]
TestApp["Application"]
TestInit["Init Service"]
end
TestInit --> TestDB
TestInit --> Migrations["Run Migrations"]
TestInit --> TestData["Load Test Data"]
TestApp --> TestDB
TestApp --> TestRedis
TestDB --> |Health Check| TestInit
TestDB --> |Health Check| TestApp
TestRedis --> |Health Check| TestApp
style TestDB fill:#f9f,stroke:#333
style TestRedis fill:#f9f,stroke:#333
style TestApp fill:#bbf,stroke:#333
style TestInit fill:#ff9,stroke:#333
```

**Diagram sources**
- [docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)

**Section sources**
- [docker-compose.test.yml](file://test/config/docker-compose.test.yml#L1-L112)

## Test Setup and Teardown Procedures

The test setup and teardown procedures are orchestrated through shell scripts in the `test/scripts` directory. The `run_all.sh` script coordinates the entire test lifecycle, including environment setup, service startup, test execution, and cleanup.

The setup process includes:
1. Ensuring Go module dependencies are up-to-date
2. Creating necessary test directories
3. Starting Docker containers with `docker-compose.test.yml`
4. Waiting for services to become healthy
5. Running database migrations

The teardown process cleans up resources by:
1. Stopping and removing Docker containers
2. Removing associated volumes
3. Resetting the database state when using external services

The scripts support both local development and CI environments through configuration flags like `SKIP_DOCKER_SETUP` and `USE_EXTERNAL_SERVICES`, allowing flexibility in how the test environment is provisioned.

```mermaid
sequenceDiagram
participant User
participant RunAll as run_all.sh
participant Docker as Docker Compose
participant DB as Database
User->>RunAll : Execute script
RunAll->>RunAll : setup_test_environment()
RunAll->>RunAll : start_test_services()
RunAll->>Docker : docker-compose up -d
Docker->>DB : Start PostgreSQL
Docker->>TestRedis : Start Redis
Docker->>TestInit : Start Init Service
TestInit->>DB : Apply migrations
TestInit->>DB : Load test data
RunAll->>RunAll : wait for services
RunAll->>RunAll : run_integration_tests()
RunAll->>RunAll : cleanup_test_environment()
RunAll->>Docker : docker-compose down -v
Docker->>DB : Stop and remove
Docker->>TestRedis : Stop and remove
Docker-->>RunAll : Success
RunAll-->>User : Test results
```

**Diagram sources**
- [run_all.sh](file://test/scripts/run_all.sh#L183-L436)

**Section sources**
- [run_all.sh](file://test/scripts/run_all.sh#L183-L436)

## Configuration Management in Tests

Configuration management in the integration tests follows a hierarchical approach with multiple layers of configuration sources:

1. **Default Configuration**: Hardcoded defaults in `auth_integration_test.go` for local development
2. **Environment Variables**: Overridable settings via `TEST_DATABASE_URL` and `TEST_REDIS_URL`
3. **CI Configuration**: GitHub Actions services with environment-specific URLs
4. **Application Configuration**: Base settings from `config.yaml`

The `config.yaml` file provides the baseline configuration for the application, including database and Redis connection details. During integration tests, these values are overridden by test-specific environment variables to ensure tests run against the test database rather than production or development instances.

The configuration loading process in the test suite prioritizes environment variables over defaults, allowing for flexible test execution across different environments. This approach ensures that tests can run in CI/CD pipelines with externally provisioned services while maintaining simplicity for local development.

**Section sources**
- [config.yaml](file://configs/config.yaml#L1-L38)
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L50-L85)

## Concurrency and Edge Case Testing

The integration test suite includes `TestConcurrentOperations` to validate system behavior under concurrent access. This test creates multiple goroutines that perform operations on the data layer simultaneously, simulating real-world concurrent usage patterns.

The test verifies that:
- Database transactions are properly isolated
- Concurrent operations do not interfere with each other
- The system handles race conditions appropriately
- Performance remains acceptable under load

The test creates a base user and then spawns multiple goroutines, each performing a series of operations on unique usernames derived from the base username. Each goroutine performs save and retrieve operations with a small delay between operations to simulate realistic usage patterns.

Additionally, the test suite covers various edge cases:
- Retrieving non-existent users (expected to return `ErrUserNotFound`)
- Creating users with duplicate usernames (handled by database constraints)
- Handling expired or used captchas
- Managing account lock states with timestamps
- Processing refresh tokens that have been invalidated

These edge case tests ensure the system behaves predictably when faced with invalid or unexpected inputs, providing robust error handling and recovery mechanisms.

```mermaid
flowchart TD
Start([TestConcurrentOperations]) --> CreateBaseUser["Create Base User"]
CreateBaseUser --> SpawnGoroutines["Spawn N Goroutines"]
subgraph "Each Goroutine"
direction TB
Init["Initialize Unique Username"]
Loop["For i = 1 to M"]
SaveLock["Save Account Lock"]
GetLock["Get Account Lock"]
Sleep["Sleep 10ms"]
Loop --> SaveLock
SaveLock --> GetLock
GetLock --> Sleep
Sleep --> Loop
end
SpawnGoroutines --> EachGoroutine
EachGoroutine --> WaitForCompletion["Wait for All Goroutines"]
WaitForCompletion --> VerifyResults["Verify All Operations Succeeded"]
VerifyResults --> End([Test Complete])
```

**Diagram sources**
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L322-L365)

**Section sources**
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L322-L365)

## Execution Commands and Test Orchestration

Integration tests are executed using the standard Go test command with build tags to filter integration tests:

```bash
go test -tags=integration ./test/integration/...
```

The test orchestration is managed by the `run_all.sh` script, which provides a comprehensive interface for running different test types:

```bash
# Run all tests
./test/scripts/run_all.sh --integration --unit --e2e

# Run only integration tests
./test/scripts/run_all.sh --integration

# Run with verbose output
./test/scripts/run_all.sh --integration --verbose
```

The script accepts various flags to control test execution:
- `--integration`: Run integration tests
- `--e2e`: Run end-to-end tests
- `--verbose`: Enable verbose output
- `--skip-docker`: Use external services instead of Docker
- `--cleanup-after`: Clean up test environment after execution

For local development, tests can also be run directly from an IDE or using Go test commands with the appropriate environment variables set:

```bash
TEST_DATABASE_URL="postgresql://postgres:postgres@localhost:5433/test_db?sslmode=disable" \
TEST_REDIS_URL="localhost:6380" \
go test -tags=integration -v ./test/integration/database/
```

The test suite uses the `testing.Short()` function to skip integration tests when the `-short` flag is provided, allowing for faster test runs during development.

**Section sources**
- [run_all.sh](file://test/scripts/run_all.sh#L338-L370)

## Performance and Best Practices

The integration testing framework in kratos-boilerplate follows several best practices to ensure reliable and maintainable tests:

1. **Test Isolation**: Each test case is independent and does not rely on the state created by other tests
2. **Resource Cleanup**: All resources are properly cleaned up after tests to prevent test pollution
3. **Configuration Flexibility**: Tests can run in different environments with appropriate configuration
4. **Comprehensive Coverage**: Tests cover both happy paths and edge cases
5. **Concurrency Testing**: System behavior under load is validated

Performance considerations for the integration test suite include:

- **Test Duration**: The use of `sleep` statements in setup/teardown should be minimized and replaced with health checks where possible
- **Database Operations**: Tests should minimize expensive database operations and use transactions where appropriate
- **Parallel Execution**: Tests that do not share state can be run in parallel to reduce execution time
- **Resource Management**: Database connections and other resources should be properly closed to prevent leaks

The framework could be enhanced with transaction rollback strategies to further improve test isolation and speed. Currently, the `cleanupTestData` method is responsible for cleaning up test data, but using database transactions with rollback could provide faster cleanup and better isolation.

Additionally, fixture data loading could be optimized by using a shared setup phase that loads common test data once for the entire suite, reducing redundant database operations across test cases.

**Section sources**
- [auth_integration_test.go](file://test/integration/database/auth_integration_test.go#L130-L145)
- [run_all.sh](file://test/scripts/run_all.sh#L237-L269)