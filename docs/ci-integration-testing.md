# CI集成测试架构设计

## 概述

本文档描述了项目中循序渐进的CI集成测试架构实现，采用GitHub Actions Services方案，实现了真正的数据库集成测试。

## 架构设计

### 测试分层策略

```
     🔺 E2E Tests (计划中)
    🔺🔺 Integration Tests (已实现)
   🔺🔺🔺 BDD Tests (已完成)
  🔺🔺🔺🔺 Unit Tests (已完成)
```

### CI工作流程

#### 阶段1: 快速反馈测试 (fast-tests)
- **单元测试**: 验证核心业务逻辑
- **BDD测试**: 验证业务场景和用户故事
- **执行时间**: ~2-3分钟
- **触发条件**: 每次push和PR

#### 阶段2: 集成测试 (integration-tests)
- **数据库集成**: 真实PostgreSQL数据库交互
- **缓存集成**: 真实Redis缓存操作
- **执行时间**: ~3-5分钟
- **触发条件**: 快速测试通过后

## 技术实现

### GitHub Actions配置

#### 服务配置
```yaml
services:
  postgres:
    image: postgres:15
    env:
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: test_db
    options: >-
      --health-cmd pg_isready
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5
    ports:
      - 5432:5432
  
  redis:
    image: redis:7
    options: >-
      --health-cmd "redis-cli ping"
      --health-interval 10s
      --health-timeout 5s
      --health-retries 5
    ports:
      - 6379:6379
```

#### 数据库迁移
```yaml
- name: Run database migrations
  run: |
    migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/test_db?sslmode=disable" up
  env:
    DATABASE_URL: postgresql://postgres:postgres@localhost:5432/test_db?sslmode=disable
```

### 智能环境检测

测试脚本能够自动检测运行环境并调整配置：

```bash
# 检测CI环境类型
detect_ci_environment() {
    if [ "$GITHUB_ACTIONS" = "true" ]; then
        CI_TYPE="github_actions"
        USE_EXTERNAL_SERVICES=true
        SKIP_DOCKER_SETUP=true
        log_info "检测到GitHub Actions环境"
    elif [ "$CI" = "true" ]; then
        CI_TYPE="generic_ci"
        USE_EXTERNAL_SERVICES=false
        SKIP_DOCKER_SETUP=false
        log_info "检测到通用CI环境"
    else
        CI_TYPE="local"
        USE_EXTERNAL_SERVICES=false
        SKIP_DOCKER_SETUP=false
        log_info "检测到本地开发环境"
    fi
}
```

### 集成测试配置

#### 环境变量动态配置
```go
// 从环境变量获取数据库配置
databaseURL := os.Getenv("TEST_DATABASE_URL")
if databaseURL == "" {
    // 默认配置（本地开发环境）
    databaseURL = "postgres://postgres:postgres@localhost:5433/test_db?sslmode=disable"
}

redisURL := os.Getenv("TEST_REDIS_URL")
if redisURL == "" {
    // 默认配置（本地开发环境）
    redisURL = "localhost:6380"
}
```

#### 数据库连接验证
```bash
# 确保数据库连接可用
if [ "$USE_EXTERNAL_SERVICES" = true ]; then
    log_info "验证数据库连接..."
    if ! pg_isready -h localhost -p 5432 -U postgres 2>/dev/null; then
        log_error "数据库连接失败"
        return 1
    fi
fi
```

## 测试覆盖范围

### 集成测试场景

1. **用户管理集成测试**
   - 创建用户 (CreateUser)
   - 查询用户 (GetUser)
   - 邮箱查询 (GetUserByEmail)
   - 用户数据完整性验证

2. **账户锁定集成测试**
   - 锁定状态管理
   - 锁定时间验证
   - 失败次数统计

3. **令牌管理集成测试**
   - 刷新令牌存储
   - 令牌过期处理
   - 令牌清理机制

4. **验证码集成测试**
   - 验证码生成和存储
   - 验证码验证逻辑
   - 验证码过期处理

5. **并发操作测试**
   - 多用户并发创建
   - 并发锁定操作
   - 数据一致性验证

## 性能优化

### 并行执行策略
- **快速测试**: 单元测试和BDD测试并行
- **集成测试**: 依赖快速测试完成
- **失败快速**: 任何阶段失败立即停止后续测试

### 资源管理
- **服务健康检查**: 确保服务完全启动后再运行测试
- **数据库迁移**: 自动运行必要的数据库结构更新
- **数据清理**: 测试后自动清理测试数据

## 报告和监控

### 测试报告
- **快速测试报告**: 单元测试覆盖率和BDD测试结果
- **集成测试报告**: 数据库交互测试结果
- **Artifacts保存**: 30天保留期，便于问题排查

### 覆盖率集成
- **Codecov集成**: 自动上传覆盖率数据
- **覆盖率阈值**: 确保代码质量标准

## 使用指南

### 本地开发
```bash
# 运行所有测试
./test/scripts/run_all.sh --all

# 运行特定类型测试
./test/scripts/run_all.sh --unit-only
./test/scripts/run_all.sh --bdd-only
./test/scripts/run_all.sh --integration
```

### CI环境
- **自动触发**: Push到main分支或创建PR
- **分阶段执行**: 快速测试 → 集成测试
- **报告生成**: 自动生成并保存测试报告

## 未来扩展

### 阶段3: 端到端测试 (计划中)
- **应用启动**: 在CI中启动完整应用
- **API测试**: 测试完整的HTTP API
- **业务流程**: 测试关键业务场景

### 性能测试 (计划中)
- **负载测试**: 验证系统性能
- **压力测试**: 测试系统极限
- **性能回归**: 监控性能变化

## 最佳实践

1. **环境隔离**: 每个测试job使用独立的数据库
2. **失败快速**: 单元测试失败立即停止
3. **数据清理**: 确保测试之间的数据隔离
4. **日志详细**: 提供足够的调试信息
5. **缓存优化**: 缓存Go模块和构建产物

## 总结

通过循序渐进的方式，我们成功实现了：

✅ **快速反馈**: 2-3分钟内获得基础测试结果
✅ **真实集成**: 使用真实数据库和缓存进行集成测试
✅ **智能适配**: 自动适配CI和本地环境
✅ **完整报告**: 详细的测试报告和覆盖率分析
✅ **可扩展性**: 为未来的端到端测试奠定基础

这个架构为项目提供了强大的测试基础设施，确保代码质量的同时提供快速的开发反馈。