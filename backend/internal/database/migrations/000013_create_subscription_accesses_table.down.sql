-- Drop subscription_accesses table
DROP INDEX IF EXISTS idx_subscription_accesses_is_suspicious;
DROP INDEX IF EXISTS idx_subscription_accesses_accessed_at;
DROP INDEX IF EXISTS idx_subscription_accesses_user_id;
DROP TABLE IF EXISTS subscription_accesses;
