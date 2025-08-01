# BDD测试框架设计文档

## 概述

本文档描述了为Kratos项目设计的完整BDD（行为驱动开发）测试框架。该框架支持多种测试类型，包括单元测试、BDD测试、集成测试和端到端测试，提供了完整的测试解决方案。

## 框架特性

### 🎯 核心特性

- **多层次测试支持**: 单元测试、BDD测试、集成测试、端到端测试
- **工具集成**: Go testing、Ginkgo、Gomega、testify
- **Mock支持**: 完整的Mock仓储和服务
- **自动化环境**: Docker Compose自动化测试环境
- **覆盖率报告**: 自动生成HTML覆盖率报告
- **并发测试**: 支持并发安全性测试
- **CI/CD集成**: 与现有CI/CD流程无缝集成

### 🏗️ 架构设计

```
test/
├── README.md                    # 框架说明文档
├── bdd/                         # BDD测试套件
│   ├── auth/                    # 认证相关BDD测试
│   │   └── auth_bdd_test.go
│   ├── greeter/                 # Greeter相关BDD测试
│   └── shared/                  # 共享的测试工具和助手
│       ├── test_suite.go        # 测试套件基础设施
│       └── mocks.go             # Mock定义
├── integration/                 # 集成测试
│   ├── database/                # 数据库集成测试
│   │   └── auth_integration_test.go
│   ├── redis/                   # Redis集成测试
│   └── api/                     # API集成测试
├── e2e/                         # 端到端测试
│   ├── scenarios/               # 测试场景
│   │   └── auth_e2e_test.go
│   ├── fixtures/                # 测试数据
│   └── helpers/                 # 测试助手
│       ├── api_client.go        # HTTP API客户端
│       └── test_user.go         # 测试用户管理
├── config/                      # 测试配置
│   ├── test.yaml               # 测试环境配置
│   └── docker-compose.test.yml # 测试环境Docker配置
└── scripts/                     # 测试脚本
    ├── setup.sh                # 测试环境设置
    ├── teardown.sh             # 测试环境清理
    └── run_all.sh              # 运行所有测试
```

## 测试类型详解

### 1. 单元测试 (TDD)

**位置**: 与被测代码同目录  
**命名**: `*_test.go`  
**工具**: Go testing + testify  
**覆盖**: 业务逻辑层、数据访问层  

```go
// 示例: internal/biz/auth_test.go
func TestAuthUsecase_Login(t *testing.T) {
    // 使用testify进行断言
    assert.Equal(t, expected, actual)
    require.NoError(t, err)
}
```

### 2. BDD测试

**位置**: `test/bdd/` 和 `internal/service/`  
**命名**: `*_bdd_test.go`  
**工具**: Ginkgo + Gomega  
**覆盖**: 服务层业务流程  

```go
// 示例: test/bdd/auth/auth_bdd_test.go
var _ = Describe("认证服务", func() {
    Context("当提供有效的用户名和密码时", func() {
        It("应该返回访问令牌", func() {
            // 使用Gomega进行断言
            Expect(response.AccessToken).NotTo(BeEmpty())
        })
    })
})
```

### 3. 集成测试

**位置**: `test/integration/`  
**命名**: `*_integration_test.go`  
**工具**: Go testing + testify + testcontainers  
**覆盖**: 与外部依赖的交互  

```go
// 示例: test/integration/database/auth_integration_test.go
//go:build integration

func (suite *AuthIntegrationTestSuite) TestCreateUser() {
    // 测试真实数据库交互
    err := suite.userRepo.CreateUser(suite.ctx, user)
    assert.NoError(suite.T(), err)
}
```

### 4. 端到端测试

**位置**: `test/e2e/`  
**命名**: `*_e2e_test.go`  
**工具**: Ginkgo + Gomega + HTTP客户端  
**覆盖**: 完整的用户场景  

```go
// 示例: test/e2e/scenarios/auth_e2e_test.go
//go:build e2e

var _ = Describe("用户注册流程", func() {
    It("应该能够成功注册新用户", func() {
        By("获取图片验证码")
        // 完整的HTTP API调用
    })
})
```

## 核心组件

### TestSuite - 测试套件基础设施

```go
// test/bdd/shared/test_suite.go
type TestSuite struct {
    Logger   log.Logger
    Ctx      context.Context
    Data     *data.Data
    Cleanup  func()
    Services *Services
}

// 支持Mock和真实数据层
func (ts *TestSuite) SetupWithMocks() *MockRepositories
func (ts *TestSuite) SetupWithRealData(config *conf.Data) error
```

### Mock系统

```go
// test/bdd/shared/mocks.go
type MockUserRepo struct {
    mock.Mock
}

// 实现所有biz.UserRepo接口方法
func (m *MockUserRepo) GetUser(ctx context.Context, username string) (*biz.User, error) {
    args := m.Called(ctx, username)
    // ...
}
```

### API客户端

