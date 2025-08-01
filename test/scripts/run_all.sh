#!/bin/bash

# BDD测试框架 - 运行所有测试脚本
# 这个脚本会依次运行单元测试、BDD测试、集成测试和端到端测试

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEST_DIR="$PROJECT_ROOT/test"

# 默认配置
RUN_UNIT=true
RUN_BDD=true
RUN_INTEGRATION=false
RUN_E2E=false
GENERATE_COVERAGE=true
CLEANUP_AFTER=true
VERBOSE=false

# CI环境检测
CI_TYPE="local"
USE_EXTERNAL_SERVICES=false
SKIP_DOCKER_SETUP=false

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --unit-only)
            RUN_UNIT=true
            RUN_BDD=false
            RUN_INTEGRATION=false
            RUN_E2E=false
            shift
            ;;
        --bdd-only)
            RUN_UNIT=false
            RUN_BDD=true
            RUN_INTEGRATION=false
            RUN_E2E=false
            shift
            ;;
        --integration)
            RUN_INTEGRATION=true
            shift
            ;;
        --e2e)
            RUN_E2E=true
            shift
            ;;
        --all)
            RUN_UNIT=true
            RUN_BDD=true
            RUN_INTEGRATION=true
            RUN_E2E=true
            shift
            ;;
        --no-coverage)
            GENERATE_COVERAGE=false
            shift
            ;;
        --no-cleanup)
            CLEANUP_AFTER=false
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            echo "用法: $0 [选项]"
            echo "选项:"
            echo "  --unit-only     仅运行单元测试"
            echo "  --bdd-only      仅运行BDD测试"
            echo "  --integration   运行集成测试"
            echo "  --e2e           运行端到端测试"
            echo "  --all           运行所有测试"
            echo "  --no-coverage   不生成覆盖率报告"
            echo "  --no-cleanup    测试后不清理环境"
            echo "  --verbose       详细输出"
            echo "  --help          显示帮助信息"
            exit 0
            ;;
        *)
            log_error "未知选项: $1"
            exit 1
            ;;
    esac
done

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

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    # 检测CI环境
    detect_ci_environment
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        log_error "Go未安装"
        exit 1
    fi
    
    # 检查Ginkgo（如果需要运行BDD测试）
    if [ "$RUN_BDD" = true ] || [ "$RUN_E2E" = true ]; then
        if ! command -v ginkgo &> /dev/null; then
            log_warning "Ginkgo未安装，正在安装..."
            go install github.com/onsi/ginkgo/v2/ginkgo@latest
        fi
    fi
    
    # 检查Docker（如果需要运行集成测试或端到端测试）
    if [ "$RUN_INTEGRATION" = true ] || [ "$RUN_E2E" = true ]; then
        if [ "$SKIP_DOCKER_SETUP" = false ]; then
            if ! command -v docker &> /dev/null; then
                log_error "Docker未安装，集成测试和端到端测试需要Docker"
                exit 1
            fi
            
            if ! command -v docker-compose &> /dev/null; then
                log_error "Docker Compose未安装"
                exit 1
            fi
        else
            log_info "CI环境检测到，使用外部服务，跳过Docker检查"
        fi
    fi
    
    # 检查数据库迁移工具（如果需要运行集成测试）
    if [ "$RUN_INTEGRATION" = true ] && [ "$USE_EXTERNAL_SERVICES" = true ]; then
        if ! command -v migrate &> /dev/null; then
            log_warning "migrate工具未安装，集成测试可能需要数据库迁移"
        fi
    fi
    
    log_success "依赖检查完成"
}

# 设置测试环境
setup_test_environment() {
    log_info "设置测试环境..."
    
    cd "$PROJECT_ROOT"
    
    # 确保依赖是最新的
    go mod tidy
    
    # 创建测试输出目录
    mkdir -p "$TEST_DIR/output"
    
    log_success "测试环境设置完成"
}

# 等待数据库就绪
wait_for_database() {
    if [ "$RUN_INTEGRATION" = true ] && [ "$USE_EXTERNAL_SERVICES" = true ]; then
        log_info "等待数据库就绪..."
        
        local max_attempts=30
        local attempt=1
        
        while [ $attempt -le $max_attempts ]; do
            if pg_isready -h localhost -p 5432 -U postgres 2>/dev/null; then
                log_success "数据库已就绪"
                return 0
            fi
            
            log_info "等待数据库启动... ($attempt/$max_attempts)"
            sleep 2
            attempt=$((attempt + 1))
        done
        
        log_error "数据库启动超时"
        return 1
    fi
}

