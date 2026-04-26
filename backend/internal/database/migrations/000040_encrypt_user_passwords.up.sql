-- This migration encrypts existing plaintext user passwords
-- The actual encryption is done in Go code (see migration hook)
ALTER TABLE users ADD COLUMN password_encrypted BOOLEAN DEFAULT FALSE;