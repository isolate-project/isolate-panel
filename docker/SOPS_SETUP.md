# SOPS + age + Docker Secrets Setup

## Overview

This guide configures secret encryption for production deployments using:
- **SOPS** (Secrets OPerationS) — encrypts `.env` files with age keys
- **age** — modern encryption tool (replaces PGP)
- **Docker Compose secrets** — mounts secrets as files in the container

## Quick Setup

### 1. Generate age key pair

```bash
age-keygen -o age.key.txt
```

This creates:
- `age.key.txt` — private key (KEEP SECRET, never commit)
- Public key printed to stdout (used for encryption)

### 2. Add to .gitignore

```bash
echo "age.key.txt" >> .gitignore
echo "*.enc" >> .gitignore
echo "secrets/" >> .gitignore
```

### 3. Encrypt your .env file

```bash
sops --encrypt --age $(cat age.key.txt | grep "public key" | cut -d" " -f3) .env > .env.enc
```

### 4. Create secrets directory for Docker

```bash
mkdir -p secrets
chmod 700 secrets

# Extract secrets from .env and write to individual files
# These files are mounted via Docker Compose secrets
echo "your-jwt-secret-min-64-characters-long!!!" > secrets/jwt_secret.txt
echo "your-password-pepper" > secrets/password_pepper.txt
echo "your-admin-password" > secrets/admin_password.txt

chmod 600 secrets/*.txt
```

### 5. Deploy with encrypted .env

```bash
# Decrypt .env for deployment
sops --decrypt .env.enc > .env

# Or deploy directly with encrypted file
docker compose up -d
```

## Decrypt in CI/CD

```bash
# In GitHub Actions / GitLab CI
export SOPS_AGE_KEY=$(cat age.key.txt | grep "AGE-SECRET-KEY" | cut -d" " -f3)
sops --decrypt .env.enc > .env
```

## Security Notes

- `age.key.txt` — store in password manager or HSM, never in Git
- `secrets/*.txt` — these are mounted read-only in the container at `/run/secrets/`
- The Go backend reads secrets from files when `_FILE` env vars are set
- `.env.enc` — safe to commit to Git (encrypted)
