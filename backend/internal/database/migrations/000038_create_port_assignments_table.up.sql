-- Create port_assignments table
CREATE TABLE IF NOT EXISTS port_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inbound_id INTEGER NOT NULL,
    user_listen_port INTEGER NOT NULL,
    user_listen_address VARCHAR(50) DEFAULT '0.0.0.0',
    backend_port INTEGER NOT NULL,
    core_type VARCHAR(20) NOT NULL,
    use_haproxy BOOLEAN DEFAULT 1,
    sni_match VARCHAR(255),
    path_match VARCHAR(255),
    send_proxy_protocol BOOLEAN DEFAULT 0,
    proxy_protocol_version INTEGER,
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (inbound_id) REFERENCES inbounds(id) ON DELETE CASCADE,
    UNIQUE(inbound_id)
);

CREATE INDEX idx_port_assignments_user_port ON port_assignments(user_listen_port);
CREATE INDEX idx_port_assignments_backend ON port_assignments(backend_port);
CREATE INDEX idx_port_assignments_core ON port_assignments(core_type);
