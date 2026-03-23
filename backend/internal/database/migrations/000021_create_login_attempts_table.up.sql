-- Create login_attempts table
CREATE TABLE IF NOT EXISTS login_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip_address VARCHAR(50) NOT NULL,
    username VARCHAR(50),
    success BOOLEAN DEFAULT 0,
    attempted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    user_agent VARCHAR(255)
);

CREATE INDEX idx_login_attempts_ip ON login_attempts(ip_address, attempted_at);
CREATE INDEX idx_login_attempts_username ON login_attempts(username, attempted_at);
