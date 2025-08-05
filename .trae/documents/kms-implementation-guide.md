# KMS系统实现指南

## 1. 实现概述

本文档提供了在现有Kratos项目中实现KMS密钥管理系统和日志脱敏功能的详细实现指南，包括代码实现、配置修改和迁移步骤。

## 2. 项目结构调整

### 2.1 新增目录结构

```
internal/pkg/
├── kms/
│   ├── manager.go          # KMS管理器
│   ├── datakey.go          # 数据密钥管理
│   ├── rootkey.go          # 根密钥生成
│   ├── crypto_service.go   # 加解密服务
│   ├── storage.go          # 密钥存储
│   ├── config.go           # KMS配置
│   └── interfaces.go       # 接口定义
├── sensitive/
│   ├── anonymizer.go       # 脱敏处理器
│   ├── interfaces.go       # 脱敏接口
│   ├── rules.go           # 脱敏规则
│   ├── middleware.go      # 日志脱敏中间件
│   └── logger.go          # 安全日志
└── crypto/
    ├── aes.go             # 现有AES实现（保持兼容）
    ├── sm3.go             # 现有SM3实现
    └── adapter.go         # KMS适配器
```

## 3. 核心代码实现

### 3.1 KMS接口定义

**文件：`internal/pkg/kms/interfaces.go`**

```go
package kms

import (
	"context"
	"time"
)

// KMSManager 密钥管理系统接口
type KMSManager interface {
	// 初始化KMS系统
	Initialize(ctx context.Context, config *Config) error
	
	// 获取当前活跃的数据密钥
	GetActiveDataKey(ctx context.Context) (*DataKey, error)
	
	// 根据版本获取数据密钥
	GetDataKeyByVersion(ctx context.Context, version string) (*DataKey, error)
	
	// 轮换数据密钥
	RotateDataKey(ctx context.Context) (*DataKey, error)
	
	// 关闭KMS系统
	Close() error
}

// CryptoService 统一加解密服务
type CryptoService interface {
	// 加密敏感字段
	EncryptField(ctx context.Context, fieldName string, value []byte) (*EncryptedField, error)
	
	// 解密敏感字段
	DecryptField(ctx context.Context, encryptedField *EncryptedField) ([]byte, error)
	
	// 计算哈希值（用于检索）
	HashField(value []byte) string
	
	// 批量加密
	EncryptBatch(ctx context.Context, fields map[string][]byte) (map[string]*EncryptedField, error)
	
	// 批量解密
	DecryptBatch(ctx context.Context, fields map[string]*EncryptedField) (map[string][]byte, error)
}

// KeyStorage 密钥存储接口
type KeyStorage interface {
	// 保存数据密钥
	SaveDataKey(ctx context.Context, key *DataKey) error
	
	// 获取活跃的数据密钥
	GetActiveDataKey(ctx context.Context) (*DataKey, error)
	
	// 根据版本获取数据密钥
	GetDataKeyByVersion(ctx context.Context, version string) (*DataKey, error)
	
	// 列出所有数据密钥
	ListDataKeys(ctx context.Context) ([]*DataKey, error)
	
	// 更新密钥状态
	UpdateKeyStatus(ctx context.Context, version string, isActive bool) error
}
```

### 3.2 数据结构定义

**文件：`internal/pkg/kms/config.go`**

```go
package kms

import (
	"time"
)

// Config KMS配置
type Config struct {
	Seed           string        `yaml:"seed"`            // 种子值
	Salt           string        `yaml:"salt"`            // 盐值
	Iterations     int           `yaml:"iterations"`      // PBKDF2迭代次数
	KeyLength      int           `yaml:"key_length"`      // 密钥长度
	RotateInterval time.Duration `yaml:"rotate_interval"` // 轮换间隔
	Algorithm      string        `yaml:"algorithm"`       // 默认算法
	StorageType    string        `yaml:"storage_type"`    // 存储类型：database/file
}

// DataKey 数据密钥
type DataKey struct {
	ID           string    `json:"id" db:"id"`                     // 密钥ID
	Version      string    `json:"version" db:"version"`           // 版本号
	Algorithm    string    `json:"algorithm" db:"algorithm"`       // 算法类型
	Key          []byte    `json:"-"`                              // 实际密钥（不序列化）
	EncryptedKey []byte    `json:"encrypted_key" db:"encrypted_key"` // 加密后的密钥
	CreatedAt    time.Time `json:"created_at" db:"created_at"`     // 创建时间
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`     // 过期时间
	IsActive     bool      `json:"is_active" db:"is_active"`       // 是否活跃
}

// EncryptedData 加密数据
type EncryptedData struct {
	KeyVersion string `json:"key_version"` // 密钥版本
	Algorithm  string `json:"algorithm"`   // 算法标识
	Nonce      []byte `json:"nonce"`       // 随机数
	Ciphertext []byte `json:"ciphertext"`  // 密文
	AuthTag    []byte `json:"auth_tag"`    // 认证标签（GCM模式）
}

// EncryptedField 加密字段
type EncryptedField struct {
	FieldName     string         `json:"field_name"`
	EncryptedData *EncryptedData `json:"encrypted_data"`
	Hash          string         `json:"hash"` // 用于检索的哈希值
}
```

### 3.3 根密钥生成

**文件：`internal/pkg/kms/rootkey.go`**

```go
package kms

