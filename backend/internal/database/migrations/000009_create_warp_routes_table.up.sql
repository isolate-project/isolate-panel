-- Create warp_routes table
CREATE TABLE IF NOT EXISTS warp_routes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    resource_type VARCHAR(20) NOT NULL,
    resource_value VARCHAR(255) NOT NULL,
    description TEXT,
    
    is_enabled BOOLEAN DEFAULT 1,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_warp_routes_resource_type ON warp_routes(resource_type);
