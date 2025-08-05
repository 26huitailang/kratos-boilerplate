package biz

import (
	"context"
	"errors"
	"time"
)

// DataKey 数据密钥结构体
type DataKey struct {
	ID           string    `json:"id"`
	Version      string    `json:"version"`
	Algorithm    string    `json:"algorithm"`
	Key          []byte    `json:"-"` // 明文密钥，不序列化
	EncryptedKey []byte    `json:"encrypted_key"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	IsActive     bool      `json:"is_active"`
}

// KMSRepo KMS数据仓库接口
type KMSRepo interface {
	// 保存数据密钥
	SaveDataKey(ctx context.Context, key *DataKey) error
	
	// 获取活跃的数据密钥
	GetActiveDataKey(ctx context.Context) (*DataKey, error)
	
	// 根据版本获取数据密钥
	GetDataKeyByVersion(ctx context.Context, version string) (*DataKey, error)
	
	// 列出所有数据密钥
	ListDataKeys(ctx context.Context) ([]*DataKey, error)
	
	// 根据ID获取数据密钥
	GetDataKey(ctx context.Context, keyID string) (*DataKey, error)
	
	// 更新数据密钥
	UpdateDataKey(ctx context.Context, dataKey *DataKey) error
	
	// 更新密钥状态
	UpdateKeyStatus(ctx context.Context, version string, isActive bool) error
	
	// 清理过期密钥
	CleanupExpiredKeys(ctx context.Context) error
	
	// 获取密钥统计信息
	GetKeyStatistics(ctx context.Context) (*KeyStatistics, error)
}

// KMSUseCase KMS业务逻辑接口
type KMSUseCase interface {
	// 初始化KMS系统
	Initialize(ctx context.Context, config *KMSConfig) error
	
	// 获取当前活跃的数据密钥
	GetActiveDataKey(ctx context.Context) (*DataKey, error)
	
	// 根据版本获取数据密钥
	GetDataKeyByVersion(ctx context.Context, version string) (*DataKey, error)
	
	// 轮换数据密钥
	RotateDataKey(ctx context.Context) (*DataKey, error)
	
	// 加密敏感字段
	EncryptField(ctx context.Context, fieldName string, value []byte) (*EncryptedField, error)
	
	// 解密敏感字段
	DecryptField(ctx context.Context, encryptedField *EncryptedField) ([]byte, error)
	
	// 计算哈希值
	HashField(value []byte) string
	
	// 关闭KMS系统
	Close() error
}

// KMSConfig KMS配置
type KMSConfig struct {
	Seed           string        `yaml:"seed"`            // 种子值
	Salt           string        `yaml:"salt"`            // 盐值
	Iterations     int           `yaml:"iterations"`      // PBKDF2迭代次数
	KeyLength      int           `yaml:"key_length"`      // 密钥长度
	RotateInterval time.Duration `yaml:"rotate_interval"` // 轮换间隔
	Algorithm      string        `yaml:"algorithm"`       // 默认算法
	StorageType    string        `yaml:"storage_type"`    // 存储类型：database/file
}

// EncryptedField 加密字段结构
type EncryptedField struct {
	Value     []byte `json:"value"`     // 加密后的值
	Version   string `json:"version"`   // 密钥版本
	Algorithm string `json:"algorithm"` // 加密算法
}

// KeyStatistics 密钥统计信息
type KeyStatistics struct {
	TotalKeys    int64 `json:"total_keys"`
	ActiveKeys   int64 `json:"active_keys"`
	ExpiredKeys  int64 `json:"expired_keys"`
}

// Validate 验证KMS配置
func (c *KMSConfig) Validate() error {
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

// KMS错误定义
var (
	ErrKeyNotFound        = errors.New("数据密钥未找到")
	ErrNoActiveKey        = errors.New("没有活跃的数据密钥")
	ErrKeyExpired         = errors.New("数据密钥已过期")
	ErrInvalidKeyVersion  = errors.New("无效的密钥版本")
	ErrKeyEncryptionFail  = errors.New("密钥加密失败")
	ErrKeyDecryptionFail  = errors.New("密钥解密失败")
	ErrInvalidCiphertext  = errors.New("无效的密文")
	ErrKMSAlreadyInit     = errors.New("KMS系统已初始化")
	ErrKMSNotInitialized  = errors.New("KMS系统未初始化")
	ErrSystemShutdown     = errors.New("系统已关闭")
	ErrInvalidSeed        = errors.New("无效的种子值")
	ErrInvalidSalt        = errors.New("无效的盐值")
	ErrInvalidIterations  = errors.New("无效的迭代次数")
	ErrInvalidKeyLength   = errors.New("无效的密钥长度")
	ErrInvalidAlgorithm   = errors.New("无效的算法")
	ErrKeyGenerationFail  = errors.New("密钥生成失败")
	ErrStorageOperation   = errors.New("存储操作失败")
)