# Isolate Panel Development Guide

Complete guide for local development setup and workflow.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Development Setup](#local-development-setup)
3. [Backend Development](#backend-development)
4. [Frontend Development](#frontend-development)
5. [CLI Development](#cli-development)
6. [Docker Development](#docker-development)
7. [Testing](#testing)
8. [Debugging](#debugging)
9. [Common Issues](#common-issues)

---

## Prerequisites

### Required Software

| Software | Version | Purpose |
|----------|---------|---------|
| Go | 1.23+ | Backend development |
| Node.js | 20.x | Frontend development |
| Git | Latest | Version control |
| Docker | 20.10+ | Containerization (optional) |
| SQLite | 3.x | Database (included with Go) |

### Optional Tools

| Tool | Purpose |
|------|---------|
| VS Code | Recommended IDE |
| Go extension | Go development |
| Prettier | Code formatting |
| Docker Desktop | Docker management |

---

## Local Development Setup

### 1. Clone Repository

```bash
git clone https://github.com/your-org/isolate-panel.git
cd isolate-panel
```

### 2. Backend Setup

```bash
cd backend

# Download dependencies
go mod download

# Verify installation
go version
go list -m all
```

### 3. Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Verify installation
npm --version
node --version
```

### 4. Initialize Database

```bash
cd backend

# Run migrations
go run cmd/migrate/main.go up

# Seed initial data (optional)
go run cmd/migrate/main.go seed
```

---

## Backend Development

### Running the Server

```bash
cd backend

# Development mode
go run cmd/server/main.go

# With hot reload (air)
air

# Production build
go build -o server ./cmd/server/main.go
./server
```

### Environment Variables

Create `.env` file in `backend/` directory:

```bash
# Development .env
APP_ENV=development
PORT=8080
JWT_SECRET=your-development-secret-key
DATABASE_PATH=./data/isolate-panel.db
LOG_LEVEL=debug
MONITORING_MODE=lite
```

### Project Structure

```
backend/
├── cmd/
│   ├── server/          # Main entry point
│   └── migrate/         # Database migrations
├── internal/
│   ├── api/             # HTTP handlers
│   ├── auth/            # Authentication
│   ├── config/          # Configuration
│   ├── cores/           # Core generators
│   ├── database/        # Database layer
│   ├── middleware/      # HTTP middleware
│   ├── models/          # GORM models
│   ├── services/        # Business logic
│   └── ...
└── tests/               # Test files
```

### Common Commands

```bash
# Format code
go fmt ./...

# Lint code
go vet ./...

# Run tests
go test ./... -v

# Run specific test
go test ./tests/unit/services/... -v -run TestUserService

# Check dependencies
go mod tidy
go mod verify

# Build binary
go build -o bin/server ./cmd/server/main.go
```

---

## Frontend Development

### Running Dev Server

```bash
cd frontend

# Start Vite dev server
npm run dev

# With custom port
npm run dev -- --port 3000

# Production preview
npm run build
npm run preview
```

### Environment Variables

Create `.env` file in `frontend/` directory:

```bash
# Development .env
VITE_API_URL=http://localhost:8080/api
VITE_APP_TITLE=Isolate Panel (Dev)
```

### Project Structure

```
frontend/
├── src/
│   ├── api/             # API client
│   ├── components/      # UI components
│   ├── pages/           # Page components
│   ├── stores/          # Zustand stores
│   ├── i18n/            # Translations
│   ├── utils/           # Utilities
│   ├── app.tsx          # Root component
│   └── main.tsx         # Entry point
├── public/              # Static assets
└── tests/               # Test files
```

### Common Commands

```bash
# Format code
npm run format

# Lint code
npm run lint

# Run tests
npm run test

# Build for production
npm run build

# Type check
npm run type-check
```

---

## CLI Development

### Building CLI

```bash
cd cli

# Build binary
go build -o bin/isolate-panel .

# Install globally
go install .

# Run CLI
isolate-panel --help
```

### Testing CLI

```bash
cd cli

# Run tests
go test ./...

# Test specific command
go test ./cmd/... -v -run TestLogin
```

---

## Docker Development

### Development Mode

```bash
cd docker

# Build and run dev container
docker compose -f docker-compose.dev.yml up --build

# View logs
docker compose -f docker-compose.dev.yml logs -f

# Stop container
docker compose -f docker-compose.dev.yml down
```

### Production Mode

```bash
cd docker

# Build production image
docker build -f Dockerfile -t isolate-panel:latest ..

# Run production container
docker compose up -d

# View logs
docker compose logs -f
```

### Docker Commands

```bash
# List containers
docker ps -a

# View container logs
docker logs isolate-panel

# Execute command in container
docker exec -it isolate-panel sh

# Remove container
docker compose down -v
```

---

## Testing

### Backend Tests

```bash
cd backend

# Run all tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/services/... -v

# Run integration tests
go test ./tests/integration/... -v

# Run E2E tests (slow)
go test ./tests/e2e/... -v
```

### Frontend Tests

```bash
cd frontend

# Run all tests
npm run test

# Run with coverage
npm run test -- --coverage

# Run specific test file
npm run test -- src/components/ui/Button.test.tsx

# Watch mode
npm run test -- --watch
```

### CLI Tests

```bash
cd cli

# Run all tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
```

---

## Debugging

### Backend Debugging

**VS Code (delve):**

Create `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/backend/cmd/server/main.go",
      "env": {
        "APP_ENV": "development",
        "PORT": "8080"
      }
    }
  ]
}
```

**Delve CLI:**
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug server
dlv debug ./cmd/server/main.go

# Debug with breakpoints
dlv debug ./cmd/server/main.go --headless --listen=:2345
```

### Frontend Debugging

**Chrome DevTools:**
1. Open Chrome DevTools (F12)
2. Go to Sources tab
3. Set breakpoints in TypeScript files
4. Refresh page

**VS Code Debugger:**

Create `.vscode/launch.json`:
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "type": "chrome",
      "request": "launch",
      "name": "Launch Chrome",
      "url": "http://localhost:5173",
      "webRoot": "${workspaceFolder}/frontend/src"
    }
  ]
}
```

### Database Debugging

```bash
# Connect to SQLite database
sqlite3 backend/data/isolate-panel.db

# List tables
.tables

# Query users
SELECT * FROM users LIMIT 10;

# Check settings
SELECT * FROM settings;
```

---

## Common Issues

### Backend Issues

**Issue: "package not found"**
```bash
# Solution: Download dependencies
go mod download
go mod tidy
```

**Issue: "database locked"**
```bash
# Solution: Close other connections
# Or remove database file (development only)
rm backend/data/isolate-panel.db
```

**Issue: "port already in use"**
```bash
# Solution: Change port in .env
PORT=8081

# Or kill process using port 8080
lsof -ti:8080 | xargs kill -9
```

### Frontend Issues

**Issue: "npm install fails"**
```bash
# Solution: Clear cache
npm cache clean --force
rm -rf node_modules package-lock.json
npm install
```

**Issue: "Vite dev server not starting"**
```bash
# Solution: Check Node.js version
node --version  # Should be 20.x

# Reinstall dependencies
rm -rf node_modules
npm install
```

**Issue: "TypeScript errors"**
```bash
# Solution: Run type check
npm run type-check

# Fix common issues
npm run lint -- --fix
```

### Docker Issues

**Issue: "container won't start"**
```bash
# Solution: Check logs
docker compose logs

# Solution: Rebuild image
docker compose build --no-cache
docker compose up -d
```

**Issue: "permission denied"**
```bash
# Solution: Fix permissions
sudo chown -R $(whoami) ./data
```

---

## Development Workflow

### Daily Workflow

1. **Pull latest changes**
   ```bash
   git pull origin main
   ```

2. **Install dependencies**
   ```bash
   cd backend && go mod download
   cd frontend && npm install
   ```

3. **Run tests**
   ```bash
   cd backend && go test ./...
   cd frontend && npm run test
   ```

4. **Start development servers**
   ```bash
   # Terminal 1: Backend
   cd backend && go run cmd/server/main.go
   
   # Terminal 2: Frontend
   cd frontend && npm run dev
   ```

5. **Make changes and test**
   - Edit code
   - Run tests
   - Commit changes

### Git Workflow

```bash
# Create feature branch
git checkout -b feat/your-feature

# Make changes
git add .
git commit -m "feat: add your feature"

# Push to remote
git push origin feat/your-feature

# Create pull request on GitHub
```

---

## Performance Tips

### Backend

- Use connection pooling for database
- Enable query logging in development only
- Use caching for frequently accessed data
- Profile with `go tool pprof`

### Frontend

- Use React.memo for expensive components
- Lazy load large components
- Optimize bundle size with `npm run build -- --analyze`
- Use production build for testing

---

**Development Guide Version:** 0.1.0  
**Last Updated:** March 2026
