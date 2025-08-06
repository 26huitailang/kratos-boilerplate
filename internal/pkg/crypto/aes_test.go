package crypto

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAESEncryptor(t *testing.T) {
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
			encryptor, err := NewAESEncryptor(tt.key)
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

func TestAESEncryptor_Encrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := NewAESEncryptor(key)
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
			plaintext: []byte("this is a very long text that should be encrypted properly"),
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
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
			assert.Contains(t, parts, AlgoAESGCM)
			assert.Contains(t, parts, ".")
		})
	}
}

func TestAESEncryptor_Decrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := NewAESEncryptor(key)
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
			plaintext: []byte("this is a very long text that should be encrypted and decrypted properly"),
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先加密
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			require.NoError(t, err)

			// 再解密
			decrypted, err := encryptor.Decrypt(ciphertext)
			assert.NoError(t, err)
			if len(tt.plaintext) == 0 {
				assert.Empty(t, decrypted)
			} else {
				assert.Equal(t, tt.plaintext, decrypted)
			}
		})
	}
}

func TestAESEncryptor_Decrypt_InvalidFormat(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext []byte
		wantErr    string
	}{
		{
			name:       "invalid format - no dots",
			ciphertext: []byte("invalidformat"),
			wantErr:    "invalid ciphertext format",
		},
		{
			name:       "invalid format - only one dot",
			ciphertext: []byte("01.something"),
			wantErr:    "invalid ciphertext format",
		},
		{
			name:       "unsupported algorithm",
			ciphertext: []byte("99.dGVzdA==.dGVzdA=="),
			wantErr:    "unsupported algorithm",
		},
		{
			name:       "invalid nonce",
			ciphertext: []byte("01.invalid_base64.dGVzdA=="),
			wantErr:    "invalid nonce",
		},
		{
			name:       "invalid ciphertext data",
			ciphertext: []byte("01.dGVzdA==.invalid_base64"),
			wantErr:    "invalid ciphertext",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrypted, err := encryptor.Decrypt(tt.ciphertext)
			assert.Error(t, err)
			assert.Nil(t, decrypted)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestAESEncryptor_Hash(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewAESEncryptor(key)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := encryptor.Hash(tt.data)
			assert.NotEmpty(t, hash)
			assert.Len(t, hash, 64) // SHA256 产生 64 个十六进制字符

			// 相同数据应该产生相同的哈希
			hash2 := encryptor.Hash(tt.data)
			assert.Equal(t, hash, hash2)
		})
	}
}

func TestNewEncryptor(t *testing.T) {
	key := make([]byte, 32)

	tests := []struct {
		name    string
		algo    string
		key     []byte
		wantErr bool
	}{
		{
			name:    "AES-GCM algorithm",
			algo:    AlgoAESGCM,
			key:     key,
			wantErr: false,
		},
		{
			name:    "SM3 algorithm",
			algo:    AlgoSM3,
			key:     key,
			wantErr: false,
		},
		{
			name:    "unsupported algorithm",
			algo:    "99",
			key:     key,
			wantErr: true,
		},
		{
			name:    "invalid key for AES",
			algo:    AlgoAESGCM,
			key:     make([]byte, 16),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encryptor, err := NewEncryptor(tt.algo, tt.key)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, encryptor)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, encryptor)
			}
		})
	}
}

