# Isolate Panel CLI Documentation

## Installation

### Build from Source

```bash
cd cli
go build -o isolate-panel
sudo mv isolate-panel /usr/local/bin/
```

## Quick Start

### 1. Login to your panel

```bash
# Interactive login (prompts for username/password)
isolate-panel login

# Non-interactive login
isolate-panel login --url http://192.168.1.100:8080 --username admin --password secret

# Login to a named profile
isolate-panel login production --url http://192.168.1.100:8080
```

### 2. Manage profiles

```bash
# List all profiles
isolate-panel profile list

# Switch to a different profile
isolate-panel profile switch production

# Show current profile
isolate-panel profile current

# Delete a profile
isolate-panel profile delete production
```

### 3. List users

```bash
# Table format (default)
isolate-panel user list

# JSON format
isolate-panel user list --format json

# CSV format (for export)
isolate-panel user list --format csv

# Quiet mode (usernames only, for scripting)
isolate-panel user list --format quiet
```

## Global Flags

```bash
--config string       config file path (default: ~/.isolate-panel/config.json)
--url string          panel URL (default: "http://localhost:8080")
--token string        API token (overrides profile token)
--no-color            Disable colored output
--format string       Output format: table, json, csv, quiet (default "table")
-h, --help            Help for command
-v, --version         Version information
```

## Configuration

Config file location: `~/.isolate-panel/config.json`

```json
{
  "current_profile": "default",
  "profiles": {
    "default": {
      "panel_url": "http://localhost:8080",
      "username": "admin",
      "access_token": "eyJhbGc...",
      "refresh_token": "eyJhbGc...",
      "token_expires_at": "2026-03-24T23:59:59Z"
    },
    "production": {
      "panel_url": "http://192.168.1.100:8080",
      "username": "admin",
      "access_token": "eyJhbGc...",
      "refresh_token": "eyJhbGc...",
      "token_expires_at": "2026-03-24T23:59:59Z"
    }
  }
}
```

## Commands Reference

### Authentication

```bash
isolate-panel login [profile-name]
isolate-panel logout [profile-name]
```

### Profile Management

```bash
isolate-panel profile list
isolate-panel profile switch <name>
isolate-panel profile current
isolate-panel profile delete <name>
```

### User Management

```bash
isolate-panel user list [--format table|json|csv|quiet]
isolate-panel user show <username|id>
isolate-panel user create <username> [--email=<email>] [--traffic-limit=<bytes>] [--expiry=<date>]
isolate-panel user update <username|id> [--traffic-limit=<bytes>] [--expiry=<date>] [--active=<bool>]
isolate-panel user delete <username|id> [--force]
isolate-panel user credentials <username|id>
isolate-panel user regenerate <username|id> [--force]
```

### Inbound Management

```bash
isolate-panel inbound list [--format table|json|csv|quiet]
isolate-panel inbound show <id|name>
isolate-panel inbound create --core=<core> --name=<name> --port=<port> --protocol=<protocol>
isolate-panel inbound update <id|name> [--name=<name>] [--port=<port>]
isolate-panel inbound delete <id|name> [--force]
isolate-panel inbound add-users <inbound-id> <user-id1> [user-id2] [...]
isolate-panel inbound remove-user <inbound-id> <user-id>
isolate-panel inbound users <inbound-id>
```

### Outbound Management

```bash
isolate-panel outbound list [--format table|json|csv|quiet]
isolate-panel outbound show <id>
isolate-panel outbound create --name=<name> --type=<type>
isolate-panel outbound update <id> [--name=<name>]
isolate-panel outbound delete <id>
```

### Core Management

```bash
isolate-panel core list [--format table|json|csv|quiet]
isolate-panel core status [core-name]
isolate-panel core start <singbox|xray|mihomo>
isolate-panel core stop <singbox|xray|mihomo>
isolate-panel core restart <singbox|xray|mihomo>
isolate-panel core logs <singbox|xray|mihomo> [--tail <n>] [--follow]
```

