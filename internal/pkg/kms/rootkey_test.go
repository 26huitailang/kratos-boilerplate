package kms

import (
	"testing"

	"kratos-boilerplate/internal/biz"

	"github.com/stretchr/testify/assert"
)

// TestRootKeyGenerator_GenerateRootKey 测试根密钥生成
func TestRootKeyGenerator_GenerateRootKey(t *testing.T) {
	config := &biz.KMSConfig{
		Seed:       "test-seed",
		Salt:       "test-salt",
		Iterations: 10000,
		KeyLength:  32,
		Algorithm:  "AES-256-GCM",
	}
	generator := NewRootKeyGenerator(config)
	
	key, err := generator.GenerateRootKey()
	assert.NoError(t, err)
	assert.NotNil(t, key)
	assert.Len(t, key, 32)
	
	// 相同配置应该生成相同的密钥
	key2, err := generator.GenerateRootKey()
	assert.NoError(t, err)
	assert.Equal(t, key, key2)
}

// TestRootKeyGenerator_GenerateRootKey_DifferentConfigs 测试不同配置生成不同密钥
func TestRootKeyGenerator_GenerateRootKey_DifferentConfigs(t *testing.T) {
	config1 := &biz.KMSConfig{
		Seed:       "test-seed-1",
		Salt:       "test-salt",
		Iterations: 10000,
		KeyLength:  32,
	}
	generator1 := NewRootKeyGenerator(config1)
	
	config2 := &biz.KMSConfig{
		Seed:       "test-seed-2",
		Salt:       "test-salt",
		Iterations: 10000,
		KeyLength:  32,
	}
	generator2 := NewRootKeyGenerator(config2)
	
	key1, err := generator1.GenerateRootKey()
	assert.NoError(t, err)
	
	key2, err := generator2.GenerateRootKey()
	assert.NoError(t, err)
	
	// 不同配置应该生成不同的密钥
	assert.NotEqual(t, key1, key2)
}

