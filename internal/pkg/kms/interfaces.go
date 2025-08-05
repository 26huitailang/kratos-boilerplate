package kms

import (
	"context"
	"kratos-boilerplate/internal/biz"
)

// KMSManager 密钥管理系统接口
type KMSManager interface {
	// 初始化KMS系统
	Initialize(ctx context.Context, config *biz.KMSConfig) error
	
	// 获取当前活跃的数据密钥
	GetActiveDataKey(ctx context.Context) (*biz.DataKey, error)
	
	// 根据版本获取数据密钥
	GetDataKeyByVersion(ctx context.Context, version string) (*biz.DataKey, error)
	
	// 轮换数据密钥
	RotateDataKey(ctx context.Context) (*biz.DataKey, error)
	
	// 获取加解密服务
	GetCryptoService() CryptoService
	
	// 关闭KMS系统
	Close() error
}

// CryptoService 统一加解密服务
type CryptoService interface {
	// 加密敏感字段
	EncryptField(ctx context.Context, fieldName string, value []byte) (*biz.EncryptedField, error)
	
	// 解密敏感字段
	DecryptField(ctx context.Context, encryptedField *biz.EncryptedField) ([]byte, error)
	
	// 计算哈希值（用于检索）
	HashField(value []byte) string
	
	// 批量加密
	EncryptBatch(ctx context.Context, fields map[string][]byte) (map[string]*biz.EncryptedField, error)
	
	// 批量解密
	DecryptBatch(ctx context.Context, fields map[string]*biz.EncryptedField) (map[string][]byte, error)
}

// KeyStorage 密钥存储接口 - 使用Repository模式
type KeyStorage interface {
	// 保存数据密钥
	SaveDataKey(ctx context.Context, key *biz.DataKey) error
	
	// 获取活跃的数据密钥
	GetActiveDataKey(ctx context.Context) (*biz.DataKey, error)
	
	// 根据版本获取数据密钥
	GetDataKeyByVersion(ctx context.Context, version string) (*biz.DataKey, error)
	
	// 列出所有数据密钥
	ListDataKeys(ctx context.Context) ([]*biz.DataKey, error)
	
	// 更新密钥状态
	UpdateKeyStatus(ctx context.Context, version string, isActive bool) error
	
	// 清理过期密钥
	CleanupExpiredKeys(ctx context.Context) error
	
	// 获取密钥统计信息
	GetKeyStatistics(ctx context.Context) (*biz.KeyStatistics, error)
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
	GenerateDataKey(ctx context.Context, algorithm string) (*biz.DataKey, error)
	
	// 获取当前活跃的数据密钥
	GetActiveDataKey(ctx context.Context) (*biz.DataKey, error)
	
	// 根据版本获取数据密钥
	GetDataKeyByVersion(ctx context.Context, version string) (*biz.DataKey, error)
	
	// 轮换数据密钥
	RotateDataKey(ctx context.Context) (*biz.DataKey, error)
	
	// 使用数据密钥加密
	EncryptWithDataKey(ctx context.Context, plaintext []byte, keyVersion string) (*biz.EncryptedField, error)
	
	// 使用数据密钥解密
	DecryptWithDataKey(ctx context.Context, encryptedField *biz.EncryptedField) ([]byte, error)
}