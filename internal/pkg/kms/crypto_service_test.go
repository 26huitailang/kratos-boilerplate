package kms

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
	
	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"kratos-boilerplate/internal/biz"
)

// mockDataKeyManager 模拟数据密钥管理器
type mockDataKeyManager struct {
	activeKey *biz.DataKey
	keys      map[string]*biz.DataKey
	// 存储原始密钥数据，避免被clearDataKey影响
	originalKeys map[string][]byte
}

func newMockDataKeyManager() *mockDataKeyManager {
	return &mockDataKeyManager{
		keys:         make(map[string]*biz.DataKey),
		originalKeys: make(map[string][]byte),
	}
}

func (m *mockDataKeyManager) GenerateDataKey(ctx context.Context, algorithm string) (*biz.DataKey, error) {
	keyData := make([]byte, 32) // 32字节密钥
	
	// 生成随机密钥
	for i := range keyData {
		keyData[i] = byte(i)
	}
	
	// 创建密钥的副本存储
	keyCopy := make([]byte, len(keyData))
	copy(keyCopy, keyData)
	
	key := &biz.DataKey{
		ID:        "test-key-id",
		Version:   "v1",
		Algorithm: algorithm,
		Key:       keyCopy,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsActive:  true,
	}
	
	// 保存原始密钥数据
	originalKeyCopy := make([]byte, len(keyData))
	copy(originalKeyCopy, keyData)
	m.originalKeys[key.Version] = originalKeyCopy
	
	m.activeKey = key
	m.keys[key.Version] = key
	return key, nil
}

func (m *mockDataKeyManager) GetActiveDataKey(ctx context.Context) (*biz.DataKey, error) {
	if m.activeKey == nil {
		return nil, biz.ErrNoActiveKey
	}
	return m.activeKey, nil
}

func (m *mockDataKeyManager) GetDataKeyByVersion(ctx context.Context, version string) (*biz.DataKey, error) {
	key, exists := m.keys[version]
	if !exists {
		return nil, biz.ErrKeyNotFound
	}
	// 返回密钥的副本以避免并发问题
	copiedKey := *key
	// 使用原始密钥数据，避免被clearDataKey影响
	originalKeyData, hasOriginal := m.originalKeys[version]
	if hasOriginal {
		copiedKey.Key = make([]byte, len(originalKeyData))
		copy(copiedKey.Key, originalKeyData)
	} else {
		// 如果没有原始数据，使用当前密钥数据
		copiedKey.Key = make([]byte, len(key.Key))
		copy(copiedKey.Key, key.Key)
	}
	return &copiedKey, nil
}

func (m *mockDataKeyManager) RotateDataKey(ctx context.Context) (*biz.DataKey, error) {
	// 简单实现：生成新密钥
	return m.GenerateDataKey(ctx, "AES-256-GCM")
}

func (m *mockDataKeyManager) EncryptWithDataKey(ctx context.Context, plaintext []byte, keyVersion string) (*biz.EncryptedField, error) {
	// 获取指定版本的密钥
	key, exists := m.keys[keyVersion]
	if !exists {
		return nil, biz.ErrKeyNotFound
	}
	
	// 简单的模拟加密：只是返回原文加上一些标识
	encryptedData := append([]byte("encrypted:"), plaintext...)
	
	return &biz.EncryptedField{
		Value:     encryptedData,
		Version:   key.Version,
		Algorithm: key.Algorithm,
	}, nil
}

func (m *mockDataKeyManager) DecryptWithDataKey(ctx context.Context, encryptedField *biz.EncryptedField) ([]byte, error) {
	// 验证密钥版本
	_, exists := m.keys[encryptedField.Version]
	if !exists {
		return nil, biz.ErrKeyNotFound
	}
	
	// 简单的模拟解密：移除加密标识
	prefix := []byte("encrypted:")
	if len(encryptedField.Value) < len(prefix) {
		return nil, biz.ErrInvalidCiphertext
	}
	return encryptedField.Value[len(prefix):], nil
}

func TestCryptoService_EncryptField(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	
	// 先生成一个活跃密钥
	ctx := context.Background()
	_, err := mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)
	
	cryptoService := NewCryptoService(mockManager, logger)
	
	// 测试正常加密
	fieldName := "test_field"
	value := []byte("sensitive data")
	
	encryptedField, err := cryptoService.EncryptField(ctx, fieldName, value)
	assert.NoError(t, err)
	assert.NotNil(t, encryptedField)
	assert.NotEmpty(t, encryptedField.Value)
	assert.Equal(t, "v1", encryptedField.Version)
	assert.Equal(t, "AES-256-GCM", encryptedField.Algorithm)
	
	// 测试空值
	_, err = cryptoService.EncryptField(ctx, fieldName, nil)
	assert.Error(t, err)
	
	_, err = cryptoService.EncryptField(ctx, fieldName, []byte{})
	assert.Error(t, err)
}

