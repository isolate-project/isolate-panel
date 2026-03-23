-- Create subscription_short_urls table
CREATE TABLE IF NOT EXISTS subscription_short_urls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    short_code VARCHAR(8) UNIQUE NOT NULL,
    full_url TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_subscription_short_urls_user_id ON subscription_short_urls(user_id);
CREATE UNIQUE INDEX idx_subscription_short_urls_code ON subscription_short_urls(short_code);
