# Testing Guide

This document describes the testing infrastructure for Isolate Panel.

## Overview

Isolate Panel uses a multi-layered testing approach:

1. **Backend Unit Tests** - Go tests for services, handlers, and utilities
2. **Frontend Component Tests** - Vitest + Testing Library for UI components
3. **E2E Tests** - Playwright for browser-based end-to-end testing
4. **CLI Tests** - Smoke and integration tests for CLI commands

## Backend Tests

### Run All Backend Tests

```bash
cd backend
go test ./... -v
```

### Run Specific Test Suites

```bash
# Unit tests for services
go test ./tests/unit/services/... -v

# API handler tests
go test ./tests/unit/api/... -v

# Integration tests
go test ./tests/integration/... -v

# E2E tests
go test ./tests/e2e/... -v
```

### Test Coverage

```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# View coverage in terminal
go test ./... -cover
```

### Test Structure

```
backend/tests/
├── testutil/           # Test utilities
│   ├── database.go     # In-memory SQLite setup
│   ├── fixtures.go     # Test data loaders
│   └── assertions.go   # Custom assertions
├── fixtures/           # JSON test data
├── unit/              # Unit tests
│   ├── services/      # Service layer tests
│   └── api/           # Handler tests
├── integration/       # Integration tests
└── e2e/              # End-to-end tests
```

## Frontend Tests

### Run Component Tests

```bash
cd frontend
npm run test
```

### Run Tests in Watch Mode

```bash
npm run test -- --watch
```

### Run Tests with Coverage

```bash
npm run test:coverage
```

### Run E2E Tests

```bash
# Run all E2E tests
npm run test:e2e

# Run with UI
npm run test:e2e:ui

# Show HTML report
npm run test:e2e:report
```

### Test Structure

```
frontend/
├── src/
│   ├── test/
│   │   └── setup.ts       # Test setup and mocks
│   └── components/
│       └── ui/
│           ├── Button.test.tsx
│           ├── Card.test.tsx
│           └── ...
└── e2e/
    ├── auth.spec.ts       # Authentication tests
    ├── dashboard.spec.ts  # Dashboard tests
    ├── users.spec.ts      # User management tests
    ├── settings.spec.ts   # Settings tests
    └── inbounds.spec.ts   # Inbounds tests
```

### Mocking

The test setup (`src/test/setup.ts`) provides automatic mocks for:

- **i18next** - Returns translation keys as-is
- **zustand stores** - Returns mock state
- **lucide-preact icons** - Returns null components

## CLI Tests

### Run All CLI Tests

```bash
cd cli
go test ./... -v
```

### Run Smoke Tests

```bash
go test ./cmd/smoke_test.go -v
```

### Run Integration Tests

```bash
go test ./cmd/integration_test.go -v
```

### Test Structure

```
cli/
├── cmd/
│   ├── smoke_test.go       # Command existence tests
│   └── integration_test.go # API integration tests
└── pkg/
    └── config_test.go      # Config tests (if any)
```

## CI/CD Integration

Tests are automatically run on GitHub Actions on every push and PR.

### Workflow Jobs

1. **backend-tests** - Runs all backend tests
2. **frontend-tests** - Runs frontend component tests
3. **e2e-tests** - Runs Playwright E2E tests
4. **cli-tests** - Runs CLI smoke and integration tests
5. **build-verify** - Verifies all builds complete successfully
6. **lint** - Runs linters on all code

### Running CI Locally

You can run the same checks locally:

```bash
# Backend
cd backend && go test ./... && go build ./...

# Frontend
cd frontend && npm run test -- --run && npm run build

# CLI
cd cli && go test ./... && go build ./...

# Linting
cd backend && golangci-lint run
cd frontend && npm run lint
```

## Writing Tests

### Backend Test Example

```go
func TestUserService_CreateUser(t *testing.T) {
    db := testutil.SetupTestDB(t)
    defer testutil.TeardownTestDB(t, db)

    service := services.NewUserService(db, nil)

    req := &services.CreateUserRequest{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "password123",
    }

    user, err := service.CreateUser(req, 1)

    require.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "testuser", user.Username)
}
```

### Frontend Component Test Example

```tsx
import { render, screen } from '@testing-library/preact'
import { Button } from './Button'

test('renders button with text', () => {
  render(<Button>Click me</Button>)
  
  expect(screen.getByText('Click me')).toBeInTheDocument()
})

test('calls onClick when clicked', () => {
  const handleClick = vi.fn()
  render(<Button onClick={handleClick}>Click</Button>)
  
  fireEvent.click(screen.getByText('Click'))
  expect(handleClick).toHaveBeenCalledTimes(1)
})
```

### E2E Test Example

```typescript
import { test, expect } from '@playwright/test'

test('should login with valid credentials', async ({ page }) => {
  await page.goto('/')
  
  // Mock API
  await page.route('**/api/auth/login', async route => {
    await route.fulfill({
      status: 200,
      body: JSON.stringify({ success: true, token: 'mock-token' }),
    })
  })
  
  await page.fill('input[type="email"]', 'admin@example.com')
  await page.fill('input[type="password"]', 'password123')
  await page.click('button[type="submit"]')
  
  await expect(page).toHaveURL(/\/dashboard/)
})
```

## Test Coverage Targets

| Component | Target | Current |
|-----------|--------|---------|
| Backend Services | >80% | ~85% |
| Backend API Handlers | >80% | ~90% |
| Frontend Components | >80% | ~85% |
| E2E Critical Flows | 100% | ~100% |
| CLI Commands | >70% | ~75% |

## Troubleshooting

### Backend Tests Fail

1. **Database locked**: Ensure no other tests are running
2. **Port conflicts**: Check if test server ports are available
3. **Missing dependencies**: Run `go mod download`

### Frontend Tests Fail

1. **Icon mock errors**: Check `src/test/setup.ts` for icon mocks
2. **i18n errors**: Verify `react-i18next` mock is correct
3. **Store errors**: Check zustand store mocks

### E2E Tests Fail

1. **Browser not found**: Run `npx playwright install`
2. **Port conflicts**: Ensure dev server is not running on port 3000
3. **Timeout errors**: Increase timeout in `playwright.config.ts`

### CLI Tests Fail

1. **Config path issues**: Check temp directory permissions
2. **Mock server errors**: Verify HTTP mock responses
3. **Import cycles**: Ensure test files don't import main package

## Best Practices

1. **Use test utilities** - Always use `testutil.SetupTestDB()` for backend tests
2. **Mock external dependencies** - Never call real APIs in unit tests
3. **Clean up resources** - Always defer cleanup functions
4. **Test edge cases** - Test error conditions and boundary values
5. **Keep tests independent** - Tests should not depend on each other
6. **Use descriptive names** - Test names should describe the expected behavior

## Additional Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Vitest Documentation](https://vitest.dev/)
- [Playwright Documentation](https://playwright.dev/)
- [Testing Library](https://testing-library.com/)
