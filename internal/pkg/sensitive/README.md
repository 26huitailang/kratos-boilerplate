# 统一结构化日志使用指南

本包提供了统一的结构化日志功能，支持自动脱敏、敏感信息检测和与Kratos log.Helper的完全兼容。

## 功能特性

- ✅ 支持Infow、Debugw、Errorw、Warnw等结构化日志方法
- ✅ 集成敏感信息自动检测和脱敏功能
- ✅ 支持自定义脱敏规则和字段映射
- ✅ 保持与Kratos log.Helper的完全兼容性
- ✅ 支持上下文和字段链式调用
- ✅ 提供多种环境预设配置

## 快速开始

### 基本使用

```go
package main

import (
    "github.com/go-kratos/kratos/v2/log"
    "your-project/internal/pkg/sensitive"
)

func main() {
    // 创建标准输出日志器
    logger := log.With(log.NewStdLogger(os.Stdout),
        "ts", log.DefaultTimestamp,
        "caller", log.DefaultCaller,
    )
    helper := log.NewHelper(logger)
    
    // 创建结构化日志器
    slogger := sensitive.NewStructuredLoggerQuick(helper)
    
    // 使用结构化日志
    slogger.Infow("用户登录", 
        "user_id", "12345",
        "email", "user@example.com",  // 自动脱敏
        "ip", "192.168.1.1",
    )
}
```

### 替代现有日志方法

#### 从格式化日志迁移到结构化日志

**旧方式（格式化日志）：**
```go
// 不推荐的方式
helper.Infof("用户 %s 登录成功，IP: %s", userID, clientIP)
helper.Errorf("用户 %s 登录失败: %v", userID, err)
```

**新方式（结构化日志）：**
```go
// 推荐的方式
slogger.Infow("用户登录成功",
    "user_id", userID,
    "client_ip", clientIP,
)
slogger.Errorw("用户登录失败",
    "user_id", userID,
    "error", err,
)
```

#### 兼容性方法

如果需要渐进式迁移，结构化日志器仍然支持原有的格式化方法：

```go
// 兼容原有方法
slogger.Info("系统启动")
slogger.Infof("服务监听端口: %d", port)
slogger.Error("数据库连接失败")
slogger.Errorf("连接超时: %v", err)
```

## 配置选项

### 环境预设配置

```go
// 生产环境配置
slogger := sensitive.NewStructuredLoggerForProduction(helper)

// 开发环境配置
slogger := sensitive.NewStructuredLoggerForDevelopment(helper)

// 测试环境配置
slogger := sensitive.NewStructuredLoggerForTesting(helper)
```

### 自定义配置

```go
config := &sensitive.StructuredLogConfig{
    Enabled:           true,
    AutoDetect:        true,
    CustomRules:       customRules,
    SensitiveKeys:     []string{"password", "token", "secret"},
    MaxValueLength:    1000,
    TruncateThreshold: 500,
}

slogger := sensitive.NewStructuredLoggerWithCustomRules(helper, config)
```

## 敏感信息脱敏

### 自动检测脱敏

系统会自动检测并脱敏以下类型的敏感信息：

- 邮箱地址
- 手机号码
- 身份证号
- 银行卡号
- IP地址
- URL中的敏感参数

```go
slogger.Infow("用户信息",
    "data", "邮箱: user@example.com, 手机: 13812345678",
)
// 输出: 邮箱: u***@***.com, 手机: 138****5678
```

### 按字段名脱敏

预定义的敏感字段会自动脱敏：

```go
slogger.Infow("用户认证",
    "password", "mypassword123",    // 自动脱敏
    "token", "abc123def456",       // 自动脱敏
    "user_id", "12345",           // 不脱敏
)
```

### 自定义脱敏规则

```go
customRules := map[string]sensitive.AnonymizeRule{
    "credit_card": {
        FieldName: "credit_card",
        KeepStart: 4,
        KeepEnd:   4,
        MaskChar:  "*",
    },
}

slogger.UpdateRules(customRules)
slogger.Infow("支付信息", "credit_card", "1234567890123456")
// 输出: 1234********3456
```

## 上下文和字段链式调用

### 添加上下文

```go
ctx := context.WithValue(context.Background(), "trace_id", "abc123")
contextLogger := slogger.WithContext(ctx)
contextLogger.Infow("处理请求", "action", "create_user")
```

