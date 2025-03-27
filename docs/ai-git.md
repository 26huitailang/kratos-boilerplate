# Git 提交规范

## 提交信息格式

每个提交信息都应该包含以下三个部分：

```
<type>(<scope>): <subject>

<body>

<footer>
```

### 类型（Type）

提交类型必须是以下之一：

- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码格式（不影响代码运行的变动）
- `refactor`: 重构（既不是新增功能，也不是修改bug的代码变动）
- `perf`: 性能优化
- `test`: 增加测试
- `chore`: 构建过程或辅助工具的变动
- `revert`: 回退

### 范围（Scope）

范围是可选的，用于说明提交影响的范围。例如：
- `feat(auth)`: 认证相关
- `fix(api)`: API相关
- `style(ui)`: UI相关

### 主题（Subject）

- 使用现在时态（"change" 而不是 "changed" 或 "changes"）
- 首字母不要大写
- 结尾不加句号
- 不超过50个字符

### 正文（Body）

- 使用现在时态
- 说明代码变动的动机，以及与以前行为的对比
- 每行不超过72个字符

### 页脚（Footer）

- 关闭 Issue：`Closes #123, #456`
- 重大变更：`BREAKING CHANGE: 描述变更内容`

## 示例

```
feat(auth): add user login functionality

- Add login form component
- Implement JWT authentication
- Add login API endpoint

Closes #123
```

```
fix(api): handle null response in user service

- Add null check for user response
- Return default user object when null
- Update error handling

Fixes #456
```

```
docs(readme): update installation instructions

- Add Docker installation steps
- Update dependency versions
- Fix broken links
```

## 注意事项

1. 提交信息应该清晰明了，便于其他开发者理解
2. 每个提交应该只做一件事
3. 提交前请确保代码已经通过测试
4. 提交前请进行代码格式化
5. 避免提交敏感信息（如密码、密钥等）