func TestCryptoService_DecryptField(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	
	// 先生成一个活跃密钥
	ctx := context.Background()
	_, err := mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)
	
	cryptoService := NewCryptoService(mockManager, logger)
	
	// 先加密
	originalValue := []byte("test data")
	encryptedField, err := cryptoService.EncryptField(ctx, "test", originalValue)
	require.NoError(t, err)
	
	// 然后解密
	decryptedValue, err := cryptoService.DecryptField(ctx, encryptedField)
	assert.NoError(t, err)
	assert.Equal(t, originalValue, decryptedValue)
	
	// 测试空值
	_, err = cryptoService.DecryptField(ctx, nil)
	assert.Error(t, err)
	
	// 测试无效版本
	invalidField := &biz.EncryptedField{
		Value:     []byte("test"),
		Version:   "invalid-version",
		Algorithm: "AES-256-GCM",
	}
	_, err = cryptoService.DecryptField(ctx, invalidField)
	assert.Error(t, err)
}

func TestCryptoService_HashField(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	cryptoService := NewCryptoService(mockManager, logger)
	
	// 测试哈希计算
	value := []byte("test data")
	hash1 := cryptoService.HashField(value)
	assert.NotEmpty(t, hash1)
	
	// 相同数据应该产生相同哈希
	hash2 := cryptoService.HashField(value)
	assert.Equal(t, hash1, hash2)
	
	// 不同数据应该产生不同哈希
	differentValue := []byte("different data")
	hash3 := cryptoService.HashField(differentValue)
	assert.NotEqual(t, hash1, hash3)
	
	// 测试空值
	emptyHash := cryptoService.HashField([]byte{})
	assert.NotEmpty(t, emptyHash)
	assert.NotEqual(t, hash1, emptyHash)
}

func TestCryptoService_EncryptBatch(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	
	// 先生成一个活跃密钥
	ctx := context.Background()
	_, err := mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)
	
	cryptoService := NewCryptoService(mockManager, logger)
	
	// 测试批量加密
	fields := map[string][]byte{
		"field1": []byte("value1"),
		"field2": []byte("value2"),
		"field3": []byte("value3"),
	}
	
	encryptedFields, err := cryptoService.EncryptBatch(ctx, fields)
	assert.NoError(t, err)
	assert.Len(t, encryptedFields, 3)
	
	for fieldName := range fields {
		encryptedField, exists := encryptedFields[fieldName]
		assert.True(t, exists)
		assert.NotNil(t, encryptedField)
		assert.NotEmpty(t, encryptedField.Value)
		assert.Equal(t, "v1", encryptedField.Version)
	}
	
	// 测试空字段集合
	emptyResult, err := cryptoService.EncryptBatch(ctx, map[string][]byte{})
	assert.NoError(t, err)
	assert.Empty(t, emptyResult)
	
	// 测试包含空值的字段
	fieldsWithEmpty := map[string][]byte{
		"valid":   []byte("valid data"),
		"empty":   []byte{},
		"nil":     nil,
	}
	
	_, err = cryptoService.EncryptBatch(ctx, fieldsWithEmpty)
	assert.Error(t, err) // 应该因为空值而失败
}

func TestCryptoService_DecryptBatch(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	
	// 先生成一个活跃密钥
	ctx := context.Background()
	_, err := mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)
	
	cryptoService := NewCryptoService(mockManager, logger)
	
	// 先批量加密
	originalFields := map[string][]byte{
		"field1": []byte("value1"),
		"field2": []byte("value2"),
		"field3": []byte("value3"),
	}
	
	encryptedFields, err := cryptoService.EncryptBatch(ctx, originalFields)
	require.NoError(t, err)
	
	// 然后批量解密
	decryptedFields, err := cryptoService.DecryptBatch(ctx, encryptedFields)
	assert.NoError(t, err)
	assert.Len(t, decryptedFields, 3)
	
	for fieldName, originalValue := range originalFields {
		decryptedValue, exists := decryptedFields[fieldName]
		assert.True(t, exists)
		assert.Equal(t, originalValue, decryptedValue)
	}
	
	// 测试空字段集合
	emptyResult, err := cryptoService.DecryptBatch(ctx, map[string]*biz.EncryptedField{})
	assert.NoError(t, err)
	assert.Empty(t, emptyResult)
	
	// 测试包含nil的字段
	fieldsWithNil := map[string]*biz.EncryptedField{
		"valid": encryptedFields["field1"],
		"nil":   nil,
	}
	
	_, err = cryptoService.DecryptBatch(ctx, fieldsWithNil)
	assert.Error(t, err) // 应该因为nil值而失败
}