import (
	"crypto/sha256"
	"fmt"
	
	"golang.org/x/crypto/pbkdf2"
)

// RootKeyGenerator 根密钥生成器
type RootKeyGenerator struct {
	config *Config
}

// NewRootKeyGenerator 创建根密钥生成器
func NewRootKeyGenerator(config *Config) *RootKeyGenerator {
	return &RootKeyGenerator{
		config: config,
	}
}

// GenerateRootKey 生成根密钥
func (g *RootKeyGenerator) GenerateRootKey() ([]byte, error) {
	if g.config.Seed == "" {
		return nil, fmt.Errorf("seed cannot be empty")
	}
	
	if g.config.Salt == "" {
		return nil, fmt.Errorf("salt cannot be empty")
	}
	
	if g.config.Iterations <= 0 {
		g.config.Iterations = 100000 // 默认迭代次数
	}
	
	if g.config.KeyLength <= 0 {
		g.config.KeyLength = 32 // 默认256位密钥
	}
	
	// 使用PBKDF2生成根密钥
	rootKey := pbkdf2.Key(
		[]byte(g.config.Seed),
		[]byte(g.config.Salt),
		g.config.Iterations,
		g.config.KeyLength,
		sha256.New,
	)
	
	return rootKey, nil
}

// ValidateConfig 验证配置
func (g *RootKeyGenerator) ValidateConfig() error {
	if g.config.Seed == "" {
		return fmt.Errorf("seed is required")
	}
	
	if g.config.Salt == "" {
		return fmt.Errorf("salt is required")
	}
	
	if g.config.Iterations < 10000 {
		return fmt.Errorf("iterations should be at least 10000")
	}
	
	if g.config.KeyLength < 16 {
		return fmt.Errorf("key length should be at least 16 bytes")
	}
	
	return nil
}
```

### 3.4 数据密钥管理

**文件：`internal/pkg/kms/datakey.go`**

```go
package kms

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"time"
	
	"github.com/google/uuid"
)

// DataKeyManager 数据密钥管理器
type DataKeyManager struct {
	rootKey []byte
	storage KeyStorage
	config  *Config
}

// NewDataKeyManager 创建数据密钥管理器
func NewDataKeyManager(rootKey []byte, storage KeyStorage, config *Config) *DataKeyManager {
	return &DataKeyManager{
		rootKey: rootKey,
		storage: storage,
		config:  config,
	}
}

// GenerateDataKey 生成新的数据密钥
func (m *DataKeyManager) GenerateDataKey(ctx context.Context) (*DataKey, error) {
	// 1. 生成随机密钥
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}
	
	// 2. 使用根密钥加密数据密钥
	encryptedKey, err := m.encryptWithRootKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data key: %w", err)
	}
	
	// 3. 创建数据密钥对象
	dataKey := &DataKey{
		ID:           uuid.New().String(),
		Version:      generateVersion(),
		Algorithm:    m.config.Algorithm,
		Key:          key,
		EncryptedKey: encryptedKey,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(m.config.RotateInterval),
		IsActive:     true,
	}
	
	// 4. 保存到存储
	if err := m.storage.SaveDataKey(ctx, dataKey); err != nil {
		return nil, fmt.Errorf("failed to save data key: %w", err)
	}
	
	return dataKey, nil
}

// GetActiveDataKey 获取活跃的数据密钥
func (m *DataKeyManager) GetActiveDataKey(ctx context.Context) (*DataKey, error) {
	dataKey, err := m.storage.GetActiveDataKey(ctx)
	if err != nil {
		return nil, err
	}
	
	// 解密数据密钥
	if err := m.decryptDataKey(dataKey); err != nil {
		return nil, fmt.Errorf("failed to decrypt data key: %w", err)
	}
	
	return dataKey, nil
}

// GetDataKeyByVersion 根据版本获取数据密钥
func (m *DataKeyManager) GetDataKeyByVersion(ctx context.Context, version string) (*DataKey, error) {
	dataKey, err := m.storage.GetDataKeyByVersion(ctx, version)
	if err != nil {
		return nil, err
	}
	
	// 解密数据密钥
	if err := m.decryptDataKey(dataKey); err != nil {
		return nil, fmt.Errorf("failed to decrypt data key: %w", err)
	}
	
	return dataKey, nil
}

// RotateDataKey 轮换数据密钥
func (m *DataKeyManager) RotateDataKey(ctx context.Context) (*DataKey, error) {
	// 1. 将当前活跃密钥设为非活跃
	currentKey, err := m.storage.GetActiveDataKey(ctx)
	if err == nil && currentKey != nil {
		if err := m.storage.UpdateKeyStatus(ctx, currentKey.Version, false); err != nil {
			return nil, fmt.Errorf("failed to deactivate current key: %w", err)
		}
	}
	
	// 2. 生成新的数据密钥
	newKey, err := m.GenerateDataKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new data key: %w", err)
	}
	
	return newKey, nil
}

