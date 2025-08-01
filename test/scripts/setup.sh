#!/bin/bash

# BDD测试框架 - 环境设置脚本

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

# 检查并安装依赖
install_dependencies() {
    log_info "检查并安装测试依赖..."
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        log_error "Go未安装，请先安装Go"
        exit 1
    fi
    
    log_info "Go版本: $(go version)"
    
    # 安装Ginkgo
    if ! command -v ginkgo &> /dev/null; then
        log_info "安装Ginkgo..."
        go install github.com/onsi/ginkgo/v2/ginkgo@latest
    else
        log_info "Ginkgo已安装: $(ginkgo version)"
    fi
    
    # 检查Docker
    if command -v docker &> /dev/null; then
        log_info "Docker版本: $(docker --version)"
    else
        log_warning "Docker未安装，集成测试和端到端测试将无法运行"
    fi
    
    # 检查Docker Compose
    if command -v docker-compose &> /dev/null; then
        log_info "Docker Compose版本: $(docker-compose --version)"
    else
        log_warning "Docker Compose未安装，集成测试和端到端测试将无法运行"
    fi
    
    log_success "依赖检查完成"
}

# 创建测试目录结构
create_test_directories() {
    log_info "创建测试目录结构..."
    
    # 创建主要测试目录
    mkdir -p "$TEST_DIR/output"
    mkdir -p "$TEST_DIR/fixtures"
    mkdir -p "$TEST_DIR/logs"
    
    # 创建BDD测试目录
    mkdir -p "$TEST_DIR/bdd/auth"
    mkdir -p "$TEST_DIR/bdd/greeter"
    mkdir -p "$TEST_DIR/bdd/shared"
    
    # 创建集成测试目录
    mkdir -p "$TEST_DIR/integration/database"
    mkdir -p "$TEST_DIR/integration/redis"
    mkdir -p "$TEST_DIR/integration/api"
    
    # 创建端到端测试目录
    mkdir -p "$TEST_DIR/e2e/scenarios"
    mkdir -p "$TEST_DIR/e2e/fixtures"
    mkdir -p "$TEST_DIR/e2e/helpers"
    
    # 创建配置和脚本目录
    mkdir -p "$TEST_DIR/config"
    mkdir -p "$TEST_DIR/scripts"
    
    log_success "测试目录结构创建完成"
}

# 初始化Go模块依赖
init_go_dependencies() {
    log_info "初始化Go模块依赖..."
    
    cd "$PROJECT_ROOT"
    
    # 确保所有依赖都是最新的
    go mod tidy
    
    # 下载测试依赖
    go mod download
    
    # 验证模块
    go mod verify
    
    log_success "Go模块依赖初始化完成"
}

# 设置测试数据库
setup_test_database() {
    log_info "设置测试数据库..."
    
    if command -v docker &> /dev/null && command -v docker-compose &> /dev/null; then
        cd "$TEST_DIR/config"
        
        # 启动测试数据库
        docker-compose -f docker-compose.test.yml up -d test-db test-redis
        
        # 等待数据库启动
        log_info "等待数据库启动..."
        sleep 10
        
        # 检查数据库连接
        if docker-compose -f docker-compose.test.yml exec -T test-db pg_isready -U postgres; then
            log_success "测试数据库启动成功"
        else
            log_warning "测试数据库启动可能有问题，请检查Docker日志"
        fi
    else
        log_warning "Docker未安装，跳过测试数据库设置"
    fi
}

