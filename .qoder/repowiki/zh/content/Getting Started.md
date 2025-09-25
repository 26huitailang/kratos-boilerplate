# Getting Started

<cite>
**本文档中引用的文件**   
- [README.md](file://README.md)
- [Makefile](file://Makefile)
- [docker-compose.yml](file://docker-compose.yml)
- [docker-compose.dev.yml](file://docker-compose.dev.yml)
- [test/scripts/run_all.sh](file://test/scripts/run_all.sh)
- [configs/config.yaml](file://configs/config.yaml)
- [cmd/kratos-boilerplate/main.go](file://cmd/kratos-boilerplate/main.go)
- [frontend/package.json](file://frontend/package.json)
</cite>

## 目录
1. [介绍](#介绍)
2. [先决条件和环境设置](#先决条件和环境设置)
3. [快速入门指南](#快速入门指南)
4. [运行应用程序](#运行应用程序)
5. [前端和API访问](#前端和api访问)
6. [Makefile命令概述](#makefile命令概述)
7. [常见问题排查](#常见问题排查)
8. [结论](#结论)

## 介绍
本指南为新加入 **kratos-boilerplate** 项目的开发者提供了全面的入门流程。涵盖了建立完整功能开发环境所需的所有步骤，包括先决条件、安装、配置以及使用直接执行和基于Docker的方法进行工作流操作。目标是让任何开发者在10分钟内拥有一个可工作的应用程序实例。

该项目使用Go语言和Kratos框架构建，包含Vue 3前端，并使用PostgreSQL作为主要数据库。它遵循模块化架构，使用Protocol Buffers定义API，并支持基于插件的可扩展性。

**Section sources**
- [README.md](file://README.md#L1-L92)

## 先决条件和环境设置
在开始项目之前，请确保系统上已安装以下工具：

- **Go 1.22或更高版本**：项目使用Go编写，需要1.22+版本。
- **Node.js（含npm）**：用于构建和运行前端应用。
- **Docker和Docker Compose**：用于管理PostgreSQL数据库和可选的容器化执行。
- **Protocol Buffers编译器（protoc）**：需要从`.proto`文件生成API代码。
- **Kratos CLI**：Go-Kratos项目的官方命令行工具。

要安装所需的工具和依赖项，请运行以下命令：

```bash
make init
```

此命令安装以下必要的Go工具：
- `protoc-gen-go`
- `protoc-gen-go-grpc`
- `kratos` CLI
- `protoc-gen-go-http`
- `protoc-gen-openapi`
- `wire`

此外，前端使用 **Vue 3**、**Vite**、**Pinia** 和 **Element Plus**，这些通过`npm`管理。当您运行前端特定命令时，它们将自动安装。

**Section sources**
- [README.md](file://README.md#L10-L15)
- [Makefile](file://Makefile#L10-L20)

## 快速入门指南
按照以下步骤快速启动并运行项目。

### 步骤1：克隆仓库
```bash
git clone https://github.com/26huitailang/kratos-boilerplate.git
cd kratos-boilerplate
```

### 步骤2：使用Docker启动数据库
项目使用PostgreSQL作为其数据库。使用Docker Compose启动数据库容器：

```bash
docker compose up db -d
```

此命令启动`docker-compose.yml`中定义的PostgreSQL服务，在端口`5432`上暴露。数据库容器名为`cross-redline-db`，并以用户名`postgres`、密码`postgres`和数据库名`test`初始化。

```mermaid
flowchart TD
A["启动Docker Compose"] --> B["启动PostgreSQL容器"]
B --> C["暴露端口5432"]
C --> D["初始化数据库: test"]
D --> E["准备应用程序连接"]
```

**Diagram sources**
- [docker-compose.yml](file://docker-compose.yml#L15-L34)

### 步骤3：生成API和配置文件
从`.proto`定义生成必要的Go和API文件：

```bash
make api
```

此命令运行`protoc`来生成：
- Go结构体（`*.pb.go`）
- HTTP和gRPC服务器绑定
- OpenAPI/Swagger规范

您还可以生成内部配置文件：

```bash
make config
```

并生成所有代码（API + 配置 + wire DI）：

```bash
make all
```

**Section sources**
- [README.md](file://README.md#L17-L25)
- [Makefile](file://Makefile#L22-L41)

## 运行应用程序
设置好环境并生成必要文件后，可以构建和运行应用程序。

### 构建应用程序
```bash
make build
```

这会将Go应用程序编译成位于`./bin/kratos-boilerplate`的二进制文件。

### 运行应用程序
```bash
./bin/kratos-boilerplate -conf ./configs
```

`-conf`标志指定配置目录。应用程序从此目录读取`config.yaml`以配置服务器、数据库连接、插件和功能开关。

或者，您可以使用Docker运行整个堆栈：

```bash
docker compose up --build
```

这会使用`Dockerfile`构建应用程序镜像，并启动`docker-compose.yml`中定义的应用程序和数据库容器。

```mermaid
sequenceDiagram
participant User as 开发者
participant Make as Makefile
participant App as 应用程序
participant DB as 数据库
User->>Make : make build
Make->>App : 编译Go代码
App-->>Make : 输出二进制文件到 ./bin/
User->>App : ./bin/kratos-boilerplate -conf ./configs
App->>DB : 通过PostgreSQL URL连接
DB-->>App : 建立连接
App-->>User : HTTP服务器在 : 8000 上运行
```

**Diagram sources**
- [Makefile](file://Makefile#L43-L50)
- [configs/config.yaml](file://configs/config.yaml#L1-L38)
- [cmd/kratos-boilerplate/main.go](file://cmd/kratos-boilerplate/main.go#L1-L91)

## 前端和API访问
一旦应用程序运行起来，就可以访问以下端点：

### 前端应用程序
前端通过后端在端口`8000`提供服务。打开浏览器并导航至：

```
http://localhost:8000
```

前端使用Vue 3构建，并使用Vite进行开发。您也可以独立运行前端：

```bash
cd frontend
npm install
npm run dev
```

这会在`http://localhost:5173`启动Vite开发服务器。

### API文档
Swagger UI可在以下位置获得：

```
http://localhost:8000/q/swagger-ui/
```

此端点在`main.go`文件中使用`openapiv2.NewHandler()`中间件自动注册。

### 健康检查端点
基本健康检查可在以下位置获得：

```
GET http://localhost:8000/health
```

## Makefile命令概述
`Makefile`提供了几个方便的开发命令：

| 命令 | 描述 |
|--------|-------------|
| `make init` | 安装所有必需的Go工具 |
| `make api` | 从`.proto`定义生成API文件 |
| `make config` | 生成内部配置Go文件 |
| `make all` | 运行`api`、`config`和`generate` |
| `make build` | 构建应用程序二进制文件 |
| `make test` | 运行所有Go测试 |
| `make test-coverage` | 运行测试并生成HTML覆盖率报告 |
| `make logcheck` | 使用自定义工具检查日志使用合规性 |

对于测试，可以使用脚本`scripts/run_tests.sh`运行所有测试，包括BDD和集成测试：

```bash
scripts/run_tests.sh
```

此脚本处理环境设置、数据库初始化、测试执行和清理。

**Section sources**
- [Makefile](file://Makefile#L1-L140)
- [README.md](file://README.md#L70-L92)
- [test/scripts/run_all.sh](file://test/scripts/run_all.sh#L183-L436)

## 常见问题排查
以下是常见设置问题的解决方案：

### 端口冲突
如果遇到类似`port is already allocated`的错误，请确保没有其他服务正在使用端口`8000`、`9000`或`5432`：

```bash
lsof -i :8000
lsof -i :5432
```

停止冲突进程或修改`docker-compose.yml`中的端口。

### 依赖安装失败
如果`make init`由于网络问题（例如在中国）失败，请配置Go代理：

```bash
go env -w GOPROXY=https://goproxy.io,direct
```

然后重试`make init`。

### 数据库初始化问题
如果应用程序无法连接到数据库：
1. 确保数据库容器正在运行：
   ```bash
   docker compose ps
   ```
2. 检查日志：
   ```bash
   docker compose logs db
   ```
3. 验证`configs/config.yaml`中的连接字符串是否与服务名称`cross-redline-db`匹配。

### 缺少迁移文件
确保`migrations/`目录中的迁移文件格式正确并已应用：

```bash
# 如有必要手动应用迁移
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/test?sslmode=disable" up
```

### 代码生成错误
如果`make api`失败：
- 确保已安装`protoc`且在PATH中。
- 验证所有`.proto`文件语法正确。
- 确认存在`third_party`依赖项。

**Section sources**
- [configs/config.yaml](file://configs/config.yaml#L15-L20)
- [docker-compose.yml](file://docker-compose.yml#L15-L34)
- [test/scripts/run_all.sh](file://test/scripts/run_all.sh#L237-L269)

## 结论
本指南已带您完成从零开始设置 **kratos-boilerplate** 项目的全过程。您已经学习了如何：
- 使用`make init`安装先决条件
- 使用Docker启动PostgreSQL数据库
- 生成API和配置文件
- 构建和运行应用程序
- 访问前端和API文档
- 使用Makefile命令进行高效开发
- 排查常见问题

有了这个基础，您就可以开始开发功能、编写测试，并使用其插件架构和模块化设计扩展应用程序。