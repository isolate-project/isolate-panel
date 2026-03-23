-- Drop login_attempts table
DROP INDEX IF EXISTS idx_login_attempts_username;
DROP INDEX IF EXISTS idx_login_attempts_ip;
DROP TABLE IF EXISTS login_attempts;