# 创建测试配置文件
create_test_configs() {
    log_info "创建测试配置文件..."
    
    # 创建.gitignore文件
    cat > "$TEST_DIR/.gitignore" << EOF
# 测试输出
output/
logs/
*.log
*.out
*.html

# 临时文件
*.tmp
*.temp

# IDE文件
.vscode/
.idea/
*.swp
*.swo
EOF
    
    # 创建Makefile
    cat > "$TEST_DIR/Makefile" << EOF
# BDD测试框架 Makefile

.PHONY: help setup test test-unit test-bdd test-integration test-e2e test-all clean

# 默认目标
help:
	@echo "BDD测试框架可用命令:"
	@echo "  setup           - 设置测试环境"
	@echo "  test            - 运行单元测试和BDD测试"
	@echo "  test-unit       - 仅运行单元测试"
	@echo "  test-bdd        - 仅运行BDD测试"
	@echo "  test-integration - 运行集成测试"
	@echo "  test-e2e        - 运行端到端测试"
	@echo "  test-all        - 运行所有测试"
	@echo "  clean           - 清理测试环境"

# 设置测试环境
setup:
	./scripts/setup.sh

# 运行基础测试
test:
	./scripts/run_all.sh --unit-only --bdd-only

# 运行单元测试
test-unit:
	./scripts/run_all.sh --unit-only

# 运行BDD测试
test-bdd:
	./scripts/run_all.sh --bdd-only

# 运行集成测试
test-integration:
	./scripts/run_all.sh --integration

# 运行端到端测试
test-e2e:
	./scripts/run_all.sh --e2e

# 运行所有测试
test-all:
	./scripts/run_all.sh --all

# 清理测试环境
clean:
	./scripts/teardown.sh
EOF
    
    log_success "测试配置文件创建完成"
}

# 设置权限
set_permissions() {
    log_info "设置脚本权限..."
    
    # 设置脚本可执行权限
    chmod +x "$TEST_DIR/scripts/"*.sh
    
    log_success "脚本权限设置完成"
}

# 验证设置
validate_setup() {
    log_info "验证测试环境设置..."
    
    # 检查目录结构
    local required_dirs=(
        "$TEST_DIR/bdd"
        "$TEST_DIR/integration"
        "$TEST_DIR/e2e"
        "$TEST_DIR/config"
        "$TEST_DIR/scripts"
        "$TEST_DIR/output"
    )
    
    for dir in "${required_dirs[@]}"; do
        if [ ! -d "$dir" ]; then
            log_error "目录不存在: $dir"
            return 1
        fi
    done
    
    # 检查脚本文件
    local required_scripts=(
        "$TEST_DIR/scripts/run_all.sh"
        "$TEST_DIR/scripts/setup.sh"
        "$TEST_DIR/scripts/teardown.sh"
    )
    
    for script in "${required_scripts[@]}"; do
        if [ ! -f "$script" ]; then
            log_error "脚本文件不存在: $script"
            return 1
        fi
        
        if [ ! -x "$script" ]; then
            log_error "脚本文件不可执行: $script"
            return 1
        fi
    done
    
    # 运行简单的Go测试验证
    cd "$PROJECT_ROOT"
    if go test -run=^$ ./internal/...; then
        log_success "Go测试环境验证通过"
    else
        log_error "Go测试环境验证失败"
        return 1
    fi
    
    log_success "测试环境设置验证通过"
}

# 显示使用说明
show_usage() {
    log_info "BDD测试框架设置完成！"
    echo ""
    echo "使用方法:"
    echo "  cd $TEST_DIR"
    echo "  make help                    # 查看所有可用命令"
    echo "  make test                    # 运行基础测试"
    echo "  make test-all                # 运行所有测试"
    echo "  ./scripts/run_all.sh --help  # 查看详细选项"
    echo ""
    echo "测试目录结构:"
    echo "  bdd/          - BDD测试套件"
    echo "  integration/  - 集成测试"
    echo "  e2e/          - 端到端测试"
    echo "  config/       - 测试配置"
    echo "  scripts/      - 测试脚本"
    echo "  output/       - 测试输出和报告"
    echo ""
}

# 主函数
main() {
    log_info "开始设置BDD测试框架..."
    
    install_dependencies
    create_test_directories
    init_go_dependencies
    create_test_configs
    set_permissions
    setup_test_database
    validate_setup
    show_usage
    
    log_success "BDD测试框架设置完成！"
}

# 运行主函数
main "$@"