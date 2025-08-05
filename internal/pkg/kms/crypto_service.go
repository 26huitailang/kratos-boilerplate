package kms

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
	
	"github.com/go-kratos/kratos/v2/log"
	"kratos-boilerplate/internal/biz"
)

// cryptoService 统一加解密服务实现
type cryptoService struct {
	dataKeyManager DataKeyManager
	log            *log.Helper
	cache          *keyCache
}

// keyCache 密钥缓存
type keyCache struct {
	mu         sync.RWMutex
	activeKey  *biz.DataKey
	versionMap map[string]*biz.DataKey
}

// NewCryptoService 创建加解密服务
func NewCryptoService(dataKeyManager DataKeyManager, logger log.Logger) CryptoService {
	return &cryptoService{
		dataKeyManager: dataKeyManager,
		log:            log.NewHelper(logger),
		cache: &keyCache{
			versionMap: make(map[string]*biz.DataKey),
		},
	}
}

// EncryptField 加密敏感字段
func (s *cryptoService) EncryptField(ctx context.Context, fieldName string, value []byte) (*biz.EncryptedField, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf("value cannot be empty")
	}
	
	// 获取活跃的数据密钥
	dataKey, err := s.getActiveDataKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active data key: %w", err)
	}
	
	// 执行加密
	encryptedValue, err := s.encryptWithDataKey(dataKey, value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", biz.ErrKeyEncryptionFail, err)
	}
	
	encryptedField := &biz.EncryptedField{
		Value:     encryptedValue,
		Version:   dataKey.Version,
		Algorithm: dataKey.Algorithm,
	}
	
	s.log.Debugf("Field encrypted successfully: %s", fieldName)
	return encryptedField, nil
}

// DecryptField 解密敏感字段
func (s *cryptoService) DecryptField(ctx context.Context, encryptedField *biz.EncryptedField) ([]byte, error) {
	if encryptedField == nil {
		return nil, fmt.Errorf("encrypted field cannot be nil")
	}
	
	// 根据版本获取数据密钥
	dataKey, err := s.getDataKeyByVersion(ctx, encryptedField.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get data key: %w", err)
	}
	
	// 执行解密
	plaintext, err := s.decryptWithDataKey(dataKey, encryptedField.Value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", biz.ErrKeyDecryptionFail, err)
	}
	
	s.log.Debugf("Field decrypted successfully")
	return plaintext, nil
}

// HashField 计算哈希值（用于检索）
func (s *cryptoService) HashField(value []byte) string {
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}

// EncryptBatch 批量加密
func (s *cryptoService) EncryptBatch(ctx context.Context, fields map[string][]byte) (map[string]*biz.EncryptedField, error) {
	if len(fields) == 0 {
		return make(map[string]*biz.EncryptedField), nil
	}
	
	// 获取活跃的数据密钥（一次获取，批量使用）
	dataKey, err := s.getActiveDataKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active data key: %w", err)
	}
	
	result := make(map[string]*biz.EncryptedField, len(fields))
	var encryptionErrors []string
	
	for fieldName, value := range fields {
		if len(value) == 0 {
			encryptionErrors = append(encryptionErrors, fmt.Sprintf("field %s: value cannot be empty", fieldName))
			continue
		}
		
		// 执行加密
		encryptedValue, err := s.encryptWithDataKey(dataKey, value)
		if err != nil {
			encryptionErrors = append(encryptionErrors, fmt.Sprintf("field %s: %v", fieldName, err))
			continue
		}
		
		result[fieldName] = &biz.EncryptedField{
			Value:     encryptedValue,
			Version:   dataKey.Version,
			Algorithm: dataKey.Algorithm,
		}
	}
	
	if len(encryptionErrors) > 0 {
		s.log.Errorf("Batch encryption errors: %v", encryptionErrors)
		return result, fmt.Errorf("batch encryption failed with %d errors", len(encryptionErrors))
	}
	
	s.log.Infof("Batch encryption completed: %d fields", len(result))
	return result, nil
}

