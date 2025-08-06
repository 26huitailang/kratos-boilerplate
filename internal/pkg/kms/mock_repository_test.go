package kms

import (
	"context"
	"testing"
	"time"

	"kratos-boilerplate/internal/biz"

	"github.com/stretchr/testify/assert"
)

// TestMockKMSRepo_GetDataKey 测试获取数据密钥
func TestMockKMSRepo_GetDataKey(t *testing.T) {
	repo := NewMockKMSRepo()
	ctx := context.Background()
	
	// 生成一个密钥
	key, err := repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	assert.NotNil(t, key)
	
	// 获取密钥
	retrievedKey, err := repo.GetDataKey(ctx, key.ID)
	assert.NoError(t, err)
	assert.Equal(t, key.ID, retrievedKey.ID)
	assert.Equal(t, key.Version, retrievedKey.Version)
	
	// 获取不存在的密钥
	_, err = repo.GetDataKey(ctx, "non-existent")
	assert.Error(t, err)
	assert.Equal(t, biz.ErrKeyNotFound, err)
}

// TestMockKMSRepo_UpdateDataKey 测试更新数据密钥
func TestMockKMSRepo_UpdateDataKey(t *testing.T) {
	repo := NewMockKMSRepo()
	ctx := context.Background()
	
	// 生成一个密钥
	key, err := repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	
	// 更新密钥
	key.IsActive = false
	err = repo.UpdateDataKey(ctx, key)
	assert.NoError(t, err)
	
	// 验证更新
	updatedKey, err := repo.GetDataKey(ctx, key.ID)
	assert.NoError(t, err)
	assert.False(t, updatedKey.IsActive)
	
	// 更新不存在的密钥
	nonExistentKey := &biz.DataKey{
		ID:      "non-existent",
		Version: "v999",
	}
	err = repo.UpdateDataKey(ctx, nonExistentKey)
	assert.Error(t, err)
	assert.Equal(t, biz.ErrKeyNotFound, err)
}

// TestMockKMSRepo_DeactivateDataKey 测试停用数据密钥
func TestMockKMSRepo_DeactivateDataKey(t *testing.T) {
	repo := NewMockKMSRepo()
	ctx := context.Background()
	
	// 生成一个密钥
	key, err := repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	assert.True(t, key.IsActive)
	
	// 停用密钥
	err = repo.DeactivateDataKey(ctx, key.ID)
	assert.NoError(t, err)
	
	// 验证密钥已停用
	deactivatedKey, err := repo.GetDataKey(ctx, key.ID)
	assert.NoError(t, err)
	assert.False(t, deactivatedKey.IsActive)
	
	// 停用不存在的密钥
	err = repo.DeactivateDataKey(ctx, "non-existent")
	assert.Error(t, err)
	assert.Equal(t, biz.ErrKeyNotFound, err)
}

// TestMockKMSRepo_ListDataKeys 测试列出数据密钥
func TestMockKMSRepo_ListDataKeys(t *testing.T) {
	repo := NewMockKMSRepo()
	ctx := context.Background()
	
	// 生成多个密钥
	key1, err := repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	
	key2, err := repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	
	// 列出所有密钥
	keys, err := repo.ListDataKeys(ctx)
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	
	// 验证密钥ID
	keyIDs := make(map[string]bool)
	for _, key := range keys {
		keyIDs[key.ID] = true
	}
	assert.True(t, keyIDs[key1.ID])
	assert.True(t, keyIDs[key2.ID])
}

// TestMockKMSRepo_GetKeyStatistics 测试获取密钥统计信息
func TestMockKMSRepo_GetKeyStatistics(t *testing.T) {
	repo := NewMockKMSRepo()
	ctx := context.Background()
	
	// 初始统计信息
	stats, err := repo.GetKeyStatistics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalKeys)
	assert.Equal(t, int64(0), stats.ActiveKeys)
	assert.Equal(t, int64(0), stats.ExpiredKeys)
	
	// 生成一些密钥
	key1, err := repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	
	key2, err := repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	
	// 设置一个密钥为过期
	key2.ExpiresAt = time.Now().Add(-1 * time.Hour)
	err = repo.UpdateDataKey(ctx, key2)
	assert.NoError(t, err)
	
	// 停用一个密钥
	err = repo.DeactivateDataKey(ctx, key1.ID)
	assert.NoError(t, err)
	
	// 获取更新后的统计信息
	stats, err = repo.GetKeyStatistics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), stats.TotalKeys)
	assert.Equal(t, int64(0), stats.ActiveKeys) // 两个密钥都不活跃了
	assert.Equal(t, int64(1), stats.ExpiredKeys)
}

// TestMockKMSRepo_Reset 测试重置仓库
func TestMockKMSRepo_Reset(t *testing.T) {
	repo := NewMockKMSRepo()
	ctx := context.Background()
	
	// 生成一些密钥
	_, err := repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	
	_, err = repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	
	// 验证密钥存在
	stats, err := repo.GetKeyStatistics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), stats.TotalKeys)
	
	// 重置仓库
	repo.Reset()
	
	// 验证密钥已清空
	stats, err = repo.GetKeyStatistics(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalKeys)
	assert.Equal(t, int64(0), stats.ActiveKeys)
	assert.Equal(t, int64(0), stats.ExpiredKeys)
	
	// 验证活跃密钥也被清空
	_, err = repo.GetActiveDataKey(ctx)
	assert.Error(t, err)
	assert.Equal(t, biz.ErrNoActiveKey, err)
}

// TestMockKMSRepo_SetActiveKey 测试设置活跃密钥
func TestMockKMSRepo_SetActiveKey(t *testing.T) {
	repo := NewMockKMSRepo()
	ctx := context.Background()
	
	// 生成一个密钥
	key, err := repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	
	// 停用密钥
	err = repo.DeactivateDataKey(ctx, key.ID)
	assert.NoError(t, err)
	
	// 验证没有活跃密钥
	_, err = repo.GetActiveDataKey(ctx)
	assert.Error(t, err)
	assert.Equal(t, biz.ErrNoActiveKey, err)
	
	// 设置活跃密钥
	err = repo.SetActiveKey(key.ID)
	assert.NoError(t, err)
	
	// 验证密钥已激活
	activeKey, err := repo.GetActiveDataKey(ctx)
	assert.NoError(t, err)
	assert.Equal(t, key.ID, activeKey.ID)
	assert.True(t, activeKey.IsActive)
	
	// 设置不存在的密钥为活跃
	err = repo.SetActiveKey("non-existent")
	assert.Error(t, err)
	assert.Equal(t, biz.ErrKeyNotFound, err)
}

// TestMockKMSRepo_GetKeyCount 测试获取密钥数量
func TestMockKMSRepo_GetKeyCount(t *testing.T) {
	repo := NewMockKMSRepo()
	ctx := context.Background()
	
	// 初始数量应该为0
	count := repo.GetKeyCount()
	assert.Equal(t, 0, count)
	
	// 生成一些密钥
	_, err := repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)

	_, err = repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)

	_, err = repo.GenerateDataKey(ctx, "AES-256-GCM")
	assert.NoError(t, err)
	
	// 验证数量
	count = repo.GetKeyCount()
	assert.Equal(t, 3, count)
}