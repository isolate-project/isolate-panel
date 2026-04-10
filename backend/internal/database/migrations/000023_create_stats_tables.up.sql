-- Recreate traffic_stats table with updated schema (adding core_id, new column names)
-- SQLite does not support ALTER TABLE ADD COLUMN with FOREIGN KEY, so we recreate.
-- Preserve existing data during migration.

CREATE TABLE IF NOT EXISTS traffic_stats_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    core_id INTEGER NOT NULL DEFAULT 0,
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

-- Migrate existing data (upload_bytes -> upload, download_bytes -> download)
INSERT OR IGNORE INTO traffic_stats_new (id, user_id, inbound_id, core_id, upload, download, total, recorded_at, granularity, created_at)
SELECT id, user_id, inbound_id, 0, COALESCE(upload_bytes, 0), COALESCE(download_bytes, 0),
       COALESCE(upload_bytes, 0) + COALESCE(download_bytes, 0), COALESCE(recorded_at, CURRENT_TIMESTAMP), 'raw', COALESCE(recorded_at, CURRENT_TIMESTAMP)
FROM traffic_stats;

DROP TABLE IF EXISTS traffic_stats;
ALTER TABLE traffic_stats_new RENAME TO traffic_stats;

CREATE INDEX IF NOT EXISTS idx_traffic_stats_user_id ON traffic_stats(user_id);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_inbound_id ON traffic_stats(inbound_id);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_core_id ON traffic_stats(core_id);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_recorded_at ON traffic_stats(recorded_at);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_granularity ON traffic_stats(granularity);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_user_recorded ON traffic_stats(user_id, recorded_at);

-- Recreate active_connections table with updated schema
CREATE TABLE IF NOT EXISTS active_connections_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    core_id INTEGER NOT NULL DEFAULT 0,
    core_name TEXT NOT NULL DEFAULT '',
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

-- Migrate existing data (upload_bytes -> upload, download_bytes -> download, connected_at -> started_at)
INSERT OR IGNORE INTO active_connections_new (id, user_id, inbound_id, core_id, source_ip, source_port, destination_ip, destination_port, started_at, last_activity, upload, download, created_at, updated_at)
SELECT id, user_id, inbound_id, 0, source_ip, source_port, destination_ip, destination_port,
       COALESCE(connected_at, CURRENT_TIMESTAMP), last_activity_at, COALESCE(upload_bytes, 0), COALESCE(download_bytes, 0),
       COALESCE(connected_at, CURRENT_TIMESTAMP), COALESCE(last_activity_at, CURRENT_TIMESTAMP)
FROM active_connections;

DROP TABLE IF EXISTS active_connections;
ALTER TABLE active_connections_new RENAME TO active_connections;

CREATE INDEX IF NOT EXISTS idx_active_connections_user_id ON active_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_active_connections_inbound_id ON active_connections(inbound_id);
CREATE INDEX IF NOT EXISTS idx_active_connections_started_at ON active_connections(started_at);
