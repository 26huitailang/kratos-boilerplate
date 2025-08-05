-- 创建KMS数据密钥表
CREATE TABLE IF NOT EXISTS kms_data_keys (
    id VARCHAR(36) PRIMARY KEY,
    version INTEGER NOT NULL,
    algorithm VARCHAR(50) NOT NULL DEFAULT 'AES-256-GCM',
    encrypted_key BYTEA NOT NULL,
    key_hash VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    
    -- 索引
    CONSTRAINT kms_data_keys_status_check CHECK (status IN ('active', 'inactive', 'expired', 'revoked')),
    CONSTRAINT kms_data_keys_algorithm_check CHECK (algorithm IN ('AES-256-GCM', 'AES-256-CBC', 'ChaCha20-Poly1305', 'SM4-GCM'))
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_kms_data_keys_version ON kms_data_keys(version);
CREATE INDEX IF NOT EXISTS idx_kms_data_keys_status ON kms_data_keys(status);
CREATE INDEX IF NOT EXISTS idx_kms_data_keys_created_at ON kms_data_keys(created_at);
CREATE INDEX IF NOT EXISTS idx_kms_data_keys_expires_at ON kms_data_keys(expires_at);
CREATE INDEX IF NOT EXISTS idx_kms_data_keys_key_hash ON kms_data_keys(key_hash);

-- 创建唯一约束
CREATE UNIQUE INDEX IF NOT EXISTS idx_kms_data_keys_version_status ON kms_data_keys(version, status) 
WHERE status = 'active';

-- 添加注释
COMMENT ON TABLE kms_data_keys IS 'KMS数据密钥存储表';
COMMENT ON COLUMN kms_data_keys.id IS '密钥唯一标识符';
COMMENT ON COLUMN kms_data_keys.version IS '密钥版本号';
COMMENT ON COLUMN kms_data_keys.algorithm IS '加密算法';
COMMENT ON COLUMN kms_data_keys.encrypted_key IS '使用根密钥加密后的数据密钥';
COMMENT ON COLUMN kms_data_keys.key_hash IS '密钥哈希值，用于验证';
COMMENT ON COLUMN kms_data_keys.status IS '密钥状态：active-活跃, inactive-非活跃, expired-过期, revoked-撤销';
COMMENT ON COLUMN kms_data_keys.created_at IS '创建时间';
COMMENT ON COLUMN kms_data_keys.updated_at IS '更新时间';
COMMENT ON COLUMN kms_data_keys.expires_at IS '过期时间';
COMMENT ON COLUMN kms_data_keys.metadata IS '密钥元数据，JSON格式';

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_kms_data_keys_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 创建触发器
CREATE TRIGGER trigger_update_kms_data_keys_updated_at
    BEFORE UPDATE ON kms_data_keys
    FOR EACH ROW
    EXECUTE FUNCTION update_kms_data_keys_updated_at();

-- 创建密钥统计视图
CREATE OR REPLACE VIEW kms_key_stats AS
SELECT 
    status,
    algorithm,
    COUNT(*) as key_count,
    MIN(created_at) as earliest_created,
    MAX(created_at) as latest_created,
    COUNT(CASE WHEN expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP THEN 1 END) as expired_count
FROM kms_data_keys
GROUP BY status, algorithm;

COMMENT ON VIEW kms_key_stats IS 'KMS密钥统计视图';