```go
// test/e2e/helpers/api_client.go
type APIClient struct {
    baseURL    string
    httpClient *http.Client
}

// 封装所有API调用
func (c *APIClient) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginReply, error)
```

### 测试用户管理

```go
// test/e2e/helpers/test_user.go
type TestUser struct {
    Username     string
    Password     string
    Email        string
    AccessToken  string
    RefreshToken string
}

func NewTestUser() *TestUser
```

## 使用指南

### 快速开始

1. **设置测试环境**
   ```bash
   cd test
   ./scripts/setup.sh
   ```

2. **运行基础测试**
   ```bash
   make test
   # 或
   ./scripts/run_all.sh
   ```

3. **运行所有测试**
   ```bash
   make test-all
   # 或
   ./scripts/run_all.sh --all
   ```

### 详细命令

```bash
# 查看所有可用命令
make help
./scripts/run_all.sh --help

# 运行特定类型的测试
make test-unit           # 仅单元测试
make test-bdd            # 仅BDD测试
make test-integration    # 集成测试
make test-e2e            # 端到端测试

# 使用原有脚本（兼容模式）
../../scripts/run_tests.sh
../../scripts/run_tests.sh --bdd-framework --all
```

### 环境管理

```bash
# 设置测试环境
./scripts/setup.sh

# 清理测试环境
./scripts/teardown.sh

# 强制清理
./scripts/teardown.sh --force
```

## 编写测试指南

### BDD测试最佳实践

1. **使用描述性的测试名称**
   ```go
   Describe("用户登录", func() {
       Context("当提供有效的用户名和密码时", func() {
           It("应该返回访问令牌", func() {
               // 测试逻辑
           })
       })
   })
   ```

2. **使用BeforeEach和AfterEach**
   ```go
   BeforeEach(func() {
       testSuite = shared.BeforeEachSetup()
       mocks = testSuite.SetupWithMocks()
   })
   
   AfterEach(func() {
       shared.AfterEachTeardown(testSuite)
   })
   ```

3. **使用By步骤分解复杂测试**
   ```go
   It("应该能够完成完整的注册流程", func() {
       By("获取验证码")
       // 步骤1
       
       By("提交注册信息")
       // 步骤2
       
       By("验证用户创建成功")
       // 步骤3
   })
   ```

### 集成测试最佳实践

1. **使用Build Tags**
   ```go
   //go:build integration
   
   package database_test
   ```

2. **使用测试套件**
   ```go
   type AuthIntegrationTestSuite struct {
       suite.Suite
       data     *data.Data
       cleanup  func()
   }
   
   func TestAuthIntegrationSuite(t *testing.T) {
       suite.Run(t, new(AuthIntegrationTestSuite))
   }
   ```

3. **数据清理**
   ```go
   func (suite *AuthIntegrationTestSuite) SetupTest() {
       suite.cleanupTestData()
   }
   
   func (suite *AuthIntegrationTestSuite) TearDownTest() {
       suite.cleanupTestData()
   }
   ```

### 端到端测试最佳实践

1. **使用Build Tags**
   ```go
   //go:build e2e
   
   package scenarios_test
   ```

2. **等待服务就绪**
   ```go
   BeforeSuite(func() {
       Eventually(func() error {
           return client.HealthCheck(ctx)
       }, 30*time.Second, 1*time.Second).Should(Succeed())
   })
   ```

3. **使用测试用户管理**
   ```go
   BeforeEach(func() {
       testUser = helpers.NewTestUser()
   })
   
   AfterEach(func() {
       if testUser != nil {
           client.CleanupUser(ctx, testUser.Username)
       }
   })
   ```

## 配置说明

### 测试配置文件

```yaml
# test/config/test.yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s

data:
  database:
    driver: postgres
    source: postgresql://postgres:postgres@localhost:5432/test_db?sslmode=disable
  redis:
    addr: localhost:6379

auth:
  jwt_secret_key: "test-secret-key-for-testing-only"
  captcha_enabled: true
  max_login_attempts: 5

test:
  fixed_captcha_code: "123456"
  auto_cleanup: true
  timeout:
    unit: "30s"
    integration: "2m"
    e2e: "5m"
```

### Docker Compose配置

```yaml
# test/config/docker-compose.test.yml
services:
  test-db:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: test_db
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5433:5432"
    
  test-redis:
    image: redis:7-alpine
    ports:
      - "6380:6379"
    
  test-app:
    build:
      context: ../..
      dockerfile: Dockerfile.dev
    ports:
      - "8001:8000"
    depends_on:
      - test-db
      - test-redis
```

## CI/CD集成

### GitHub Actions示例

```yaml
# .github/workflows/test.yml
name: BDD Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version: 1.21
      
      - name: Install dependencies
        run: |
          go install github.com/onsi/ginkgo/v2/ginkgo@latest
      
      - name: Run unit and BDD tests
        run: |
          cd test
          ./scripts/run_all.sh --unit-only --bdd-only
      
      - name: Run integration tests
        run: |
          cd test
          ./scripts/run_all.sh --integration
      
      - name: Upload coverage reports
        uses: codecov/codecov-action@v3
        with:
          file: ./test/output/unit_coverage.out
```

