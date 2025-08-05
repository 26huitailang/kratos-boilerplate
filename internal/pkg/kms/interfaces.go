package kms

import (
	"context"
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
	
	// 获取加解密服务
	GetCryptoService() CryptoService
	
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

// RootKeyGenerator 根密钥生成器接口
type RootKeyGenerator interface {
	// 生成根密钥
	GenerateRootKey() ([]byte, error)
	
	// 验证配置
	ValidateConfig() error
}

// DataKeyManager 数据密钥管理器接口
type DataKeyManager interface {
	// 生成新的数据密钥
	GenerateDataKey(ctx context.Context) (*DataKey, error)
	
	// 获取活跃的数据密钥
	GetActiveDataKey(ctx context.Context) (*DataKey, error)
	
	// 根据版本获取数据密钥
	GetDataKeyByVersion(ctx context.Context, version string) (*DataKey, error)
	
	// 轮换数据密钥
	RotateDataKey(ctx context.Context) (*DataKey, error)
}