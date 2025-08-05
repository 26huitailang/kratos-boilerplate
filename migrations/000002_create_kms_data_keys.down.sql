-- 删除KMS数据密钥相关对象

-- 删除视图
DROP VIEW IF EXISTS kms_key_stats;

-- 删除触发器
DROP TRIGGER IF EXISTS trigger_update_kms_data_keys_updated_at ON kms_data_keys;

-- 删除触发器函数
DROP FUNCTION IF EXISTS update_kms_data_keys_updated_at();

-- 删除索引（表删除时会自动删除，但为了明确性还是列出）
DROP INDEX IF EXISTS idx_kms_data_keys_version_status;
DROP INDEX IF EXISTS idx_kms_data_keys_key_hash;
DROP INDEX IF EXISTS idx_kms_data_keys_expires_at;
DROP INDEX IF EXISTS idx_kms_data_keys_created_at;
DROP INDEX IF EXISTS idx_kms_data_keys_status;
DROP INDEX IF EXISTS idx_kms_data_keys_version;

-- 删除表
DROP TABLE IF EXISTS kms_data_keys;