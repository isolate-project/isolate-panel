-- Create active_connections table
CREATE TABLE IF NOT EXISTS active_connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    
    -- Connection info
    source_ip VARCHAR(50),
    source_port INTEGER,
    destination_ip VARCHAR(50),
    destination_port INTEGER,
    protocol VARCHAR(50),
    
    -- Statistics
    upload_bytes BIGINT DEFAULT 0,
    download_bytes BIGINT DEFAULT 0,
    
    -- Timestamps
    connected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_activity_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (inbound_id) REFERENCES inbounds(id) ON DELETE CASCADE
);

CREATE INDEX idx_active_connections_user_id ON active_connections(user_id);
CREATE INDEX idx_active_connections_inbound_id ON active_connections(inbound_id);
