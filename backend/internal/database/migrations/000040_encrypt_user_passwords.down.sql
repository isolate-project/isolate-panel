-- Rollback password encryption migration
ALTER TABLE users DROP COLUMN password_encrypted;