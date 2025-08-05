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

func TestKMSManager_Initialize(t *testing.T) {
	mockRepo := NewMockKMSRepo()
	logger := log.NewStdLogger(os.Stdout)
	
	config := &biz.KMSConfig{
		Seed:           "test-seed",
		Salt:           "test-salt",
		Iterations:     10000,
		KeyLength:      32,
		RotateInterval: time.Hour,
		Algorithm:      "AES-256-GCM",
	}
	
	manager := NewKMSManager(mockRepo, config, logger)
	require.NotNil(t, manager)
	
	// 测试初始化后应该有一个活跃的数据密钥
	ctx := context.Background()
	activeKey, err := manager.GetActiveDataKey(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, activeKey)
	assert.True(t, activeKey.IsActive)
}

func TestKMSManager_EncryptDecrypt(t *testing.T) {
	mockRepo := NewMockKMSRepo()
	logger := log.NewStdLogger(os.Stdout)
	
	config := &biz.KMSConfig{
		Seed:           "test-seed",
		Salt:           "test-salt",
		Iterations:     10000,
		KeyLength:      32,
		RotateInterval: time.Hour,
		Algorithm:      "AES-256-GCM",
	}
	
	manager := NewKMSManager(mockRepo, config, logger)
	ctx := context.Background()
	
	// 获取加解密服务
	cryptoService := manager.GetCryptoService()
	
	// 测试数据
	fieldName := "test_field"
	originalData := []byte("sensitive data")
	
	// 加密
	encryptedField, err := cryptoService.EncryptField(ctx, fieldName, originalData)
	assert.NoError(t, err)
	assert.NotNil(t, encryptedField)
	assert.NotEmpty(t, encryptedField.Value)
	assert.NotEmpty(t, encryptedField.Version)
	assert.Equal(t, config.Algorithm, encryptedField.Algorithm)
	
	// 解密
	decryptedData, err := cryptoService.DecryptField(ctx, encryptedField)
	assert.NoError(t, err)
	assert.Equal(t, originalData, decryptedData)
}

func TestKMSManager_HashField(t *testing.T) {
	mockRepo := NewMockKMSRepo()
	logger := log.NewStdLogger(os.Stdout)
	
	config := &biz.KMSConfig{
		Seed:           "test-seed",
		Salt:           "test-salt",
		Iterations:     10000,
		KeyLength:      32,
		RotateInterval: time.Hour,
		Algorithm:      "AES-256-GCM",
	}
	
	manager := NewKMSManager(mockRepo, config, logger)
	
	// 获取加解密服务
	cryptoService := manager.GetCryptoService()
	
	// 测试数据
	data := []byte("test data")
	
	// 计算哈希
	hash1 := cryptoService.HashField(data)
	assert.NotEmpty(t, hash1)
	
	// 相同数据应该产生相同哈希
	hash2 := cryptoService.HashField(data)
	assert.Equal(t, hash1, hash2)
	
	// 不同数据应该产生不同哈希
	differentData := []byte("different data")
	hash3 := cryptoService.HashField(differentData)
	assert.NotEqual(t, hash1, hash3)
}

func TestKMSManager_RotateDataKey(t *testing.T) {
	mockRepo := NewMockKMSRepo()
	logger := log.NewStdLogger(os.Stdout)
	
	config := &biz.KMSConfig{
		Seed:           "test-seed",
		Salt:           "test-salt",
		Iterations:     10000,
		KeyLength:      32,
		RotateInterval: time.Hour,
		Algorithm:      "AES-256-GCM",
	}
	
	manager := NewKMSManager(mockRepo, config, logger)
	ctx := context.Background()
	
	// 获取初始活跃密钥
	initialKey, err := manager.GetActiveDataKey(ctx)
	assert.NoError(t, err)
	initialVersion := initialKey.Version
	
	// 轮换密钥
	newKey, err := manager.RotateDataKey(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, newKey)
	assert.True(t, newKey.IsActive)
	assert.NotEqual(t, initialVersion, newKey.Version)
	
	// 验证新密钥是活跃的
	currentActiveKey, err := manager.GetActiveDataKey(ctx)
	assert.NoError(t, err)
	assert.Equal(t, newKey.Version, currentActiveKey.Version)
	
	// 验证可以通过版本获取旧密钥
	oldKey, err := manager.GetDataKeyByVersion(ctx, initialVersion)
	assert.NoError(t, err)
	assert.NotNil(t, oldKey)
	assert.False(t, oldKey.IsActive)
}

