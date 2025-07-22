# Kratos Project Template

[![codecov](https://codecov.io/gh/26huitailang/kratos-boilerplate/branch/main/graph/badge.svg)](https://codecov.io/gh/26huitailang/kratos-boilerplate)

## Install Kratos
```
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
```

## Project Setup
1. Install dependencies and tools:
```bash
# Download and update dependencies
make init
```

2. Generate API files:
```bash
# Generate API files (include: pb.go, http, grpc, validate, swagger)
make api
```

3. Build and run:
```bash
# Build
make build

# Run
./bin/xredline -conf ./configs
```

## Development

### Prerequisites
- Go 1.22+
- Protocol Buffers v3
- Docker (for running PostgreSQL)

### Database Setup
```bash
# Start PostgreSQL
docker compose up db -d
```

### Code Generation
```bash
# Generate all files (API, configs, etc.)
make all

# Generate API files only
make api

# Generate config only
make config
```

### Build and Run
```bash
# Build
make build

# Run
./bin/xredline -conf ./configs
```

## Docker
```bash
# build
docker build -t <your-docker-image-name> .

# run
docker run --rm -p 8000:8000 -p 9000:9000 -v </path/to/your/configs>:/data/conf <your-docker-image-name>
```

## API Documentation
After starting the service, you can access the Swagger UI at:
```
http://localhost:8000/q/swagger-ui/
```

## 测试规范与运行

本项目采用TDD（底层逻辑）+ BDD（业务逻辑）混合测试策略：
- TDD：底层模块使用Go原生testing+testify，见internal/biz、internal/pkg等目录。
- BDD：业务流程使用ginkgo/gomega，见internal/service目录。

一键运行所有测试：
```sh
scripts/run_tests.sh
```
- 运行后会自动生成 coverage.html 覆盖率报告，可用浏览器打开查看详细覆盖情况。
- 覆盖率建议 >80%，详细规范见 docs/测试规范.md。
