#!/bin/bash

set -e

# 记录开始时间
start_time=$(date +%s)

# 输出彩色文本
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
ROOT_DIR=$(pwd)

echo -e "${BLUE}开始运行测试...${NC}"

# 运行认证模块测试
echo -e "${YELLOW}测试认证模块 (internal/biz)...${NC}"
go test -v -cover ./internal/biz/...

echo -e "${YELLOW}测试验证码模块 (internal/pkg/captcha)...${NC}"
go test -v -cover ./internal/pkg/captcha/...

echo -e "${YELLOW}测试数据层 (internal/data)...${NC}"
go test -v -cover ./internal/data/...

# 计算覆盖率
echo -e "${YELLOW}计算总体测试覆盖率...${NC}"
go test -cover ./internal/... | grep -v "no test files" | awk '{sum+=$5; count++} END {print "平均覆盖率:", sum/count, "%"}'

# 记录结束时间并计算总用时
end_time=$(date +%s)
duration=$((end_time - start_time))
echo -e "${GREEN}测试完成! 总用时: ${duration}秒${NC}"
