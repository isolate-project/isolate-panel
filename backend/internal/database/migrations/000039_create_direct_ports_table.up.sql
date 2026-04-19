-- Create direct_ports table
CREATE TABLE IF NOT EXISTS direct_ports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    inbound_id INTEGER NOT NULL,
    listen_port INTEGER NOT NULL,
    listen_address VARCHAR(50) DEFAULT '0.0.0.0',
    core_type VARCHAR(20) NOT NULL,
    backend_port INTEGER NOT NULL,
    is_active BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (inbound_id) REFERENCES inbounds(id) ON DELETE CASCADE,
    UNIQUE(inbound_id)
);

CREATE INDEX idx_direct_ports_listen_port ON direct_ports(listen_port);
