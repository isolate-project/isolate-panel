-- Update warp_routes table: add core_id and priority columns
ALTER TABLE warp_routes ADD COLUMN core_id INTEGER NOT NULL DEFAULT 1;
ALTER TABLE warp_routes ADD COLUMN priority INTEGER DEFAULT 50;

-- Create index for core_id
CREATE INDEX idx_warp_routes_core_id ON warp_routes(core_id);
CREATE INDEX idx_warp_routes_priority ON warp_routes(priority);

-- Create geo_rules table
CREATE TABLE IF NOT EXISTS geo_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    core_id INTEGER NOT NULL,
    type VARCHAR(20) NOT NULL,  -- 'geoip' or 'geosite'
    code VARCHAR(50) NOT NULL,  -- country code or category name
    action VARCHAR(20) NOT NULL,  -- 'proxy', 'direct', 'block', 'warp'
    priority INTEGER DEFAULT 50,
    is_enabled BOOLEAN DEFAULT 1,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_geo_rules_core_id ON geo_rules(core_id);
CREATE INDEX idx_geo_rules_type ON geo_rules(type);
CREATE INDEX idx_geo_rules_priority ON geo_rules(priority);
