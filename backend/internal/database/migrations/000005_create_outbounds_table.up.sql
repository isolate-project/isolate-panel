-- Create outbounds table
CREATE TABLE IF NOT EXISTS outbounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    protocol VARCHAR(50) NOT NULL,
    core_id INTEGER NOT NULL,
    
    -- Configuration (JSON)
    config_json TEXT NOT NULL,
    
    -- Priority for routing
    priority INTEGER DEFAULT 0,
    
    -- Status
    is_enabled BOOLEAN DEFAULT 1,
    
    -- Metadata
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (core_id) REFERENCES cores(id) ON DELETE CASCADE
);

CREATE INDEX idx_outbounds_protocol ON outbounds(protocol);
CREATE INDEX idx_outbounds_core_id ON outbounds(core_id);
