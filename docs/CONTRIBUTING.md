# Contributing to Isolate Panel

---

## Table of Contents

1. [Development Setup](#development-setup)
2. [Project Structure](#project-structure)
3. [Code Style](#code-style)
4. [Commit Messages](#commit-messages)
5. [Pull Request Process](#pull-request-process)
6. [Testing](#testing)
7. [Adding a New API Endpoint](#adding-a-new-api-endpoint)

---

## Development Setup

### Prerequisites

| Tool | Version | Notes |
|---|---|---|
| Go | 1.25+ | CGO required (SQLite via `go-sqlite3`) |
| Node.js | 22.x | For frontend |
| gcc / libc6-dev | any | CGO build dependency |
| Docker | 20.10+ | For containerized dev |

### Quickstart (Docker — recommended)

```bash
git clone https://github.com/isolate-project/isolate-panel.git
cd isolate-panel/docker
cp .env.example .env       # set JWT_SECRET and ADMIN_PASSWORD
docker-compose -f docker-compose.dev.yml up --build
```

- Backend hot-reloads via [Air](https://github.com/air-verse/air) on `:8080`
- Frontend dev server (Vite) on `:5173` — proxies `/api` to `:8080`

### Manual setup

```bash
# Backend
cd backend
go mod download
make run                  # starts server on :8080

# Frontend (separate terminal)
cd frontend
npm install
npm run dev               # starts on :5173
```

Access the panel via SSH tunnel (see README).

---

## Project Structure

```
isolate-panel/
├── backend/
│   ├── internal/api/       # Add new handlers here
│   ├── internal/services/  # Add new business logic here
│   ├── internal/models/    # GORM models
│   └── internal/database/migrations/  # SQL migrations (append-only)
├── frontend/
│   ├── src/pages/          # Add new page components here
│   ├── src/hooks/          # Data-fetching hooks
│   └── src/api/endpoints/  # API client functions
├── docs/                   # Documentation
└── cli/                    # CLI tool (separate go.mod)
```

---

## Code Style

### Backend (Go)

- Follow standard Go formatting: `gofmt`, `goimports`
- Run `make lint` before submitting — must pass with zero warnings
- Linters active: `govet`, `errcheck`, `staticcheck`, `gosec`, `misspell`, `prealloc`
- Fiber v3 context type is `fiber.Ctx` (value, not pointer — different from v2)
- Always handle errors; don't use `_` for errors in production paths
- Keep handlers thin — business logic belongs in `services/`

### Frontend (TypeScript / Preact)

- `npm run typecheck` must pass (`tsc --noEmit`)
- `npm run lint` must pass with zero warnings (`--max-warnings 0`)
- Use Preact imports (`preact`, `preact/hooks`, `preact/compat`), not React
- No `any` types — `@typescript-eslint/no-explicit-any` is set to `error`
- New pages must be lazy-loaded via `preact/compat`'s `lazy()`:
  ```tsx
  const MyPage = lazy(() => import('./pages/MyPage').then(m => ({ default: m.MyPage })))
  ```

---

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short summary>

[optional body]
```

**Types:** `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `perf`

**Examples:**

```
feat(users): add bulk deactivation endpoint
fix(websocket): handle disconnect race condition
refactor(app): extract route registration to routes.go
test(api): add coverage for TOTP setup flow
docs(arch): update architecture diagram for v1.0.0
```

---

## Pull Request Process

1. Fork the repository and create a feature branch from `master`
2. Write or update tests for your changes
3. Ensure all checks pass:
   ```bash
   # Backend
   cd backend && make test && make lint

   # Frontend
   cd frontend && npm run typecheck && npm run lint && npm test
   ```
4. Add a clear PR description explaining **what** changed and **why**
5. Reference any related issues: `Closes #123`
6. Keep PRs focused — one feature or fix per PR

### Database migrations

- Never modify existing migration files
- Add a new numbered migration file: `000030_description.up.sql`
- There is no down migration support (append-only)
- Test the migration on a fresh database before submitting

---

## Testing

### Backend tests

```bash
cd backend

make test                                    # all tests
go test -v -run TestFoo ./internal/api/...   # single test
go test ./... -coverprofile=coverage.out     # coverage report
```

Tests are located:
- `internal/*/` — unit tests colocated with source
- `tests/integration/` — integration tests (require DB)
- `tests/e2e/` — end-to-end flow tests
- `tests/edgecases/` — edge cases

### Frontend tests

```bash
cd frontend

npm test              # Vitest (unit + component tests, jsdom env)
npm run test:e2e      # Playwright (Chromium, requires running backend)
```

### Writing a new backend test

```go
// File: internal/api/my_handler_test.go
package api_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestMyHandler_Create(t *testing.T) {
    // use in-memory SQLite: ":memory:"
    // use fiber.New() + app.Test()
}
```

---

## Adding a New API Endpoint

### 1. Add the handler

In `internal/api/myfeature.go`:

```go
// MyHandler handles my feature
//
// @Summary      Do something
// @Tags         myfeature
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /myfeature [get]
// @Security     BearerAuth
func (h *MyHandler) DoSomething(c fiber.Ctx) error {
    return c.JSON(fiber.Map{"ok": true})
}
```

### 2. Register the route

In `internal/app/routes.go`:

```go
protected.Get("/myfeature", a.MyH.DoSomething)
```

### 3. Update Swagger docs

```bash
cd backend
~/go/bin/swag init -g cmd/server/main.go -o docs/swagger --parseDependency --parseInternal
```

### 4. Add the frontend API call

In `frontend/src/api/endpoints/index.ts`:

```ts
export const myFeatureApi = {
  doSomething: () => apiClient.get('/myfeature'),
}
```

### 5. Write a test

Add `internal/api/myfeature_test.go` with at least a happy-path test.
