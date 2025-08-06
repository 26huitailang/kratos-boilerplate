package crypto

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSM3Encryptor(t *testing.T) {
	tests := []struct {
		name    string
		key     []byte
		wantErr bool
	}{
		{
			name:    "valid 32-byte key",
			key:     make([]byte, 32),
			wantErr: false,
		},
		{
			name:    "invalid 16-byte key",
			key:     make([]byte, 16),
			wantErr: true,
		},
		{
			name:    "invalid 24-byte key",
			key:     make([]byte, 24),
			wantErr: true,
		},
		{
			name:    "empty key",
			key:     []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor, err := NewSM3Encryptor(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, encryptor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encryptor)
				assert.Equal(t, tt.key, encryptor.key)
			}
		})
	}
}

func TestSM3Encryptor_Encrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := NewSM3Encryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "simple text",
			plaintext: []byte("hello world"),
		},
		{
			name:      "empty text",
			plaintext: []byte(""),
		},
		{
			name:      "long text",
			plaintext: []byte("this is a very long text that should be processed with SM3"),
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
		},
		{
			name:      "chinese text",
			plaintext: []byte("你好世界"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			assert.NoError(t, err)
			assert.NotNil(t, ciphertext)
			assert.NotEqual(t, tt.plaintext, ciphertext)

			// 验证加密结果格式：算法标记位.IV.密文
			parts := string(ciphertext)
			assert.Contains(t, parts, AlgoSM3)
			assert.Contains(t, parts, ".")

			// SM3是哈希算法，相同输入应该产生相同输出（除了IV部分）
			ciphertext2, err := encryptor.Encrypt(tt.plaintext)
			assert.NoError(t, err)
			// 由于IV是随机的，所以两次加密结果应该不同
			assert.NotEqual(t, ciphertext, ciphertext2)
		})
	}
}

func TestSM3Encryptor_Decrypt(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewSM3Encryptor(key)
	require.NoError(t, err)

	// SM3是哈希算法，不支持解密
	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{
			name:       "valid SM3 format",
			ciphertext: []byte("02.dGVzdA==.dGVzdA=="),
		},
		{
			name:       "invalid format",
			ciphertext: []byte("invalid"),
		},
		{
			name:       "wrong algorithm",
			ciphertext: []byte("01.dGVzdA==.dGVzdA=="),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrypted, err := encryptor.Decrypt(tt.ciphertext)
			assert.Error(t, err)
			assert.Nil(t, decrypted)

			if string(tt.ciphertext) == "02.dGVzdA==.dGVzdA==" {
				// 对于正确的SM3格式，应该返回特定的错误信息
				assert.Contains(t, err.Error(), "SM3 is a hash algorithm, decryption is not supported")
			} else if string(tt.ciphertext) == "01.dGVzdA==.dGVzdA==" {
				// 对于错误的算法标记，应该返回不支持的算法错误
				assert.Contains(t, err.Error(), "unsupported algorithm")
			} else {
				// 对于无效格式，应该返回格式错误
				assert.Contains(t, err.Error(), "invalid ciphertext format")
			}
		})
	}
}

func TestSM3Encryptor_Hash(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewSM3Encryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "simple text",
			data: []byte("hello world"),
		},
		{
			name: "empty data",
			data: []byte(""),
		},
		{
			name: "binary data",
			data: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
		},
		{
			name: "chinese text",
			data: []byte("你好世界"),
		},
		{
			name: "long text",
			data: []byte("this is a very long text to test SM3 hash function with various input lengths"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := encryptor.Hash(tt.data)
			assert.NotEmpty(t, hash)
			assert.Len(t, hash, 64) // SM3 产生 64 个十六进制字符 (256位)

			// 相同数据应该产生相同的哈希
			hash2 := encryptor.Hash(tt.data)
			assert.Equal(t, hash, hash2)

			// 不同数据应该产生不同的哈希
			if len(tt.data) > 0 {
				differentData := append(tt.data, 0x01)
				differentHash := encryptor.Hash(differentData)
				assert.NotEqual(t, hash, differentHash)
			}
		})
	}
}

// 测试SM3哈希的一致性
func TestSM3Encryptor_Hash_Consistency(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 1) // 不同的密钥
	}

	encryptor1, err := NewSM3Encryptor(key1)
	require.NoError(t, err)

	encryptor2, err := NewSM3Encryptor(key2)
	require.NoError(t, err)

	data := []byte("test data for consistency")

	// 不同的加密器实例对相同数据应该产生相同的哈希
	// 因为SM3哈希不依赖于密钥，只是为了接口一致性
	hash1 := encryptor1.Hash(data)
	hash2 := encryptor2.Hash(data)

	// SM3哈希应该是确定性的，不依赖于密钥
	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64)
	assert.Len(t, hash2, 64)
}

