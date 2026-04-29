-- Add per-node API key fields to cores table (Phase 5.4)
ALTER TABLE cores ADD COLUMN api_key_hash TEXT DEFAULT '';
ALTER TABLE cores ADD COLUMN api_key_salt TEXT DEFAULT '';
ALTER TABLE cores ADD COLUMN api_key_hint TEXT DEFAULT '';
