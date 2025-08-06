# 日志规范检查工具使用指南

## 概述

日志规范检查工具是一个基于Go AST的静态代码分析工具，用于检测项目中不符合结构化日志规范的代码，并生成详细的告警报告。

## 功能特性

- 🔍 **AST静态分析**: 基于Go语法树进行精确的代码分析
- 📊 **多种输出格式**: 支持控制台、JSON、HTML格式的报告
- ⚙️ **灵活配置**: 支持忽略规则、白名单、敏感关键词等配置
- 🎯 **智能检测**: 自动识别应该使用结构化日志的场景
- 📈 **严重程度分级**: 根据日志级别和内容确定问题严重程度
- 🔧 **CI/CD集成**: 无缝集成到持续集成流程中

## 安装和使用

### 本地使用

#### 1. 基本使用

```bash
# 检查internal目录
make logcheck

# 生成JSON格式报告
make logcheck-json

# 生成HTML格式报告
make logcheck-html

# 安装到系统PATH
make logcheck-install
```

#### 2. 直接使用工具

```bash
# 进入工具目录
cd tools/logchecker

# 编译工具
go build -o logchecker .

# 基本使用
./logchecker -dir ../../internal

# 使用配置文件
./logchecker -dir ../../internal -config logchecker.json

# 指定输出格式
./logchecker -dir ../../internal -output json
./logchecker -dir ../../internal -output html

# 启用详细输出
./logchecker -dir ../../internal -verbose
```

### 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-dir` | 要扫描的目录路径 | `.` |
| `-config` | 配置文件路径 | 使用默认配置 |
| `-output` | 输出格式：console/json/html | `console` |
| `-verbose` | 启用详细输出 | `false` |

## 配置文件

工具支持JSON格式的配置文件，默认配置文件为 `tools/logchecker/logchecker.json`：

```json
{
  "skip_test_files": true,
  "ignore_files": [
    "*.pb.go",
    "*.gen.go",
    "vendor/*",
    ".git/*"
  ],
  "ignore_dirs": [
    "vendor",
    ".git",
    "node_modules",
    "third_party"
  ],
  "whitelist_files": [],
  "whitelist_methods": [
    "Printf"
  ],
  "sensitive_keywords": [
    "password", "token", "secret", "key", "auth",
    "email", "phone", "mobile", "card", "id",
    "user", "account", "login", "credential"
  ],
  "min_message_length": 50,
  "severity_config": {
    "error_methods": ["Errorf", "Error"],
    "warn_methods": ["Warnf", "Warn", "Warningf", "Warning"],
    "info_methods": ["Infof", "Info", "Debugf", "Debug"]
  }
}
```

### 配置项说明

- `skip_test_files`: 是否跳过测试文件（`*_test.go`）
- `ignore_files`: 忽略的文件模式列表
- `ignore_dirs`: 忽略的目录列表
- `whitelist_files`: 白名单文件（允许使用格式化日志）
- `whitelist_methods`: 白名单方法（允许的格式化日志方法）
- `sensitive_keywords`: 敏感关键词列表
- `min_message_length`: 消息长度阈值（超过此长度建议使用结构化日志）
- `severity_config`: 严重程度配置

## 检测规则

工具会检测以下情况并建议使用结构化日志：

### 1. 格式化日志方法

检测使用以下方法的代码：
- `Infof`, `Info`
- `Debugf`, `Debug`
- `Errorf`, `Error`
- `Warnf`, `Warn`, `Warningf`, `Warning`
- `Printf`, `Print`, `Println`

### 2. 建议使用结构化日志的场景

- **多参数**: 日志调用包含多个参数
- **格式化占位符**: 消息包含 `%` 格式化占位符
- **长消息**: 消息长度超过配置的阈值
- **敏感信息**: 消息包含敏感关键词

### 3. 严重程度分级

- **高严重程度**: Error级别的日志方法
- **中等严重程度**: Warn级别的日志方法
- **低严重程度**: Info/Debug级别的日志方法

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
     问题: 使用了格式化日志方法 'Errorf'，建议使用结构化日志方法 'Errorw' 以获得更好的日志脱敏和分析能力
     建议: 使用 Errorw 替代 Errorf

