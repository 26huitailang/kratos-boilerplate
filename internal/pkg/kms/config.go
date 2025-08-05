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

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Iterations:     100000,
		KeyLength:      32,
		RotateInterval: 24 * time.Hour,
		Algorithm:      "AES-GCM-256",
		StorageType:    "database",
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Seed == "" {
		return ErrInvalidSeed
	}
	
	if c.Salt == "" {
		return ErrInvalidSalt
	}
	
	if c.Iterations < 10000 {
		return ErrInvalidIterations
	}
	
	if c.KeyLength < 16 {
		return ErrInvalidKeyLength
	}
	
	return nil
}