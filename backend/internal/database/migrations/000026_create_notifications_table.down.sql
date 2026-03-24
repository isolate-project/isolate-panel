-- Down migration for 000026: Remove notification system tables
DROP TABLE IF EXISTS notification_settings;
DROP TABLE IF EXISTS notifications;
