-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100),
    
    -- Universal Credentials
    uuid VARCHAR(36) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    token VARCHAR(64) UNIQUE,
    subscription_token VARCHAR(64) UNIQUE NOT NULL,
    
    -- Quotas
    traffic_limit_bytes BIGINT DEFAULT NULL,
    traffic_used_bytes BIGINT DEFAULT 0,
    expiry_date DATETIME DEFAULT NULL,
    
    -- Status
    is_active BOOLEAN DEFAULT 1,
    is_online BOOLEAN DEFAULT 0,
    
    -- Metadata
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_connected_at DATETIME,
    
    -- Relations
    created_by_admin_id INTEGER,
    FOREIGN KEY (created_by_admin_id) REFERENCES admins(id) ON DELETE SET NULL
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_uuid ON users(uuid);
CREATE INDEX idx_users_token ON users(token);
CREATE INDEX idx_users_subscription_token ON users(subscription_token);
CREATE INDEX idx_users_is_active ON users(is_active);
