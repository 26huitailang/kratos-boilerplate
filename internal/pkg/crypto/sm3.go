package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/tjfoc/gmsm/sm3"
)

// SM3Encryptor SM3 加密实现
type SM3Encryptor struct {
	key []byte
}

// NewSM3Encryptor 创建新的 SM3 加密器
func NewSM3Encryptor(key []byte) (*SM3Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key size: need 32 bytes")
	}
	return &SM3Encryptor{key: key}, nil
}

// Encrypt 加密数据，返回格式: 算法标记位.IV.密文
func (e *SM3Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	// 生成随机IV
	iv := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// 使用SM3进行加密
	h := sm3.New()
	h.Write(plaintext)
	ciphertext := h.Sum(nil)

	// 组装最终的加密字符串：算法标记位.IV.密文
	result := fmt.Sprintf("%s.%s.%s",
		AlgoSM3,
		base64.StdEncoding.EncodeToString(iv),
		base64.StdEncoding.EncodeToString(ciphertext),
	)

	return []byte(result), nil
}

// Decrypt 解密数据
func (e *SM3Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	// 解析加密字符串
	parts := strings.Split(string(ciphertext), ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid ciphertext format")
	}

	// 验证算法标记位
	if parts[0] != AlgoSM3 {
		return nil, fmt.Errorf("unsupported algorithm: %s", parts[0])
	}

	// SM3是哈希算法，不支持解密
	return nil, fmt.Errorf("SM3 is a hash algorithm, decryption is not supported")
}

// Hash 计算数据的哈希值
func (e *SM3Encryptor) Hash(data []byte) string {
	h := sm3.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