// encryptWithRootKey 使用根密钥加密
func (m *DataKeyManager) encryptWithRootKey(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.rootKey)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptDataKey 解密数据密钥
func (m *DataKeyManager) decryptDataKey(dataKey *DataKey) error {
	if len(dataKey.Key) > 0 {
		return nil // 已经解密
	}
	
	block, err := aes.NewCipher(m.rootKey)
	if err != nil {
		return err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	
	nonceSize := gcm.NonceSize()
	if len(dataKey.EncryptedKey) < nonceSize {
		return fmt.Errorf("encrypted key too short")
	}
	
	nonce := dataKey.EncryptedKey[:nonceSize]
	ciphertext := dataKey.EncryptedKey[nonceSize:]
	
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}
	
	dataKey.Key = plaintext
	return nil
}

// generateVersion 生成版本号
func generateVersion() string {
	return fmt.Sprintf("v%d", time.Now().Unix())
}
```

### 3.5 密钥存储实现

**文件：`internal/pkg/kms/storage.go`**

```go
package kms

import (
	"context"
	"database/sql"
	"fmt"
	
	"github.com/go-kratos/kratos/v2/log"
)

// DatabaseKeyStorage 数据库密钥存储
type DatabaseKeyStorage struct {
	db  *sql.DB
	log *log.Helper
}

// NewDatabaseKeyStorage 创建数据库密钥存储
func NewDatabaseKeyStorage(db *sql.DB, logger log.Logger) KeyStorage {
	return &DatabaseKeyStorage{
		db:  db,
		log: log.NewHelper(logger),
	}
}

