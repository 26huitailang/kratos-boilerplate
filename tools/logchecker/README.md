# 日志规范检查工具 (Log Compliance Checker)

基于Go AST的静态代码分析工具，用于检测项目中不符合结构化日志规范的代码。

## 快速开始

### 构建工具

```bash
go build -o logchecker .
```

### 基本使用

```bash
# 检查当前目录
./logchecker

# 检查指定目录
./logchecker -dir ../../internal

# 使用配置文件
./logchecker -dir ../../internal -config logchecker.json

# 生成JSON报告
./logchecker -dir ../../internal -output json

# 生成HTML报告
./logchecker -dir ../../internal -output html

# 启用详细输出
./logchecker -dir ../../internal -verbose
```

### 通过Makefile使用

在项目根目录执行：

```bash
# 运行日志检查
make logcheck

# 生成JSON报告
make logcheck-json

# 生成HTML报告
make logcheck-html

# 安装到系统PATH
make logcheck-install
```

## 检测规则

工具会检测以下日志方法的使用并建议替换为结构化日志：

- `Infof` → `Infow`
- `Debugf` → `Debugw`
- `Errorf` → `Errorw`
- `Warnf` → `Warnw`
- `Info` → `Infow` (在特定条件下)
- `Debug` → `Debugw` (在特定条件下)
- `Error` → `Errorw` (在特定条件下)
- `Warn` → `Warnw` (在特定条件下)

### 触发检测的条件

- 多个参数
- 包含格式化占位符 (`%`)
- 消息长度超过阈值
- 包含敏感关键词

## 配置文件

默认配置文件 `logchecker.json`：

```json
{
  "skip_test_files": true,
  "ignore_files": ["*.pb.go", "*.gen.go"],
  "ignore_dirs": ["vendor", ".git"],
  "whitelist_methods": ["Printf"],
  "sensitive_keywords": ["password", "token", "secret"],
  "min_message_length": 50
}
```

## 输出格式

### 控制台输出

```
=== 日志规范检查报告 ===
扫描时间: 2024-01-15 10:30:45
总文件数: 25
已扫描文件数: 20
发现问题数: 8

🔴 高严重程度问题 (2个):
  📁 internal/service/auth.go:45:3
     问题: 使用了格式化日志方法 'Errorf'，建议使用结构化日志方法 'Errorw'
     建议: 使用 Errorw 替代 Errorf
```

### JSON输出

```json
{
  "timestamp": "2024-01-15T10:30:45Z",
  "total_files": 25,
  "scanned_files": 20,
  "total_issues": 8,
  "issues": [
    {
      "file": "internal/service/auth.go",
      "line": 45,
      "column": 3,
      "method": "Errorf",
      "suggestion": "Errorw",
      "description": "使用了格式化日志方法 'Errorf'，建议使用结构化日志方法 'Errorw'",
      "severity": "high"
    }
  ]
}
```

## 测试

```bash
# 运行单元测试
go test -v

# 运行测试并查看覆盖率
go test -v -cover
```

## 文件结构

```
tools/logchecker/
├── main.go              # 命令行入口
├── checker.go           # 核心检查器
├── visitor.go           # AST访问器
├── config.go            # 配置管理
├── reporter.go          # 报告生成器
├── checker_test.go      # 单元测试
├── logchecker.json      # 默认配置
├── testdata/            # 测试数据
│   └── sample.go
├── go.mod
└── README.md
```

## 集成到CI/CD

工具已集成到项目的GitHub Actions工作流中，会在每次推送和PR时自动运行检查。

## 相关文档

- [详细使用指南](../../docs/log-compliance-checker.md)
- [结构化日志指南](../../docs/structured-logging-guide.md)