#!/bin/bash

# 增强的测试运行脚本 - 集成BDD测试框架

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
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 检查是否存在新的BDD测试框架
if [ -f "$PROJECT_ROOT/test/scripts/run_all.sh" ]; then
    log_info "检测到BDD测试框架，使用增强的测试运行器"
    
    # 检查命令行参数
    if [ "$1" = "--bdd-framework" ]; then
        log_info "使用完整的BDD测试框架"
        exec "$PROJECT_ROOT/test/scripts/run_all.sh" "${@:2}"
    elif [ "$1" = "--help" ]; then
        echo "用法: $0 [选项]"
        echo "选项:"
        echo "  --bdd-framework  使用完整的BDD测试框架"
        echo "  --help           显示帮助信息"
        echo ""
        echo "默认行为: 运行传统的TDD+BDD测试"
        echo "BDD框架选项: 运行 $PROJECT_ROOT/test/scripts/run_all.sh --help 查看更多选项"
        exit 0
    else
        log_info "运行传统的TDD+BDD测试（兼容模式）"
    fi
else
    log_info "使用传统测试模式"
fi

log_info "运行Go单元测试 (TDD, 排除service层)..."
go test $(go list ./internal/... | grep -v '/service') -coverprofile=coverage.out > tdd_test.log
cat tdd_test.log

log_info "生成覆盖率报告..."
go tool cover -html=coverage.out -o coverage.html
log_success "覆盖率报告已生成: coverage.html (在浏览器中打开查看详情)"

COVERAGE=$(go tool cover -func=coverage.out | grep total: | awk '{print $3}')
log_info "总覆盖率 (仅TDD，排除service层): $COVERAGE"

log_info "运行Ginkgo BDD测试 (service层)..."
if command -v ginkgo &> /dev/null; then
    if ginkgo -r ./internal/service; then
        log_success "Service层BDD测试通过"
    else
        log_error "Service层BDD测试失败"
        exit 1
    fi
    
    # 如果存在BDD测试框架，也运行其中的BDD测试
    if [ -d "$PROJECT_ROOT/test/bdd" ]; then
        log_info "运行BDD测试框架中的额外测试..."
        if ginkgo -r "$PROJECT_ROOT/test/bdd"; then
            log_success "BDD测试框架测试通过"
        else
            log_warning "BDD测试框架测试失败，但不影响主要测试结果"
        fi
    fi
else
    log_error "Ginkgo未安装，请运行: go install github.com/onsi/ginkgo/v2/ginkgo@latest"
    exit 1
fi

log_success "所有测试完成！"

# 如果存在BDD测试框架，提示用户
if [ -f "$PROJECT_ROOT/test/scripts/run_all.sh" ]; then
    echo ""
    log_info "提示: 您可以使用完整的BDD测试框架获得更多功能:"
    echo "  $0 --bdd-framework --help    # 查看BDD框架选项"
    echo "  $0 --bdd-framework --all     # 运行所有类型的测试"
    echo "  cd test && make help         # 查看Makefile命令"
fi
