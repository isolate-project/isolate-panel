-- Create notifications table for Phase 10: Notification System
CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry_at DATETIME,
    sent_at DATETIME,
    error TEXT,
    metadata TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_event_type ON notifications(event_type);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);

-- Create notification_settings table
CREATE TABLE IF NOT EXISTS notification_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    webhook_enabled BOOLEAN DEFAULT FALSE,
    webhook_url VARCHAR(255),
    webhook_secret VARCHAR(255),
    telegram_enabled BOOLEAN DEFAULT FALSE,
    telegram_bot_token VARCHAR(255),
    telegram_chat_id VARCHAR(100),
    notify_quota_exceeded BOOLEAN DEFAULT TRUE,
    notify_expiry_warning BOOLEAN DEFAULT TRUE,
    notify_cert_renewed BOOLEAN DEFAULT TRUE,
    notify_core_error BOOLEAN DEFAULT TRUE,
    notify_failed_login BOOLEAN DEFAULT TRUE,
    notify_user_created BOOLEAN DEFAULT TRUE,
    notify_user_deleted BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert default settings
INSERT INTO notification_settings (
    webhook_enabled, webhook_url, webhook_secret,
    telegram_enabled, telegram_bot_token, telegram_chat_id,
    notify_quota_exceeded, notify_expiry_warning, notify_cert_renewed,
    notify_core_error, notify_failed_login, notify_user_created, notify_user_deleted
) VALUES (
    FALSE, '', '',
    FALSE, '', '',
    TRUE, TRUE, TRUE,
    TRUE, TRUE, TRUE, FALSE
);
