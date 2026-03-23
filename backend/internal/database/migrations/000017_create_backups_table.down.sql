-- Drop backups table
DROP INDEX IF EXISTS idx_backups_status;
DROP INDEX IF EXISTS idx_backups_created_at;
DROP TABLE IF EXISTS backups;