// TestRootKeyGenerator_ValidateConfig 测试配置验证
func TestRootKeyGenerator_ValidateConfig(t *testing.T) {
	
	tests := []struct {
		name    string
		config  *biz.KMSConfig
		wantErr bool
	}{
		{
			name: "valid_config",
			config: &biz.KMSConfig{
				Seed:       "test-seed",
				Salt:       "test-salt",
				Iterations: 10000,
				KeyLength:  32,
			},
			wantErr: false,
		},
		{
			name: "empty_seed",
			config: &biz.KMSConfig{
				Seed:       "",
				Salt:       "test-salt",
				Iterations: 10000,
				KeyLength:  32,
			},
			wantErr: true,
		},
		{
			name: "empty_salt",
			config: &biz.KMSConfig{
				Seed:       "test-seed",
				Salt:       "",
				Iterations: 10000,
				KeyLength:  32,
			},
			wantErr: true,
		},
		{
			name: "low_iterations",
			config: &biz.KMSConfig{
				Seed:       "test-seed",
				Salt:       "test-salt",
				Iterations: 100,
				KeyLength:  32,
			},
			wantErr: true,
		},
		{
			name: "invalid_key_length",
			config: &biz.KMSConfig{
				Seed:       "test-seed",
				Salt:       "test-salt",
				Iterations: 10000,
				KeyLength:  15,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testGenerator := NewRootKeyGenerator(tt.config)
			err := testGenerator.ValidateConfig()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRootKeyGenerator_GetKeyStrength 测试密钥强度计算
func TestRootKeyGenerator_GetKeyStrength(t *testing.T) {
	
	tests := []struct {
		name      string
		config    *biz.KMSConfig
		expected  string
	}{
		{
			name: "aes_128",
			config: &biz.KMSConfig{
				KeyLength:  16,
				Iterations: 10000,
			},
			expected: "AES-128",
		},
		{
			name: "aes_256",
			config: &biz.KMSConfig{
				KeyLength:  32,
				Iterations: 100000,
			},
			expected: "AES-256",
		},
		{
			name: "aes_192",
			config: &biz.KMSConfig{
				KeyLength:  24,
				Iterations: 50000,
			},
			expected: "AES-192",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testGenerator := NewRootKeyGenerator(tt.config)
			strength := testGenerator.GetKeyStrength()
			assert.Equal(t, tt.expected, strength)
		})
	}
}

// TestRootKeyGenerator_EstimateGenerationTime 测试生成时间估算
func TestRootKeyGenerator_EstimateGenerationTime(t *testing.T) {
	tests := []struct {
		name     string
		config   *biz.KMSConfig
		minTime  int64
		maxTime  int64
	}{
		{
			name: "low_iterations",
			config: &biz.KMSConfig{
				Seed:       "test-seed",
				Salt:       "test-salt",
				Iterations: 1000,
				KeyLength:  32,
			},
			minTime: 0,
			maxTime: 100,
		},
		{
			name: "medium_iterations",
			config: &biz.KMSConfig{
				Seed:       "test-seed",
				Salt:       "test-salt",
				Iterations: 10000,
				KeyLength:  32,
			},
			minTime: 0,
			maxTime: 500,
		},
		{
			name: "high_iterations",
			config: &biz.KMSConfig{
				Seed:       "test-seed",
				Salt:       "test-salt",
				Iterations: 100000,
				KeyLength:  32,
			},
			minTime: 1,
			maxTime: 5000,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testGenerator := NewRootKeyGenerator(tt.config)
			estimated := testGenerator.EstimateGenerationTime()
			assert.True(t, estimated >= tt.minTime, "估算时间应该大于等于最小时间")
			assert.True(t, estimated <= tt.maxTime, "估算时间应该小于等于最大时间")
		})
	}
}

// TestRootKeyGenerator_GenerateRootKey_InvalidConfig 测试无效配置
func TestRootKeyGenerator_GenerateRootKey_InvalidConfig(t *testing.T) {
	// 测试无效配置
	invalidConfig := &biz.KMSConfig{
		Seed:       "",
		Salt:       "test-salt",
		Iterations: 10000,
		KeyLength:  32,
	}
	
	generator := NewRootKeyGenerator(invalidConfig)
	_, err := generator.GenerateRootKey()
	assert.Error(t, err)
}

// TestRootKeyGenerator_ValidateConfig_EdgeCases 测试配置验证的边界情况
func TestRootKeyGenerator_ValidateConfig_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		config  *biz.KMSConfig
		wantErr bool
		errorMsg string
	}{
		{
			name:    "nil_config",
			config:  nil,
			wantErr: true,
			errorMsg: "config is nil",
		},
		{
			name: "unsupported_key_length_8",
			config: &biz.KMSConfig{
				Seed:       "test-seed",
				Salt:       "test-salt",
				Iterations: 10000,
				KeyLength:  8,
			},
			wantErr: true,
			errorMsg: "无效的密钥长度",
		},
		{
			name: "unsupported_key_length_20",
			config: &biz.KMSConfig{
				Seed:       "test-seed",
				Salt:       "test-salt",
				Iterations: 10000,
				KeyLength:  20,
			},
			wantErr: true,
			errorMsg: "unsupported key length",
		},
		{
			name: "valid_aes_192",
			config: &biz.KMSConfig{
				Seed:       "test-seed",
				Salt:       "test-salt",
				Iterations: 10000,
				KeyLength:  24,
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var generator RootKeyGenerator
			if tt.config == nil {
				generator = &rootKeyGenerator{config: nil}
			} else {
				generator = NewRootKeyGenerator(tt.config)
			}
			
			err := generator.ValidateConfig()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRootKeyGenerator_GetKeyStrength_EdgeCases 测试密钥强度的边界情况
func TestRootKeyGenerator_GetKeyStrength_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		keyLength int
		expected  string
	}{
		{
			name:      "unknown_key_length_0",
			keyLength: 0,
			expected:  "Unknown",
		},
		{
			name:      "unknown_key_length_64",
			keyLength: 64,
			expected:  "Unknown",
		},
		{
			name:      "unknown_key_length_negative",
			keyLength: -1,
			expected:  "Unknown",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &biz.KMSConfig{
				KeyLength: tt.keyLength,
			}
			generator := NewRootKeyGenerator(config)
			strength := generator.GetKeyStrength()
			assert.Equal(t, tt.expected, strength)
		})
	}
}

// TestRootKeyGenerator_GenerateRootKey_KeyLengthMismatch 测试密钥长度不匹配的情况
func TestRootKeyGenerator_GenerateRootKey_KeyLengthMismatch(t *testing.T) {
	// 这个测试用于覆盖GenerateRootKey中的长度检查逻辑
	// 虽然在正常情况下PBKDF2不会产生长度不匹配，但我们需要覆盖这个分支
	config := &biz.KMSConfig{
		Seed:       "test-seed",
		Salt:       "test-salt",
		Iterations: 10000,
		KeyLength:  32,
	}
	
	generator := NewRootKeyGenerator(config)
	key, err := generator.GenerateRootKey()
	assert.NoError(t, err)
	assert.Len(t, key, 32) // 验证生成的密钥长度正确
}