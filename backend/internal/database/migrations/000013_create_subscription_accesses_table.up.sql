-- Create subscription_accesses table
CREATE TABLE IF NOT EXISTS subscription_accesses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    user_agent VARCHAR(512),
    country VARCHAR(2),
    format VARCHAR(20),
    is_suspicious BOOLEAN DEFAULT 0,
    response_time_ms INTEGER DEFAULT 0,
    accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_subscription_accesses_user_id ON subscription_accesses(user_id);
CREATE INDEX idx_subscription_accesses_accessed_at ON subscription_accesses(accessed_at);
CREATE INDEX idx_subscription_accesses_is_suspicious ON subscription_accesses(is_suspicious);
