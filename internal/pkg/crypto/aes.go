package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

const (
	AlgoAESGCM = "01" // AES-GCM 算法标记位
	AlgoSM3    = "02" // SM3 算法标记位
)

// Encryptor 加密器接口
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
	Hash(plaintext []byte) string
}

// AESEncryptor AES-GCM 加密实现
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor 创建新的 AES 加密器
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	if len(key) != 32 { // AES-256 需要 32 字节密钥
		return nil, fmt.Errorf("invalid key size: need 32 bytes")
	}
	return &AESEncryptor{key: key}, nil
}

// Encrypt 加密数据，返回格式: 算法标记位.IV.密文
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// 组装最终的加密字符串：算法标记位.IV.密文
	result := fmt.Sprintf("%s.%s.%s",
		AlgoAESGCM,
		base64.StdEncoding.EncodeToString(nonce),
		base64.StdEncoding.EncodeToString(ciphertext),
	)

	return []byte(result), nil
}

// Decrypt 解密数据
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	// 解析加密字符串
	parts := strings.Split(string(ciphertext), ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid ciphertext format")
	}

	// 验证算法标记位
	if parts[0] != AlgoAESGCM {
		return nil, fmt.Errorf("unsupported algorithm: %s", parts[0])
	}

	// 解码 IV 和密文
	nonce, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %v", err)
	}

	encrypted, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext: %v", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, encrypted, nil)
}

// Hash 计算数据的哈希值
func (e *AESEncryptor) Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// NewEncryptor 根据算法标记位创建对应的加密器
func NewEncryptor(algo string, key []byte) (Encryptor, error) {
	switch algo {
	case AlgoAESGCM:
		return NewAESEncryptor(key)
	case AlgoSM3:
		return NewSM3Encryptor(key)
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", algo)
	}
}

// Anonymize 匿名化处理
func Anonymize(data string, keepStart, keepEnd int) string {
	runes := []rune(data)
	length := len(runes)

	if length <= keepStart+keepEnd {
		return string(runes)
	}

	start := string(runes[:keepStart])
	end := string(runes[length-keepEnd:])
	stars := ""
	for i := 0; i < length-keepStart-keepEnd; i++ {
		stars += "*"
	}

	return start + stars + end
}
