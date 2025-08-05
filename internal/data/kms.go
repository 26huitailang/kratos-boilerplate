package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"kratos-boilerplate/internal/biz"
)

// kmsRepo KMS数据仓库实现
type kmsRepo struct {
	data *Data
	log  *log.Helper
}

// NewKMSRepo 创建KMS数据仓库
func NewKMSRepo(data *Data, logger log.Logger) biz.KMSRepo {
	return &kmsRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// SaveDataKey 保存数据密钥
func (r *kmsRepo) SaveDataKey(ctx context.Context, dataKey *biz.DataKey) error {
	query := `
		INSERT INTO data_keys (id, version, algorithm, key_data, encrypted_key, created_at, expires_at, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := r.data.db.ExecContext(ctx, query,
		dataKey.ID,
		dataKey.Version,
		dataKey.Algorithm,
		dataKey.Key,
		dataKey.EncryptedKey,
		dataKey.CreatedAt,
		dataKey.ExpiresAt,
		dataKey.IsActive,
	)
	
	if err != nil {
		r.log.Errorf("Failed to save data key: %v", err)
		return fmt.Errorf("failed to save data key: %w", err)
	}
	
	r.log.Infof("Data key saved successfully: %s", dataKey.ID)
	return nil
}

// GetDataKey 根据ID获取数据密钥
func (r *kmsRepo) GetDataKey(ctx context.Context, keyID string) (*biz.DataKey, error) {
	query := `
		SELECT id, version, algorithm, key_data, encrypted_key, created_at, expires_at, is_active
		FROM data_keys
		WHERE id = ?
	`
	
	var dataKey biz.DataKey
	err := r.data.db.QueryRowContext(ctx, query, keyID).Scan(
		&dataKey.ID,
		&dataKey.Version,
		&dataKey.Algorithm,
		&dataKey.Key,
		&dataKey.EncryptedKey,
		&dataKey.CreatedAt,
		&dataKey.ExpiresAt,
		&dataKey.IsActive,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, biz.ErrKeyNotFound
		}
		r.log.Errorf("Failed to get data key: %v", err)
		return nil, fmt.Errorf("failed to get data key: %w", err)
	}
	
	return &dataKey, nil
}

// GetActiveDataKey 获取活跃的数据密钥
func (r *kmsRepo) GetActiveDataKey(ctx context.Context) (*biz.DataKey, error) {
	query := `
		SELECT id, version, algorithm, key_data, encrypted_key, created_at, expires_at, is_active
		FROM data_keys
		WHERE is_active = true AND expires_at > ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	
	var dataKey biz.DataKey
	err := r.data.db.QueryRowContext(ctx, query, time.Now()).Scan(
		&dataKey.ID,
		&dataKey.Version,
		&dataKey.Algorithm,
		&dataKey.Key,
		&dataKey.EncryptedKey,
		&dataKey.CreatedAt,
		&dataKey.ExpiresAt,
		&dataKey.IsActive,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, biz.ErrNoActiveKey
		}
		r.log.Errorf("Failed to get active data key: %v", err)
		return nil, fmt.Errorf("failed to get active data key: %w", err)
	}
	
	return &dataKey, nil
}

// GetDataKeyByVersion 根据版本获取数据密钥
func (r *kmsRepo) GetDataKeyByVersion(ctx context.Context, version string) (*biz.DataKey, error) {
	query := `
		SELECT id, version, algorithm, key_data, encrypted_key, created_at, expires_at, is_active
		FROM data_keys
		WHERE version = ?
	`
	
	var dataKey biz.DataKey
	err := r.data.db.QueryRowContext(ctx, query, version).Scan(
		&dataKey.ID,
		&dataKey.Version,
		&dataKey.Algorithm,
		&dataKey.Key,
		&dataKey.EncryptedKey,
		&dataKey.CreatedAt,
		&dataKey.ExpiresAt,
		&dataKey.IsActive,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, biz.ErrKeyNotFound
		}
		r.log.Errorf("Failed to get data key by version: %v", err)
		return nil, fmt.Errorf("failed to get data key by version: %w", err)
	}
	
	return &dataKey, nil
}

