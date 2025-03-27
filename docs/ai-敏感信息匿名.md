# 敏感信息加密存储需求文档

## 1. 需求背景

在用户认证系统中，需要安全地存储用户的敏感信息（如邮箱、手机号、姓名等），同时又要支持这些信息的检索功能。

## 2. 功能需求

### 2.1 数据加密存储

- 使用加密算法对敏感信息进行加密，要兼容国密和国际算法，不能使用低安全算法
  - 算法支持aes-gcm，sm3
- 加密字段包括：
  - 邮箱（email）
  - 手机号（phone）
  - 姓名（name）
- 每个敏感信息同时存储加密值和哈希值
- 使用string 类型存储加密数据，有算法标记位，便于使用不同算法解密
- 使用 VARCHAR(64) 存储哈希值

### 2.2 数据检索功能

- 支持通过以下方式检索用户：
  - 用户名（明文存储）
  - 邮箱（通过哈希值匹配）
  - 手机号（通过哈希值匹配）
  - 姓名（通过哈希值匹配）
- 使用哈希索引优化检索性能

### 2.3 数据匿名化

- 提供敏感信息的匿名化处理功能
- 支持自定义保留前N位和后N位字符
- 中间部分使用星号(*)替代

## 3. 技术实现

### 3.1 数据库设计

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    email_encrypted VARCHAR(255),
    email_hash VARCHAR(64),
    phone_encrypted VARCHAR(255),
    phone_hash VARCHAR(64),
    name_encrypted VARCHAR(255),
    name_hash VARCHAR(64),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 索引
CREATE INDEX idx_users_email_hash ON users(email_hash);
CREATE INDEX idx_users_phone_hash ON users(phone_hash);
CREATE INDEX idx_users_name_hash ON users(name_hash);
```

### 3.3 用户仓储层

- 实现用户信息的加密存储
- 实现基于哈希的用户信息检索
- 提供用户信息的更新功能

## 4. 安全考虑

- 使用随机 nonce 确保加密安全性
- 使用哈希值进行检索，避免明文存储
- 密钥管理（实际应用中应从配置或环境变量获取）

## 5. 性能考虑

- 使用哈希索引优化检索性能
- 加密/解密操作在应用层进行，不影响数据库性能
- 哈希值用于快速匹配，避免解密操作
