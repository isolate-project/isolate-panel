-- Create certificates table
CREATE TABLE IF NOT EXISTS certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,
    
    -- File paths
    cert_path VARCHAR(255) NOT NULL,
    key_path VARCHAR(255) NOT NULL,
    
    -- ACME settings
    acme_provider VARCHAR(50),
    acme_email VARCHAR(100),
    acme_challenge_type VARCHAR(20),
    
    -- Status
    is_valid BOOLEAN DEFAULT 1,
    auto_renew BOOLEAN DEFAULT 1,
    
    -- Dates
    issued_at DATETIME,
    expires_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_certificates_domain ON certificates(domain);
CREATE INDEX idx_certificates_expires_at ON certificates(expires_at);