### System & Statistics

```bash
isolate-panel stats [--format table|json|csv|quiet]
isolate-panel connections [--user=<id>] [--core=<core>]
```

### Certificate Management

```bash
isolate-panel cert list [--format table|json|csv|quiet]
isolate-panel cert request <domain> [--email=<email>] [--wildcard]
isolate-panel cert show <id>
isolate-panel cert renew <id>
isolate-panel cert delete <id>
```

### Backup Management

```bash
isolate-panel backup create [--no-encryption] [--no-cores] [--no-certs] [--no-warp]
isolate-panel backup list [--format table|json|csv|quiet]
isolate-panel backup restore <backup-id> [--force]
isolate-panel backup delete <backup-id>
isolate-panel backup download <backup-id> [--output <path>]
isolate-panel backup schedule [cron-expression]
```

### Shell Completion

```bash
# Bash
isolate-panel completion bash > /etc/bash_completion.d/isolate-panel

# Zsh
isolate-panel completion zsh > /usr/local/share/zsh/site-functions/_isolate-panel

# Fish
isolate-panel completion fish > ~/.config/fish/completions/isolate-panel.fish
```

## Output Formats

### Table (default)

Human-readable table with aligned columns:

```
ID  USERNAME  EMAIL              STATUS    TRAFFIC  EXPIRY
1   user1     user1@example.com  Active    1.2 GB   2026-12-31
2   user2     user2@example.com  Inactive  500 MB   2026-06-30
```

### JSON

Machine-readable JSON:

```json
[
  {
    "id": 1,
    "username": "user1",
    "email": "user1@example.com",
    "is_active": true,
    "traffic_used_bytes": 1288490188,
    "expiry_date": "2026-12-31T23:59:59Z"
  }
]
```

### CSV

CSV for export to spreadsheets:

```csv
id,username,email,is_active,traffic_used_bytes,expiry_date
1,user1,user1@example.com,true,1288490188,2026-12-31T23:59:59Z
```

### Quiet

Minimal output (values only) for scripting:

```
user1
user2
```

## Exit Codes

| Code | Name | Description |
|------|------|-------------|
| 0 | ExitSuccess | Command completed successfully |
| 1 | ExitGeneralError | General error |
| 2 | ExitAuthError | Authentication error |
| 3 | ExitNotFoundError | Resource not found |
| 4 | ExitValidationError | Validation error |
| 5 | ExitNetworkError | Network error |
| 6 | ExitPermissionError | Permission denied |

## Examples

### Create a user with 100GB traffic limit

```bash
isolate-panel user create alice --email=alice@example.com --traffic-limit=107374182400
```

### List inactive users

```bash
isolate-panel user list --format json | jq '.[] | select(.is_active == false)'
```

### Create inbound on port 443

```bash
isolate-panel inbound create --core=singbox --name=vmess-443 --port=443 --protocol=vmess
```

### Backup with custom schedule

```bash
# Create manual backup
isolate-panel backup create

# Set daily backup at 3 AM
isolate-panel backup schedule "0 3 * * *"

# List backups
isolate-panel backup list --format table
```

### Export users to CSV

```bash
isolate-panel user list --format csv > users.csv
```

### Script-friendly user listing

```bash
# Get list of usernames for a loop
for user in $(isolate-panel user list --format quiet); do
  echo "Processing $user"
  isolate-panel user show "$user"
done
```

## Troubleshooting

### "no profile selected" error

Run `isolate-panel login` first to create a profile.

### "API error: 401 Unauthorized"

Your token has expired. Run `isolate-panel login` to get a new token.

### "API error: 404 Not Found"

Check that the panel URL is correct and the API endpoint exists.

### Colors not working in terminal

Check if `NO_COLOR` environment variable is set. Use `--no-color` flag to disable colors explicitly.

## Post-MVP Features

The following features are planned for future releases:

- Interactive wizards with prompts
- QR code display in terminal
- Credential export to file
- Automatic token refresh
- Progress indicators for long operations
- Man pages
