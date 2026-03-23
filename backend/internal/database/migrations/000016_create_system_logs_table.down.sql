-- Drop system_logs table
DROP INDEX IF EXISTS idx_system_logs_created_at;
DROP INDEX IF EXISTS idx_system_logs_category;
DROP INDEX IF EXISTS idx_system_logs_level;
DROP TABLE IF EXISTS system_logs;
