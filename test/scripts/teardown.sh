#!/bin/bash

# BDD测试框架 - 环境清理脚本

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
CLEAN_DOCKER=true
CLEAN_OUTPUT=true
CLEAN_LOGS=true
FORCE_CLEAN=false

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-docker)
            CLEAN_DOCKER=false
            shift
            ;;
        --no-output)
            CLEAN_OUTPUT=false
            shift
            ;;
        --no-logs)
            CLEAN_LOGS=false
            shift
            ;;
        --force)
            FORCE_CLEAN=true
            shift
            ;;
        --help)
            echo "用法: $0 [选项]"
            echo "选项:"
            echo "  --no-docker   不清理Docker容器和卷"
            echo "  --no-output   不清理测试输出文件"
            echo "  --no-logs     不清理日志文件"
            echo "  --force       强制清理，不询问确认"
            echo "  --help        显示帮助信息"
            exit 0
            ;;
        *)
            log_error "未知选项: $1"
            exit 1
            ;;
    esac
done

# 确认清理操作
confirm_cleanup() {
    if [ "$FORCE_CLEAN" = false ]; then
        echo "即将清理以下内容:"
        [ "$CLEAN_DOCKER" = true ] && echo "  - Docker容器和卷"
        [ "$CLEAN_OUTPUT" = true ] && echo "  - 测试输出文件"
        [ "$CLEAN_LOGS" = true ] && echo "  - 日志文件"
        echo ""
        read -p "确认继续？(y/N): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "取消清理操作"
            exit 0
        fi
    fi
}

# 停止并清理Docker容器
cleanup_docker() {
    if [ "$CLEAN_DOCKER" = true ]; then
        log_info "清理Docker容器和卷..."
        
        if command -v docker-compose &> /dev/null; then
            cd "$TEST_DIR/config"
            
            # 停止并删除容器
            if [ -f "docker-compose.test.yml" ]; then
                docker-compose -f docker-compose.test.yml down -v --remove-orphans 2>/dev/null || true
                log_success "Docker容器已停止并删除"
            fi
            
            # 清理未使用的卷（可选）
            if [ "$FORCE_CLEAN" = true ]; then
                docker volume prune -f 2>/dev/null || true
                log_info "未使用的Docker卷已清理"
            fi
            
            # 清理未使用的网络（可选）
            if [ "$FORCE_CLEAN" = true ]; then
                docker network prune -f 2>/dev/null || true
                log_info "未使用的Docker网络已清理"
            fi
        else
            log_warning "Docker Compose未安装，跳过Docker清理"
        fi
    fi
}

# 清理测试输出文件
cleanup_output() {
    if [ "$CLEAN_OUTPUT" = true ]; then
        log_info "清理测试输出文件..."
        
        # 清理输出目录
        if [ -d "$TEST_DIR/output" ]; then
            rm -rf "$TEST_DIR/output"/*
            log_success "测试输出文件已清理"
        fi
        
        # 清理项目根目录下的覆盖率文件
        cd "$PROJECT_ROOT"
        rm -f coverage.out coverage.html
        
        # 清理测试缓存
        go clean -testcache
        
        log_success "Go测试缓存已清理"
    fi
}

# 清理日志文件
cleanup_logs() {
    if [ "$CLEAN_LOGS" = true ]; then
        log_info "清理日志文件..."
        
        # 清理日志目录
        if [ -d "$TEST_DIR/logs" ]; then
            rm -rf "$TEST_DIR/logs"/*
            log_success "日志文件已清理"
        fi
        
        # 清理项目中的日志文件
        find "$PROJECT_ROOT" -name "*.log" -type f -delete 2>/dev/null || true
        
        log_success "项目日志文件已清理"
    fi
}

# 清理临时文件
cleanup_temp_files() {
    log_info "清理临时文件..."
    
    # 清理临时文件
    find "$TEST_DIR" -name "*.tmp" -type f -delete 2>/dev/null || true
    find "$TEST_DIR" -name "*.temp" -type f -delete 2>/dev/null || true
    find "$TEST_DIR" -name ".DS_Store" -type f -delete 2>/dev/null || true
    
    # 清理空目录
    find "$TEST_DIR/output" -type d -empty -delete 2>/dev/null || true
    find "$TEST_DIR/logs" -type d -empty -delete 2>/dev/null || true
    
    log_success "临时文件已清理"
}

# 停止可能运行的测试进程
stop_test_processes() {
    log_info "停止可能运行的测试进程..."
    
    # 查找并停止Ginkgo进程
    pkill -f "ginkgo" 2>/dev/null || true
    
    # 查找并停止Go测试进程
    pkill -f "go test" 2>/dev/null || true
    
    # 等待进程完全停止
    sleep 2
    
    log_success "测试进程已停止"
}

# 重置测试环境
reset_test_environment() {
    log_info "重置测试环境..."
    
    # 重新创建必要的目录
    mkdir -p "$TEST_DIR/output"
    mkdir -p "$TEST_DIR/logs"
    
    # 重置权限
    if [ -d "$TEST_DIR/scripts" ]; then
        chmod +x "$TEST_DIR/scripts/"*.sh 2>/dev/null || true
    fi
    
    log_success "测试环境已重置"
}

# 显示清理结果
show_cleanup_result() {
    log_info "清理操作完成！"
    echo ""
    echo "已清理的内容:"
    [ "$CLEAN_DOCKER" = true ] && echo "  ✓ Docker容器和卷"
    [ "$CLEAN_OUTPUT" = true ] && echo "  ✓ 测试输出文件"
    [ "$CLEAN_LOGS" = true ] && echo "  ✓ 日志文件"
    echo "  ✓ 临时文件"
    echo "  ✓ 测试进程"
    echo ""
    echo "测试环境已重置，可以重新运行测试。"
    echo ""
}

# 错误处理
handle_error() {
    log_error "清理过程中发生错误: $1"
    log_info "尝试继续清理其他组件..."
}

# 主函数
main() {
    log_info "开始清理BDD测试环境..."
    
    # 确认清理操作
    confirm_cleanup
    
    # 执行清理操作（即使某个步骤失败也继续执行其他步骤）
    stop_test_processes || handle_error "停止测试进程"
    cleanup_docker || handle_error "清理Docker"
    cleanup_output || handle_error "清理输出文件"
    cleanup_logs || handle_error "清理日志文件"
    cleanup_temp_files || handle_error "清理临时文件"
    reset_test_environment || handle_error "重置测试环境"
    
    # 显示结果
    show_cleanup_result
    
    log_success "BDD测试环境清理完成！"
}

# 捕获中断信号
trap 'log_warning "清理操作被中断"' INT TERM

# 运行主函数
main "$@"