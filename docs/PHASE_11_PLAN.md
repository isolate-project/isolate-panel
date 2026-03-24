# Phase 11: CLI Interface

## Decisions

| Aspect | Decision |
|--------|----------|
| Order | By plan: 11.1 → 11.2 → 11.3 → 11.4 → 11.5 → 11.6 → 11.7 |
| Token Refresh | ✅ Automatic when token expires |
| Interactive Login | ✅ Prompt username/password |
| Output Formats | ✅ --json, --csv, --quiet, --no-color |
| User Credentials | Admin has access to all (existing decision) |
| Wizards | ❌ Post-MVP (only non-interactive with flags for MVP) |
| Completion | ✅ bash, zsh, fish |

## Implementation Plan

### 11.1 CLI Authentication & Core Framework (3 days)

**Tasks:**
- [x] CLI framework (cobra) — already done in Phase 9
- [ ] Multi-profile configuration system (`~/.isolate-panel/config.json`)
- [ ] Authentication commands (`login`, `logout`)
- [ ] Interactive login prompts (username/password)
- [ ] Profile management (`switch`, `list`, `current`)
- [ ] Automatic token refresh
- [ ] Error handling with exit codes

**Config Structure:**
```json
{
  "current_profile": "local",
  "profiles": {
    "local": {
      "panel_url": "http://localhost:8080",
      "access_token": "...",
      "refresh_token": "...",
      "token_expires_at": "2026-03-22T10:00:00Z"
    },
    "production": {
      "panel_url": "http://192.168.1.100:8080",
      "access_token": "...",
      "refresh_token": "...",
      "token_expires_at": "2026-03-22T10:00:00Z"
    }
  }
}
```

**Exit Codes:**
```go
const (
  ExitSuccess           = 0
  ExitGeneralError      = 1
  ExitAuthError         = 2
  ExitNotFoundError     = 3
  ExitValidationError   = 4
  ExitNetworkError      = 5
  ExitPermissionError   = 6
)
```

### 11.2 Output Formatters (2 days)

**Tasks:**
- [ ] Table formatter (human-readable, default)
- [ ] JSON formatter (`--json` flag)
- [ ] CSV formatter (`--csv` flag)
- [ ] Quiet mode (`--quiet` flag, values only)
- [ ] Colored output with `--no-color` option
- [ ] Progress indicators for long operations

**Global Flags:**
```bash
--json                Output in JSON format
--csv                 Output in CSV format
--quiet               Minimal output (values only)
--no-color            Disable colored output
--profile <name>      Use specific profile
--config <path>       Config file path
```

### 11.3 User Management Commands (2 days)

**Commands:**
```bash
isolate-panel user list [--active] [--expired] [--limit=<n>] [--json|--csv|--quiet]
isolate-panel user show <username|id> [--json|--csv|--quiet]
isolate-panel user create <username> [--email=<email>] [--traffic-limit=<GB>] [--expiry=<date>]
isolate-panel user update <username|id> [--traffic-limit=<GB>] [--expiry=<date>] [--active=<bool>]
isolate-panel user delete <username|id> [--force]
isolate-panel user credentials <username|id>  # Show credentials (admin only)
isolate-panel user regenerate <username|id> [--force]  # Regenerate credentials
```

### 11.4 Inbound/Outbound Management (2 days)

**Note:** Interactive wizards moved to Post-MVP. Only non-interactive mode with flags.

**Commands:**
```bash
# Inbound management
isolate-panel inbound list [--core <singbox|xray|mihomo>] [--json|--csv|--quiet]
isolate-panel inbound show <id|name> [--json|--csv|--quiet]
isolate-panel inbound create --core=<core> --protocol=<protocol> --name=<name> --port=<port>
isolate-panel inbound update <id|name> [--name <name>] [--port <port>]
isolate-panel inbound delete <id|name> [--force]
isolate-panel inbound add-users <inbound-id> <user-id1> [user-id2] [...]
isolate-panel inbound remove-user <inbound-id> <user-id>
isolate-panel inbound users <inbound-id>

# Outbound management
isolate-panel outbound list [--json|--csv|--quiet]
isolate-panel outbound show <id> [--json|--csv|--quiet]
isolate-panel outbound create --name=<name> --type=<type>
isolate-panel outbound update <id> [--name <name>]
isolate-panel outbound delete <id>
```

### 11.5 Core & System Management (1 day)

**Commands:**
```bash
# Core management
isolate-panel core list [--json|--csv|--quiet]
isolate-panel core status [<singbox|xray|mihomo>] [--json|--csv|--quiet]
isolate-panel core start <singbox|xray|mihomo>
isolate-panel core stop <singbox|xray|mihomo>
isolate-panel core restart <singbox|xray|mihomo>
isolate-panel core logs <singbox|xray|mihomo> [--tail <n>] [--follow]
isolate-panel core validate <singbox|xray|mihomo>

# System management
isolate-panel system status [--json|--csv|--quiet]
isolate-panel system restart
isolate-panel system logs [--tail <n>] [--level <level>] [--follow]

# Statistics
isolate-panel stats [--json|--csv|--quiet]
isolate-panel stats export --format <csv|json> --output <file>

# Active connections
isolate-panel connections [--user <id>] [--core <core>] [--json|--csv|--quiet]

# Settings
isolate-panel settings list [--json|--csv|--quiet]
isolate-panel settings get <key> [--json|--csv|--quiet]
isolate-panel settings set <key> <value>
```

### 11.6 Backup & Certificates (1 day)

**Commands:**
```bash
# Backup management (already implemented in Phase 9)
isolate-panel backup create [--no-encryption] [--no-cores] [--no-certs] [--no-warp]
isolate-panel backup list [--json|--csv|--quiet]
isolate-panel backup restore <backup-id> [--force]
isolate-panel backup delete <backup-id>
isolate-panel backup download <backup-id> [--output <path>]
isolate-panel backup schedule [cron-expression]

# Certificate management
isolate-panel cert list [--json|--csv|--quiet]
isolate-panel cert request <domain> [--email <email>] [--wildcard]
isolate-panel cert show <id> [--json|--csv|--quiet]
isolate-panel cert renew <id>
isolate-panel cert revoke <id> [--force]
isolate-panel cert delete <id> [--force]
isolate-panel cert upload --cert <file> --key <file> --domain <domain>
```

### 11.7 Completion & Documentation (1 day)

**Tasks:**
- [ ] Bash completion script
- [ ] Zsh completion script
- [ ] Fish completion script
- [ ] Man pages (optional)
- [ ] CLI documentation (markdown)

**Installation:**
```bash
# Bash
isolate-panel completion bash > /etc/bash_completion.d/isolate-panel

# Zsh
isolate-panel completion zsh > /usr/local/share/zsh/site-functions/_isolate-panel

# Fish
isolate-panel completion fish > ~/.config/fish/completions/isolate-panel.fish
```

## Deliverables

- [ ] Multi-profile configuration with automatic token refresh
- [ ] Interactive login with username/password prompts
- [ ] All output formats: table, JSON, CSV, quiet
- [ ] Colored output with --no-color option
- [ ] User management commands (full CRUD)
- [ ] Inbound/Outbound management (non-interactive only)
- [ ] Core & System management commands
- [ ] Certificate management commands
- [ ] Backup commands (already done)
- [ ] Shell completion for bash, zsh, fish
- [ ] Exit codes for automation
- [ ] Comprehensive CLI documentation

## Post-MVP (Deferred)

- ❌ Interactive wizards with prompts
- ❌ QR code display in terminal
- ❌ Credential export to file
- ❌ Man pages
