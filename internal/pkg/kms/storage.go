package kms

import (
	"context"
	"database/sql"
	"fmt"
	
	"github.com/go-kratos/kratos/v2/log"
)

// databaseKeyStorage 数据库密钥存储实现
type databaseKeyStorage struct {
	db  *sql.DB
	log *log.Helper
}

// NewDatabaseKeyStorage 创建数据库密钥存储
func NewDatabaseKeyStorage(db *sql.DB, logger log.Logger) KeyStorage {
	return &databaseKeyStorage{
		db:  db,
		log: log.NewHelper(logger),
	}
}

// SaveDataKey 保存数据密钥
func (s *databaseKeyStorage) SaveDataKey(ctx context.Context, key *DataKey) error {
	query := `
		INSERT INTO kms_data_keys (id, version, algorithm, encrypted_key, created_at, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		key.ID, key.Version, key.Algorithm, key.EncryptedKey,
		key.CreatedAt, key.ExpiresAt, key.IsActive)
	
	if err != nil {
		s.log.Errorf("Failed to save data key: %v", err)
		return fmt.Errorf("%w: %v", ErrStorageOperation, err)
	}
	
	s.log.Infof("Data key saved successfully: %s", key.Version)
	return nil
}

// GetActiveDataKey 获取活跃的数据密钥
func (s *databaseKeyStorage) GetActiveDataKey(ctx context.Context) (*DataKey, error) {
	query := `
		SELECT id, version, algorithm, encrypted_key, created_at, expires_at, is_active
		FROM kms_data_keys
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT 1
	`
	
	var key DataKey
	err := s.db.QueryRowContext(ctx, query).Scan(
		&key.ID, &key.Version, &key.Algorithm, &key.EncryptedKey,
		&key.CreatedAt, &key.ExpiresAt, &key.IsActive)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNoActiveKey
		}
		s.log.Errorf("Failed to get active data key: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrStorageOperation, err)
	}
	
	return &key, nil
}

// GetDataKeyByVersion 根据版本获取数据密钥
func (s *databaseKeyStorage) GetDataKeyByVersion(ctx context.Context, version string) (*DataKey, error) {
	query := `
		SELECT id, version, algorithm, encrypted_key, created_at, expires_at, is_active
		FROM kms_data_keys
		WHERE version = $1
	`
	
	var key DataKey
	err := s.db.QueryRowContext(ctx, query, version).Scan(
		&key.ID, &key.Version, &key.Algorithm, &key.EncryptedKey,
		&key.CreatedAt, &key.ExpiresAt, &key.IsActive)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: version %s", ErrKeyNotFound, version)
		}
		s.log.Errorf("Failed to get data key by version: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrStorageOperation, err)
	}
	
	return &key, nil
}

// ListDataKeys 列出所有数据密钥
func (s *databaseKeyStorage) ListDataKeys(ctx context.Context) ([]*DataKey, error) {
	query := `
		SELECT id, version, algorithm, encrypted_key, created_at, expires_at, is_active
		FROM kms_data_keys
		ORDER BY created_at DESC
	`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		s.log.Errorf("Failed to list data keys: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrStorageOperation, err)
	}
	defer rows.Close()
	
	var keys []*DataKey
	for rows.Next() {
		var key DataKey
		err := rows.Scan(
			&key.ID, &key.Version, &key.Algorithm, &key.EncryptedKey,
			&key.CreatedAt, &key.ExpiresAt, &key.IsActive)
		if err != nil {
			s.log.Errorf("Failed to scan data key: %v", err)
			return nil, fmt.Errorf("%w: %v", ErrStorageOperation, err)
		}
		keys = append(keys, &key)
	}
	
	if err := rows.Err(); err != nil {
		s.log.Errorf("Row iteration error: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrStorageOperation, err)
	}
	
	return keys, nil
}

// UpdateKeyStatus 更新密钥状态
func (s *databaseKeyStorage) UpdateKeyStatus(ctx context.Context, version string, isActive bool) error {
	query := `UPDATE kms_data_keys SET is_active = $1 WHERE version = $2`
	
	result, err := s.db.ExecContext(ctx, query, isActive, version)
	if err != nil {
		s.log.Errorf("Failed to update key status: %v", err)
		return fmt.Errorf("%w: %v", ErrStorageOperation, err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.log.Errorf("Failed to get rows affected: %v", err)
		return fmt.Errorf("%w: %v", ErrStorageOperation, err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("%w: version %s", ErrKeyNotFound, version)
	}
	
	s.log.Infof("Key status updated: %s -> %t", version, isActive)
	return nil
}

// CleanupExpiredKeys 清理过期密钥（可选的维护操作）
func (s *databaseKeyStorage) CleanupExpiredKeys(ctx context.Context) error {
	query := `DELETE FROM kms_data_keys WHERE expires_at < NOW() AND is_active = false`
	
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		s.log.Errorf("Failed to cleanup expired keys: %v", err)
		return fmt.Errorf("%w: %v", ErrStorageOperation, err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		s.log.Errorf("Failed to get rows affected: %v", err)
		return fmt.Errorf("%w: %v", ErrStorageOperation, err)
	}
	
	s.log.Infof("Cleaned up %d expired keys", rowsAffected)
	return nil
}

// GetKeyStatistics 获取密钥统计信息
func (s *databaseKeyStorage) GetKeyStatistics(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN is_active = true THEN 1 END) as active,
			COUNT(CASE WHEN expires_at < NOW() THEN 1 END) as expired
		FROM kms_data_keys
	`
	
	var total, active, expired int64
	err := s.db.QueryRowContext(ctx, query).Scan(&total, &active, &expired)
	if err != nil {
		s.log.Errorf("Failed to get key statistics: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrStorageOperation, err)
	}
	
	stats := map[string]int64{
		"total":   total,
		"active":  active,
		"expired": expired,
		"inactive": total - active,
	}
	
	return stats, nil
}