# Security Audit Plan - Isolate Panel

## Overview

This document outlines the comprehensive security audit plan for Isolate Panel Phase 14.

## Audit Scope

### Components to Audit

1. **Backend (Go)**
   - Authentication & Authorization
   - API endpoints
   - Database operations
   - File operations
   - Configuration management

2. **Frontend (Preact/TypeScript)**
   - Input validation
   - XSS prevention
   - CSRF protection
   - Session management

3. **Infrastructure**
   - Docker configuration
   - Supervisord configuration
   - File permissions
   - Environment variables

## Automated Security Scanning

### 1. Backend Scanning

#### gosec - Go Security Scanner

```bash
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run scan on backend
cd backend
gosec -exclude=G104 ./...

# Generate JSON report
gosec -fmt=json -out=../security-reports/gosec-report.json ./...

# Full scan with all checks
gosec -exclude="" ./...
```

**Common Issues to Check:**
- G104: Errors unhandled (acceptable in some cases)
- G107: HTTP requests made with variable URL (SSRF)
- G109: Potential Integer overflow
- G114: Use of net/http without timeout
- G201/G202: SQL query construction (SQL injection)
- G301/G302: File/directory permissions
- G401/G402: Weak cryptographic functions
- G501/G502/G503: Weak hash functions

#### govulncheck - Go Vulnerability Checker

```bash
# Install govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# Run scan
cd backend
govulncheck ./...

# Generate report
govulncheck -json ./... > ../security-reports/govulncheck-report.json
```

### 2. Frontend Scanning

#### npm audit

```bash
# Install dependencies
cd frontend
npm install

# Run audit
npm audit --audit-level=moderate

# Generate JSON report
npm audit --json > ../security-reports/npm-audit-report.json

# Auto-fix issues
npm audit fix
npm audit fix --force
```

#### Snyk Security Scanner

```bash
# Install Snyk
npm install -g snyk

# Authenticate
snyk auth

# Run scan
cd frontend
snyk test

# Generate report
snyk test --json > ../security-reports/snyk-report.json

# Monitor for new vulnerabilities
snyk monitor
```

### 3. CI/CD Integration

Create `.github/workflows/security-scan.yml`:

```yaml
name: Security Scan

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 2 * * 1'  # Weekly on Monday at 2 AM

jobs:
  security-scan:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.25'
    
    - name: Run gosec
      uses: securego/gosec@master
      with:
        args: ./...
    
    - name: Run govulncheck
      run: |
        go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./...
    
    - name: Set up Node.js
      uses: actions/setup-node@v4
      with:
        node-version: '22'
    
    - name: Install dependencies
      run: cd frontend && npm ci
    
    - name: Run npm audit
      run: cd frontend && npm audit --audit-level=moderate
    
    - name: Run Snyk
      uses: snyk/actions/node@master
      env:
        SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
      with:
        directory: frontend
```

## Manual Penetration Testing

### Authentication & Authorization

#### JWT Token Security

- [ ] Test JWT token manipulation (change payload, verify signature fails)
- [ ] Test expired token handling
- [ ] Test token with invalid signature
- [ ] Test token with wrong algorithm (none algorithm attack)
- [ ] Test token reuse after logout
- [ ] Test refresh token rotation
- [ ] Test concurrent session handling

#### Privilege Escalation

- [ ] Attempt to access admin endpoints as regular user
- [ ] Attempt to modify other users' data (IDOR)
- [ ] Attempt to elevate privileges via API parameters
- [ ] Test vertical privilege escalation
- [ ] Test horizontal privilege escalation

#### Session Security

- [ ] Test session fixation attacks
- [ ] Test session hijacking
- [ ] Test concurrent sessions limits
- [ ] Test session timeout enforcement
- [ ] Test logout invalidates tokens

### Input Validation

#### SQL Injection

Test all API endpoints with SQL injection payloads:

```
' OR '1'='1
'; DROP TABLE users; --
' UNION SELECT NULL, NULL, NULL--
' AND 1=1--
' AND 1=2--
```

**Endpoints to Test:**
- [ ] `/api/auth/login`
- [ ] `/api/users`
- [ ] `/api/inbounds`
- [ ] `/api/settings`
- [ ] `/api/stats`
- [ ] `/sub/{token}`

#### XSS (Cross-Site Scripting)

Test all input fields with XSS payloads:

```
<script>alert('XSS')</script>
<img src=x onerror=alert('XSS')>
<svg onload=alert('XSS')>
javascript:alert('XSS')
```

**Fields to Test:**
- [ ] Username fields
- [ ] Email fields
- [ ] Inbound names
- [ ] Configuration JSON
- [ ] Settings values

#### Command Injection

Test CLI integration points:

```
; ls -la
| cat /etc/passwd
`whoami`
$(whoami)
```

**Areas to Test:**
- [ ] CLI commands
- [ ] File operations
- [ ] Core lifecycle management
- [ ] Backup operations

#### Path Traversal

Test file operations:

```
../../../etc/passwd
..\\..\\..\\windows\\system32\\config\\sam
....//....//etc/passwd
%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd
```

**Endpoints to Test:**
- [ ] `/api/backups/download`
- [ ] `/api/certs/download`
- [ ] File upload endpoints
- [ ] Log viewing endpoints

#### SSRF (Server-Side Request Forgery)

Test URL parameters:

```
http://169.254.169.254/latest/meta-data/ (AWS metadata)
http://localhost:8080/admin
http://127.0.0.1:22
file:///etc/passwd
```

**Endpoints to Test:**
- [ ] Webhook URL configuration
- [ ] Certificate request (ACME)
- [ ] External API calls
- [ ] Backup destination URLs

### API Security

#### IDOR (Insecure Direct Object Reference)

