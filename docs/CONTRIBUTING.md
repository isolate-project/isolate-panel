# Contributing to Isolate Panel

Thank you for your interest in contributing to Isolate Panel! This document provides guidelines and instructions for contributing.

---

## Table of Contents

1. [Getting Started](#getting-started)
2. [Development Setup](#development-setup)
3. [Code Style](#code-style)
4. [Commit Messages](#commit-messages)
5. [Pull Request Process](#pull-request-process)
6. [Issue Reporting](#issue-reporting)
7. [Code Review Guidelines](#code-review-guidelines)

---

## Getting Started

### Ways to Contribute

- **Bug Reports**: Found a bug? Open an issue
- **Feature Requests**: Have an idea? Open an issue
- **Documentation**: Improve docs, fix typos
- **Code**: Fix bugs, implement features
- **Testing**: Write tests, test new features
- **Translations**: Add or improve i18n translations

### First Time Contributors

1. Look for issues labeled `good first issue` or `help wanted`
2. Fork the repository
3. Create a branch
4. Make your changes
5. Submit a pull request

---

## Development Setup

### Prerequisites

- **Go**: 1.23 or later
- **Node.js**: 20.x or later
- **Git**: Latest version
- **Docker**: 20.10+ (optional, for containerized development)

### Backend Setup

```bash
# Clone repository
git clone https://github.com/your-org/isolate-panel.git
cd isolate-panel

# Navigate to backend
cd backend

# Install dependencies
go mod download

# Run server
go run cmd/server/main.go
```

### Frontend Setup

```bash
# Navigate to frontend
cd frontend

# Install dependencies
npm install

# Run dev server
npm run dev
```

### Running Tests

```bash
# Backend tests
cd backend
go test ./...

# Frontend tests
cd frontend
npm run test
```

---

## Code Style

### Go Code Style

Follow [Effective Go](https://golang.org/doc/effective_go.html) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

**Formatting:**
```bash
# Format code
go fmt ./...

# Lint code
go vet ./...
```

**Naming Conventions:**
- Use `CamelCase` for exported names
- Use `lowercase` for unexported names
- Use `ALL_CAPS` for constants
- Acronyms: `UUID`, `HTTP`, `JWT` (not `Uuid`, `Http`, `Jwt`)

**Example:**
```go
// Good
type UserService struct {
    db *gorm.DB
}

func (us *UserService) CreateUser(req *CreateUserRequest) (*models.User, error) {
    // Implementation
}

// Bad
type user_service struct {  // Should be UserService
    DB *gorm.DB  // Should be db (unexported)
}
```

### Frontend Code Style

**TypeScript/TSX:**
- Use TypeScript for all new code
- Use functional components with hooks
- Use arrow functions for component definitions

**Example:**
```tsx
// Good
export function Users() {
  const [users, setUsers] = useState<User[]>([])
  
  const handleDelete = async (id: number) => {
    // Implementation
  }
  
  return <div>...</div>
}

// Bad
const Users = class extends Component {  // Use functional component
  render() {
    return <div>...</div>
  }
}
```

**CSS/Tailwind:**
- Use Tailwind utility classes
- Avoid custom CSS when possible
- Use consistent spacing scale

---

## Commit Messages

### Conventional Commits

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.

**Format:**
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Build/config changes

### Examples

```bash
# Feature
feat(phase12): add production Dockerfile

# Bug fix
fix(auth): resolve JWT token expiration issue

# Documentation
docs: update API reference with new endpoints

# Tests
test(phase13): add UserService unit tests

# Refactor
refactor(services): extract notification logic to separate service
```

### Commit Guidelines

- Keep subject line under 72 characters
- Use imperative mood ("add" not "added")
- Don't end subject line with period
- Reference issues/PRs in body when applicable

---

## Pull Request Process

### Before Submitting

1. **Fork the repository**
2. **Create a branch**:
   ```bash
   git checkout -b feat/your-feature-name
   ```
3. **Make changes**
4. **Run tests**:
   ```bash
   # Backend
   go test ./...
   
   # Frontend
   npm run test
   ```
5. **Format code**:
   ```bash
   # Backend
   go fmt ./...
   
   # Frontend
   npm run format
   ```
6. **Update documentation** if needed
7. **Commit changes** with conventional commits

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests pass
- [ ] New tests added (if applicable)

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings
```

### PR Review

1. **Label PR** appropriately
2. **Request reviews** from maintainers
3. **Address feedback** promptly
4. **Keep PR size** reasonable (< 400 lines preferred)
5. **Resolve conflicts** with main branch

---

## Issue Reporting

### Bug Reports

**Template:**
```markdown
**Describe the bug**
Clear description of the bug

**To Reproduce**
Steps to reproduce:
1. Go to '...'
2. Click on '...'
3. See error

**Expected behavior**
What should happen

**Screenshots**
If applicable

**Environment:**
- OS: [e.g., Ubuntu 22.04]
- Go version: [e.g., 1.23]
- Node version: [e.g., 20.x]
- Browser: [e.g., Chrome 120]

**Additional context**
Any other details
```

### Feature Requests

**Template:**
```markdown
**Is your feature request related to a problem?**
Clear description

**Describe the solution you'd like**
What you want to happen

**Describe alternatives you've considered**
Other solutions you've thought about

**Additional context**
Any other details
```

---

## Code Review Guidelines

### For Reviewers

**Be constructive:**
- Suggest, don't command
- Explain reasoning
- Acknowledge good code

**Review checklist:**
- [ ] Code follows style guidelines
- [ ] Tests are included (if applicable)
- [ ] Documentation is updated
- [ ] No security issues introduced
- [ ] Performance impact considered
- [ ] Backward compatibility maintained

### For Contributors

**Responding to feedback:**
- Be professional and courteous
- Ask for clarification if needed
- Make requested changes promptly
- Push updates to the same branch

---

## Development Workflow

### Branch Naming

- `feat/feature-name` - New features
- `fix/bug-name` - Bug fixes
- `docs/document-name` - Documentation
- `refactor/component-name` - Refactoring
- `test/component-name` - Tests

### Release Process

1. **Version bump** (following SemVer)
2. **Update CHANGELOG.md**
3. **Create release branch**
4. **Final testing**
5. **Merge to main**
6. **Create Git tag**
7. **Build and publish**

---

## Questions?

- **General questions**: Open a GitHub Discussion
- **Bug reports**: Open an Issue
- **Chat**: Join our community (if applicable)

---

**Thank you for contributing to Isolate Panel!** 🎉

---

**Last Updated:** March 2026  
**Version:** 0.1.0
