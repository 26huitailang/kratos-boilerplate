# Cursor Commit Message 生成规范

请按照以下规范生成 Git 提交信息：

## 提交信息格式

```
<type>(<scope>): <subject> <issue-reference>

<body>
```

## 类型（Type）选择指南

根据代码变更内容，选择最合适的类型：

- `feat`: 新增功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建/工具相关
- `revert`: 回退变更

## 范围（Scope）选择指南

根据代码变更影响的范围选择：

- `auth`: 认证相关
- `api`: API 接口相关
- `ui`: 界面相关
- `core`: 核心功能
- `db`: 数据库相关
- `config`: 配置相关
- `deps`: 依赖相关
- `ci`: CI/CD 相关

## 主题（Subject）编写指南

1. 使用现在时态
2. 首字母小写
3. 不要以句号结尾
4. 不超过 50 个字符
5. 清晰描述变更内容

## Issue 关联

如果代码变更与 Issue 相关，在第一行末尾添加：

- GitHub: `#123`, `gh-123`, `github-123`
- GitLab: `!123`, `gl-123`, `gitlab-123`

## 正文（Body）编写指南

1. 使用现在时态
2. 每行不超过 72 个字符
3. 列出具体的变更内容
4. 说明变更原因（如果需要）

## 示例

```
feat(auth): add user login #123

- Add login form component
- Implement JWT authentication
- Add login API endpoint
```

```
fix(api): handle null response !456

- Add null check for user response
- Return default user object when null
- Update error handling
```

```
docs(readme): update installation steps gh-789

- Add Docker installation steps
- Update dependency versions
- Fix broken links
```

## 注意事项

1. 每个提交应该只做一件事
2. 提交信息应该清晰明了
3. 避免提交敏感信息
4. 确保代码已经通过测试
5. 提交前进行代码格式化 