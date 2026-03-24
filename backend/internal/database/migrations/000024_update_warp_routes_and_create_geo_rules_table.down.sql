-- Drop geo_rules table
DROP INDEX IF EXISTS idx_geo_rules_priority;
DROP INDEX IF EXISTS idx_geo_rules_priority;
DROP INDEX IF EXISTS idx_geo_rules_core_id;
DROP TABLE IF EXISTS geo_rules;

-- Remove new columns from warp_routes (SQLite doesn't support DROP COLUMN directly)
-- We need to recreate the table
CREATE TABLE warp_routes_backup AS SELECT id, resource_type, resource_value, description, is_enabled, created_at, updated_at FROM warp_routes;
DROP TABLE warp_routes;

CREATE TABLE warp_routes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    resource_type VARCHAR(20) NOT NULL,
    resource_value VARCHAR(255) NOT NULL,
    description TEXT,
    is_enabled BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO warp_routes (id, resource_type, resource_value, description, is_enabled, created_at, updated_at)
SELECT id, resource_type, resource_value, description, is_enabled, created_at, updated_at FROM warp_routes_backup;

DROP TABLE warp_routes_backup;
DROP INDEX IF EXISTS idx_warp_routes_priority;
DROP INDEX IF EXISTS idx_warp_routes_core_id;
