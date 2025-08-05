package kms

import (
	"context"
	"sync"
	"time"

	"kratos-boilerplate/internal/biz"
)

// MockKMSRepo 模拟KMS数据仓库，用于测试
type MockKMSRepo struct {
	mu       sync.RWMutex
	dataKeys map[string]*biz.DataKey
	activeID string
}

// NewMockKMSRepo 创建模拟KMS数据仓库
func NewMockKMSRepo() *MockKMSRepo {
	return &MockKMSRepo{
		dataKeys: make(map[string]*biz.DataKey),
	}
}

// SaveDataKey 保存数据密钥
func (m *MockKMSRepo) SaveDataKey(ctx context.Context, dataKey *biz.DataKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 复制数据密钥以避免外部修改
	copiedKey := *dataKey
	m.dataKeys[dataKey.ID] = &copiedKey
	
	// 如果是活跃密钥，更新活跃ID
	if dataKey.IsActive {
		// 先将其他密钥设为非活跃
		for _, key := range m.dataKeys {
			key.IsActive = false
		}
		m.activeID = dataKey.ID
		m.dataKeys[dataKey.ID].IsActive = true
	}
	
	return nil
}

// GetDataKey 根据ID获取数据密钥
func (m *MockKMSRepo) GetDataKey(ctx context.Context, keyID string) (*biz.DataKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	key, exists := m.dataKeys[keyID]
	if !exists {
		return nil, biz.ErrKeyNotFound
	}
	
	// 返回副本以避免外部修改
	copiedKey := *key
	return &copiedKey, nil
}

// GetActiveDataKey 获取活跃的数据密钥
func (m *MockKMSRepo) GetActiveDataKey(ctx context.Context) (*biz.DataKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.activeID == "" {
		return nil, biz.ErrNoActiveKey
	}
	
	key, exists := m.dataKeys[m.activeID]
	if !exists {
		return nil, biz.ErrNoActiveKey
	}
	
	// 检查是否过期
	if time.Now().After(key.ExpiresAt) {
		return nil, biz.ErrNoActiveKey
	}
	
	// 返回副本
	copiedKey := *key
	return &copiedKey, nil
}

// GetDataKeyByVersion 根据版本获取数据密钥
func (m *MockKMSRepo) GetDataKeyByVersion(ctx context.Context, version string) (*biz.DataKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, key := range m.dataKeys {
		if key.Version == version {
			// 返回副本
			copiedKey := *key
			return &copiedKey, nil
		}
	}
	
	return nil, biz.ErrKeyNotFound
}

// UpdateDataKey 更新数据密钥
func (m *MockKMSRepo) UpdateDataKey(ctx context.Context, dataKey *biz.DataKey) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	_, exists := m.dataKeys[dataKey.ID]
	if !exists {
		return biz.ErrKeyNotFound
	}
	
	// 更新数据密钥
	copiedKey := *dataKey
	m.dataKeys[dataKey.ID] = &copiedKey
	
	// 如果是活跃密钥，更新活跃ID
	if dataKey.IsActive {
		// 先将其他密钥设为非活跃
		for id, key := range m.dataKeys {
			if id != dataKey.ID {
				key.IsActive = false
			}
		}
		m.activeID = dataKey.ID
	} else if m.activeID == dataKey.ID {
		// 如果当前活跃密钥被设为非活跃，清除活跃ID
		m.activeID = ""
	}
	
	return nil
}

// UpdateKeyStatus 更新密钥状态
func (m *MockKMSRepo) UpdateKeyStatus(ctx context.Context, version string, isActive bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, key := range m.dataKeys {
		if key.Version == version {
			key.IsActive = isActive
			if isActive {
				// 先将其他密钥设为非活跃
				for _, otherKey := range m.dataKeys {
					if otherKey.Version != version {
						otherKey.IsActive = false
					}
				}
				m.activeID = key.ID
			} else if m.activeID == key.ID {
				m.activeID = ""
			}
			return nil
		}
	}
	
	return biz.ErrKeyNotFound
}

// CleanupExpiredKeys 清理过期的数据密钥
func (m *MockKMSRepo) CleanupExpiredKeys(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	for id, key := range m.dataKeys {
		if now.After(key.ExpiresAt) && !key.IsActive {
			delete(m.dataKeys, id)
		}
	}
	
	return nil
}

// GetKeyStatistics 获取密钥统计信息
func (m *MockKMSRepo) GetKeyStatistics(ctx context.Context) (*biz.KeyStatistics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := &biz.KeyStatistics{
		TotalKeys:   int64(len(m.dataKeys)),
		ActiveKeys:  0,
		ExpiredKeys: 0,
	}
	
	now := time.Now()
	for _, key := range m.dataKeys {
		if key.IsActive {
			stats.ActiveKeys++
		}
		if now.After(key.ExpiresAt) {
			stats.ExpiredKeys++
		}
	}
	
	return stats, nil
}

// DeactivateDataKey 停用数据密钥
func (m *MockKMSRepo) DeactivateDataKey(ctx context.Context, keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key, exists := m.dataKeys[keyID]
	if !exists {
		return biz.ErrKeyNotFound
	}
	
	key.IsActive = false
	if m.activeID == keyID {
		m.activeID = ""
	}
	
	return nil
}

// ListDataKeys 列出数据密钥
func (m *MockKMSRepo) ListDataKeys(ctx context.Context) ([]*biz.DataKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// 将所有密钥转换为切片
	var keys []*biz.DataKey
	for _, key := range m.dataKeys {
		copiedKey := *key
		keys = append(keys, &copiedKey)
	}
	
	return keys, nil
}

// Reset 重置模拟仓库（仅用于测试）
func (m *MockKMSRepo) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.dataKeys = make(map[string]*biz.DataKey)
	m.activeID = ""
}

// SetActiveKey 设置活跃密钥（仅用于测试）
func (m *MockKMSRepo) SetActiveKey(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key, exists := m.dataKeys[keyID]
	if !exists {
		return biz.ErrKeyNotFound
	}
	
	// 先将所有密钥设为非活跃
	for _, k := range m.dataKeys {
		k.IsActive = false
	}
	
	// 设置指定密钥为活跃
	key.IsActive = true
	m.activeID = keyID
	
	return nil
}

// GetKeyCount 获取密钥数量（仅用于测试）
func (m *MockKMSRepo) GetKeyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.dataKeys)
}