// SaveDataKey 保存数据密钥
func (s *DatabaseKeyStorage) SaveDataKey(ctx context.Context, key *DataKey) error {
	query := `
		INSERT INTO kms_data_keys (id, version, algorithm, encrypted_key, created_at, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		key.ID, key.Version, key.Algorithm, key.EncryptedKey,
		key.CreatedAt, key.ExpiresAt, key.IsActive)
	
	if err != nil {
		s.log.Errorf("Failed to save data key: %v", err)
		return fmt.Errorf("failed to save data key: %w", err)
	}
	
	s.log.Infof("Data key saved successfully: %s", key.Version)
	return nil
}

// GetActiveDataKey 获取活跃的数据密钥
func (s *DatabaseKeyStorage) GetActiveDataKey(ctx context.Context) (*DataKey, error) {
	query := `
		SELECT id, version, algorithm, encrypted_key, created_at, expires_at, is_active
		FROM kms_data_keys
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`
	
	var key DataKey
	err := s.db.QueryRowContext(ctx, query).Scan(
		&key.ID, &key.Version, &key.Algorithm, &key.EncryptedKey,
		&key.CreatedAt, &key.ExpiresAt, &key.IsActive)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active data key found")
		}
		return nil, fmt.Errorf("failed to get active data key: %w", err)
	}
	
	return &key, nil
}

// GetDataKeyByVersion 根据版本获取数据密钥
func (s *DatabaseKeyStorage) GetDataKeyByVersion(ctx context.Context, version string) (*DataKey, error) {
	query := `
		SELECT id, version, algorithm, encrypted_key, created_at, expires_at, is_active
		FROM kms_data_keys
		WHERE version = $1
	`
	
	var key DataKey
	err := s.db.QueryRowContext(ctx, query, version).Scan(
		&key.ID, &key.Version, &key.Algorithm, &key.EncryptedKey,
		&key.CreatedAt, &key.ExpiresAt, &key.IsActive)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("data key not found: %s", version)
		}
		return nil, fmt.Errorf("failed to get data key: %w", err)
	}
	
	return &key, nil
}

// ListDataKeys 列出所有数据密钥
func (s *DatabaseKeyStorage) ListDataKeys(ctx context.Context) ([]*DataKey, error) {
	query := `
		SELECT id, version, algorithm, encrypted_key, created_at, expires_at, is_active
		FROM kms_data_keys
		ORDER BY created_at DESC
	`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list data keys: %w", err)
	}
	defer rows.Close()
	
	var keys []*DataKey
	for rows.Next() {
		var key DataKey
		err := rows.Scan(
			&key.ID, &key.Version, &key.Algorithm, &key.EncryptedKey,
			&key.CreatedAt, &key.ExpiresAt, &key.IsActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan data key: %w", err)
		}
		keys = append(keys, &key)
	}
	
	return keys, nil
}

// UpdateKeyStatus 更新密钥状态
func (s *DatabaseKeyStorage) UpdateKeyStatus(ctx context.Context, version string, isActive bool) error {
	query := `UPDATE kms_data_keys SET is_active = $1 WHERE version = $2`
	
	_, err := s.db.ExecContext(ctx, query, isActive, version)
	if err != nil {
		s.log.Errorf("Failed to update key status: %v", err)
		return fmt.Errorf("failed to update key status: %w", err)
	}
	
	s.log.Infof("Key status updated: %s -> %t", version, isActive)
	return nil
}
```

### 3.6 敏感信息脱敏接口

**文件：`internal/pkg/sensitive/interfaces.go`**

```go
package sensitive

// MakeSensitive 敏感信息脱敏接口
type MakeSensitive interface {
	// 获取敏感字段列表
	GetSensitiveFields() []string
	
	// 脱敏处理
	Anonymize() interface{}
	
	// 获取脱敏规则
	GetAnonymizeRules() map[string]AnonymizeRule
}

// LogSafeStringer 日志安全字符串接口
type LogSafeStringer interface {
	// 返回日志安全的字符串表示
	LogSafeString() string
}

// AnonymizeRule 脱敏规则
type AnonymizeRule struct {
	FieldName  string                `json:"field_name"`  // 字段名
	KeepStart  int                   `json:"keep_start"`  // 保留开头字符数
	KeepEnd    int                   `json:"keep_end"`    // 保留结尾字符数
	MaskChar   string                `json:"mask_char"`   // 掩码字符，默认为*
	CustomFunc func(string) string   `json:"-"`           // 自定义脱敏函数
}

// Anonymizer 脱敏处理器接口
type Anonymizer interface {
	// 脱敏字符串
	AnonymizeString(value string, rule AnonymizeRule) string
	
	// 脱敏对象
	AnonymizeObject(obj MakeSensitive) interface{}
	
	// 批量脱敏
	AnonymizeBatch(objects []MakeSensitive) []interface{}
}
```

### 3.7 脱敏处理器实现

**文件：`internal/pkg/sensitive/anonymizer.go`**

```go
package sensitive

import (
	"reflect"
	"strings"
)

// DefaultAnonymizer 默认脱敏处理器
type DefaultAnonymizer struct {
	defaultMaskChar string
}

// NewDefaultAnonymizer 创建默认脱敏处理器
func NewDefaultAnonymizer() Anonymizer {
	return &DefaultAnonymizer{
		defaultMaskChar: "*",
	}
}

// AnonymizeString 脱敏字符串
func (a *DefaultAnonymizer) AnonymizeString(value string, rule AnonymizeRule) string {
	if rule.CustomFunc != nil {
		return rule.CustomFunc(value)
	}
	
	runes := []rune(value)
	length := len(runes)
	
	// 如果字符串长度小于等于保留字符数，直接返回
	if length <= rule.KeepStart+rule.KeepEnd {
		return value
	}
	
	// 确定掩码字符
	maskChar := rule.MaskChar
	if maskChar == "" {
		maskChar = a.defaultMaskChar
	}
	
	// 构建脱敏字符串
	start := string(runes[:rule.KeepStart])
	end := string(runes[length-rule.KeepEnd:])
	maskLength := length - rule.KeepStart - rule.KeepEnd
	mask := strings.Repeat(maskChar, maskLength)
	
	return start + mask + end
}

// AnonymizeObject 脱敏对象
func (a *DefaultAnonymizer) AnonymizeObject(obj MakeSensitive) interface{} {
	return obj.Anonymize()
}

// AnonymizeBatch 批量脱敏
func (a *DefaultAnonymizer) AnonymizeBatch(objects []MakeSensitive) []interface{} {
	result := make([]interface{}, len(objects))
	for i, obj := range objects {
		result[i] = a.AnonymizeObject(obj)
	}
	return result
}

// AnonymizeValue 通用值脱敏
func (a *DefaultAnonymizer) AnonymizeValue(value interface{}, rules map[string]AnonymizeRule) interface{} {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		return value
	}
	
	// 创建新的结构体副本
	newValue := reflect.New(v.Type()).Elem()
	newValue.Set(v)
	
	// 遍历字段进行脱敏
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		fieldValue := newValue.Field(i)
		
		if !fieldValue.CanSet() {
			continue
		}
		
		// 检查是否需要脱敏
		if rule, exists := rules[strings.ToLower(field.Name)]; exists {
			if fieldValue.Kind() == reflect.String {
				originalValue := fieldValue.String()
				anonymizedValue := a.AnonymizeString(originalValue, rule)
				fieldValue.SetString(anonymizedValue)
			}
		}
	}
	
	return newValue.Interface()
}
```

### 3.8 日志脱敏中间件

**文件：`internal/pkg/sensitive/middleware.go`**

```go
package sensitive

import (
	"context"
	"fmt"
	
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
)

// LogSanitizeMiddleware 日志脱敏中间件
func LogSanitizeMiddleware(logger log.Logger, anonymizer Anonymizer) middleware.Middleware {
	logHelper := log.NewHelper(logger)
	
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			// 请求日志脱敏
			if sensitive, ok := req.(MakeSensitive); ok {
				anonymizedReq := anonymizer.AnonymizeObject(sensitive)
				logHelper.Infof("Request: %+v", anonymizedReq)
			} else if logSafe, ok := req.(LogSafeStringer); ok {
				logHelper.Infof("Request: %s", logSafe.LogSafeString())
			} else {
				logHelper.Infof("Request: %+v", req)
			}
			
			// 执行处理器
			reply, err = handler(ctx, req)
			
			// 响应日志脱敏
			if err != nil {
				logHelper.Errorf("Error: %v", err)
			} else {
				if sensitive, ok := reply.(MakeSensitive); ok {
					anonymizedReply := anonymizer.AnonymizeObject(sensitive)
					logHelper.Infof("Response: %+v", anonymizedReply)
				} else if logSafe, ok := reply.(LogSafeStringer); ok {
					logHelper.Infof("Response: %s", logSafe.LogSafeString())
				} else {
					logHelper.Infof("Response: %+v", reply)
				}
			}
			
			return reply, err
		}
	}
}

// OperationLogSanitizer 操作日志脱敏器
type OperationLogSanitizer struct {
	anonymizer Anonymizer
}

// NewOperationLogSanitizer 创建操作日志脱敏器
func NewOperationLogSanitizer(anonymizer Anonymizer) *OperationLogSanitizer {
	return &OperationLogSanitizer{
		anonymizer: anonymizer,
	}
}

// SanitizeLogContent 脱敏日志内容
func (s *OperationLogSanitizer) SanitizeLogContent(content interface{}) string {
	if sensitive, ok := content.(MakeSensitive); ok {
		anonymized := s.anonymizer.AnonymizeObject(sensitive)
		return fmt.Sprintf("%+v", anonymized)
	}
	
	if logSafe, ok := content.(LogSafeStringer); ok {
		return logSafe.LogSafeString()
	}
	
	return fmt.Sprintf("%+v", content)
}
```

## 4. 业务层集成

### 4.1 用户结构体改造

**文件：`internal/biz/user.go`（修改现有文件）**

```go
// 在现有User结构体中添加脱敏接口实现

// GetSensitiveFields 获取敏感字段列表
func (u *User) GetSensitiveFields() []string {
	return []string{"email", "phone", "name"}
}

// GetAnonymizeRules 获取脱敏规则
func (u *User) GetAnonymizeRules() map[string]sensitive.AnonymizeRule {
	return map[string]sensitive.AnonymizeRule{
		"email": {FieldName: "email", KeepStart: 2, KeepEnd: 4, MaskChar: "*"},
		"phone": {FieldName: "phone", KeepStart: 3, KeepEnd: 4, MaskChar: "*"},
		"name":  {FieldName: "name", KeepStart: 1, KeepEnd: 1, MaskChar: "*"},
	}
}

// Anonymize 脱敏处理
func (u *User) Anonymize() interface{} {
	rules := u.GetAnonymizeRules()
	anonymizer := sensitive.NewDefaultAnonymizer()
	
	anonymized := &User{
		ID:       u.ID,
		Username: u.Username,
		Email:    anonymizer.AnonymizeString(u.Email, rules["email"]),
		Phone:    anonymizer.AnonymizeString(u.Phone, rules["phone"]),
		Name:     anonymizer.AnonymizeString(u.Name, rules["name"]),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	return anonymized
}

// LogSafeString 返回日志安全的字符串表示
func (u *User) LogSafeString() string {
	anonymized := u.Anonymize().(*User)
	return fmt.Sprintf("User{ID:%d, Username:%s, Email:%s, Phone:%s, Name:%s}",
		anonymized.ID, anonymized.Username, anonymized.Email, anonymized.Phone, anonymized.Name)
}
```

### 4.2 数据层改造

**文件：`internal/data/user.go`（修改现有文件）**

```go
// 修改userRepo结构体
type userRepo struct {
	data   *Data
	log    *log.Helper
	crypto kms.CryptoService  // 替换原来的enc字段
}

// 修改NewUserRepo函数
func NewUserRepo(data *Data, logger log.Logger, cryptoService kms.CryptoService) (biz.UserRepo, error) {
	return &userRepo{
		data:   data,
		log:    log.NewHelper(logger),
		crypto: cryptoService,
	}, nil
}

// 修改CreateUser方法
func (r *userRepo) CreateUser(ctx context.Context, u *biz.User) error {
	// 使用新的加密服务
	fields := map[string][]byte{
		"email": []byte(u.Email),
		"phone": []byte(u.Phone),
		"name":  []byte(u.Name),
	}
	
	encryptedFields, err := r.crypto.EncryptBatch(ctx, fields)
	if err != nil {
		return err
	}
	
	// 构建数据库记录
	user := &User{
		Username:      u.Username,
		PasswordHash:  u.PasswordHash,
		EmailEncrypted: encryptedFields["email"].EncryptedData,
		EmailHash:     encryptedFields["email"].Hash,
		PhoneEncrypted: encryptedFields["phone"].EncryptedData,
		PhoneHash:     encryptedFields["phone"].Hash,
		NameEncrypted:  encryptedFields["name"].EncryptedData,
		NameHash:      encryptedFields["name"].Hash,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	// 插入数据库
	result := r.data.db.Create(user)
	if result.Error != nil {
		return result.Error
	}
	
	u.ID = user.ID
	return nil
}

// 类似地修改其他方法...
```

## 5. 配置文件修改

### 5.1 主配置文件

**文件：`configs/config.yaml`**

```yaml
# 现有配置保持不变...

# 新增KMS配置
kms:
  seed: "${KMS_SEED}"  # 从环境变量获取
  salt: "${KMS_SALT}"  # 从环境变量获取
  iterations: 100000
  key_length: 32
  rotate_interval: "24h"
  algorithm: "AES-GCM-256"
  storage_type: "database"

# 新增敏感信息配置
sensitive:
  log_level: "info"
  default_mask_char: "*"
  enable_anonymize: true
  enable_log_sanitize: true
```

### 5.2 环境变量配置

**文件：`.env`（新建）**

```bash
# KMS密钥配置
KMS_SEED=your-super-secret-seed-value-here
KMS_SALT=your-unique-salt-value-here

# 数据库配置（如果需要）
DB_HOST=localhost
DB_PORT=5432
DB_NAME=kratos_boilerplate
DB_USER=postgres
DB_PASSWORD=password

# Redis配置（如果需要）
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
```

## 6. 数据库迁移

### 6.1 KMS密钥表

**文件：`migrations/000002_add_kms_tables.up.sql`**

```sql
-- KMS数据密钥表
CREATE TABLE IF NOT EXISTS kms_data_keys (
    id VARCHAR(36) PRIMARY KEY,
    version VARCHAR(50) UNIQUE NOT NULL,
    algorithm VARCHAR(50) NOT NULL,
    encrypted_key BYTEA NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT false
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_kms_data_keys_version ON kms_data_keys(version);
CREATE INDEX IF NOT EXISTS idx_kms_data_keys_active ON kms_data_keys(is_active);
CREATE INDEX IF NOT EXISTS idx_kms_data_keys_created_at ON kms_data_keys(created_at DESC);

-- 确保只有一个活跃密钥
CREATE UNIQUE INDEX IF NOT EXISTS idx_kms_data_keys_unique_active 
ON kms_data_keys(is_active) WHERE is_active = true;
```

**文件：`migrations/000002_add_kms_tables.down.sql`**

```sql
DROP TABLE IF EXISTS kms_data_keys;
```

## 7. 依赖注入配置

### 7.1 Wire配置

**文件：`internal/data/data.go`（修改）**

```go
// 添加KMS相关的Provider
var ProviderSet = wire.NewSet(
	NewData, 
	NewGreeterRepo,
	NewUserRepo,
	NewOperationLogRepo,
	// 新增KMS相关Provider
	NewKMSManager,
	NewCryptoService,
	NewKeyStorage,
	NewAnonymizer,
)

// NewKMSManager 创建KMS管理器
func NewKMSManager(conf *conf.Data, logger log.Logger) (kms.KMSManager, error) {
	// 从配置创建KMS配置
	kmsConfig := &kms.Config{
		Seed:           conf.Kms.Seed,
		Salt:           conf.Kms.Salt,
		Iterations:     int(conf.Kms.Iterations),
		KeyLength:      int(conf.Kms.KeyLength),
		RotateInterval: conf.Kms.RotateInterval.AsDuration(),
		Algorithm:      conf.Kms.Algorithm,
		StorageType:    conf.Kms.StorageType,
	}
	
	// 创建根密钥生成器
	rootKeyGen := kms.NewRootKeyGenerator(kmsConfig)
	if err := rootKeyGen.ValidateConfig(); err != nil {
		return nil, err
	}
	
	rootKey, err := rootKeyGen.GenerateRootKey()
	if err != nil {
		return nil, err
	}
	
	// 创建密钥存储
	keyStorage := NewKeyStorage(conf, logger)
	
	// 创建数据密钥管理器
	dataKeyManager := kms.NewDataKeyManager(rootKey, keyStorage, kmsConfig)
	
	// 创建KMS管理器
	kmsManager := kms.NewManager(dataKeyManager, kmsConfig, logger)
	
	return kmsManager, nil
}

// NewCryptoService 创建加密服务
func NewCryptoService(kmsManager kms.KMSManager, logger log.Logger) kms.CryptoService {
	return kms.NewCryptoService(kmsManager, logger)
}

// NewKeyStorage 创建密钥存储
func NewKeyStorage(conf *conf.Data, logger log.Logger) kms.KeyStorage {
	// 这里使用现有的数据库连接
	db := // 获取数据库连接
	return kms.NewDatabaseKeyStorage(db, logger)
}

// NewAnonymizer 创建脱敏处理器
func NewAnonymizer() sensitive.Anonymizer {
	return sensitive.NewDefaultAnonymizer()
}
```

## 8. 中间件注册

### 8.1 HTTP服务器配置

**文件：`internal/server/http.go`（修改）**

```go
func NewHTTPServer(c *conf.Server, 
	greeter *service.GreeterService,
	auth *service.AuthService,
	anonymizer sensitive.Anonymizer,
	logger log.Logger) *http.Server {
	
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			logging.Server(logger),
			// 添加日志脱敏中间件
			sensitive.LogSanitizeMiddleware(logger, anonymizer),
		),
	}
	
	// 其余配置保持不变...
}
```

### 8.2 gRPC服务器配置

**文件：`internal/server/grpc.go`（修改）**

```go
func NewGRPCServer(c *conf.Server, 
	greeter *service.GreeterService,
	auth *service.AuthService,
	anonymizer sensitive.Anonymizer,
	logger log.Logger) *grpc.Server {
	
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			tracing.Server(),
			logging.Server(logger),
			// 添加日志脱敏中间件
			sensitive.LogSanitizeMiddleware(logger, anonymizer),
		),
	}
	
	// 其余配置保持不变...
}
```

## 9. 迁移步骤

### 9.1 阶段1：准备工作

1. **创建新的包结构**

   ```bash
   mkdir -p internal/pkg/kms
   mkdir -p internal/pkg/sensitive
   ```

2. **添加依赖**

   ```bash
   go get golang.org/x/crypto/pbkdf2
   go get github.com/google/uuid
   ```

3. **运行数据库迁移**

   ```bash
   migrate -path migrations -database "postgres://user:pass@localhost/db?sslmode=disable" up
   ```

### 9.2 阶段2：部署KMS系统

1. **配置环境变量**

   ```bash
   export KMS_SEED="your-super-secret-seed-value"
   export KMS_SALT="your-unique-salt-value"
   ```

2. **更新配置文件**

   * 添加KMS和敏感信息配置到`config.yaml`

3. **部署新版本**

   * KMS系统与现有加密系统并行运行

### 9.3 阶段3：渐进式迁移

1. **新数据使用KMS**

   * 新用户注册使用KMS加密

   * 现有用户更新时迁移到KMS

2. **历史数据迁移**

   ```go
   // 数据迁移脚本示例
   func migrateUserData(ctx context.Context, oldRepo, newRepo UserRepo) error {
       users, err := oldRepo.ListAllUsers(ctx)
       if err != nil {
           return err
       }
       
       for _, user := range users {
           // 解密旧数据
           decryptedUser, err := oldRepo.DecryptUser(ctx, user)
           if err != nil {
               continue
           }
           
           // 使用新系统加密
           err = newRepo.UpdateUserWithKMS(ctx, decryptedUser)
           if err != nil {
               log.Errorf("Failed to migrate user %d: %v", user.ID, err)
           }
       }
       
       return nil
   }
   ```

### 9.4 阶段4：清理工作

1. **移除旧的加密代码**

   * 删除硬编码密钥

   * 移除旧的加密逻辑

2. **验证迁移完成**

   * 确保所有数据都使用KMS加密

   * 验证日志脱敏功能正常

## 10. 测试验证

### 10.1 单元测试

**文件：`internal/pkg/kms/manager_test.go`**

```go
package kms

import (
	"context"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKMSManager(t *testing.T) {
	config := &Config{
		Seed:           "test-seed",
		Salt:           "test-salt",
		Iterations:     10000,
		KeyLength:      32,
		RotateInterval: time.Hour,
		Algorithm:      "AES-GCM-256",
	}
	
	// 测试根密钥生成
	rootKeyGen := NewRootKeyGenerator(config)
	rootKey, err := rootKeyGen.GenerateRootKey()
	require.NoError(t, err)
	assert.Len(t, rootKey, 32)
	
	// 测试数据密钥生成
	mockStorage := &MockKeyStorage{}
	dataKeyManager := NewDataKeyManager(rootKey, mockStorage, config)
	
	ctx := context.Background()
	dataKey, err := dataKeyManager.GenerateDataKey(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, dataKey.ID)
	assert.NotEmpty(t, dataKey.Version)
	assert.Len(t, dataKey.Key, 32)
}
```

### 10.2 集成测试

**文件：`internal/pkg/sensitive/anonymizer_test.go`**

```go
package sensitive

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
)

func TestAnonymizer(t *testing.T) {
	anonymizer := NewDefaultAnonymizer()
	
	tests := []struct {
		name     string
		value    string
		rule     AnonymizeRule
		expected string
	}{
		{
			name:  "email",
			value: "user@example.com",
			rule:  AnonymizeRule{KeepStart: 2, KeepEnd: 4, MaskChar: "*"},
			expected: "us***********.com",
		},
		{
			name:  "phone",
			value: "13800138000",
			rule:  AnonymizeRule{KeepStart: 3, KeepEnd: 4, MaskChar: "*"},
			expected: "138****8000",
		},
		{
			name:  "chinese_name",
			value: "张三李四王五",
			rule:  AnonymizeRule{KeepStart: 1, KeepEnd: 1, MaskChar: "*"},
			expected: "张****五",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := anonymizer.AnonymizeString(tt.value, tt.rule)
			assert.Equal(t, tt.expected, result)
		})
	}
}
```

## 11. 监控和运维

### 11.1 健康检查

**文件：`internal/service/health.go`（新建）**

```go
package service

import (
	"context"
	
	"github.com/go-kratos/kratos/v2/log"
	v1 "your-project/api/health/v1"
	"your-project/internal/pkg/kms"
)

type HealthService struct {
	v1.UnimplementedHealthServer
	
	kms kms.KMSManager
	log *log.Helper
}

func NewHealthService(kmsManager kms.KMSManager, logger log.Logger) *HealthService {
	return &HealthService{
		kms: kmsManager,
		log: log.NewHelper(logger),
	}
}

func (s *HealthService) Check(ctx context.Context, req *v1.HealthCheckRequest) (*v1.HealthCheckResponse, error) {
	// 检查KMS系统状态
	_, err := s.kms.GetActiveDataKey(ctx)
	if err != nil {
		s.log.Errorf("KMS health check failed: %v", err)
		return &v1.HealthCheckResponse{
			Status: v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}
	
	return &v1.HealthCheckResponse{
		Status: v1.HealthCheckResponse_SERVING,
	}, nil
}
```

### 11.2 监控指标

**文件：`internal/pkg/kms/metrics.go`（新建）**

```go
package kms

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// 加密操作计数器
	encryptionCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kms_encryption_operations_total",
			Help: "Total number of encryption operations",
		},
		[]string{"operation", "status"},
	)
	
	// 密钥轮换计数器
	keyRotationCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kms_key_rotation_total",
			Help: "Total number of key rotations",
		},
		[]string{"status"},
	)
	
	// 加密操作延迟
	encryptionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "kms_encryption_duration_seconds",
			Help: "Duration of encryption operations",
		},
		[]string{"operation"},
	)
)

// RecordEncryption 记录加密操作
func RecordEncryption(operation, status string) {
	encryptionCounter.WithLabelValues(operation, status).Inc()
}

// RecordKeyRotation 记录密钥轮换
func RecordKeyRotation(status string) {
	keyRotationCounter.WithLabelValues(status).Inc()
}

// RecordEncryptionDuration 记录加密操作延迟
func RecordEncryptionDuration(operation string, duration float64) {
	encryptionDuration.WithLabelValues(operation).Observe(duration)
}
```

## 12. 安全最佳实践

### 12.1 密钥安全

1. **环境变量管理**
   - 使用专门的密钥管理服务（如AWS KMS、Azure Key Vault）
   - 避免在代码中硬编码任何密钥信息
   - 定期轮换seed和salt值

2. **内存安全**
   ```go
   // 安全清理内存中的密钥
   func secureZero(b []byte) {
       for i := range b {
           b[i] = 0
       }
   }
   
   // 在密钥使用完毕后清理
   defer secureZero(rootKey)
   ```

3. **访问控制**
   - 限制KMS接口的访问权限
   - 实现审计日志记录
   - 使用RBAC控制密钥操作权限

### 12.2 数据保护

1. **传输安全**
   - 所有API调用使用HTTPS/TLS
   - 内部服务间通信加密

2. **存储安全**
   - 数据库连接加密
   - 定期备份加密数据
   - 实现数据完整性校验

## 13. 故障处理

### 13.1 常见问题

1. **密钥轮换失败**
   ```go
   // 实现回滚机制
   func (m *KMSManager) rollbackKeyRotation(ctx context.Context, oldVersion string) error {
       return m.storage.UpdateKeyStatus(ctx, oldVersion, true)
   }
   ```

2. **解密失败**
   ```go
   // 尝试使用历史密钥解密
   func (s *CryptoService) decryptWithFallback(ctx context.Context, data *EncryptedData) ([]byte, error) {
       // 首先尝试当前密钥
       result, err := s.decrypt(ctx, data)
       if err == nil {
           return result, nil
       }
       
       // 尝试历史密钥
       if data.KeyVersion != "" {
           oldKey, err := s.kms.GetDataKeyByVersion(ctx, data.KeyVersion)
           if err == nil {
               return s.decryptWithKey(data, oldKey)
           }
       }
       
       return nil, fmt.Errorf("decryption failed with all available keys")
   }
   ```

### 13.2 监控告警

```yaml
# prometheus告警规则
groups:
- name: kms_alerts
  rules:
  - alert: KMSEncryptionFailureRate
    expr: rate(kms_encryption_operations_total{status="error"}[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "KMS encryption failure rate is high"
      
  - alert: KMSKeyRotationFailed
    expr: kms_key_rotation_total{status="error"} > 0
    for: 0m
    labels:
      severity: critical
    annotations:
      summary: "KMS key rotation failed"
```

## 14. 性能优化建议

### 14.1 缓存策略

```go
// 密钥缓存实现
type KeyCache struct {
	cache map[string]*DataKey
	mutex sync.RWMutex
	ttl   time.Duration
}

func (c *KeyCache) Get(version string) (*DataKey, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	key, exists := c.cache[version]
	return key, exists
}

func (c *KeyCache) Set(version string, key *DataKey) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.cache[version] = key
	// 设置过期清理逻辑
}
```

### 14.2 批量操作优化

```go
// 批量加密优化
func (s *CryptoService) EncryptBatchOptimized(ctx context.Context, fields map[string][]byte) (map[string]*EncryptedField, error) {
	// 获取一次密钥，用于所有加密操作
	dataKey, err := s.kms.GetActiveDataKey(ctx)
	if err != nil {
		return nil, err
	}
	
	results := make(map[string]*EncryptedField)
	
	// 并发加密
	var wg sync.WaitGroup
	var mutex sync.Mutex
	errorChan := make(chan error, len(fields))
	
	for fieldName, value := range fields {
		wg.Add(1)
		go func(name string, val []byte) {
			defer wg.Done()
			
			encrypted, err := s.encryptWithKey(val, dataKey)
			if err != nil {
				errorChan <- err
				return
			}
			
			mutex.Lock()
			results[name] = encrypted
			mutex.Unlock()
		}(fieldName, value)
	}
	
	wg.Wait()
	close(errorChan)
	
	// 检查错误
	if len(errorChan) > 0 {
		return nil, <-errorChan
	}
	
	return results, nil
}
```

## 15. 总结

本实现指南提供了完整的KMS密钥管理系统和日志脱敏功能的实现方案，包括：

✅ **分层密钥架构**：Root Key + Data Key设计确保密钥安全
✅ **统一加解密接口**：简化业务层调用，提高代码复用性
✅ **自动密钥轮换**：定期轮换密钥，提高长期安全性
✅ **敏感信息脱敏**：保护日志中的隐私数据，符合合规要求
✅ **渐进式迁移**：平滑替换现有硬编码密钥系统
✅ **完整的监控**：提供性能指标和告警机制
✅ **故障恢复**：支持密钥回滚和多版本兼容

### 实施建议

1. **分阶段实施**：按照文档中的4个阶段逐步实施，确保系统稳定性
2. **充分测试**：在生产环境部署前进行充分的单元测试和集成测试
3. **监控部署**：确保监控和告警系统正常工作
4. **备份策略**：制定完整的数据备份和恢复策略
5. **安全审计**：定期进行安全审计和渗透测试

通过实施本方案，系统将获得企业级的密钥管理能力和数据保护水平，为业务的长期发展提供坚实的安全基础。
```