// 测试SM3加密器与NewEncryptor工厂函数的集成
func TestSM3Encryptor_Integration(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	// 通过工厂函数创建SM3加密器
	encryptor, err := NewEncryptor(AlgoSM3, key)
	require.NoError(t, err)
	require.NotNil(t, encryptor)

	// 测试类型断言
	sm3Encryptor, ok := encryptor.(*SM3Encryptor)
	assert.True(t, ok)
	assert.Equal(t, key, sm3Encryptor.key)

	// 测试基本功能
	data := []byte("integration test data")

	// 测试加密
	ciphertext, err := encryptor.Encrypt(data)
	assert.NoError(t, err)
	assert.NotNil(t, ciphertext)
	assert.Contains(t, string(ciphertext), AlgoSM3)

	// 测试哈希
	hash := encryptor.Hash(data)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)

	// 测试解密（应该失败）
	decrypted, err := encryptor.Decrypt(ciphertext)
	assert.Error(t, err)
	assert.Nil(t, decrypted)
	assert.Contains(t, err.Error(), "SM3 is a hash algorithm, decryption is not supported")
}

// 测试SM3加密器的边界情况
func TestSM3Encryptor_EdgeCases(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewSM3Encryptor(key)
	require.NoError(t, err)

	// 测试极大数据
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	ciphertext, err := encryptor.Encrypt(largeData)
	assert.NoError(t, err)
	assert.NotNil(t, ciphertext)

	hash := encryptor.Hash(largeData)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)

	// 测试特殊字符
	specialData := []byte("\x00\x01\x02\xFF\xFE\xFD")
	ciphertext, err = encryptor.Encrypt(specialData)
	assert.NoError(t, err)
	assert.NotNil(t, ciphertext)

	hash = encryptor.Hash(specialData)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)
}

// 测试SM3解密的各种错误情况
func TestSM3Encryptor_DecryptErrors(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewSM3Encryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext []byte
		errorMsg   string
	}{
		{
			name:       "empty ciphertext",
			ciphertext: []byte(""),
			errorMsg:   "invalid ciphertext format",
		},
		{
			name:       "single dot",
			ciphertext: []byte("02.something"),
			errorMsg:   "invalid ciphertext format",
		},
		{
			name:       "no dots",
			ciphertext: []byte("02something"),
			errorMsg:   "invalid ciphertext format",
		},
		{
			name:       "wrong algorithm",
			ciphertext: []byte("01.dGVzdA==.dGVzdA=="),
			errorMsg:   "unsupported algorithm",
		},
		{
			name:       "invalid algorithm format",
			ciphertext: []byte("abc.dGVzdA==.dGVzdA=="),
			errorMsg:   "unsupported algorithm",
		},
		{
			name:       "valid SM3 format",
			ciphertext: []byte("02.dGVzdA==.dGVzdA=="),
			errorMsg:   "SM3 is a hash algorithm, decryption is not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrypted, err := encryptor.Decrypt(tt.ciphertext)
			assert.Error(t, err)
			assert.Nil(t, decrypted)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

// 测试SM3哈希的性能和一致性
func TestSM3Encryptor_HashPerformance(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewSM3Encryptor(key)
	require.NoError(t, err)

	// 测试不同大小数据的哈希性能
	sizes := []int{100, 1000, 10000, 100000}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			data := make([]byte, size)
			for i := range data {
				data[i] = byte(i % 256)
			}

			// 多次哈希应该产生相同结果
			hash1 := encryptor.Hash(data)
			hash2 := encryptor.Hash(data)
			hash3 := encryptor.Hash(data)

			assert.Equal(t, hash1, hash2)
			assert.Equal(t, hash2, hash3)
			assert.Len(t, hash1, 64)
		})
	}
}

// 测试SM3并发安全性
func TestSM3Encryptor_ConcurrentSafety(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewSM3Encryptor(key)
	require.NoError(t, err)

	const goroutines = 20
	const iterations = 50

	results := make(chan string, goroutines*iterations)
	errors := make(chan error, goroutines*iterations)

	// 并发执行哈希操作
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < iterations; j++ {
				data := []byte(fmt.Sprintf("concurrent test %d-%d", id, j))

				// 测试哈希
				hash := encryptor.Hash(data)
				if len(hash) != 64 {
					errors <- fmt.Errorf("invalid hash length: %d", len(hash))
					return
				}
				results <- hash

				// 测试加密
				ciphertext, err := encryptor.Encrypt(data)
				if err != nil {
					errors <- err
					return
				}
				if !strings.Contains(string(ciphertext), AlgoSM3) {
					errors <- fmt.Errorf("invalid ciphertext format")
					return
				}
			}
		}(i)
	}

	// 收集结果
	for i := 0; i < goroutines*iterations; i++ {
		select {
		case <-results:
			// 成功
		case err := <-errors:
			t.Fatalf("并发测试失败: %v", err)
		case <-time.After(10 * time.Second):
			t.Fatal("并发测试超时")
		}
	}
}