func TestKMSManager_GetDataKeyByVersion(t *testing.T) {
	mockRepo := NewMockKMSRepo()
	logger := log.NewStdLogger(os.Stdout)
	
	config := &biz.KMSConfig{
		Seed:           "test-seed",
		Salt:           "test-salt",
		Iterations:     10000,
		KeyLength:      32,
		RotateInterval: time.Hour,
		Algorithm:      "AES-256-GCM",
	}
	
	manager := NewKMSManager(mockRepo, config, logger)
	ctx := context.Background()
	
	// 获取活跃密钥
	activeKey, err := manager.GetActiveDataKey(ctx)
	assert.NoError(t, err)
	
	// 通过版本获取密钥
	keyByVersion, err := manager.GetDataKeyByVersion(ctx, activeKey.Version)
	assert.NoError(t, err)
	assert.Equal(t, activeKey.Version, keyByVersion.Version)
	assert.Equal(t, activeKey.Algorithm, keyByVersion.Algorithm)
	
	// 测试不存在的版本
	_, err = manager.GetDataKeyByVersion(ctx, "non-existent-version")
	assert.Error(t, err)
	assert.Equal(t, biz.ErrKeyNotFound, err)
}

func TestKMSManager_CryptoService(t *testing.T) {
	mockRepo := NewMockKMSRepo()
	logger := log.NewStdLogger(os.Stdout)
	
	config := &biz.KMSConfig{
		Seed:           "test-seed",
		Salt:           "test-salt",
		Iterations:     10000,
		KeyLength:      32,
		RotateInterval: time.Hour,
		Algorithm:      "AES-256-GCM",
	}
	
	manager := NewKMSManager(mockRepo, config, logger)
	
	// 获取加解密服务
	cryptoService := manager.GetCryptoService()
	assert.NotNil(t, cryptoService)
	
	ctx := context.Background()
	
	// 测试批量加密
	fields := map[string][]byte{
		"field1": []byte("value1"),
		"field2": []byte("value2"),
		"field3": []byte("value3"),
	}
	
	encryptedFields, err := cryptoService.EncryptBatch(ctx, fields)
	assert.NoError(t, err)
	assert.Len(t, encryptedFields, 3)
	
	// 测试批量解密
	decryptedFields, err := cryptoService.DecryptBatch(ctx, encryptedFields)
	assert.NoError(t, err)
	assert.Len(t, decryptedFields, 3)
	
	for fieldName, originalValue := range fields {
		decryptedValue, exists := decryptedFields[fieldName]
		assert.True(t, exists)
		assert.Equal(t, originalValue, decryptedValue)
	}
}

func TestKMSManager_Close(t *testing.T) {
	mockRepo := NewMockKMSRepo()
	logger := log.NewStdLogger(os.Stdout)
	
	config := &biz.KMSConfig{
		Seed:           "test-seed",
		Salt:           "test-salt",
		Iterations:     10000,
		KeyLength:      32,
		RotateInterval: time.Hour,
		Algorithm:      "AES-256-GCM",
	}
	
	manager := NewKMSManager(mockRepo, config, logger)
	
	// 关闭KMS系统
	err := manager.Close()
	assert.NoError(t, err)
	
	// 再次关闭应该不会出错
	err = manager.Close()
	assert.NoError(t, err)
}

func TestKMSManager_ErrorHandling(t *testing.T) {
	mockRepo := NewMockKMSRepo()
	logger := log.NewStdLogger(os.Stdout)
	
	config := &biz.KMSConfig{
		Seed:           "test-seed",
		Salt:           "test-salt",
		Iterations:     10000,
		KeyLength:      32,
		RotateInterval: time.Hour,
		Algorithm:      "AES-256-GCM",
	}
	
	manager := NewKMSManager(mockRepo, config, logger)
	ctx := context.Background()
	
	// 获取加解密服务
	cryptoService := manager.GetCryptoService()
	
	// 测试空值加密
	_, err := cryptoService.EncryptField(ctx, "test", nil)
	assert.Error(t, err)
	
	_, err = cryptoService.EncryptField(ctx, "test", []byte{})
	assert.Error(t, err)
	
	// 测试空值解密
	_, err = cryptoService.DecryptField(ctx, nil)
	assert.Error(t, err)
	
	// 测试无效的加密字段
	invalidField := &biz.EncryptedField{
		Value:     []byte("invalid"),
		Version:   "non-existent",
		Algorithm: "AES-256-GCM",
	}
	_, err = cryptoService.DecryptField(ctx, invalidField)
	assert.Error(t, err)
}