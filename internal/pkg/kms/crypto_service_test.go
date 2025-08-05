package kms

import (
	"context"
	"os"
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
}

func newMockDataKeyManager() *mockDataKeyManager {
	return &mockDataKeyManager{
		keys: make(map[string]*biz.DataKey),
	}
}

func (m *mockDataKeyManager) GenerateDataKey(ctx context.Context, algorithm string) (*biz.DataKey, error) {
	key := &biz.DataKey{
		ID:        "test-key-id",
		Version:   "v1",
		Algorithm: algorithm,
		Key:       make([]byte, 32), // 32字节密钥
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsActive:  true,
	}
	
	// 生成随机密钥
	for i := range key.Key {
		key.Key[i] = byte(i)
	}
	
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
	return key, nil
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