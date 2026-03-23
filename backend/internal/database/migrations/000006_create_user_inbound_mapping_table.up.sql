-- Create user_inbound_mapping table
CREATE TABLE IF NOT EXISTS user_inbound_mapping (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    inbound_id INTEGER NOT NULL,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (inbound_id) REFERENCES inbounds(id) ON DELETE CASCADE,
    
    UNIQUE(user_id, inbound_id)
);

CREATE INDEX idx_user_inbound_user_id ON user_inbound_mapping(user_id);
CREATE INDEX idx_user_inbound_inbound_id ON user_inbound_mapping(inbound_id);
