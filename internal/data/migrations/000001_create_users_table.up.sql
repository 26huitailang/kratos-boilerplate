CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    email_encrypted VARCHAR(255),     -- 改为 VARCHAR 类型
    email_hash VARCHAR(64),           -- 用于检索
    phone_encrypted VARCHAR(255),     -- 改为 VARCHAR 类型
    phone_hash VARCHAR(64),           -- 用于检索
    name_encrypted VARCHAR(255),      -- 改为 VARCHAR 类型
    name_hash VARCHAR(64),            -- 用于检索
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
); 