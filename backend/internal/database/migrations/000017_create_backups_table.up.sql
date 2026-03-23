-- Create backups table
CREATE TABLE IF NOT EXISTS backups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT,
    
    -- Type and destination
    backup_type VARCHAR(20) NOT NULL,
    destination VARCHAR(20) NOT NULL,
    
    -- Status
    status VARCHAR(20) DEFAULT 'pending',
    error_message TEXT,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

CREATE INDEX idx_backups_created_at ON backups(created_at);
CREATE INDEX idx_backups_status ON backups(status);
