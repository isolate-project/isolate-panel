-- Create inbounds table
CREATE TABLE IF NOT EXISTS inbounds (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(100) NOT NULL,
    protocol VARCHAR(50) NOT NULL,
    core_id INTEGER NOT NULL,
    
    -- Network settings
    listen_address VARCHAR(50) DEFAULT '0.0.0.0',
    port INTEGER NOT NULL,
    
    -- Configuration (JSON)
    config_json TEXT NOT NULL,
    
    -- TLS/REALITY
    tls_enabled BOOLEAN DEFAULT 0,
    tls_cert_id INTEGER,
    reality_enabled BOOLEAN DEFAULT 0,
    reality_config_json TEXT,
    
    -- Status
    is_enabled BOOLEAN DEFAULT 1,
    
    -- Metadata
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (core_id) REFERENCES cores(id) ON DELETE CASCADE,
    FOREIGN KEY (tls_cert_id) REFERENCES certificates(id) ON DELETE SET NULL
);

CREATE INDEX idx_inbounds_protocol ON inbounds(protocol);
CREATE INDEX idx_inbounds_port ON inbounds(port);
CREATE INDEX idx_inbounds_core_id ON inbounds(core_id);