## 性能和最佳实践

### 测试性能优化

1. **并行测试执行**
   ```bash
   ginkgo -p -r ./test/bdd  # 并行运行BDD测试
   go test -parallel 4 ./internal/...  # 并行运行单元测试
   ```

2. **测试缓存利用**
   ```bash
   go test -cache ./internal/...  # 利用测试缓存
   ```

3. **选择性测试运行**
   ```bash
   ginkgo -focus="登录" ./test/bdd  # 只运行包含"登录"的测试
   go test -run TestAuth ./internal/biz  # 只运行特定测试
   ```

### 测试数据管理

1. **使用事务回滚**
   ```go
   func (suite *IntegrationTestSuite) SetupTest() {
       suite.tx = suite.db.Begin()
   }
   
   func (suite *IntegrationTestSuite) TearDownTest() {
       suite.tx.Rollback()
   }
   ```

2. **测试数据隔离**
   ```go
   func NewTestUser() *TestUser {
       suffix := generateRandomString(8)
       return &TestUser{
           Username: fmt.Sprintf("testuser_%s", suffix),
           // ...
       }
   }
   ```

3. **资源清理**
   ```go
   func (ts *TestSuite) Teardown() {
       if ts.Cleanup != nil {
           ts.Cleanup()
       }
   }
   ```

## 故障排除

### 常见问题

1. **Ginkgo未安装**
   ```bash
   go install github.com/onsi/ginkgo/v2/ginkgo@latest
   ```

2. **Docker服务启动失败**
   ```bash
   docker-compose -f test/config/docker-compose.test.yml logs
   ```

3. **测试数据库连接失败**
   ```bash
   # 检查数据库状态
   docker-compose -f test/config/docker-compose.test.yml ps
   
   # 重启数据库
   docker-compose -f test/config/docker-compose.test.yml restart test-db
   ```

4. **权限问题**
   ```bash
   chmod +x test/scripts/*.sh
   ```

### 调试技巧

1. **详细输出**
   ```bash
   ./scripts/run_all.sh --verbose
   ginkgo -v -r ./test/bdd
   ```

2. **单独运行失败的测试**
   ```bash
   ginkgo -focus="特定测试名称" ./test/bdd
   go test -v -run TestSpecificFunction ./internal/biz
   ```

3. **查看测试日志**
   ```bash
   cat test/output/test_report.txt
   cat test/logs/*.log
   ```

## 扩展指南

### 添加新的测试类型

1. **创建新的测试目录**
   ```bash
   mkdir -p test/performance
   mkdir -p test/security
   ```

2. **更新运行脚本**
   ```bash
   # 在 test/scripts/run_all.sh 中添加新的测试类型
   run_performance_tests() {
       # 性能测试逻辑
   }
   ```

3. **更新Makefile**
   ```makefile
   # 在 test/Makefile 中添加新命令
   test-performance:
       ./scripts/run_all.sh --performance
   ```

### 自定义Mock

1. **扩展Mock接口**
   ```go
   // test/bdd/shared/mocks.go
   type MockCustomService struct {
       mock.Mock
   }
   
   func (m *MockCustomService) CustomMethod(ctx context.Context, param string) error {
       args := m.Called(ctx, param)
       return args.Error(0)
   }
   ```

2. **集成到TestSuite**
   ```go
   type MockRepositories struct {
       // 现有Mock
       CustomService *MockCustomService
   }
   ```

### 添加新的测试助手

1. **创建助手函数**
   ```go
   // test/e2e/helpers/custom_helper.go
   func WaitForCondition(condition func() bool, timeout time.Duration) error {
       // 实现逻辑
   }
   ```

2. **在测试中使用**
   ```go
   It("应该等待条件满足", func() {
       err := helpers.WaitForCondition(func() bool {
           return someCondition
       }, 30*time.Second)
       Expect(err).NotTo(HaveOccurred())
   })
   ```

## 总结

这个BDD测试框架提供了：

✅ **完整的测试覆盖**: 从单元测试到端到端测试  
✅ **现代化工具链**: Ginkgo、Gomega、testify  
✅ **自动化环境**: Docker Compose自动化  
✅ **易于使用**: 简单的命令行接口  
✅ **可扩展性**: 模块化设计，易于扩展  
✅ **CI/CD友好**: 与现有流程无缝集成  
✅ **详细文档**: 完整的使用和扩展指南  

通过这个框架，开发团队可以：
- 提高代码质量和测试覆盖率
- 加快开发和部署速度
- 减少生产环境bug
- 提升团队协作效率
- 建立可持续的测试文化

框架设计遵循了测试金字塔原则，平衡了测试速度、可靠性和维护成本，为项目的长期发展提供了坚实的测试基础。