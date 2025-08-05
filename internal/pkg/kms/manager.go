package kms

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
	
	"github.com/go-kratos/kratos/v2/log"
)

// kmsManager KMS管理器实现
type kmsManager struct {
	config         *Config
	rootKeyGen     RootKeyGenerator
	dataKeyManager DataKeyManager
	cryptoService  CryptoService
	storage        KeyStorage
	log            *log.Helper
	logger         log.Logger // 原始logger
	
	// 状态管理
	mu          sync.RWMutex
	initialized bool
	shutdown    bool
	
	// 自动轮换
	rotationTicker *time.Ticker
	rotationDone   chan struct{}
}

// NewKMSManager 创建KMS管理器
func NewKMSManager(db *sql.DB, logger log.Logger) KMSManager {
	return &kmsManager{
		log:          log.NewHelper(logger),
		storage:      NewDatabaseKeyStorage(db, logger),
		rotationDone: make(chan struct{}),
		logger:       logger, // 保存原始logger
	}
}

// Initialize 初始化KMS系统
func (m *kmsManager) Initialize(ctx context.Context, config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.initialized {
		return ErrKMSAlreadyInit
	}
	
	if m.shutdown {
		return ErrSystemShutdown
	}
	
	// 验证配置
	if err := config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	
	m.config = config
	
	// 初始化根密钥生成器
	m.rootKeyGen = NewRootKeyGenerator(config)
	
	// 生成根密钥
	rootKey, err := m.rootKeyGen.GenerateRootKey()
	if err != nil {
		return fmt.Errorf("failed to generate root key: %w", err)
	}
	
	// 初始化数据密钥管理器
	m.dataKeyManager = NewDataKeyManager(rootKey, m.storage, config)
	
	// 初始化加解密服务
	m.cryptoService = NewCryptoService(m.dataKeyManager, m.logger)
	
	// 检查是否存在活跃的数据密钥，如果没有则生成一个
	if err := m.ensureActiveDataKey(ctx); err != nil {
		return fmt.Errorf("failed to ensure active data key: %w", err)
	}
	
	// 启动自动轮换
	if config.RotateInterval > 0 {
		m.startAutoRotation()
	}
	
	m.initialized = true
	m.log.Info("KMS system initialized successfully")
	return nil
}

// GetActiveDataKey 获取当前活跃的数据密钥
func (m *kmsManager) GetActiveDataKey(ctx context.Context) (*DataKey, error) {
	if err := m.checkInitialized(); err != nil {
		return nil, err
	}
	
	return m.dataKeyManager.GetActiveDataKey(ctx)
}

// GetDataKeyByVersion 根据版本获取数据密钥
func (m *kmsManager) GetDataKeyByVersion(ctx context.Context, version string) (*DataKey, error) {
	if err := m.checkInitialized(); err != nil {
		return nil, err
	}
	
	return m.dataKeyManager.GetDataKeyByVersion(ctx, version)
}

// RotateDataKey 轮换数据密钥
func (m *kmsManager) RotateDataKey(ctx context.Context) (*DataKey, error) {
	if err := m.checkInitialized(); err != nil {
		return nil, err
	}
	
	m.log.Info("Starting data key rotation")
	
	newKey, err := m.dataKeyManager.RotateDataKey(ctx)
	if err != nil {
		m.log.Errorf("Data key rotation failed: %v", err)
		return nil, err
	}
	
	// 清除加解密服务的缓存，强制使用新密钥
	if cryptoSvc, ok := m.cryptoService.(*cryptoService); ok {
		cryptoSvc.ClearCache()
	}
	
	m.log.Infof("Data key rotation completed: %s", newKey.Version)
	return newKey, nil
}

// GetCryptoService 获取加解密服务
func (m *kmsManager) GetCryptoService() CryptoService {
	return m.cryptoService
}