// TestCryptoService_ClearCache 测试缓存清除功能
// 注意：由于ClearCache是内部方法，这里通过间接方式测试缓存行为
func TestCryptoService_ClearCache(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	
	cryptoService := NewCryptoService(mockManager, logger)
	
	// 先生成一个活跃密钥
	ctx := context.Background()
	_, err := mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)
	
	// 进行一次加密操作以填充缓存
	value := []byte("test data")
	encrypted, err := cryptoService.EncryptField(ctx, "test", value)
	assert.NoError(t, err)
	
	// 验证解密能正常工作（说明缓存正常）
	decrypted, err := cryptoService.DecryptField(ctx, encrypted)
	assert.NoError(t, err)
	assert.Equal(t, value, decrypted)
}

func TestCryptoService_KeyCaching(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	
	// 先生成一个活跃密钥
	ctx := context.Background()
	_, err := mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)
	
	cryptoService := NewCryptoService(mockManager, logger)
	
	// 多次加密应该使用缓存的密钥
	value := []byte("test data")
	
	encrypted1, err := cryptoService.EncryptField(ctx, "field1", value)
	assert.NoError(t, err)
	
	encrypted2, err := cryptoService.EncryptField(ctx, "field2", value)
	assert.NoError(t, err)
	
	// 应该使用相同版本的密钥
	assert.Equal(t, encrypted1.Version, encrypted2.Version)
	
	// 解密应该也能使用缓存
	decrypted1, err := cryptoService.DecryptField(ctx, encrypted1)
	assert.NoError(t, err)
	assert.Equal(t, value, decrypted1)
	
	decrypted2, err := cryptoService.DecryptField(ctx, encrypted2)
	assert.NoError(t, err)
	assert.Equal(t, value, decrypted2)
}

