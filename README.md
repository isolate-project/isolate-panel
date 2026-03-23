# Isolate Panel

Lightweight proxy core management panel for Xray, Sing-box, and Mihomo.

## 🎯 Overview

Isolate Panel is a minimalist web-based management interface for proxy cores, designed to run efficiently on VPS with limited resources (1 CPU / 1GB RAM). The panel is accessible only through SSH tunnel for maximum security.

## 🛠️ Tech Stack

**Backend:**
- Go 1.25+ with Fiber v3
- GORM + SQLite
- JWT authentication with Argon2id password hashing
- Zerolog for structured logging

**Frontend:**
- Preact 10.29 (lightweight React alternative)
- TypeScript 5.9
- Vite 6.x
- Tailwind CSS 4.2
- Zustand for state management

**Infrastructure:**
- Docker + Docker Compose
- Supervisord for process management
- Alpine Linux base image

## 📋 Requirements

- Docker 20.10+
- Docker Compose 2.0+

## 🚀 Quick Start

### Development

```bash
# Clone the repository
git clone https://github.com/vovk4morkovk4/isolate-panel.git
cd isolate-panel

# Start development environment
./scripts/dev.sh

# Or manually:
cd docker
docker-compose up --build
```

The panel will be available at:
- Backend API: http://localhost:8080
- Frontend Dev Server: http://localhost:5173

### Project Structure

```
isolate-panel/
├── backend/          # Go backend
│   ├── cmd/          # Entry points
│   ├── internal/     # Internal packages
│   ├── configs/      # Configuration files
│   └── Makefile      # Build commands
├── frontend/         # Preact frontend
│   ├── src/          # Source code
│   └── public/       # Static assets
├── docker/           # Docker configuration
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── supervisord.conf
├── docs/             # Documentation
├── scripts/          # Helper scripts
└── data/             # SQLite database (created at runtime)
```

## 📚 Documentation

Full documentation is available in the `docs/` directory:

- [PROJECT_PLAN.md](docs/PROJECT_PLAN.md) - Complete project specification
- [SECURITY_PLAN.md](docs/SECURITY_PLAN.md) - Security architecture and implementation
- [DECISIONS_SUMMARY.md](docs/DECISIONS_SUMMARY.md) - Key architectural decisions
- [PROTOCOL_SMART_FORMS_PLAN.md](docs/PROTOCOL_SMART_FORMS_PLAN.md) - Protocol-aware forms system

## 🔒 Security

- Panel accessible only via SSH tunnel (localhost:8080)
- Multi-level rate limiting for subscription endpoints
- Argon2id password hashing
- JWT-based authentication
- IP blocking and anomaly detection

## 🎯 Current Status

**Phase 0: Setup** ✅ Complete
- Project structure initialized
- Development environment configured
- Basic backend and frontend scaffolding

**Phase 1: MVP Backend** 🚧 In Progress
- Database migrations
- Authentication system
- Core management
- User management

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details

## 🤝 Contributing

This is a personal project. Contributions are welcome via pull requests.

## 📧 Contact

For issues and questions, please use GitHub Issues.
