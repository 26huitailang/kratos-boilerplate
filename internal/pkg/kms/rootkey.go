package kms

import (
	"crypto/sha256"
	"fmt"
	
	"golang.org/x/crypto/pbkdf2"
)

// rootKeyGenerator 根密钥生成器实现
type rootKeyGenerator struct {
	config *Config
}

// NewRootKeyGenerator 创建根密钥生成器
func NewRootKeyGenerator(config *Config) RootKeyGenerator {
	return &rootKeyGenerator{
		config: config,
	}
}

// GenerateRootKey 生成根密钥
func (g *rootKeyGenerator) GenerateRootKey() ([]byte, error) {
	if err := g.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	// 使用PBKDF2生成根密钥
	rootKey := pbkdf2.Key(
		[]byte(g.config.Seed),
		[]byte(g.config.Salt),
		g.config.Iterations,
		g.config.KeyLength,
		sha256.New,
	)
	
	if len(rootKey) != g.config.KeyLength {
		return nil, fmt.Errorf("generated key length mismatch: expected %d, got %d", g.config.KeyLength, len(rootKey))
	}
	
	return rootKey, nil
}

// ValidateConfig 验证配置
func (g *rootKeyGenerator) ValidateConfig() error {
	if g.config == nil {
		return ErrInvalidConfig
	}
	
	if g.config.Seed == "" {
		return ErrInvalidSeed
	}
	
	if g.config.Salt == "" {
		return ErrInvalidSalt
	}
	
	if g.config.Iterations < 10000 {
		return ErrInvalidIterations
	}
	
	if g.config.KeyLength < 16 {
		return ErrInvalidKeyLength
	}
	
	// 确保密钥长度是合理的（16, 24, 32字节对应AES-128, AES-192, AES-256）
	if g.config.KeyLength != 16 && g.config.KeyLength != 24 && g.config.KeyLength != 32 {
		return fmt.Errorf("unsupported key length: %d (supported: 16, 24, 32)", g.config.KeyLength)
	}
	
	return nil
}

// GetKeyStrength 获取密钥强度描述
func (g *rootKeyGenerator) GetKeyStrength() string {
	switch g.config.KeyLength {
	case 16:
		return "AES-128"
	case 24:
		return "AES-192"
	case 32:
		return "AES-256"
	default:
		return "Unknown"
	}
}

// EstimateGenerationTime 估算密钥生成时间（毫秒）
func (g *rootKeyGenerator) EstimateGenerationTime() int64 {
	// 基于迭代次数的粗略估算
	// 假设每10000次迭代大约需要1毫秒（实际会因硬件而异）
	return int64(g.config.Iterations / 10000)
}