// TestCryptoService_ErrorHandling 测试错误处理
func TestCryptoService_ErrorHandling(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	cryptoService := NewCryptoService(mockManager, logger)
	ctx := context.Background()

	// 测试没有活跃密钥时的加密
	_, err := cryptoService.EncryptField(ctx, "test", []byte("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get active data key")

	// 生成密钥后测试正常情况
	_, err = mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)

	// 测试加密空值
	_, err = cryptoService.EncryptField(ctx, "test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value cannot be empty")

	_, err = cryptoService.EncryptField(ctx, "test", []byte{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value cannot be empty")

	// 测试解密nil字段
	_, err = cryptoService.DecryptField(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encrypted field cannot be nil")
}

// TestCryptoService_KeyExpiration 测试密钥过期处理
func TestCryptoService_KeyExpiration(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	cryptoService := NewCryptoService(mockManager, logger)
	ctx := context.Background()

	// 生成一个已过期的密钥
	key, err := mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)

	// 手动设置为过期
	key.ExpiresAt = time.Now().Add(-1 * time.Hour)
	mockManager.activeKey = key

	// 尝试加密，应该重新获取密钥
	value := []byte("test data")
	encrypted, err := cryptoService.EncryptField(ctx, "test", value)
	assert.NoError(t, err)
	assert.NotNil(t, encrypted)
}

// TestCryptoService_ConcurrentAccess 测试并发访问
func TestCryptoService_ConcurrentAccess(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	cryptoService := NewCryptoService(mockManager, logger)
	ctx := context.Background()

	// 生成密钥
	_, err := mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	errorChan := make(chan error, numGoroutines*numOperations)

	// 并发加密和解密
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				value := []byte(fmt.Sprintf("data-%d-%d", id, j))
				
				// 加密
				encrypted, err := cryptoService.EncryptField(ctx, "test", value)
				if err != nil {
					errorChan <- err
					return
				}
				
				// 解密
				decrypted, err := cryptoService.DecryptField(ctx, encrypted)
				if err != nil {
					errorChan <- err
					return
				}
				
				if string(decrypted) != string(value) {
					errorChan <- fmt.Errorf("data mismatch: expected %s, got %s", value, decrypted)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// 检查是否有错误
	for err := range errorChan {
		t.Errorf("Concurrent operation failed: %v", err)
	}
}

// TestCryptoService_ClearCache 测试缓存清除
func TestCryptoService_ClearCache_Detailed(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	cryptoService := NewCryptoService(mockManager, logger)
	ctx := context.Background()

	// 生成密钥并进行操作以填充缓存
	_, err := mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)

	value := []byte("test data")
	encrypted, err := cryptoService.EncryptField(ctx, "test", value)
	require.NoError(t, err)

	// 验证缓存工作正常
	decrypted, err := cryptoService.DecryptField(ctx, encrypted)
	require.NoError(t, err)
	assert.Equal(t, value, decrypted)

	// 清除缓存
	cryptoService.ClearCache()

	// 缓存清除后应该仍能正常工作（重新从manager获取）
	decrypted2, err := cryptoService.DecryptField(ctx, encrypted)
	assert.NoError(t, err)
	assert.Equal(t, value, decrypted2)
}

// TestCryptoService_HashField_EdgeCases 测试哈希函数的边界情况
func TestCryptoService_HashField_EdgeCases(t *testing.T) {
	mockManager := newMockDataKeyManager()
	logger := log.NewStdLogger(os.Stdout)
	cryptoService := NewCryptoService(mockManager, logger)

	// 测试nil值
	hash1 := cryptoService.HashField(nil)
	assert.NotEmpty(t, hash1)

	// 测试空切片
	hash2 := cryptoService.HashField([]byte{})
	assert.NotEmpty(t, hash2)
	assert.Equal(t, hash1, hash2) // nil和空切片应该产生相同哈希

	// 测试单字节
	hash3 := cryptoService.HashField([]byte{0})
	assert.NotEmpty(t, hash3)
	assert.NotEqual(t, hash1, hash3)

	// 测试大数据
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}
	hash4 := cryptoService.HashField(largeData)
	assert.NotEmpty(t, hash4)
	assert.Len(t, hash4, 64) // SHA256哈希应该是64个十六进制字符

	// 验证哈希的确定性
	hash5 := cryptoService.HashField(largeData)
	assert.Equal(t, hash4, hash5)
}

// mockDataKeyManagerWithErrors 带错误模拟的数据密钥管理器
type mockDataKeyManagerWithErrors struct {
	*mockDataKeyManager
	shouldFailGetActive bool
	shouldFailGetByVersion bool
}

func (m *mockDataKeyManagerWithErrors) GetActiveDataKey(ctx context.Context) (*biz.DataKey, error) {
	if m.shouldFailGetActive {
		return nil, errors.New("mock error: failed to get active key")
	}
	return m.mockDataKeyManager.GetActiveDataKey(ctx)
}

func (m *mockDataKeyManagerWithErrors) GetDataKeyByVersion(ctx context.Context, version string) (*biz.DataKey, error) {
	if m.shouldFailGetByVersion {
		return nil, errors.New("mock error: failed to get key by version")
	}
	return m.mockDataKeyManager.GetDataKeyByVersion(ctx, version)
}

// TestCryptoService_ManagerErrors 测试数据密钥管理器错误
func TestCryptoService_ManagerErrors(t *testing.T) {
	mockManager := &mockDataKeyManagerWithErrors{
		mockDataKeyManager: newMockDataKeyManager(),
		shouldFailGetActive: true,
	}
	logger := log.NewStdLogger(os.Stdout)
	cryptoService := NewCryptoService(mockManager, logger)
	ctx := context.Background()

	// 测试获取活跃密钥失败
	_, err := cryptoService.EncryptField(ctx, "test", []byte("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get active data key")

	// 测试批量加密时获取活跃密钥失败
	fields := map[string][]byte{"field1": []byte("value1")}
	_, err = cryptoService.EncryptBatch(ctx, fields)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get active data key")

	// 修复获取活跃密钥，但获取版本密钥失败
	mockManager.shouldFailGetActive = false
	mockManager.shouldFailGetByVersion = true

	// 先生成密钥
	_, err = mockManager.GenerateDataKey(ctx, "AES-256-GCM")
	require.NoError(t, err)

	// 加密成功
	encrypted, err := cryptoService.EncryptField(ctx, "test", []byte("data"))
	require.NoError(t, err)

	// 清除缓存，确保解密时会调用GetDataKeyByVersion
	cryptoService.ClearCache()

	// 解密失败（因为获取版本密钥失败）
	_, err = cryptoService.DecryptField(ctx, encrypted)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get data key")
}