-- Drop refresh_tokens table
DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_token_hash;
DROP INDEX IF EXISTS idx_refresh_tokens_admin_id;
DROP TABLE IF EXISTS refresh_tokens;
