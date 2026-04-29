-- Remove per-node API key fields from cores table
ALTER TABLE cores DROP COLUMN api_key_hash;
ALTER TABLE cores DROP COLUMN api_key_salt;
ALTER TABLE cores DROP COLUMN api_key_hint;
