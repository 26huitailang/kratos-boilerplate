# 构建与部署指南

## 环境要求

- Docker 20.10.0 或更高版本
- Docker Compose v2.0.0 或更高版本
- Go 1.24.1 或更高版本
- Node.js 和 npm（用于前端构建）
- 至少 2GB 可用内存
- 至少 10GB 可用磁盘空间

## 开发环境准备

### 安装必要的工具
```bash
make init
```
这将安装以下工具：
- protoc-gen-go
- protoc-gen-go-grpc
- kratos
- protoc-gen-go-http
- protoc-gen-openapi
- wire

### 开发构建

1. 生成 API 和配置
```bash
make all
```
这个命令会：
- 生成 API 相关的 protobuf 代码
- 生成内部配置相关的 protobuf 代码
- 执行 go generate 生成其他代码
- 更新 go.mod 依赖

2. 构建二进制文件
```bash
make build
```
这将在 `bin/` 目录下生成可执行文件。

## 部署

### 使用 Docker Compose 部署

1. 克隆代码仓库
```bash
git clone <repository_url>
cd cross-redline
```

2. 使用 Docker Compose 启动服务
```bash
docker compose up -d
```

这将启动以下服务：
- HTTP API 服务 (端口 8000)
- gRPC 服务 (端口 9000)
- PostgreSQL 数据库 (端口 5432)

### 服务说明

#### 应用服务 (cross-redline-app)
- 基于 openEuler 24.03 LTS
- 暴露端口：
  - 8000：HTTP API
  - 9000：gRPC
- 配置文件位置：`/data/conf/config.yaml`
- 时区设置：Asia/Shanghai

#### 数据库服务 (cross-redline-db)
- PostgreSQL 14
- 默认凭据：
  - 用户名：postgres
  - 密码：postgres
  - 数据库：test
- 数据持久化：使用 Docker volume (postgres_data)
- 健康检查：每 5 秒检查一次

### 配置文件

配置文件位于 `configs/config.yaml`，包含以下主要配置：
- HTTP 服务配置
- gRPC 服务配置
- 数据库连接配置
- Redis 配置（如果需要）

### 常用操作

#### 查看服务状态
```bash
docker compose ps
```

#### 查看服务日志
```bash
# 查看所有服务日志
docker compose logs

# 查看特定服务日志
docker compose logs app
docker compose logs db
```

#### 重启服务
```bash
docker compose restart
```

#### 停止服务
```bash
docker compose down
```

#### 完全清理（包括数据卷）
```bash
docker compose down -v
```

### 网络配置

服务运行在名为 `cross-redline-network` 的 Docker 网络中，这是一个桥接网络，允许容器之间通过容器名称相互访问。

### 数据持久化

数据库数据存储在名为 `postgres_data` 的 Docker volume 中，即使容器被删除，数据也会保持。

## GitHub Actions 构建

项目使用 GitHub Actions 进行自动化构建和部署。构建流程包括：

1. 代码检查
2. 依赖安装
3. 代码生成
4. 二进制构建
5. Docker 镜像构建和推送

### 构建环境

构建环境基于 openEuler 24.03 LTS，包含以下工具：
- Go 1.24.1（从官方源安装）
- Node.js 和 npm
- Git
- Make
- wget、tar 等基础工具

Go 环境配置：
- GOPROXY=https://goproxy.cn
- GO111MODULE=on
- GOROOT=/usr/local/go
- 其他必要的构建工具

## 故障排除

1. 如果应用服务无法连接到数据库：
   - 检查数据库服务是否健康：`docker compose ps`
   - 检查数据库日志：`docker compose logs db`
   - 确认配置文件中的数据库连接字符串正确

2. 如果服务无法启动：
   - 检查端口是否被占用
   - 检查配置文件是否正确
   - 查看服务日志了解详细错误信息

3. 如果构建失败：
   - 检查 Go 版本是否满足要求
   - 确保所有必要的工具都已安装
   - 检查网络连接是否正常
   - 查看构建日志了解详细错误信息