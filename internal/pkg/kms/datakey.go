package kms

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"time"
	
	"github.com/google/uuid"
	"kratos-boilerplate/internal/biz"
)

// dataKeyManager 数据密钥管理器实现
type dataKeyManager struct {
	rootKey []byte
	storage KeyStorage
	config  *biz.KMSConfig
}

// NewDataKeyManager 创建数据密钥管理器
func NewDataKeyManager(rootKey []byte, storage KeyStorage, config *biz.KMSConfig) DataKeyManager {
	return &dataKeyManager{
		rootKey: rootKey,
		storage: storage,
		config:  config,
	}
}

// GenerateDataKey 生成新的数据密钥
func (m *dataKeyManager) GenerateDataKey(ctx context.Context, algorithm string) (*biz.DataKey, error) {
	// 1. 生成随机密钥
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("%w: %v", biz.ErrKeyGenerationFail, err)
	}
	
	// 2. 使用根密钥加密数据密钥
	encryptedKey, err := m.encryptWithRootKey(key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", biz.ErrKeyEncryptionFail, err)
	}
	
	// 3. 创建数据密钥对象
	dataKey := &biz.DataKey{
		ID:           uuid.New().String(),
		Version:      generateVersion(),
		Algorithm:    algorithm,
		Key:          key,
		EncryptedKey: encryptedKey,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(m.config.RotateInterval),
		IsActive:     true,
	}
	
	// 4. 保存到存储
	if err := m.storage.SaveDataKey(ctx, dataKey); err != nil {
		return nil, fmt.Errorf("%w: failed to save data key: %v", biz.ErrStorageOperation, err)
	}
	
	return dataKey, nil
}

// GetActiveDataKey 获取活跃的数据密钥
func (m *dataKeyManager) GetActiveDataKey(ctx context.Context) (*biz.DataKey, error) {
	dataKey, err := m.storage.GetActiveDataKey(ctx)
	if err != nil {
		return nil, err
	}
	
	// 检查密钥是否过期
	if time.Now().After(dataKey.ExpiresAt) {
		return nil, biz.ErrKeyExpired
	}
	
	// 解密数据密钥
	if err := m.decryptDataKey(dataKey); err != nil {
		return nil, fmt.Errorf("%w: %v", biz.ErrKeyDecryptionFail, err)
	}
	
	return dataKey, nil
}

// GetDataKeyByVersion 根据版本获取数据密钥
func (m *dataKeyManager) GetDataKeyByVersion(ctx context.Context, version string) (*biz.DataKey, error) {
	dataKey, err := m.storage.GetDataKeyByVersion(ctx, version)
	if err != nil {
		return nil, err
	}
	
	// 解密数据密钥
	if err := m.decryptDataKey(dataKey); err != nil {
		return nil, fmt.Errorf("%w: %v", biz.ErrKeyDecryptionFail, err)
	}
	
	return dataKey, nil
}

// RotateDataKey 轮换数据密钥
func (m *dataKeyManager) RotateDataKey(ctx context.Context) (*biz.DataKey, error) {
	// 1. 先获取当前活跃密钥（如果存在）
	var currentKey *biz.DataKey
	if key, err := m.storage.GetActiveDataKey(ctx); err == nil {
		currentKey = key
	}

	// 2. 生成新的数据密钥
	newKey, err := m.GenerateDataKey(ctx, m.config.Algorithm)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new data key: %w", err)
	}

	// 3. 将之前的活跃密钥设为非活跃（如果存在且不是新密钥）
	if currentKey != nil && currentKey.Version != newKey.Version {
		if err := m.storage.UpdateKeyStatus(ctx, currentKey.Version, false); err != nil {
			// 忽略停用旧密钥的错误，因为新密钥已经生成并激活
			// 这不会影响系统的正常运行
			_ = err
		}
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
func (m *dataKeyManager) decryptDataKey(dataKey *biz.DataKey) error {
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
		return biz.ErrInvalidCiphertext
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

// EncryptWithDataKey 使用数据密钥加密
func (m *dataKeyManager) EncryptWithDataKey(ctx context.Context, plaintext []byte, keyVersion string) (*biz.EncryptedField, error) {
	// 获取指定版本的数据密钥
	dataKey, err := m.GetDataKeyByVersion(ctx, keyVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get data key: %w", err)
	}
	
	// 使用数据密钥加密
	block, err := aes.NewCipher(dataKey.Key)
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
	
	return &biz.EncryptedField{
		Value:     ciphertext,
		Version:   keyVersion,
		Algorithm: dataKey.Algorithm,
	}, nil
}

// DecryptWithDataKey 使用数据密钥解密
func (m *dataKeyManager) DecryptWithDataKey(ctx context.Context, encryptedField *biz.EncryptedField) ([]byte, error) {
	// 获取指定版本的数据密钥
	dataKey, err := m.GetDataKeyByVersion(ctx, encryptedField.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get data key: %w", err)
	}
	
	// 使用数据密钥解密
	block, err := aes.NewCipher(dataKey.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(encryptedField.Value) < nonceSize {
		return nil, biz.ErrInvalidCiphertext
	}
	
	nonce := encryptedField.Value[:nonceSize]
	ciphertext := encryptedField.Value[nonceSize:]
	
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return plaintext, nil
}

// generateVersion 生成版本号
func generateVersion() string {
	return fmt.Sprintf("v%d", time.Now().UnixNano())
}