// Close 关闭KMS系统
func (m *kmsManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.shutdown {
		return nil
	}
	
	m.shutdown = true
	
	// 停止自动轮换
	if m.rotationTicker != nil {
		m.rotationTicker.Stop()
		close(m.rotationDone)
	}
	
	// 清除敏感数据
	if cryptoSvc, ok := m.cryptoService.(*cryptoService); ok {
		cryptoSvc.ClearCache()
	}
	
	m.log.Info("KMS system closed")
	return nil
}

// ensureActiveDataKey 确保存在活跃的数据密钥
func (m *kmsManager) ensureActiveDataKey(ctx context.Context) error {
	_, err := m.dataKeyManager.GetActiveDataKey(ctx)
	if err == ErrNoActiveKey {
		// 没有活跃密钥，生成一个新的
		m.log.Info("No active data key found, generating new one")
		_, err = m.dataKeyManager.GenerateDataKey(ctx)
		if err != nil {
			return fmt.Errorf("failed to generate initial data key: %w", err)
		}
		m.log.Info("Initial data key generated successfully")
		return nil
	}
	
	return err
}

// startAutoRotation 启动自动轮换
func (m *kmsManager) startAutoRotation() {
	m.rotationTicker = time.NewTicker(m.config.RotateInterval)
	
	go func() {
		for {
			select {
			case <-m.rotationTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				if _, err := m.RotateDataKey(ctx); err != nil {
					m.log.Errorf("Auto rotation failed: %v", err)
				}
				cancel()
				
			case <-m.rotationDone:
				m.log.Info("Auto rotation stopped")
				return
			}
		}
	}()
	
	m.log.Infof("Auto rotation started with interval: %v", m.config.RotateInterval)
}

// checkInitialized 检查是否已初始化
func (m *kmsManager) checkInitialized() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.shutdown {
		return ErrSystemShutdown
	}
	
	if !m.initialized {
		return ErrKMSNotInitialized
	}
	
	return nil
}

// GetStatus 获取KMS系统状态
func (m *kmsManager) GetStatus(ctx context.Context) (*KMSStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	status := &KMSStatus{
		Initialized: m.initialized,
		Shutdown:    m.shutdown,
	}
	
	if m.initialized {
		status.Algorithm = m.config.Algorithm
		status.RotateInterval = m.config.RotateInterval
		
		// 获取活跃密钥信息
		activeKey, err := m.dataKeyManager.GetActiveDataKey(ctx)
		if err == nil {
			status.ActiveKeyVersion = activeKey.Version
			status.ActiveKeyExpiry = activeKey.ExpiresAt
		}
		
		// 获取密钥统计信息
		if storage, ok := m.storage.(*databaseKeyStorage); ok {
			stats, err := storage.GetKeyStatistics(ctx)
			if err == nil {
				status.KeyStatistics = stats
			}
		}
	}
	
	return status, nil
}

// KMSStatus KMS系统状态
type KMSStatus struct {
	Initialized      bool              `json:"initialized"`
	Shutdown         bool              `json:"shutdown"`
	Algorithm        string            `json:"algorithm,omitempty"`
	RotateInterval   time.Duration     `json:"rotate_interval,omitempty"`
	ActiveKeyVersion string            `json:"active_key_version,omitempty"`
	ActiveKeyExpiry  time.Time         `json:"active_key_expiry,omitempty"`
	KeyStatistics    map[string]int64  `json:"key_statistics,omitempty"`
}

// PerformMaintenance 执行维护操作
func (m *kmsManager) PerformMaintenance(ctx context.Context) error {
	if err := m.checkInitialized(); err != nil {
		return err
	}
	
	m.log.Info("Starting KMS maintenance")
	
	// 清理过期密钥
	if storage, ok := m.storage.(*databaseKeyStorage); ok {
		if err := storage.CleanupExpiredKeys(ctx); err != nil {
			m.log.Errorf("Failed to cleanup expired keys: %v", err)
			return err
		}
	}
	
	// 清除缓存
	if cryptoSvc, ok := m.cryptoService.(*cryptoService); ok {
		cryptoSvc.ClearCache()
	}
	
	m.log.Info("KMS maintenance completed")
	return nil
}