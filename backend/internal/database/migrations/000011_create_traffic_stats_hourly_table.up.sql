-- Create traffic_stats_hourly table
CREATE TABLE IF NOT EXISTS traffic_stats_hourly (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    hour_timestamp DATETIME NOT NULL,
    upload_bytes BIGINT DEFAULT 0,
    download_bytes BIGINT DEFAULT 0,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, hour_timestamp)
);

CREATE INDEX idx_traffic_stats_hourly_user_id ON traffic_stats_hourly(user_id);
CREATE INDEX idx_traffic_stats_hourly_timestamp ON traffic_stats_hourly(hour_timestamp);
