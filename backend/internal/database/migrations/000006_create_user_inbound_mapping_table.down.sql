-- Drop user_inbound_mapping table
DROP INDEX IF EXISTS idx_user_inbound_inbound_id;
DROP INDEX IF EXISTS idx_user_inbound_user_id;
DROP TABLE IF EXISTS user_inbound_mapping;