🟡 中等严重程度问题 (3个):
  📁 internal/biz/user.go:123:2
     问题: 使用了格式化日志方法 'Warnf'，建议使用结构化日志方法 'Warnw' 以获得更好的日志脱敏和分析能力
     建议: 使用 Warnw 替代 Warnf

=== 修改建议 ===
1. 使用结构化日志方法（如 Infow, Debugw, Errorw, Warnw）替代格式化日志方法
2. 将格式化参数转换为键值对形式，便于日志分析和脱敏
3. 对于包含敏感信息的日志，确保使用结构化日志以启用自动脱敏
4. 参考项目中的 SafeLogger 使用指南: docs/structured-logging-guide.md
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
      "description": "使用了格式化日志方法 'Errorf'，建议使用结构化日志方法 'Errorw' 以获得更好的日志脱敏和分析能力",
      "severity": "high"
    }
  ],
  "summary": {
    "High": 2,
    "Medium": 3,
    "Low": 3
  }
}
```

### HTML输出

HTML格式提供了美观的可视化报告，包含：
- 扫描统计信息
- 问题汇总
- 详细问题列表（按严重程度分组）
- 修改建议

## CI/CD集成

### GitHub Actions

工具已集成到项目的CI流程中（`.github/workflows/ci.yml`）：

```yaml
- name: Check log usage compliance
  run: |
    echo "Running log compliance check..."
    make logcheck
  continue-on-error: true

- name: Generate log compliance report
  run: |
    make logcheck-json > log-compliance-report.json || true
    make logcheck-html > log-compliance-report.html || true
  continue-on-error: true

- name: Upload log compliance reports
  uses: actions/upload-artifact@v4
  if: always()
  with:
    name: log-compliance-reports
    path: |
      log-compliance-report.json
      log-compliance-report.html
    retention-days: 30
```

### 其他CI系统

对于其他CI系统，可以使用以下命令：

```bash
# 构建工具
cd tools/logchecker && go build -o logchecker .

# 运行检查（失败时退出码为1）
./logchecker -dir ../../internal -config logchecker.json

# 生成报告（不影响退出码）
./logchecker -dir ../../internal -config logchecker.json -output json > report.json || true
```

## 最佳实践

### 1. 渐进式迁移

- 首先修复高严重程度的问题
- 逐步处理中等和低严重程度的问题
- 使用白名单临时排除难以修改的代码

### 2. 团队规范

- 将日志检查集成到代码审查流程
- 定期运行检查并生成报告
- 建立日志规范培训和文档

### 3. 配置优化

- 根据项目特点调整敏感关键词列表
- 设置合适的消息长度阈值
- 维护白名单，排除合理的例外情况

### 4. 持续改进

- 定期审查和更新检测规则
- 收集团队反馈，优化工具配置
- 监控日志质量指标的改善情况

## 故障排除

### 常见问题

1. **编译失败**
   ```bash
   # 确保Go版本 >= 1.21
   go version
   
   # 更新依赖
   go mod tidy
   ```

2. **权限问题**
   ```bash
   # 确保有执行权限
   chmod +x logchecker
   ```

3. **配置文件错误**
   ```bash
   # 验证JSON格式
   cat logchecker.json | jq .
   ```

4. **路径问题**
   ```bash
   # 使用绝对路径
   ./logchecker -dir /absolute/path/to/code
   ```

### 调试模式

使用 `-verbose` 参数获取详细的执行信息：

```bash
./logchecker -dir ../../internal -verbose
```

## 扩展开发

工具采用模块化设计，主要组件包括：

- `main.go`: 命令行入口
- `checker.go`: 核心检查器
- `visitor.go`: AST访问器
- `config.go`: 配置管理
- `reporter.go`: 报告生成器

如需扩展功能，可以：

1. 添加新的检测规则到 `visitor.go`
2. 扩展配置选项到 `config.go`
3. 添加新的输出格式到 `reporter.go`
4. 增加测试用例验证功能

## 相关文档

- [结构化日志使用指南](./structured-logging-guide.md)
- [项目架构文档](./architecture.md)
- [开发规范](./development-guidelines.md)