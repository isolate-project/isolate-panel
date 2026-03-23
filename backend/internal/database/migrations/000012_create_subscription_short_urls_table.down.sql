-- Drop subscription_short_urls table
DROP INDEX IF EXISTS idx_subscription_short_urls_code;
DROP INDEX IF EXISTS idx_subscription_short_urls_user_id;
DROP TABLE IF EXISTS subscription_short_urls;