### 添加公共字段

```go
serviceLogger := slogger.WithFields(
    "service", "user-service",
    "version", "1.0.0",
)

serviceLogger.Infow("服务启动", "port", 8080)
serviceLogger.Infow("处理请求", "endpoint", "/api/users")
```

## 性能优化

### 禁用脱敏功能

在性能敏感的场景下，可以禁用脱敏功能：

```go
slogger := sensitive.NewStructuredLoggerDisabled(helper)
// 或者
config := sensitive.DefaultStructuredLogConfig()
config.Enabled = false
slogger := sensitive.NewStructuredLogger(helper, config)
```

### 值长度限制

```go
config := sensitive.DefaultStructuredLogConfig()
config.MaxValueLength = 500      // 最大值长度
config.TruncateThreshold = 200   // 截断阈值
```

## 最佳实践

### 1. 使用结构化字段而非格式化字符串

```go
// ❌ 不推荐
slogger.Infof("用户 %s 执行了 %s 操作", userID, action)

// ✅ 推荐
slogger.Infow("用户操作",
    "user_id", userID,
    "action", action,
)
```

### 2. 使用有意义的字段名

```go
// ❌ 不推荐
slogger.Infow("请求处理", "data1", userID, "data2", duration)

// ✅ 推荐
slogger.Infow("请求处理完成",
    "user_id", userID,
    "duration_ms", duration,
    "status", "success",
)
```

### 3. 合理使用日志级别

```go
// Debug: 详细的调试信息
slogger.Debugw("数据库查询", "sql", query, "duration_ms", duration)

// Info: 一般信息
slogger.Infow("用户登录", "user_id", userID)

// Warn: 警告信息
slogger.Warnw("API调用缓慢", "endpoint", "/api/users", "duration_ms", 5000)

// Error: 错误信息
slogger.Errorw("数据库连接失败", "error", err, "retry_count", retryCount)
```

### 4. 添加敏感字段

```go
// 动态添加敏感字段
slogger.AddSensitiveKey("api_key")
slogger.AddSensitiveKey("session_id")
```

## 迁移指南

### 从log.Helper迁移

1. **替换日志器创建**：
   ```go
   // 旧方式
   helper := log.NewHelper(logger)
   
   // 新方式
   slogger := sensitive.NewStructuredLoggerQuick(
       log.NewHelper(logger),
   )
   ```

2. **替换日志调用**：
   ```go
   // 旧方式
   helper.Infof("用户 %s 登录", userID)
   
   // 新方式
   slogger.Infow("用户登录", "user_id", userID)
   ```

3. **保持兼容性**：
   ```go
   // 渐进式迁移，先保持原有调用方式
   slogger.Infof("用户 %s 登录", userID)  // 仍然支持
   ```

### 批量替换建议

使用正则表达式进行批量替换：

```bash
# 替换 Infof 调用
find . -name "*.go" -exec sed -i 's/\.Infof(\(.*\))/\.Infow(\1)/g' {} \;

# 替换 Errorf 调用
find . -name "*.go" -exec sed -i 's/\.Errorf(\(.*\))/\.Errorw(\1)/g' {} \;
```

## 故障排除

### 常见问题

1. **脱敏不生效**：
   - 检查配置中的 `Enabled` 是否为 `true`
   - 确认字段名是否在敏感字段列表中
   - 验证自定义规则是否正确配置

2. **性能问题**：
   - 考虑禁用自动检测：`config.AutoDetect = false`
   - 设置合理的值长度限制
   - 在高频场景下禁用脱敏功能

3. **字段不显示**：
   - 确保使用 `NewStructuredLoggerFromLogger` 而非 `NewStructuredLogger`
   - 检查 `WithFields` 的使用方式

### 调试技巧

```go
// 获取底层helper进行调试
helper := slogger.GetHelper()
helper.Info("直接使用helper")

// 检查配置
config := slogger.GetConfig()  // 如果需要的话可以添加此方法
```

## 总结

统一结构化日志提供了以下优势：

1. **结构化数据**：便于日志分析和监控
2. **自动脱敏**：保护敏感信息安全
3. **完全兼容**：无缝替换现有日志方法
4. **灵活配置**：支持多种环境和自定义需求
5. **高性能**：可根据需要调整性能参数

建议在新项目中直接使用结构化日志，在现有项目中进行渐进式迁移。