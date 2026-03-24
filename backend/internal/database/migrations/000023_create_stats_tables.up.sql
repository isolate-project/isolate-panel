-- Create traffic_stats table
CREATE TABLE IF NOT EXISTS traffic_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    core_id INTEGER NOT NULL,
    upload INTEGER DEFAULT 0,
    download INTEGER DEFAULT 0,
    total INTEGER DEFAULT 0,
    recorded_at DATETIME NOT NULL,
    granularity TEXT DEFAULT 'raw',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (inbound_id) REFERENCES inbounds(id) ON DELETE CASCADE,
    FOREIGN KEY (core_id) REFERENCES cores(id) ON DELETE CASCADE
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_traffic_stats_user_id ON traffic_stats(user_id);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_inbound_id ON traffic_stats(inbound_id);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_core_id ON traffic_stats(core_id);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_recorded_at ON traffic_stats(recorded_at);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_granularity ON traffic_stats(granularity);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_user_recorded ON traffic_stats(user_id, recorded_at);

-- Create active_connections table
CREATE TABLE IF NOT EXISTS active_connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    core_id INTEGER NOT NULL,
    core_name TEXT NOT NULL,
    source_ip TEXT,
    source_port INTEGER,
    destination_ip TEXT,
    destination_port INTEGER,
    started_at DATETIME NOT NULL,
    last_activity DATETIME,
    upload INTEGER DEFAULT 0,
    download INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (inbound_id) REFERENCES inbounds(id) ON DELETE CASCADE,
    FOREIGN KEY (core_id) REFERENCES cores(id) ON DELETE CASCADE
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_active_connections_user_id ON active_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_active_connections_inbound_id ON active_connections(inbound_id);
CREATE INDEX IF NOT EXISTS idx_active_connections_started_at ON active_connections(started_at);