func TestAnonymize(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		keepStart int
		keepEnd   int
		expected  string
	}{
		{
			name:      "phone number",
			data:      "13800138000",
			keepStart: 3,
			keepEnd:   4,
			expected:  "138****8000",
		},
		{
			name:      "email address",
			data:      "user@example.com",
			keepStart: 2,
			keepEnd:   4,
			expected:  "us**********.com",
		},
		{
			name:      "short string - no anonymization",
			data:      "abc",
			keepStart: 2,
			keepEnd:   2,
			expected:  "abc",
		},
		{
			name:      "exact length match",
			data:      "abcd",
			keepStart: 2,
			keepEnd:   2,
			expected:  "abcd",
		},
		{
			name:      "empty string",
			data:      "",
			keepStart: 1,
			keepEnd:   1,
			expected:  "",
		},
		{
			name:      "chinese characters",
			data:      "张三李四王五",
			keepStart: 1,
			keepEnd:   1,
			expected:  "张****五",
		},
		{
			name:      "zero keep parameters",
			data:      "testdata",
			keepStart: 0,
			keepEnd:   0,
			expected:  "********",
		},
		{
			name:      "negative keep parameters",
			data:      "testdata",
			keepStart: -1,
			keepEnd:   -1,
			expected:  "********",
		},
		{
			name:      "keep parameters larger than string",
			data:      "test",
			keepStart: 10,
			keepEnd:   10,
			expected:  "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Anonymize(tt.data, tt.keepStart, tt.keepEnd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// 测试AES加密器的错误处理
func TestAESEncryptor_ErrorHandling(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	// 测试解密时GCM打开失败的情况
	// 构造一个有效格式但内容错误的密文
	// GCM需要12字节的nonce，所以使用正确长度的base64编码
	validNonce := base64.StdEncoding.EncodeToString(make([]byte, 12)) // 12字节nonce
	invalidCiphertext := []byte(fmt.Sprintf("01.%s.aW52YWxpZA==", validNonce)) // 有效nonce但无效密文
	decrypted, err := encryptor.Decrypt(invalidCiphertext)
	assert.Error(t, err)
	assert.Nil(t, decrypted)

	// 测试无效格式的密文
	invalidFormat := []byte("invalid.format")
	decrypted, err = encryptor.Decrypt(invalidFormat)
	assert.Error(t, err)
	assert.Nil(t, decrypted)
	assert.Contains(t, err.Error(), "invalid ciphertext format")

	// 测试不支持的算法
	unsupportedAlgo := []byte("99." + validNonce + ".aW52YWxpZA==")
	decrypted, err = encryptor.Decrypt(unsupportedAlgo)
	assert.Error(t, err)
	assert.Nil(t, decrypted)
	assert.Contains(t, err.Error(), "unsupported algorithm")
}

// 测试并发安全性
func TestAESEncryptor_ConcurrentAccess(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	const goroutines = 50
	const iterations = 10

	// 并发加密和解密
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < iterations; j++ {
				data := []byte(fmt.Sprintf("test data %d-%d", id, j))

				// 加密
				ciphertext, err := encryptor.Encrypt(data)
				assert.NoError(t, err)
				assert.NotNil(t, ciphertext)

				// 解密
				decrypted, err := encryptor.Decrypt(ciphertext)
				assert.NoError(t, err)
				assert.Equal(t, data, decrypted)

				// 哈希
				hash := encryptor.Hash(data)
				assert.NotEmpty(t, hash)
				assert.Len(t, hash, 64)
			}
		}(i)
	}
}

// 测试大数据量加密
func TestAESEncryptor_LargeData(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	// 测试不同大小的数据
	sizes := []int{1024, 10240, 102400, 1048576} // 1KB, 10KB, 100KB, 1MB

	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			data := make([]byte, size)
			for i := range data {
				data[i] = byte(i % 256)
			}

			// 加密
			ciphertext, err := encryptor.Encrypt(data)
			assert.NoError(t, err)
			assert.NotNil(t, ciphertext)

			// 解密
			decrypted, err := encryptor.Decrypt(ciphertext)
			assert.NoError(t, err)
			assert.Equal(t, data, decrypted)
		})
	}
}

// 测试哈希函数的特性
func TestAESEncryptor_HashProperties(t *testing.T) {
	key := make([]byte, 32)
	encryptor, err := NewAESEncryptor(key)
	require.NoError(t, err)

	// 测试哈希的确定性
	data := []byte("deterministic test")
	hash1 := encryptor.Hash(data)
	hash2 := encryptor.Hash(data)
	assert.Equal(t, hash1, hash2)

	// 测试哈希的雪崩效应（微小变化导致大幅变化）
	data1 := []byte("test data")
	data2 := []byte("test datb") // 只改变最后一个字符
	hash1 = encryptor.Hash(data1)
	hash2 = encryptor.Hash(data2)
	assert.NotEqual(t, hash1, hash2)

	// 计算汉明距离（不同位的数量）
	diffBits := 0
	for i := 0; i < len(hash1) && i < len(hash2); i++ {
		if hash1[i] != hash2[i] {
			diffBits++
		}
	}
	// 好的哈希函数应该有大约50%的位不同
	assert.Greater(t, diffBits, len(hash1)/4) // 至少25%的字符不同
}