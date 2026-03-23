-- Drop users table
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_subscription_token;
DROP INDEX IF EXISTS idx_users_token;
DROP INDEX IF EXISTS idx_users_uuid;
DROP INDEX IF EXISTS idx_users_username;
DROP TABLE IF EXISTS users;