// UpdateDataKey 更新数据密钥
func (r *kmsRepo) UpdateDataKey(ctx context.Context, dataKey *biz.DataKey) error {
	query := `
		UPDATE data_keys
		SET algorithm = ?, key_data = ?, encrypted_key = ?, expires_at = ?, is_active = ?
		WHERE id = ?
	`
	
	result, err := r.data.db.ExecContext(ctx, query,
		dataKey.Algorithm,
		dataKey.Key,
		dataKey.EncryptedKey,
		dataKey.ExpiresAt,
		dataKey.IsActive,
		dataKey.ID,
	)
	
	if err != nil {
		r.log.Errorf("Failed to update data key: %v", err)
		return fmt.Errorf("failed to update data key: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return biz.ErrKeyNotFound
	}
	
	r.log.Infof("Data key updated successfully: %s", dataKey.ID)
	return nil
}

// CleanupExpiredKeys 清理过期的数据密钥
func (r *kmsRepo) CleanupExpiredKeys(ctx context.Context) error {
	query := `
		DELETE FROM data_keys
		WHERE expires_at < ? AND is_active = false
	`
	
	result, err := r.data.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		r.log.Errorf("Failed to cleanup expired keys: %v", err)
		return fmt.Errorf("failed to cleanup expired keys: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	r.log.Infof("Cleaned up %d expired keys", rowsAffected)
	return nil
}

// GetKeyStatistics 获取密钥统计信息
func (r *kmsRepo) GetKeyStatistics(ctx context.Context) (*biz.KeyStatistics, error) {
	query := `
		SELECT
			COUNT(*) as total_keys,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active_keys,
			COUNT(CASE WHEN expires_at < ? THEN 1 END) as expired_keys
		FROM data_keys
	`
	
	var stats biz.KeyStatistics
	err := r.data.db.QueryRowContext(ctx, query, time.Now()).Scan(
		&stats.TotalKeys,
		&stats.ActiveKeys,
		&stats.ExpiredKeys,
	)
	
	if err != nil {
		r.log.Errorf("Failed to get key statistics: %v", err)
		return nil, fmt.Errorf("failed to get key statistics: %w", err)
	}
	
	return &stats, nil
}

// DeactivateDataKey 停用数据密钥
func (r *kmsRepo) DeactivateDataKey(ctx context.Context, keyID string) error {
	query := `
		UPDATE data_keys
		SET is_active = false
		WHERE id = ?
	`
	
	result, err := r.data.db.ExecContext(ctx, query, keyID)
	if err != nil {
		r.log.Errorf("Failed to deactivate data key: %v", err)
		return fmt.Errorf("failed to deactivate data key: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return biz.ErrKeyNotFound
	}
	
	r.log.Infof("Data key deactivated successfully: %s", keyID)
	return nil
}

// ListDataKeys 列出数据密钥
func (r *kmsRepo) ListDataKeys(ctx context.Context) ([]*biz.DataKey, error) {
	return r.listDataKeysWithPagination(ctx, 100, 0)
}

// listDataKeysWithPagination 列出数据密钥（支持分页）
func (r *kmsRepo) listDataKeysWithPagination(ctx context.Context, limit, offset int) ([]*biz.DataKey, error) {
	query := `
		SELECT id, version, algorithm, key_data, encrypted_key, created_at, expires_at, is_active
		FROM data_keys
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.data.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		r.log.Errorf("Failed to list data keys: %v", err)
		return nil, fmt.Errorf("failed to list data keys: %w", err)
	}
	defer rows.Close()
	
	var dataKeys []*biz.DataKey
	for rows.Next() {
		var dataKey biz.DataKey
		err := rows.Scan(
			&dataKey.ID,
			&dataKey.Version,
			&dataKey.Algorithm,
			&dataKey.Key,
			&dataKey.EncryptedKey,
			&dataKey.CreatedAt,
			&dataKey.ExpiresAt,
			&dataKey.IsActive,
		)
		if err != nil {
			r.log.Errorf("Failed to scan data key: %v", err)
			return nil, fmt.Errorf("failed to scan data key: %w", err)
		}
		dataKeys = append(dataKeys, &dataKey)
	}
	
	if err = rows.Err(); err != nil {
		r.log.Errorf("Error iterating data keys: %v", err)
		return nil, fmt.Errorf("error iterating data keys: %w", err)
	}
	
	return dataKeys, nil
}

// UpdateKeyStatus 更新密钥状态
func (r *kmsRepo) UpdateKeyStatus(ctx context.Context, version string, isActive bool) error {
	query := `
		UPDATE data_keys
		SET is_active = ?
		WHERE version = ?
	`
	
	result, err := r.data.db.ExecContext(ctx, query, isActive, version)
	if err != nil {
		r.log.Errorf("Failed to update key status: %v", err)
		return fmt.Errorf("failed to update key status: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return biz.ErrKeyNotFound
	}
	
	r.log.Infof("Key status updated successfully: %s", version)
	return nil
}