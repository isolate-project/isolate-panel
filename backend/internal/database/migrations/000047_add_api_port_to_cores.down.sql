-- Rollback: Remove api_port field from cores table
ALTER TABLE cores DROP COLUMN api_port;