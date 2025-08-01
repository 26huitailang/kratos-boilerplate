package crypto

import (
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Anonymize(tt.data, tt.keepStart, tt.keepEnd)
			assert.Equal(t, tt.expected, result)
		})
	}
}