Test object access controls:

```
GET /api/users/1
GET /api/users/2  # Try to access other users
GET /api/inbounds/1
GET /api/inbounds/999  # Try non-existent ID
```

**Endpoints to Test:**
- [ ] `/api/users/{id}`
- [ ] `/api/inbounds/{id}`
- [ ] `/api/outbounds/{id}`
- [ ] `/api/backups/{id}`

#### Mass Assignment

Test if unintended fields can be modified:

```json
// Try to set admin flag
{
  "username": "newuser",
  "email": "user@example.com",
  "is_admin": true
}

// Try to set traffic limit
{
  "username": "newuser",
  "traffic_limit_bytes": 999999999999
}
```

**Endpoints to Test:**
- [ ] `/api/users` (POST/PUT)
- [ ] `/api/admins` (POST/PUT)
- [ ] Any update endpoint

#### HTTP Method Tampering

Test HTTP method overrides:

```
GET /api/users (should list)
POST /api/users/1 (should fail or update?)
PUT /api/users/1 (should update)
DELETE /api/users/1 (should delete)
PATCH /api/users/1 (should partial update)
OPTIONS /api/users/1 (CORS preflight)
```

**Endpoints to Test:**
- [ ] All CRUD endpoints
- [ ] Read-only endpoints
- [ ] Write-only endpoints

#### Content-Type Tampering

Test content-type handling:

```
POST /api/users
Content-Type: application/json
{"username": "test"}

POST /api/users
Content-Type: application/xml
<user><username>test</username></user>

POST /api/users
Content-Type: text/plain
{"username": "test"}
```

#### CORS Configuration

Test CORS headers:

```
Origin: http://evil.com
Access-Control-Request-Method: POST
Access-Control-Request-Headers: Content-Type, Authorization
```

**Expected:**
- Only trusted origins allowed
- Credentials not exposed to untrusted origins
- Proper preflight handling

### Infrastructure Security

#### Docker Container Security

- [ ] Verify container runs as non-root user
- [ ] Check file permissions inside container
- [ ] Test container escape attempts
- [ ] Verify no unnecessary capabilities
- [ ] Check for sensitive data in environment
- [ ] Verify read-only filesystem where possible

**Commands:**
```bash
# Check user
docker exec isolate-panel whoami

# Check capabilities
docker exec isolate-panel capsh --print

# Check filesystem
docker exec isolate-panel mount | grep ro
```

#### Supervisord Security

- [ ] Verify supervisord runs as non-root
- [ ] Check process isolation
- [ ] Verify log file permissions
- [ ] Test process restart limits
- [ ] Check socket permissions

**Configuration to Review:**
```ini
[supervisord]
user=isolate

[program:isolate-panel]
user=isolate
umask=027
```

#### File Permissions

- [ ] Database file permissions (should be 600 or 640)
- [ ] Configuration file permissions
- [ ] Certificate file permissions
- [ ] Log file permissions
- [ ] Backup file permissions

**Commands:**
```bash
ls -la /app/data/
ls -la /var/log/isolate-panel/
ls -la /etc/isolate-panel/
```

#### Environment Variables

- [ ] Verify no secrets in environment
- [ ] Check for hardcoded credentials
- [ ] Test environment variable injection
- [ ] Verify .env file not exposed

**Commands:**
```bash
docker exec isolate-panel env | grep -i secret
docker exec isolate-panel env | grep -i password
docker exec isolate-panel env | grep -i token
```

## Security Checklist

### Critical (Must Fix)

- [ ] No SQL injection vulnerabilities
- [ ] No XSS vulnerabilities
- [ ] No authentication bypass
- [ ] No privilege escalation
- [ ] No sensitive data exposure
- [ ] No command injection
- [ ] No path traversal

### High (Should Fix)

- [ ] Proper rate limiting
- [ ] Input validation on all endpoints
- [ ] Secure session management
- [ ] CSRF protection
- [ ] CORS properly configured
- [ ] Security headers implemented

### Medium (Nice to Have)

- [ ] Security logging and monitoring
- [ ] Audit trail for sensitive operations
- [ ] Account lockout after failed attempts
- [ ] Password complexity requirements
- [ ] 2FA implementation

### Low (Future Improvements)

- [ ] Security documentation
- [ ] Security training for developers
- [ ] Regular security assessments
- [ ] Bug bounty program

## Remediation Plan

For each vulnerability found:

1. **Document**: Create detailed report with:
   - Vulnerability description
   - Affected component
   - Severity rating (CVSS score)
   - Steps to reproduce
   - Proof of concept
   - Recommended fix

2. **Prioritize**: Based on severity and exploitability

3. **Fix**: Implement remediation

4. **Verify**: Re-test to confirm fix

5. **Monitor**: Add detection/prevention

## Tools and Resources

### Security Tools

- **gosec**: Go source code security scanner
- **govulncheck**: Go vulnerability checker
- **npm audit**: Node.js dependency auditor
- **Snyk**: Comprehensive security scanner
- **OWASP ZAP**: Web application security scanner
- **Burp Suite**: Manual penetration testing

### References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [CWE Top 25](https://cwe.mitre.org/top25/archive/2023/2023_cwe_top25.html)
- [Go Secure Coding Practices](https://github.com/Checkmarx/secure-coding-practices)
- [Docker Security Best Practices](https://docs.docker.com/engine/security/)

## Reporting

All security findings should be documented in:

- `security-reports/penetration-testing-report.md` - Detailed findings
- `security-reports/remediation-plan.md` - Fix plan and timeline
- `security-reports/gosec-report.json` - Automated scan results
- `security-reports/npm-audit-report.json` - Dependency audit results

## Contact

For security issues, contact the development team or report via GitHub Security Advisories.
