# OpenAPI 文档使用指南

## 🎉 成功生成OpenAPI文档

恭喜！已经成功生成了 `openapi.yaml` 文件，现在可以用于测试和前端开发。

## 📋 生成的文件信息

- **文件名**: `openapi.yaml`
- **大小**: 28KB+
- **内容**: 包含认证、功能开关、欢迎服务的完整API文档
- **格式**: OpenAPI 3.0.3 规范

## 🚀 如何使用

### 1. Postman 导入

```bash
# 在Postman中：
# 1. 点击 "Import" 按钮
# 2. 选择 "File" 选项卡
# 3. 上传根目录下的 openapi.yaml 文件
# 4. Postman会自动创建完整的API集合
```

### 2. Swagger UI 在线查看

```bash
# 方式1: 使用在线编辑器
# 访问 https://editor.swagger.io/
# 复制 openapi.yaml 内容到编辑器中

# 方式2: 本地Swagger UI
# 启动服务后访问: http://localhost:8000/q/swagger-ui/
```

### 3. 其他工具导入

支持导入OpenAPI规范的工具：
- **Insomnia**: 导入 → OpenAPI → 选择文件
- **VS Code REST Client**: 可配合 OpenAPI 扩展使用
- **YApi**: 支持OpenAPI导入
- **Apifox**: 支持OpenAPI 3.0导入

## 📖 API 文档内容

生成的文档包含以下服务：

### 🔐 认证服务 (Auth)
- `GET /api/v1/auth/captcha` - 获取验证码
- `POST /api/v1/auth/captcha/verify` - 验证验证码
- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/logout` - 退出登录
- `POST /api/v1/auth/refresh` - 刷新令牌
- `GET /api/v1/auth/lock-status/{username}` - 查询锁定状态

### 🎛️ 功能开关服务 (FeatureToggle)
- `GET /api/v1/features` - 获取功能开关列表
- `GET /api/v1/features/{flag}` - 获取单个功能开关
- `PUT /api/v1/features/{flag}` - 更新功能开关
- `POST /api/v1/features/{flag}/enable` - 启用功能开关
- `POST /api/v1/features/{flag}/disable` - 禁用功能开关
- `DELETE /api/v1/features/{flag}` - 删除功能开关
- `POST /api/v1/features/{flag}/evaluate` - 评估功能开关
- `GET /api/v1/features/stats` - 获取统计信息

### 👋 欢迎服务 (Greeter)
- `GET /helloworld/{name}` - 问候服务

## 🔄 重新生成文档

当API有变更时，重新生成文档：

```bash
# 重新生成所有API文件和文档
make api

# 检查生成的文件
ls -la openapi.yaml
```

## 📝 文档特性

✅ **完整的字段说明**: 包含详细的字段描述和示例  
✅ **请求响应示例**: 提供真实可用的示例数据  
✅ **参数验证规则**: 包含字段验证约束  
✅ **错误响应格式**: 统一的错误响应结构  
✅ **认证信息**: JWT Bearer token 认证说明  

## 🛠️ 开发建议

### 前端开发
```typescript
// 可以基于 openapi.yaml 生成 TypeScript 类型
// 使用工具如: openapi-generator, swagger-codegen
npm install @openapitools/openapi-generator-cli
openapi-generator-cli generate -i openapi.yaml -g typescript-axios -o ./src/api
```

### 测试自动化
```bash
# 可以基于 OpenAPI 文档生成自动化测试
# 工具推荐: dredd, schemathesis, postman-cli
```

## 🐛 故障排除

### 生成失败
```bash
# 1. 确保安装了正确的插件
go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest

# 2. 检查 protoc 版本
protoc --version

# 3. 重新初始化环境
make init
```

### 文档内容不完整
```bash
# 检查 proto 文件是否有语法错误
make api 2>&1 | grep -i error

# 确保所有 proto 文件包含了 HTTP 注解
grep -r "google.api.http" api/
```

## 📚 相关链接

- [Kratos OpenAPI 官方文档](https://go-kratos.dev/docs/guide/openapi)
- [OpenAPI 3.0 规范](https://swagger.io/specification/)
- [Swagger Editor](https://editor.swagger.io/)
- [Postman OpenAPI 支持](https://learning.postman.com/docs/integrations/available-integrations/working-with-openAPI/)

## 🎯 下一步

1. **导入Postman**: 将 `openapi.yaml` 导入Postman创建API集合
2. **配置环境变量**: 在Postman中设置开发/测试环境的baseURL
3. **测试API**: 使用生成的请求模板测试各个接口
4. **前端集成**: 基于文档实现前端API调用
5. **持续更新**: API变更时及时更新文档

Happy coding! 🚀