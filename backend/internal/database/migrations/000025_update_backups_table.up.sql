-- Update backups table for Phase 9: Backup System
-- Add new columns for scheduling, encryption, and metadata

-- Add schedule_cron column for backup scheduling
ALTER TABLE backups ADD COLUMN schedule_cron VARCHAR(50);

-- Add encryption_enabled column
ALTER TABLE backups ADD COLUMN encryption_enabled BOOLEAN DEFAULT TRUE;

-- Add checksum_sha256 for backup integrity verification
ALTER TABLE backups ADD COLUMN checksum_sha256 VARCHAR(64);

-- Add duration_ms for tracking backup/restore duration
ALTER TABLE backups ADD COLUMN duration_ms INTEGER;

-- Add backup_source to track what was included in backup
ALTER TABLE backups ADD COLUMN backup_source TEXT; -- JSON: {"include_cores": true, "include_certs": true, ...}

-- Add metadata JSON for storing backup metadata
ALTER TABLE backups ADD COLUMN metadata TEXT; -- JSON: {"version": "1.0", "cores": ["xray", "singbox"], ...}

-- Create index for schedule_cron to quickly find scheduled backups
CREATE INDEX IF NOT EXISTS idx_backups_schedule_cron ON backups(schedule_cron) WHERE schedule_cron IS NOT NULL;

-- Update existing records to have default values
UPDATE backups SET encryption_enabled = TRUE WHERE encryption_enabled IS NULL;
UPDATE backups SET status = 'completed' WHERE status = 'pending' AND completed_at IS NOT NULL;
