-- Down migration for 000025: Remove backup system columns

-- Drop indexes
DROP INDEX IF EXISTS idx_backups_schedule_cron;

-- Remove added columns (SQLite doesn't support DROP COLUMN directly before 3.35.0)
-- We need to recreate the table without the new columns

CREATE TABLE IF NOT EXISTS backups_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT,
    backup_type VARCHAR(20) NOT NULL,
    destination VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);

-- Copy data from new table to old table (only original columns)
INSERT INTO backups_old (id, filename, file_path, file_size_bytes, backup_type, destination, status, error_message, created_at, completed_at)
SELECT id, filename, file_path, file_size_bytes, backup_type, destination, status, error_message, created_at, completed_at
FROM backups;

-- Drop the new table and rename old table
DROP TABLE backups;
ALTER TABLE backups_old RENAME TO backups;

-- Recreate original indexes
CREATE INDEX idx_backups_created_at ON backups(created_at);
CREATE INDEX idx_backups_status ON backups(status);
