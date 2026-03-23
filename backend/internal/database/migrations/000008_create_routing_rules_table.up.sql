-- Create routing_rules table
CREATE TABLE IF NOT EXISTS routing_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    core_id INTEGER NOT NULL,
    name VARCHAR(100) NOT NULL,
    
    -- Rule (JSON)
    rule_json TEXT NOT NULL,
    
    -- Action
    outbound_id INTEGER NOT NULL,
    
    -- Priority
    priority INTEGER DEFAULT 0,
    
    -- Status
    is_enabled BOOLEAN DEFAULT 1,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (core_id) REFERENCES cores(id) ON DELETE CASCADE,
    FOREIGN KEY (outbound_id) REFERENCES outbounds(id) ON DELETE CASCADE
);

CREATE INDEX idx_routing_rules_core_id ON routing_rules(core_id);
CREATE INDEX idx_routing_rules_priority ON routing_rules(priority);
