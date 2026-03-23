-- Create cores table
CREATE TABLE IF NOT EXISTS cores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(20) NOT NULL,
    version VARCHAR(20) NOT NULL,
    is_enabled BOOLEAN DEFAULT 1,
    is_running BOOLEAN DEFAULT 0,
    pid INTEGER,
    config_path VARCHAR(255),
    log_path VARCHAR(255),
    
    -- Statistics
    uptime_seconds INTEGER DEFAULT 0,
    restart_count INTEGER DEFAULT 0,
    last_error TEXT,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_cores_name ON cores(name);
