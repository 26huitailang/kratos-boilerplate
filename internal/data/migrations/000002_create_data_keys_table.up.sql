-- 创建数据密钥表
CREATE TABLE IF NOT EXISTS data_keys (
    id VARCHAR(255) PRIMARY KEY,
    version VARCHAR(255) NOT NULL UNIQUE,
    algorithm VARCHAR(50) NOT NULL,
    key_data BLOB NOT NULL,
    encrypted_key BLOB NOT NULL,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT false,
    
    INDEX idx_data_keys_version (version),
    INDEX idx_data_keys_active (is_active),
    INDEX idx_data_keys_expires (expires_at),
    INDEX idx_data_keys_created (created_at)
);

-- 创建唯一索引确保只有一个活跃密钥
CREATE UNIQUE INDEX idx_data_keys_active_unique 
ON data_keys (is_active) 
WHERE is_active = true;