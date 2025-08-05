package kms

import "errors"

// KMS相关错误定义
var (
	// 配置错误
	ErrInvalidSeed       = errors.New("invalid seed: seed cannot be empty")
	ErrInvalidSalt       = errors.New("invalid salt: salt cannot be empty")
	ErrInvalidIterations = errors.New("invalid iterations: should be at least 10000")
	ErrInvalidKeyLength  = errors.New("invalid key length: should be at least 16 bytes")
	ErrInvalidConfig     = errors.New("invalid KMS configuration")
	
	// 密钥错误
	ErrKeyNotFound       = errors.New("data key not found")
	ErrNoActiveKey       = errors.New("no active data key found")
	ErrKeyExpired        = errors.New("data key has expired")
	ErrKeyGenerationFail = errors.New("failed to generate data key")
	ErrKeyEncryptionFail = errors.New("failed to encrypt data key")
	ErrKeyDecryptionFail = errors.New("failed to decrypt data key")
	
	// 加解密错误
	ErrEncryptionFail    = errors.New("encryption operation failed")
	ErrDecryptionFail    = errors.New("decryption operation failed")
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")
	ErrInvalidNonce      = errors.New("invalid nonce size")
	
	// 存储错误
	ErrStorageOperation = errors.New("storage operation failed")
	ErrDatabaseError    = errors.New("database operation error")
	
	// 系统错误
	ErrKMSNotInitialized = errors.New("KMS system not initialized")
	ErrKMSAlreadyInit    = errors.New("KMS system already initialized")
	ErrSystemShutdown    = errors.New("KMS system is shutting down")
)