# 运行数据库迁移
run_migrations() {
    if [ "$RUN_INTEGRATION" = true ] && [ "$USE_EXTERNAL_SERVICES" = true ]; then
        log_info "运行数据库迁移..."
        
        if command -v migrate &> /dev/null && [ -n "$DATABASE_URL" ]; then
            cd "$PROJECT_ROOT"
            migrate -path migrations -database "$DATABASE_URL" up
            log_success "数据库迁移完成"
        else
            log_warning "migrate工具未安装或DATABASE_URL未设置，跳过迁移"
        fi
    fi
}

# 启动测试服务（用于集成测试和端到端测试）
start_test_services() {
    if [ "$RUN_INTEGRATION" = true ] || [ "$RUN_E2E" = true ]; then
        if [ "$SKIP_DOCKER_SETUP" = true ]; then
            log_info "使用外部服务，跳过Docker服务启动"
            wait_for_database
            run_migrations
            return 0
        fi
        
        log_info "启动测试服务..."
        
        cd "$TEST_DIR/config"
        
        # 停止可能存在的服务
        docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true
        
        # 启动服务
        docker-compose -f docker-compose.test.yml up -d
        
        # 等待服务就绪
        log_info "等待服务启动..."
        sleep 30
        
        # 检查服务健康状态
        if [ "$RUN_E2E" = true ]; then
            log_info "等待应用服务就绪..."
            timeout 60 bash -c 'until curl -f http://localhost:8001/health; do sleep 2; done'
        fi
        
        log_success "测试服务启动完成"
    fi
}

# 运行单元测试
run_unit_tests() {
    if [ "$RUN_UNIT" = true ]; then
        log_info "运行单元测试..."
        
        cd "$PROJECT_ROOT"
        
        local coverage_flag=""
        if [ "$GENERATE_COVERAGE" = true ]; then
            coverage_flag="-coverprofile=$TEST_DIR/output/unit_coverage.out"
        fi
        
        local verbose_flag=""
        if [ "$VERBOSE" = true ]; then
            verbose_flag="-v"
        fi
        
        # 运行单元测试（排除service层，因为service层使用BDD）
        if go test $verbose_flag $coverage_flag $(go list ./internal/... | grep -v '/service'); then
            log_success "单元测试通过"
            
            if [ "$GENERATE_COVERAGE" = true ]; then
                # 生成覆盖率报告
                go tool cover -html="$TEST_DIR/output/unit_coverage.out" -o "$TEST_DIR/output/unit_coverage.html"
                COVERAGE=$(go tool cover -func="$TEST_DIR/output/unit_coverage.out" | grep total: | awk '{print $3}')
                log_info "单元测试覆盖率: $COVERAGE"
            fi
        else
            log_error "单元测试失败"
            return 1
        fi
    fi
}

# 运行BDD测试
run_bdd_tests() {
    if [ "$RUN_BDD" = true ]; then
        log_info "运行BDD测试..."
        
        cd "$PROJECT_ROOT"
        
        local verbose_flag=""
        if [ "$VERBOSE" = true ]; then
            verbose_flag="-v"
        fi
        
        # 运行service层的BDD测试
        if ginkgo $verbose_flag -r ./internal/service; then
            log_success "Service层BDD测试通过"
        else
            log_error "Service层BDD测试失败"
            return 1
        fi
        
        # 运行test/bdd目录下的BDD测试
        if [ -d "$TEST_DIR/bdd" ]; then
            if ginkgo $verbose_flag -r "$TEST_DIR/bdd"; then
                log_success "BDD测试套件通过"
            else
                log_error "BDD测试套件失败"
                return 1
            fi
        fi
    fi
}