// DecryptBatch 批量解密
func (s *cryptoService) DecryptBatch(ctx context.Context, fields map[string]*biz.EncryptedField) (map[string][]byte, error) {
	if len(fields) == 0 {
		return make(map[string][]byte), nil
	}
	
	result := make(map[string][]byte, len(fields))
	var decryptionErrors []string
	
	for fieldName, encryptedField := range fields {
		if encryptedField == nil {
			decryptionErrors = append(decryptionErrors, fmt.Sprintf("field %s: encrypted field cannot be nil", fieldName))
			continue
		}
		
		// 根据版本获取数据密钥
		dataKey, err := s.getDataKeyByVersion(ctx, encryptedField.Version)
		if err != nil {
			decryptionErrors = append(decryptionErrors, fmt.Sprintf("field %s: failed to get data key: %v", fieldName, err))
			continue
		}
		
		// 执行解密
		plaintext, err := s.decryptWithDataKey(dataKey, encryptedField.Value)
		if err != nil {
			decryptionErrors = append(decryptionErrors, fmt.Sprintf("field %s: %v", fieldName, err))
			continue
		}
		
		result[fieldName] = plaintext
	}
	
	if len(decryptionErrors) > 0 {
		s.log.Errorf("Batch decryption errors: %v", decryptionErrors)
		return result, fmt.Errorf("batch decryption failed with %d errors", len(decryptionErrors))
	}
	
	s.log.Infof("Batch decryption completed: %d fields", len(result))
	return result, nil
}

// getActiveDataKey 获取活跃的数据密钥（带缓存）
func (s *cryptoService) getActiveDataKey(ctx context.Context) (*biz.DataKey, error) {
	s.cache.mu.RLock()
	if s.cache.activeKey != nil && !s.isKeyExpired(s.cache.activeKey) {
		key := s.cache.activeKey
		s.cache.mu.RUnlock()
		return key, nil
	}
	s.cache.mu.RUnlock()
	
	// 缓存未命中或已过期，重新获取
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	
	// 双重检查
	if s.cache.activeKey != nil && !s.isKeyExpired(s.cache.activeKey) {
		return s.cache.activeKey, nil
	}
	
	dataKey, err := s.dataKeyManager.GetActiveDataKey(ctx)
	if err != nil {
		return nil, err
	}
	
	// 更新缓存
	s.cache.activeKey = dataKey
	s.cache.versionMap[dataKey.Version] = dataKey
	
	return dataKey, nil
}

// getDataKeyByVersion 根据版本获取数据密钥（带缓存）
func (s *cryptoService) getDataKeyByVersion(ctx context.Context, version string) (*biz.DataKey, error) {
	s.cache.mu.RLock()
	if key, exists := s.cache.versionMap[version]; exists {
		s.cache.mu.RUnlock()
		return key, nil
	}
	s.cache.mu.RUnlock()
	
	// 缓存未命中，从存储获取
	dataKey, err := s.dataKeyManager.GetDataKeyByVersion(ctx, version)
	if err != nil {
		return nil, err
	}
	
	// 更新缓存
	s.cache.mu.Lock()
	s.cache.versionMap[version] = dataKey
	s.cache.mu.Unlock()
	
	return dataKey, nil
}

// encryptWithDataKey 使用数据密钥加密
func (s *cryptoService) encryptWithDataKey(dataKey *biz.DataKey, plaintext []byte) ([]byte, error) {
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
	return ciphertext, nil
}

// decryptWithDataKey 使用数据密钥解密
func (s *cryptoService) decryptWithDataKey(dataKey *biz.DataKey, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(dataKey.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, biz.ErrInvalidCiphertext
	}
	
	nonce := ciphertext[:nonceSize]
	ciphertextData := ciphertext[nonceSize:]
	
	plaintext, err := gcm.Open(nil, nonce, ciphertextData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return plaintext, nil
}

// isKeyExpired 检查密钥是否过期
func (s *cryptoService) isKeyExpired(dataKey *biz.DataKey) bool {
	return time.Now().After(dataKey.ExpiresAt)
}

// ClearCache 清除密钥缓存
func (s *cryptoService) ClearCache() {
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	
	// 清除敏感数据
	if s.cache.activeKey != nil {
		s.clearDataKey(s.cache.activeKey)
	}
	
	for _, key := range s.cache.versionMap {
		s.clearDataKey(key)
	}
	
	s.cache.activeKey = nil
	s.cache.versionMap = make(map[string]*biz.DataKey)
	
	s.log.Info("Crypto service cache cleared")
}

// clearDataKey 清除数据密钥中的敏感信息
func (s *cryptoService) clearDataKey(dataKey *biz.DataKey) {
	if len(dataKey.Key) > 0 {
		// 用零值覆盖密钥数据
		for i := range dataKey.Key {
			dataKey.Key[i] = 0
		}
		dataKey.Key = nil
	}
}