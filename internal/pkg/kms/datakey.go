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

// dataKeyManager 数据密钥管理器实现
type dataKeyManager struct {
	rootKey []byte
	storage KeyStorage
	config  *Config
}

// NewDataKeyManager 创建数据密钥管理器
func NewDataKeyManager(rootKey []byte, storage KeyStorage, config *Config) DataKeyManager {
	return &dataKeyManager{
		rootKey: rootKey,
		storage: storage,
		config:  config,
	}
}

// GenerateDataKey 生成新的数据密钥
func (m *dataKeyManager) GenerateDataKey(ctx context.Context) (*DataKey, error) {
	// 1. 生成随机密钥
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFail, err)
	}
	
	// 2. 使用根密钥加密数据密钥
	encryptedKey, err := m.encryptWithRootKey(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyEncryptionFail, err)
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
		return nil, fmt.Errorf("%w: failed to save data key: %v", ErrStorageOperation, err)
	}
	
	return dataKey, nil
}

// GetActiveDataKey 获取活跃的数据密钥
func (m *dataKeyManager) GetActiveDataKey(ctx context.Context) (*DataKey, error) {
	dataKey, err := m.storage.GetActiveDataKey(ctx)
	if err != nil {
		return nil, err
	}
	
	// 检查密钥是否过期
	if time.Now().After(dataKey.ExpiresAt) {
		return nil, ErrKeyExpired
	}
	
	// 解密数据密钥
	if err := m.decryptDataKey(dataKey); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyDecryptionFail, err)
	}
	
	return dataKey, nil
}

// GetDataKeyByVersion 根据版本获取数据密钥
func (m *dataKeyManager) GetDataKeyByVersion(ctx context.Context, version string) (*DataKey, error) {
	dataKey, err := m.storage.GetDataKeyByVersion(ctx, version)
	if err != nil {
		return nil, err
	}
	
	// 解密数据密钥
	if err := m.decryptDataKey(dataKey); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyDecryptionFail, err)
	}
	
	return dataKey, nil
}

// RotateDataKey 轮换数据密钥
func (m *dataKeyManager) RotateDataKey(ctx context.Context) (*DataKey, error) {
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
func (m *dataKeyManager) encryptWithRootKey(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.rootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decryptDataKey 解密数据密钥
func (m *dataKeyManager) decryptDataKey(dataKey *DataKey) error {
	if len(dataKey.Key) > 0 {
		return nil // 已经解密
	}
	
	block, err := aes.NewCipher(m.rootKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(dataKey.EncryptedKey) < nonceSize {
		return ErrInvalidCiphertext
	}
	
	nonce := dataKey.EncryptedKey[:nonceSize]
	ciphertext := dataKey.EncryptedKey[nonceSize:]
	
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}
	
	dataKey.Key = plaintext
	return nil
}

// generateVersion 生成版本号
func generateVersion() string {
	return fmt.Sprintf("v%d", time.Now().Unix())
}

// IsExpired 检查密钥是否过期
func (dk *DataKey) IsExpired() bool {
	return time.Now().After(dk.ExpiresAt)
}

// TimeToExpiry 获取距离过期的时间
func (dk *DataKey) TimeToExpiry() time.Duration {
	if dk.IsExpired() {
		return 0
	}
	return time.Until(dk.ExpiresAt)
}

// Clear 清除内存中的密钥数据
func (dk *DataKey) Clear() {
	if len(dk.Key) > 0 {
		// 用零值覆盖密钥数据
		for i := range dk.Key {
			dk.Key[i] = 0
		}
		dk.Key = nil
	}
}