# 运行集成测试
run_integration_tests() {
    if [ "$RUN_INTEGRATION" = true ]; then
        log_info "运行集成测试..."
        
        cd "$PROJECT_ROOT"
        
        local verbose_flag=""
        if [ "$VERBOSE" = true ]; then
            verbose_flag="-v"
        fi
        
        # 根据环境设置集成测试环境变量
        if [ "$USE_EXTERNAL_SERVICES" = true ]; then
            # CI环境：使用GitHub Actions services
            export TEST_DATABASE_URL="${DATABASE_URL:-postgresql://postgres:postgres@localhost:5432/test_db?sslmode=disable}"
            export TEST_REDIS_URL="${REDIS_URL:-redis://localhost:6379}"
            log_info "使用外部服务进行集成测试"
            log_info "数据库: $TEST_DATABASE_URL"
            log_info "Redis: $TEST_REDIS_URL"
        else
            # 本地环境：使用Docker Compose服务
            export TEST_DATABASE_URL="postgresql://postgres:postgres@localhost:5433/test_db?sslmode=disable"
            export TEST_REDIS_URL="localhost:6380"
            log_info "使用本地Docker服务进行集成测试"
        fi
        
        # 确保数据库连接可用
        if [ "$USE_EXTERNAL_SERVICES" = true ]; then
            log_info "验证数据库连接..."
            if ! pg_isready -h localhost -p 5432 -U postgres 2>/dev/null; then
                log_error "数据库连接失败"
                return 1
            fi
        fi
        
        # 运行集成测试
        log_info "执行集成测试..."
        if go test $verbose_flag -tags=integration "$TEST_DIR/integration/..."; then
            log_success "集成测试通过"
        else
            log_error "集成测试失败"
            return 1
        fi
    fi
}

# 运行端到端测试
run_e2e_tests() {
    if [ "$RUN_E2E" = true ]; then
        log_info "运行端到端测试..."
        
        cd "$PROJECT_ROOT"
        
        local verbose_flag=""
        if [ "$VERBOSE" = true ]; then
            verbose_flag="-v"
        fi
        
        # 设置端到端测试环境变量
        export TEST_SERVER_URL="http://localhost:8001"
        
        if ginkgo $verbose_flag -tags=e2e -r "$TEST_DIR/e2e"; then
            log_success "端到端测试通过"
        else
            log_error "端到端测试失败"
            return 1
        fi
    fi
}

# 清理测试环境
cleanup_test_environment() {
    if [ "$CLEANUP_AFTER" = true ]; then
        log_info "清理测试环境..."
        
        if [ "$RUN_INTEGRATION" = true ] || [ "$RUN_E2E" = true ]; then
            if [ "$SKIP_DOCKER_SETUP" = false ]; then
                cd "$TEST_DIR/config"
                docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true
                log_info "Docker服务已清理"
            else
                log_info "使用外部服务，跳过Docker清理"
            fi
        fi
        
        # 清理测试数据（如果使用外部服务）
        if [ "$USE_EXTERNAL_SERVICES" = true ] && [ "$RUN_INTEGRATION" = true ]; then
            log_info "清理测试数据..."
            if command -v migrate &> /dev/null && [ -n "$DATABASE_URL" ]; then
                cd "$PROJECT_ROOT"
                migrate -path migrations -database "$DATABASE_URL" down -all 2>/dev/null || true
                log_info "数据库已重置"
            fi
        fi
        
        log_success "测试环境清理完成"
    fi
}

# 生成测试报告
generate_test_report() {
    log_info "生成测试报告..."
    
    local report_file="$TEST_DIR/output/test_report.txt"
    
    {
        echo "=== BDD测试框架 - 测试报告 ==="
        echo "生成时间: $(date)"
        echo ""
        echo "测试配置:"
        echo "  单元测试: $RUN_UNIT"
        echo "  BDD测试: $RUN_BDD"
        echo "  集成测试: $RUN_INTEGRATION"
        echo "  端到端测试: $RUN_E2E"
        echo ""
        
        if [ "$GENERATE_COVERAGE" = true ] && [ -f "$TEST_DIR/output/unit_coverage.out" ]; then
            echo "覆盖率信息:"
            go tool cover -func="$TEST_DIR/output/unit_coverage.out" | tail -1
            echo ""
        fi
        
        echo "测试文件位置:"
        echo "  覆盖率报告: $TEST_DIR/output/unit_coverage.html"
        echo "  测试日志: $TEST_DIR/output/"
    } > "$report_file"
    
    log_success "测试报告已生成: $report_file"
}

# 主函数
main() {
    log_info "开始运行BDD测试框架"
    
    # 检查依赖
    check_dependencies
    
    # 设置测试环境
    setup_test_environment
    
    # 启动测试服务
    start_test_services
    
    # 运行测试
    local test_failed=false
    
    run_unit_tests || test_failed=true
    run_bdd_tests || test_failed=true
    run_integration_tests || test_failed=true
    run_e2e_tests || test_failed=true
    
    # 生成测试报告
    generate_test_report
    
    # 清理环境
    cleanup_test_environment
    
    if [ "$test_failed" = true ]; then
        log_error "部分测试失败"
        exit 1
    else
        log_success "所有测试通过！"
    fi
}

# 捕获中断信号，确保清理
trap cleanup_test_environment EXIT

# 运行主函数
main "$@"