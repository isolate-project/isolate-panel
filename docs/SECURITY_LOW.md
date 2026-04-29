# Security Vulnerability Solutions - LOW Severity

This document provides comprehensive solutions for LOW severity security vulnerabilities identified in the codebase.

---

## Table of Contents

1. [VULNERABILITY 16: Verbose Error Messages Leak Internals](#vulnerability-16-verbose-error-messages-leak-internals)
2. [VULNERABILITY 17: Missing Content-Type Validation](#vulnerability-17-missing-content-type-validation)
3. [VULNERABILITY 18: Missing API Versioning](#vulnerability-18-missing-api-versioning)
4. [VULNERABILITY 19: Hardcoded Timeouts](#vulnerability-19-hardcoded-timeouts)
5. [VULNERABILITY 20: Missing Request Size Limits](#vulnerability-20-missing-request-size-limits)
6. [VULNERABILITY 21: Log Injection via User Input](#vulnerability-21-log-injection-via-user-input)

---

## VULNERABILITY 16: Verbose Error Messages Leak Internals

### Overview

| Attribute | Value |
|-----------|-------|
| **Severity** | LOW |
| **Affected File** | `api/middleware/error.go` |
| **CWE** | CWE-209: Information Exposure Through an Error Message |
| **CVSS 3.1** | 3.7 (Low) |

### Current Vulnerable Code

```go
// api/middleware/error.go - VULNERABLE
func ErrorHandler(c *fiber.Ctx, err error) error {
    return c.Status(500).JSON(fiber.Map{
        "error": err.Error(),  // ← EXPOSES INTERNAL DETAILS
    })
}
```

### Deep Root Cause Analysis

#### The Information Disclosure Chain

When an error occurs in a Go application, the `error` interface's `Error()` method returns a string that often contains:

1. **Stack Traces**: `runtime/debug.Stack()` includes file paths like `/home/deploy/app/internal/db/user.go:47`
2. **Database Errors**: PostgreSQL errors reveal schema: `pq: relation "users" does not exist`
3. **File System Paths**: `open /etc/app/config.yml: no such file or directory`
4. **Internal Architecture**: Package names, function names, internal IP addresses
5. **Third-Party Library Details**: Version numbers, internal implementation details

#### Why This Happens

```go
// Database layer leaks schema
func GetUser(id int) (*User, error) {
    row := db.QueryRow("SELECT * FROM users WHERE id = $1", id)
    err := row.Scan(&user.ID, &user.Email)
    if err != nil {
        return nil, err  // ← Raw SQL error propagates up
    }
}

// Handler exposes everything
func GetUserHandler(c *fiber.Ctx) error {
    user, err := db.GetUser(id)
    if err != nil {
        return err  // ← Database error goes straight to client
    }
}
```

#### Attack Scenarios

**Scenario 1: Reconnaissance**
```
Attacker sends: GET /api/users/999999999999999999999
Response: {"error": "pq: value \"999999999999999999999\" is out of range for type integer"}
→ Attacker learns: PostgreSQL, integer type, no input validation
```

**Scenario 2: Path Traversal Discovery**
```
Attacker sends: GET /api/files/../../../etc/passwd
Response: {"error": "open /var/www/app/../../../etc/passwd: permission denied"}
→ Attacker learns: Application runs as non-root, file structure
```

**Scenario 3: Dependency Mapping**
```
Response: {"error": "redis: connection refused to 10.0.3.15:6379"}
→ Attacker learns: Redis cache, internal network topology
```

### The Ultimate Solution

#### Error Taxonomy Architecture

Implement a 7-category error classification system with clear separation between public and internal error information:

```
┌─────────────────────────────────────────────────────────────┐
│                    Error Taxonomy                            │
├─────────────────────────────────────────────────────────────┤
│ 1. Validation    → 400 Bad Request    → Public details      │
│ 2. Auth          → 401 Unauthorized   → Generic message     │
│ 3. Permission    → 403 Forbidden      → Generic message     │
│ 4. NotFound      → 404 Not Found      → Generic message     │
│ 5. Conflict      → 409 Conflict       → Public details      │
│ 6. RateLimit     → 429 Too Many Req   → Retry-After header  │
│ 7. Internal      → 500 Server Error   → Generic message     │
└─────────────────────────────────────────────────────────────┘
```

#### Dual Error Types

```go
// PublicError - Safe to send to clients
// Contains: error code, user-friendly message, request ID for support
// Does NOT contain: stack traces, internal details, system info

type PublicError struct {
    Code       string `json:"code"`        // Machine-readable: "VALIDATION_ERROR"
    Message    string `json:"message"`     // Human-readable: "Invalid email format"
    RequestID  string `json:"request_id"`  // For support lookup: "req_abc123"
    Field      string `json:"field,omitempty"` // Optional field reference
}

// InternalError - Full diagnostic information
// Contains: stack trace, raw error, context, timestamp, severity
// NEVER sent to clients, only to logging/monitoring

type InternalError struct {
    PublicError              // Embedded for correlation
    OriginalError string      // Raw error string
    StackTrace    string      // Full stack trace
    Context       map[string]interface{} // Request context
    Timestamp     time.Time   // When it occurred
    Severity      string      // debug, info, warn, error, fatal
    Service       string      // Which service/component
}
```

### Concrete Implementation

#### Step 1: Error Taxonomy Definition

```go
// pkg/errors/taxonomy.go
package errors

import (
    "fmt"
    "net/http"
    "time"

    "github.com/google/uuid"
)

// ErrorCategory represents the 7 error categories
type ErrorCategory int

const (
    CategoryValidation ErrorCategory = iota
    CategoryAuth
    CategoryPermission
    CategoryNotFound
    CategoryConflict
    CategoryRateLimit
    CategoryInternal
)

// HTTP status codes for each category
func (c ErrorCategory) HTTPStatus() int {
    switch c {
    case CategoryValidation:
        return http.StatusBadRequest
    case CategoryAuth:
        return http.StatusUnauthorized
    case CategoryPermission:
        return http.StatusForbidden
    case CategoryNotFound:
        return http.StatusNotFound
    case CategoryConflict:
        return http.StatusConflict
    case CategoryRateLimit:
        return http.StatusTooManyRequests
    case CategoryInternal:
        return http.StatusInternalServerError
    default:
        return http.StatusInternalServerError
    }
}

// Public-facing error codes
func (c ErrorCategory) Code() string {
    switch c {
    case CategoryValidation:
        return "VALIDATION_ERROR"
    case CategoryAuth:
        return "AUTHENTICATION_ERROR"
    case CategoryPermission:
        return "PERMISSION_DENIED"
    case CategoryNotFound:
        return "RESOURCE_NOT_FOUND"
    case CategoryConflict:
        return "RESOURCE_CONFLICT"
    case CategoryRateLimit:
        return "RATE_LIMIT_EXCEEDED"
    case CategoryInternal:
        return "INTERNAL_ERROR"
    default:
        return "UNKNOWN_ERROR"
    }
}

// Default public messages (safe, generic)
func (c ErrorCategory) DefaultMessage() string {
    switch c {
    case CategoryValidation:
        return "The request contains invalid data"
    case CategoryAuth:
        return "Authentication required"
    case CategoryPermission:
        return "You don't have permission to perform this action"
    case CategoryNotFound:
        return "The requested resource was not found"
    case CategoryConflict:
        return "The request conflicts with the current state"
    case CategoryRateLimit:
        return "Too many requests, please try again later"
    case CategoryInternal:
        return "An internal error occurred. Please try again later"
    default:
        return "An error occurred"
    }
}

// Severity returns the logging severity for the error category
func (c ErrorCategory) Severity() string {
    switch c {
    case CategoryInternal:
        return "critical"
    case CategoryAuth, CategoryPermission:
        return "warning"
    case CategoryRateLimit:
        return "warning"
    default:
        return "info"
    }
}
```

#### Step 2: PublicError Implementation

```go
// pkg/errors/public.go
package errors

// PublicError is safe to expose to API clients
type PublicError struct {
    Code      string            `json:"code"`
    Message   string            `json:"message"`
    RequestID string            `json:"request_id"`
    Field     string            `json:"field,omitempty"`
    Details   map[string]string `json:"details,omitempty"`
}

func (e PublicError) Error() string {
    return fmt.Sprintf("[%s] %s (request: %s)", e.Code, e.Message, e.RequestID)
}

// NewPublicError creates a public-safe error
func NewPublicError(category ErrorCategory, requestID string) *PublicError {
    return &PublicError{
        Code:      category.Code(),
        Message:   category.DefaultMessage(),
        RequestID: requestID,
    }
}

// WithMessage overrides the default message (use carefully)
func (e *PublicError) WithMessage(msg string) *PublicError {
    // Sanitize message - no internal details
    e.Message = sanitizePublicMessage(msg)
    return e
}

// WithField adds field context for validation errors
func (e *PublicError) WithField(field string) *PublicError {
    e.Field = field
    return e
}

// WithDetails adds safe details (validated)
func (e *PublicError) WithDetails(details map[string]string) *PublicError {
    e.Details = sanitizeDetails(details)
    return e
}

import "strings"

// sanitizePublicMessage ensures no internal info leaks
func sanitizePublicMessage(msg string) string {
    // Remove patterns that indicate internal details
    internalPatterns := []string{
        "pq:",           // PostgreSQL
        "sql:",          // SQL errors
        "dial tcp",      // Network errors
        "connection",    // Connection errors
        "timeout",       // Timeout details
        "/home/",        // File paths
        "/var/",
        "/etc/",
        ".go:",          // Source file references
        "goroutine",     // Stack traces
        "runtime.",
    }
    
    // If message contains internal patterns, return generic message
    for _, pattern := range internalPatterns {
        if contains(msg, pattern) {
            return "An error occurred processing your request"
        }
    }
    
    // Limit length
    if len(msg) > 200 {
        return msg[:200] + "..."
    }
    
    return msg
}

func sanitizeDetails(details map[string]string) map[string]string {
    sanitized := make(map[string]string)
    for k, v := range details {
        // Only allow safe detail keys
        safeKeys := map[string]bool{
            "field": true,
            "value": true,
            "constraint": true,
        }
        if safeKeys[k] {
            sanitized[k] = sanitizePublicMessage(v)
        }
    }
    return sanitized
}
```

#### Step 3: InternalError Implementation

```go
// pkg/errors/internal.go
package errors

import (
    "fmt"
    "runtime/debug"
    "time"
)

// InternalError contains full diagnostic information
// NEVER expose this to clients
type InternalError struct {
    PublicError              // Embedded for correlation
    
    OriginalError string                 `json:"-"` // Raw error
    StackTrace    string                 `json:"-"` // Full stack
    Context       map[string]interface{} `json:"-"` // Request context
    Timestamp     time.Time              `json:"-"`
    Severity      string                 `json:"-"`
    Service       string                 `json:"-"`
    Endpoint      string                 `json:"-"`
    UserID        string                 `json:"-"`
    IPAddress     string                 `json:"-"`
}

// NewInternalError wraps any error with full context
func NewInternalError(err error, category ErrorCategory, requestID string, ctx *fiber.Ctx) *InternalError {
    internal := &InternalError{
        PublicError:   *NewPublicError(category, requestID),
        OriginalError: err.Error(),
        StackTrace:    string(debug.Stack()),
        Context:       extractContext(ctx),
        Timestamp:     time.Now().UTC(),
		Severity:      category.Severity(),
		Service:       "isolate-panel",
    }
    
    if ctx != nil {
		internal.Endpoint = ctx.Path()
		internal.IPAddress = ctx.IP()
		if uid, ok := ctx.Locals("user_id").(uint); ok {
			internal.UserID = fmt.Sprintf("%d", uid)
		}
    }
    
    return internal
}

// Log sends to secure logging system
func (e *InternalError) Log(logger zerolog.Logger) {
    event := logger.Error().
        Str("error_code", e.Code).
        Str("request_id", e.RequestID).
        Str("original_error", e.OriginalError).
        Str("stack_trace", e.StackTrace).
        Str("endpoint", e.Endpoint).
        Str("user_id", e.UserID).
        Str("ip_address", e.IPAddress).
        Str("severity", e.Severity).
        Str("service", e.Service).
        Time("timestamp", e.Timestamp)
    
    // Add context
    for k, v := range e.Context {
        event = event.Interface(k, v)
    }
    
    event.Msg("Internal error occurred")
}

// SendToMonitoring sends to external monitoring (Sentry, Datadog, etc.)
// Import your monitoring SDK: "github.com/getsentry/sentry-go"
func (e *InternalError) SendToMonitoring() {
    // Example: sentry.CaptureException(fmt.Errorf("%s: %s", e.Code, e.OriginalError))
    log.Error().
        Str("code", e.Code).
        Str("original", e.OriginalError).
        Msg("Monitor: internal error")
}

func extractHeaders(ctx *fiber.Ctx) map[string]string {
    headers := make(map[string]string)
    ctx.Request().Header.VisitAll(func(key, value []byte) {
        headers[string(key)] = string(value)
    })
    return headers
}

func extractContext(ctx *fiber.Ctx) map[string]interface{} {
    if ctx == nil {
        return nil
    }
    return map[string]interface{}{
        "method":     ctx.Method(),
        "path":       ctx.Path(),
        "headers":    sanitizeHeaders(extractHeaders(ctx)),
        "query":      ctx.Queries(),
    }
}

func sanitizeHeaders(headers map[string]string) map[string]string {
    // Remove sensitive headers from logging
    sensitive := map[string]bool{
        "authorization": true,
        "cookie":        true,
        "x-api-key":     true,
    }
    
    sanitized := make(map[string]string)
    for k, v := range headers {
		if sensitive[strings.ToLower(k)] {
            sanitized[k] = "[REDACTED]"
        } else {
            sanitized[k] = v
        }
    }
    return sanitized
}
```

#### Step 4: Secure Error Handler Middleware

```go
// api/middleware/error.go
package middleware

import (
    "fmt"
    "strings"
    
    "github.com/gofiber/fiber/v3"
    "github.com/google/uuid"
    "github.com/rs/zerolog"
    
    "myapp/pkg/errors"
)

// SecureErrorHandler returns safe errors to clients, logs full details
func SecureErrorHandler(logger zerolog.Logger) fiber.ErrorHandler {
    return func(c *fiber.Ctx, err error) error {
        // Generate unique request ID for this error
        requestID := c.Get("X-Request-ID")
        if requestID == "" {
            requestID = "req_" + uuid.New().String()[:8]
        }
        
        // Set request ID header for client support
        c.Set("X-Request-ID", requestID)
        
        // Categorize the error
        category := categorizeError(err)
        
        // Create internal error with full context
        internalErr := errors.NewInternalError(err, category, requestID, c)
        
        // Log full error details (secure, internal only)
        internalErr.Log(logger)
        internalErr.SendToMonitoring()
        
        // Create public-safe error response
        publicErr := createPublicError(err, category, requestID)
        
        // Return safe error to client
        return c.Status(category.HTTPStatus()).JSON(publicErr)
    }
}

// SecureNotFoundHandler returns generic 404 without leaking path information
func SecureNotFoundHandler(c *fiber.Ctx) error {
    requestID := c.Get("X-Request-ID")
    if requestID == "" {
        requestID = "req_" + uuid.New().String()[:8]
    }
    c.Set("X-Request-ID", requestID)
    
    log.Warn().
        Str("path", c.Path()). // Log internally
        Str("ip", c.IP()).
        Str("request_id", requestID).
        Msg("Not found")
    
    return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
        "error":    "The requested resource was not found",
        "code":     "RESOURCE_NOT_FOUND",
        "request_id": requestID,
    })
}

// isAuthError checks if error is an authentication error
func isAuthError(err error) bool {
	_, ok := err.(*AuthError)
	return ok
}

// isPermissionError checks if error is a permission error
func isPermissionError(err error) bool {
	_, ok := err.(*PermissionError)
	return ok
}

// isNotFoundError checks if error is a not-found error
func isNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// isRateLimitError checks if error is a rate limit error
func isRateLimitError(err error) bool {
	_, ok := err.(*RateLimitError)
	return ok
}

// categorizeError determines the error category
func categorizeError(err error) errors.ErrorCategory {
	// Check for specific error types
	switch {
	case isValidationError(err):
		return errors.CategoryValidation
	case isAuthError(err):
		return errors.CategoryAuth
	case isPermissionError(err):
		return errors.CategoryPermission
	case isNotFoundError(err):
		return errors.CategoryNotFound
	case isConflictError(err):
		return errors.CategoryConflict
	case isRateLimitError(err):
		return errors.CategoryRateLimit
	default:
		return errors.CategoryInternal
	}
}

// createPublicError creates safe error for client
func createPublicError(err error, category errors.ErrorCategory, requestID string) *errors.PublicError {
    publicErr := errors.NewPublicError(category, requestID)
    
    // Only validation and conflict errors get specific messages
    // All others use generic messages to prevent info leakage
    switch category {
    case errors.CategoryValidation:
        if ve, ok := err.(*ValidationError); ok {
            publicErr.WithField(ve.Field).WithMessage(ve.Message)
        }
    case errors.CategoryConflict:
        if ce, ok := err.(*ConflictError); ok {
            publicErr.WithMessage(ce.Message)
        }
    }
    
    return publicErr
}

// Custom error types for categorization
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

type ConflictError struct {
    Message string
}

func (e *ConflictError) Error() string {
    return e.Message
}

type AuthError struct {
    Message string
}

func (e *AuthError) Error() string {
    return e.Message
}

type PermissionError struct {
    Message string
}

func (e *PermissionError) Error() string {
    return e.Message
}

type NotFoundError struct {
    Resource string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("resource not found: %s", e.Resource)
}

type RateLimitError struct {
    RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
    return fmt.Sprintf("rate limit exceeded, retry after %v", e.RetryAfter)
}
```

#### Step 5: Application Integration

```go
// cmd/server.go
func main() {
    app := fiber.New(fiber.Config{
        // Replace default error handler with secure one
        ErrorHandler: middleware.SecureErrorHandler(logger),
    })
    
    // Replace default not found handler to prevent path leakage
    app.Use(func(c *fiber.Ctx) error {
        if c.Path() == "/" {
            return c.Next()
        }
        return middleware.SecureNotFoundHandler(c)
    })
    
    // ... rest of setup
}
```

### Migration Path

#### Phase 1: Infrastructure (Week 1)

1. **Create error taxonomy package**
   ```bash
   mkdir -p pkg/errors
   touch pkg/errors/taxonomy.go pkg/errors/public.go pkg/errors/internal.go
   ```

2. **Implement error types without changing handlers**
   - Build the complete error system
   - Write comprehensive tests
   - Document the taxonomy

3. **Add middleware alongside existing handler**
   ```go
   // Temporary dual-handler setup
   app.Use(middleware.SecureErrorHandler(logger)) // New
   // Keep old handler as fallback during testing
   ```

#### Phase 2: Handler Updates (Week 2)

1. **Update database layer** to wrap errors:
   ```go
   // Before
   return nil, err
   
   // After
   if err != nil {
       return nil, fmt.Errorf("database query failed: %w", err)
   }
   ```

2. **Update service layer** to categorize errors:
   ```go
   // Before
   return nil, errors.New("user not found")
   
   // After
   return nil, &errors.NotFoundError{Resource: "user", ID: id}
   ```

3. **Update API handlers** to return typed errors:
   ```go
   // Before
   return err
   
   // After
   if err != nil {
       return categorizeAndWrap(err)
   }
   ```

#### Phase 3: Testing & Validation (Week 3)

1. **Security testing**:
   ```bash
   # Test that internal details don't leak
   curl -s http://localhost:3000/api/users/invalid | jq .
   # Should NOT contain: pq:, sql:, file paths, stack traces
   ```

2. **Verify logging**:
   ```bash
   # Check logs contain full details
   tail -f logs/app.log | grep "Internal error"
   # Should contain: stack traces, original errors, context
   ```

3. **Load testing** to ensure no performance regression

#### Phase 4: Production Rollout (Week 4)

1. **Canary deployment** (5% traffic)
2. **Monitor error rates** - should stay the same
3. **Monitor log volume** - will increase (expected)
4. **Full rollout** after 48 hours of stability

### Why This Is Better

| Aspect | Before | After |
|--------|--------|-------|
| **Security** | Internal details exposed | Zero information leakage |
| **Debugging** | No request correlation | Request ID links client to logs |
| **Monitoring** | Raw errors in logs | Structured, categorized errors |
| **Client Experience** | Confusing technical errors | Clear, actionable messages |
| **Compliance** | Violates security standards | Meets SOC2, ISO27001 requirements |
| **Support** | Can't trace client issues | Request ID enables full traceability |

#### Security Improvements

1. **Zero Information Leakage**: Clients never see stack traces, SQL errors, or file paths
2. **Consistent Interface**: All errors follow the same JSON schema
3. **Request Correlation**: `X-Request-ID` header allows support to find full error context
4. **Categorized Responses**: HTTP status codes are always appropriate

#### Operational Improvements

1. **Structured Logging**: Errors are JSON-formatted for log aggregation
2. **Automatic Monitoring**: Internal errors automatically sent to Sentry/Datadog
3. **Context Preservation**: Full request context captured for debugging
4. **Severity Classification**: Errors automatically prioritized by severity

#### Developer Experience

1. **Clear Taxonomy**: 7 categories cover all error scenarios
2. **Type Safety**: Compiler enforces proper error handling
3. **Easy Testing**: Can assert on error categories, not strings
4. **Documentation**: Error codes are self-documenting

---

## VULNERABILITY 17: Missing Content-Type Validation

### Overview

| Attribute | Value |
|-----------|-------|
| **Severity** | LOW |
| **Affected Files** | `api/*.go` |
| **CWE** | CWE-436: Interpretation Conflict |
| **CVSS 3.1** | 3.7 (Low) |

### Current Vulnerable Code

```go
// api/handlers.go - VULNERABLE
func CreateUser(c *fiber.Ctx) error {
    var user User
    // Accepts ANY Content-Type, no validation
    if err := c.BodyParser(&user); err != nil {
        return err
    }
    // ...
}
```

### Deep Root Cause Analysis

#### The Content-Type Confusion Problem

HTTP requests include a `Content-Type` header that tells the server how to interpret the request body. When servers don't validate this header, multiple attack vectors open:

```
┌─────────────────────────────────────────────────────────────┐
│              Content-Type Confusion Attacks                  │
├─────────────────────────────────────────────────────────────┤
│ 1. CSRF Bypass: text/plain bypasses preflight checks        │
│ 2. Parser Confusion: Same body parsed differently           │
│ 3. Security Policy Bypass: CORS/simple request tricks     │
│ 4. Cache Poisoning: Different interpretations by cache      │
│ 5. Request Smuggling: Content-Length vs Transfer-Encoding   │
└─────────────────────────────────────────────────────────────┘
```

#### CSRF Bypass via text/plain

Browsers treat `text/plain` as a "simple" content type that doesn't trigger CORS preflight:

```html
<!-- Attacker's page -->
<form action="https://api.example.com/api/users" method="POST" enctype="text/plain">
    <input name='{"admin": true, "ignore": "' value='"test"}'>
    <!-- Results in body: {"admin": true, "ignore": "="test"}" -->
</form>
<script>document.forms[0].submit()</script>
```

If the API accepts `text/plain` and tries to parse it as JSON, the attacker can inject JSON structures through form submissions.

#### Parser Confusion Example

```go
// Same body, different Content-Types, different results

// Content-Type: application/json
// Body: {"role": "user"}
// Result: User{Role: "user"}

// Content-Type: application/x-www-form-urlencoded  
// Body: role=user&role=admin  ← duplicate keys!
// Result: Depends on parser - might take last value = "admin"
```

#### Attack Scenarios

**Scenario 1: Privilege Escalation via CSRF**
```
1. User is logged into api.example.com with admin cookie
2. User visits attacker.com
3. Attacker submits form with text/plain to POST /api/users
4. Server accepts text/plain, parses as JSON
5. Attacker injects {"role": "admin"} through form field manipulation
6. User created with admin privileges
```

**Scenario 2: API Version Confusion**
```
Content-Type: application/vnd.api+json; version=2
// Server ignores version, parses as v1
// v2 fields ignored, v1 defaults applied
// Data loss or unexpected behavior
```

**Scenario 3: Security Header Bypass**
```
Content-Type: application/json; charset=utf-8
// Server checks for "application/json" exactly
// Fails check due to charset suffix
// Falls back to less secure parser
```

### The Ultimate Solution

#### Strict Content-Type Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           Content-Type Validation Pipeline                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. EARLY REJECTION                                          │
│     ├── Check Content-Length (if present)                    │
│     ├── Check Content-Type header exists                     │
│     └── 411 Length Required if missing and body present      │
│                                                              │
│  2. STRICT PARSING                                           │
│     ├── Parse media type (ignore charset, boundary params)   │
│     ├── Check against whitelist                              │
│     └── 415 Unsupported Media Type if not in whitelist       │
│                                                              │
│  3. SIZE ENFORCEMENT                                         │
│     ├── Check Content-Length against limit                   │
│     ├── Stream large bodies (don't buffer entirely)          │
│     └── 413 Payload Too Large if exceeded                    │
│                                                              │
│  4. ENDPOINT-SPECIFIC RULES                                  │
│     ├── File uploads: multipart/form-data only               │
│     ├── JSON APIs: application/json only                     │
│     └── GraphQL: application/graphql only                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### Whitelist-Based Validation

```go
// Allowed Content-Types by endpoint category
var AllowedContentTypes = map[string][]string{
    "api": {
        "application/json",
        "application/json; charset=utf-8",  // Common variation
    },
    "upload": {
        "multipart/form-data",
    },
    "webhook": {
        "application/json",
        "application/x-www-form-urlencoded", // Some webhooks use this
    },
}
```

### Concrete Implementation

#### Step 1: Content-Type Middleware

```go
// api/middleware/content_type.go
package middleware

import (
    "mime"
    "net/http"
    "strconv"
    "strings"

    "github.com/gofiber/fiber/v3"
)

// ContentTypeConfig defines validation rules
type ContentTypeConfig struct {
    // AllowedTypes is the whitelist of permitted media types
    AllowedTypes []string
    
    // MaxBodySize in bytes (0 = no limit, not recommended)
    MaxBodySize int64
    
    // RequireContentType enforces header presence
    RequireContentType bool
    
    // AllowEmptyForMethods allows empty body for these methods
    AllowEmptyForMethods []string
}

// DefaultAPIConfig for JSON APIs
var DefaultAPIConfig = ContentTypeConfig{
    AllowedTypes: []string{
        "application/json",
    },
    MaxBodySize:          10 * 1024 * 1024, // 10MB
    RequireContentType:   true,
    AllowEmptyForMethods: []string{"GET", "HEAD", "DELETE", "OPTIONS"},
}

// DefaultUploadConfig for file uploads
var DefaultUploadConfig = ContentTypeConfig{
    AllowedTypes: []string{
        "multipart/form-data",
    },
    MaxBodySize:          32 * 1024 * 1024, // 32MB
    RequireContentType:   true,
    AllowEmptyForMethods: []string{},
}

// ContentTypeValidator returns middleware that enforces Content-Type rules
func ContentTypeValidator(config ContentTypeConfig) fiber.Handler {
    // Build lookup map for O(1) checks
    allowedMap := make(map[string]bool)
    for _, t := range config.AllowedTypes {
        // Normalize: lowercase, no parameters
        mediaType, _, err := mime.ParseMediaType(t)
        if err == nil {
            allowedMap[strings.ToLower(mediaType)] = true
        }
    }
    
    // Build set of methods that allow empty body
    allowEmptyMap := make(map[string]bool)
    for _, m := range config.AllowEmptyForMethods {
        allowEmptyMap[strings.ToUpper(m)] = true
    }
    
    return func(c *fiber.Ctx) error {
        method := strings.ToUpper(c.Method())
        contentType := c.Get("Content-Type")
        contentLength := c.Get("Content-Length")
        
        // Check 1: Empty body allowed for certain methods
        if contentType == "" && allowEmptyMap[method] {
            return c.Next()
        }
        
        // Check 2: Require Content-Type header
        if config.RequireContentType && contentType == "" {
            // Check if there's actually a body
            if contentLength != "" && contentLength != "0" {
                return c.Status(http.StatusLengthRequired).JSON(fiber.Map{
                    "error": "Content-Type header is required",
                    "code":  "MISSING_CONTENT_TYPE",
                })
            }
            // No body, no Content-Type needed
            return c.Next()
        }
        
        // Check 3: Parse and validate Content-Type
        mediaType, params, err := mime.ParseMediaType(contentType)
        if err != nil {
            return c.Status(http.StatusBadRequest).JSON(fiber.Map{
                "error":   "Invalid Content-Type header",
                "code":    "INVALID_CONTENT_TYPE",
                "details": err.Error(),
            })
        }
        
        // Normalize for comparison
        mediaType = strings.ToLower(mediaType)
        
        // Check 4: Whitelist validation
        if !allowedMap[mediaType] {
            return c.Status(http.StatusUnsupportedMediaType).JSON(fiber.Map{
                "error": "Unsupported Content-Type",
                "code":  "UNSUPPORTED_MEDIA_TYPE",
                "allowed_types": config.AllowedTypes,
                "received": mediaType,
            })
        }
        
        // Check 5: Validate charset if present (security check)
        if charset, ok := params["charset"]; ok {
            charset = strings.ToLower(charset)
            if charset != "utf-8" {
                return c.Status(http.StatusUnsupportedMediaType).JSON(fiber.Map{
                    "error": "Unsupported charset",
                    "code":  "UNSUPPORTED_CHARSET",
                    "allowed": "utf-8",
                    "received": charset,
                })
            }
        }
        
        // Check 6: Body size validation
        if config.MaxBodySize > 0 && contentLength != "" {
            size, err := strconv.ParseInt(contentLength, 10, 64)
            if err == nil && size > config.MaxBodySize {
                return c.Status(http.StatusRequestEntityTooLarge).JSON(fiber.Map{
                    "error": "Request body too large",
                    "code":  "PAYLOAD_TOO_LARGE",
                    "max_size": config.MaxBodySize,
                    "received": size,
                })
            }
        }
        
        // Store validated Content-Type in context for handlers
        c.Locals("validated_content_type", mediaType)
        c.Locals("content_type_params", params)
        
        return c.Next()
    }
}
```

#### Step 2: Route-Specific Configuration

```go
// api/routes.go
package api

import (
    "github.com/gofiber/fiber/v3"
    "myapp/api/middleware"
)

func SetupRoutes(app *fiber.App) {
    // API routes - strict JSON only
    api := app.Group("/api", middleware.ContentTypeValidator(middleware.DefaultAPIConfig))
    {
        api.Post("/users", CreateUser)
        api.Put("/users/:id", UpdateUser)
        api.Post("/auth/login", Login)
    }
    
    // Upload routes - multipart only
    uploads := app.Group("/uploads", middleware.ContentTypeValidator(middleware.DefaultUploadConfig))
    {
        uploads.Post("/avatar", UploadAvatar)
        uploads.Post("/documents", UploadDocument)
    }
    
    // Webhook routes - more permissive
    webhookConfig := middleware.ContentTypeConfig{
        AllowedTypes: []string{
            "application/json",
            "application/x-www-form-urlencoded",
        },
        MaxBodySize:        1 * 1024 * 1024, // 1MB
        RequireContentType: true,
    }
    webhooks := app.Group("/webhooks", middleware.ContentTypeValidator(webhookConfig))
    {
        webhooks.Post("/stripe", StripeWebhook)
        webhooks.Post("/github", GitHubWebhook)
    }
    
    // Public read-only routes - no Content-Type required
    app.Get("/health", HealthCheck)
    app.Get("/api/users/:id", GetUser) // GET doesn't need Content-Type
}
```

#### Step 3: Body Size Limits at Server Level

```go
// cmd/server.go
func main() {
    app := fiber.New(fiber.Config{
        // Global body size limit (safety net)
        BodyLimit: 10 * 1024 * 1024, // 10MB
        
        // Disable default body parsing for more control
        // We'll use middleware to handle this
        
        // Custom error handler
        ErrorHandler: middleware.SecureErrorHandler(logger),
    })
    
    // Apply Content-Type validation globally as first middleware
    app.Use(middleware.ContentTypeValidator(middleware.ContentTypeConfig{
        AllowedTypes: []string{
            "application/json",
            "multipart/form-data",
        },
        MaxBodySize:        10 * 1024 * 1024,
        RequireContentType: false, // Allow for GET/HEAD
        AllowEmptyForMethods: []string{"GET", "HEAD", "OPTIONS"},
    }))
    
    // ... rest of setup
}
```

#### Step 4: Streaming for Large Requests

```go
// api/middleware/streaming.go
package middleware

import (
    "io"
    "net/http"

    "github.com/gofiber/fiber/v3"
)

// StreamingBodyParser parses body without full memory buffering
func StreamingBodyParser(maxSize int64) fiber.Handler {
    return func(c *fiber.Ctx) error {
        contentType := c.Locals("validated_content_type")
        if contentType != "application/json" {
            return c.Next()
        }
        
        // Use LimitReader to enforce size during streaming
        limitedReader := io.LimitReader(c.Context().RequestBodyStream(), maxSize+1)
        
        // Read with size check
        body, err := io.ReadAll(limitedReader)
        if err != nil {
            return c.Status(http.StatusBadRequest).JSON(fiber.Map{
                "error": "Failed to read request body",
                "code":  "BODY_READ_ERROR",
            })
        }
        
        // Check if limit was exceeded
        if int64(len(body)) > maxSize {
            return c.Status(http.StatusRequestEntityTooLarge).JSON(fiber.Map{
                "error": "Request body exceeds maximum size",
                "code":  "PAYLOAD_TOO_LARGE",
            })
        }
        
        // Store body in context for handler
        c.Locals("request_body", body)
        
        return c.Next()
    }
}
```

#### Step 5: Per-Endpoint Size Limits

```go
// api/handlers.go
package api

import (
    "github.com/gofiber/fiber/v3"
    "myapp/api/middleware"
)

// StrictJSONLimit enforces JSON and size limit
func StrictJSONLimit(maxSize int64) fiber.Handler {
    return middleware.ContentTypeValidator(middleware.ContentTypeConfig{
        AllowedTypes:       []string{"application/json"},
        MaxBodySize:        maxSize,
        RequireContentType: true,
    })
}

// Usage in routes
func SetupRoutes(app *fiber.App) {
    // Small payload endpoint
    app.Post("/api/login", 
        StrictJSONLimit(1*1024), // 1KB - just username/password
        LoginHandler,
    )
    
    // Medium payload endpoint
    app.Post("/api/users",
        StrictJSONLimit(10*1024), // 10KB - user profile
        CreateUserHandler,
    )
    
    // Large payload endpoint
    app.Post("/api/bulk-import",
        StrictJSONLimit(5*1024*1024), // 5MB - bulk data
        BulkImportHandler,
    )
}
```

### Migration Path

#### Phase 1: Audit Current Endpoints (Week 1)

1. **Inventory all endpoints** and their expected Content-Types:
   ```bash
   # Find all POST/PUT/PATCH handlers
   grep -r "func.*Post\|func.*Put\|func.*Patch" api/
   ```

2. **Document expected Content-Types** for each endpoint
3. **Identify file upload endpoints** (need multipart)
4. **Identify webhook endpoints** (may need form-urlencoded)

#### Phase 2: Implement Middleware (Week 2)

1. **Create middleware package** with Content-Type validator
2. **Add to development environment only**
3. **Monitor for 415 errors** - these indicate clients sending wrong Content-Type
4. **Fix any legitimate clients** that are sending incorrect headers

#### Phase 3: Gradual Enforcement (Week 3)

1. **Start with non-critical endpoints**:
   ```go
   // Add middleware to specific routes first
   app.Post("/api/test-endpoint", 
       middleware.ContentTypeValidator(config),
       Handler,
   )
   ```

2. **Monitor error rates** - should see 415 errors for bad clients
3. **Update API documentation** to specify required Content-Type
4. **Notify API consumers** of the change

#### Phase 4: Global Enforcement (Week 4)

1. **Apply global middleware**:
   ```go
   app.Use(middleware.ContentTypeValidator(middleware.ContentTypeConfig{
       AllowedTypes: []string{"application/json"},
       // ...
   }))
   ```

2. **Add exceptions** for specific routes that need different types
3. **Full production rollout**

### Why This Is Better

| Aspect | Before | After |
|--------|--------|-------|
| **Security** | CSRF possible via text/plain | CSRF blocked, strict type enforcement |
| **Predictability** | Unknown parsing behavior | Explicit, documented behavior |
| **API Contract** | Implicit | Explicit Content-Type requirements |
| **Error Handling** | Confusing parse errors | Clear 415 Unsupported Media Type |
| **DoS Protection** | Unlimited body size | Configurable limits per endpoint |
| **Standards** | Non-compliant | RFC 7231 compliant |

#### Security Improvements

1. **CSRF Prevention**: `text/plain` requests rejected, preventing form-based CSRF
2. **Parser Consistency**: Same body always parsed the same way
3. **No Confusion Attacks**: Ambiguous Content-Types rejected
4. **Size Limits**: Memory exhaustion attacks prevented

#### Operational Improvements

1. **Clear Errors**: Clients get 415 with list of allowed types
2. **Early Rejection**: Invalid requests rejected before body parsing
3. **Resource Protection**: Large requests rejected before memory allocation
4. **Standards Compliance**: Follows HTTP specification correctly

#### Developer Experience

1. **Explicit Contracts**: API documentation matches enforcement
2. **Fast Feedback**: Wrong Content-Type caught immediately
3. **Debugging**: Clear error messages indicate the problem
4. **Flexibility**: Per-endpoint configuration for special cases

---

## VULNERABILITY 18: Missing API Versioning

### Overview

| Attribute | Value |
|-----------|-------|
| **Severity** | LOW |
| **Affected File** | `cmd/server.go` |
| **CWE** | CWE-1109: Use of Same Variable for Multiple Purposes |
| **CVSS 3.1** | 3.1 (Low) |

### Current Vulnerable Code

```go
// cmd/server.go - VULNERABLE
func main() {
    app := fiber.New()
    
    // All endpoints under /api/ without version
    app.Get("/api/users", GetUsers)
    app.Post("/api/users", CreateUser)
    app.Get("/api/users/:id", GetUser)
    app.Put("/api/users/:id", UpdateUser)
    app.Delete("/api/users/:id", DeleteUser)
    
    // No versioning means breaking changes break all clients
}
```

### Deep Root Cause Analysis

#### The Breaking Change Problem

Without versioning, any API change affects all clients simultaneously:

```
Timeline of API Changes (No Versioning):

Day 1:  Client A, B, C using /api/users
        Response: {"id": 1, "name": "John"}

Day 30: Developer adds email field
        Response: {"id": 1, "name": "John", "email": "john@example.com"}
        → All clients still work (additive change)

Day 60: Developer renames 'name' to 'full_name'
        Response: {"id": 1, "full_name": "John", "email": "john@example.com"}
        → Client A breaks (expects 'name')
        → Client B breaks (expects 'name')
        → Client C breaks (expects 'name')
        → Emergency rollback required

Day 61: Developer must coordinate with ALL clients
        → Weeks of delay
        → Some clients unmaintained (break permanently)
```

#### Versioning Strategy Comparison

```
┌─────────────────────────────────────────────────────────────┐
│              API Versioning Strategies                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. URL Path (Recommended)                                   │
│     /api/v1/users, /api/v2/users                             │
│     ✓ Cache-friendly, explicit, easy to test                 │
│     ✗ Slightly longer URLs                                   │
│                                                              │
│  2. Accept Header                                            │
│     Accept: application/vnd.api+json;version=2               │
│     ✓ Clean URLs, REST purist approach                       │
│     ✗ Harder to test, not cache-friendly                      │
│                                                              │
│  3. Custom Header                                            │
│     X-API-Version: 2                                         │
│     ✓ Clean URLs                                             │
│     ✗ Not visible, harder to debug                           │
│                                                              │
│  4. Query Parameter                                          │
│     /api/users?version=2                                     │
│     ✓ Simple to implement                                    │
│     ✗ Not RESTful, cache issues                              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### The Deprecation Challenge

Without versioning, there's no way to sunset old functionality:

```
Scenario: Removing deprecated endpoint /api/legacy-feature

Without Versioning:
- Must maintain forever OR break all clients
- No way to communicate timeline to clients
- No migration path

With Versioning:
- v1: /api/v1/legacy-feature (marked deprecated)
- v2: /api/v2/new-feature (replacement)
- Deprecation headers communicate timeline
- Sunset policy gives clients 6-12 months to migrate
```

### The Ultimate Solution

#### Multi-Strategy Versioning Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           Comprehensive API Versioning System                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  PRIMARY: URL Path Versioning                                │
│  /api/v1/users, /api/v2/users                                │
│  → Required, explicit, cacheable                             │
│                                                              │
│  SECONDARY: Accept Header Negotiation                        │
│  Accept: application/vnd.myapp+json;version=2               │
│  → Optional, for content-type versioning                     │
│                                                              │
│  TERTIARY: X-API-Version Header                              │
│  X-API-Version: 2                                            │
│  → Optional, for client preference                           │
│                                                              │
│  DEPRECATION: Sunset Headers                                 │
│  Deprecation: true                                           │
│  Sunset: Sat, 31 Dec 2024 23:59:59 GMT                      │
│  → Communicates lifecycle to clients                         │
│                                                              │
│  DEFAULT: Latest Stable Version                              │
│  /api/users → redirects to /api/v2/users                    │
│  → Convenience with explicit canonical URL                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### Semantic Versioning for APIs

```
API Version Format: v{major}

v1 → Initial stable release
v2 → Breaking changes (new major version)
v3 → Next breaking changes

Minor/Patch versions NOT exposed in URL
- Internal changes only
- Additive changes go to current version
- Breaking changes require new major version
```

### Concrete Implementation

#### Step 1: Version Router Middleware

```go
// api/middleware/version.go
package middleware

import (
    "fmt"
    "net/http"
    "strconv"
    "strings"

    "github.com/gofiber/fiber/v3"
)

// VersionConfig defines versioning behavior
type VersionConfig struct {
    // Current stable version
    CurrentVersion int
    
    // Minimum supported version
    MinVersion int
    
    // Maximum supported version
    MaxVersion int
    
    // Deprecated versions with sunset dates
    DeprecatedVersions map[int]DeprecationInfo
    
    // Default to current if no version specified
    DefaultToCurrent bool
}

type DeprecationInfo struct {
    SunsetDate    string // RFC 7231 format
    MigrationPath string // URL to migration guide
}

// DefaultVersionConfig for the application
var DefaultVersionConfig = VersionConfig{
    CurrentVersion: 2,
    MinVersion:     1,
    MaxVersion:     2,
    DeprecatedVersions: map[int]DeprecationInfo{
        1: {
            SunsetDate:    "Sat, 31 Dec 2024 23:59:59 GMT",
            MigrationPath: "/docs/api/migration/v1-to-v2",
        },
    },
    DefaultToCurrent: true,
}

// VersionRouter extracts and validates API version
func VersionRouter(config VersionConfig) fiber.Handler {
    return func(c *fiber.Ctx) error {
        path := c.Path()
        
        // Extract version from URL path: /api/v1/users → 1
        version := extractVersionFromPath(path)
        
        // If no version in path, check headers
        if version == 0 {
            version = extractVersionFromHeaders(c)
        }
        
        // If still no version, use default or reject
        if version == 0 {
            if config.DefaultToCurrent {
                version = config.CurrentVersion
                // Redirect to canonical URL with version
                canonicalPath := injectVersionIntoPath(path, version)
                return c.Redirect(canonicalPath, http.StatusMovedPermanently)
            }
            return c.Status(http.StatusBadRequest).JSON(fiber.Map{
                "error": "API version required",
                "code":  "VERSION_REQUIRED",
                "message": "Please specify API version in URL (/api/v2/...) or via Accept header",
            })
        }
        
        // Validate version range
        if version < config.MinVersion || version > config.MaxVersion {
            return c.Status(http.StatusNotFound).JSON(fiber.Map{
                "error": "Unsupported API version",
                "code":  "UNSUPPORTED_VERSION",
                "requested_version": version,
                "supported_versions": fmt.Sprintf("v%d-v%d", config.MinVersion, config.MaxVersion),
                "current_version": fmt.Sprintf("v%d", config.CurrentVersion),
            })
        }
        
        // Store version in context
        c.Locals("api_version", version)
        
        // Add deprecation headers if applicable
        if depInfo, isDeprecated := config.DeprecatedVersions[version]; isDeprecated {
            c.Set("Deprecation", "true")
            c.Set("Sunset", depInfo.SunsetDate)
            c.Set("Link", fmt.Sprintf("<%s>; rel=\"migration\"", depInfo.MigrationPath))
        }
        
        // Add current version info
        c.Set("X-API-Version", fmt.Sprintf("v%d", version))
        c.Set("X-API-Latest-Version", fmt.Sprintf("v%d", config.CurrentVersion))
        
        return c.Next()
    }
}

func extractVersionFromPath(path string) int {
    // Match /api/v{N}/ or /v{N}/
    if strings.HasPrefix(path, "/api/v") {
        parts := strings.Split(path, "/")
        if len(parts) >= 3 && len(parts[2]) > 1 && parts[2][0] == 'v' {
            if v, err := strconv.Atoi(parts[2][1:]); err == nil {
                return v
            }
        }
    }
    return 0
}

func extractVersionFromHeaders(c *fiber.Ctx) int {
    // Check Accept header: application/vnd.myapp+json;version=2
    accept := c.Get("Accept")
    if strings.Contains(accept, "version=") {
        parts := strings.Split(accept, "version=")
        if len(parts) > 1 {
            vStr := strings.Split(parts[1], ";")[0]
            if v, err := strconv.Atoi(vStr); err == nil {
                return v
            }
        }
    }
    
    // Check X-API-Version header
    if vStr := c.Get("X-API-Version"); vStr != "" {
        if v, err := strconv.Atoi(vStr); err == nil {
            return v
        }
    }
    
    return 0
}

func injectVersionIntoPath(path string, version int) string {
    // /api/users → /api/v2/users
    if strings.HasPrefix(path, "/api/") {
        return fmt.Sprintf("/api/v%d%s", version, path[4:])
    }
    return path
}
```

#### Step 2: Version-Specific Route Registration

```go
// api/routes.go
package api

import (
    "github.com/gofiber/fiber/v3"
    "myapp/api/handlers/v1"
    "myapp/api/handlers/v2"
    "myapp/api/middleware"
)

func SetupRoutes(app *fiber.App) {
    // Apply version router to all /api/* routes
    api := app.Group("/api", middleware.VersionRouter(middleware.DefaultVersionConfig))
    
    // Version 1 routes (deprecated but supported)
    v1 := api.Group("/v1")
    {
        v1.Get("/users", v1handlers.GetUsers)
        v1.Post("/users", v1handlers.CreateUser)
        v1.Get("/users/:id", v1handlers.GetUser)
        v1.Put("/users/:id", v1handlers.UpdateUser)
        v1.Delete("/users/:id", v1handlers.DeleteUser)
    }
    
    // Version 2 routes (current)
    v2 := api.Group("/v2")
    {
        // Enhanced user endpoints
        v2.Get("/users", v2handlers.GetUsers)
        v2.Post("/users", v2handlers.CreateUser)
        v2.Get("/users/:id", v2handlers.GetUser)
        v2.Put("/users/:id", v2handlers.UpdateUser)
        v2.Delete("/users/:id", v2handlers.DeleteUser)
        
        // New v2-only endpoints
        v2.Get("/users/:id/profile", v2handlers.GetUserProfile)
        v2.Post("/users/:id/avatar", v2handlers.UploadAvatar)
        
        // Changed response format
        v2.Get("/dashboard", v2handlers.GetDashboard)
    }
    
    // Unversioned routes (redirect to current)
    // These redirect to canonical versioned URLs
    app.Get("/api/users", func(c *fiber.Ctx) error {
        return c.Redirect("/api/v2/users", http.StatusMovedPermanently)
    })
}
```

#### Step 3: Version-Aware Handlers

```go
// api/handlers/v2/users.go
package v2handlers

import (
    "github.com/gofiber/fiber/v3"
)

// UserResponse v2 format (breaking change from v1)
type UserResponse struct {
    ID        string `json:"id"`         // Changed from int to string
    FullName  string `json:"full_name"`  // Changed from "name"
    Email     string `json:"email"`
    CreatedAt string `json:"created_at"` // ISO 8601 format
    UpdatedAt string `json:"updated_at"`
    Links     Links  `json:"_links"`     // HATEOAS links (new)
}

type Links struct {
    Self   string `json:"self"`
    Avatar string `json:"avatar"`
}

func GetUser(c *fiber.Ctx) error {
    userID := c.Params("id")
    
    // Get user from service
    user, err := userService.GetByID(c.Context(), userID)
    if err != nil {
        return err
    }
    
    // Build v2 response format
    response := UserResponse{
        ID:        user.ID,
        FullName:  user.FullName,
        Email:     user.Email,
        CreatedAt: user.CreatedAt.Format(time.RFC3339),
        UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
        Links: Links{
            Self:   fmt.Sprintf("/api/v2/users/%s", user.ID),
            Avatar: fmt.Sprintf("/api/v2/users/%s/avatar", user.ID),
        },
    }
    
    return c.JSON(response)
}
```

#### Step 4: Version Comparison Handler

```go
// api/handlers/version.go
package handlers

import (
    "github.com/gofiber/fiber/v3"
    "myapp/api/middleware"
)

// VersionInfo provides API version information
type VersionInfo struct {
    CurrentVersion     int                 `json:"current_version"`
    SupportedVersions  []int               `json:"supported_versions"`
    DeprecatedVersions []DeprecatedVersion `json:"deprecated_versions"`
}

type DeprecatedVersion struct {
    Version       int    `json:"version"`
    SunsetDate    string `json:"sunset_date"`
    MigrationPath string `json:"migration_path"`
}

func GetVersionInfo(c *fiber.Ctx) error {
    config := middleware.DefaultVersionConfig
    
    deprecated := make([]DeprecatedVersion, 0)
    for v, info := range config.DeprecatedVersions {
        deprecated = append(deprecated, DeprecatedVersion{
            Version:       v,
            SunsetDate:    info.SunsetDate,
            MigrationPath: info.MigrationPath,
        })
    }
    
    supported := make([]int, 0)
    for v := config.MinVersion; v <= config.MaxVersion; v++ {
        supported = append(supported, v)
    }
    
    return c.JSON(VersionInfo{
        CurrentVersion:     config.CurrentVersion,
        SupportedVersions:  supported,
        DeprecatedVersions: deprecated,
    })
}
```

#### Step 5: Sunset Policy Implementation

```go
// internal/version/sunset.go
package version

import (
    "time"
)

// SunsetPolicy defines when versions are deprecated and removed
type SunsetPolicy struct {
    // Grace period after deprecation announcement
    DeprecationNoticeDays int
    
    // Time between deprecation and sunset
    SunsetGracePeriodDays int
    
    // Minimum supported versions to maintain
    MinSupportedVersions int
}

var DefaultSunsetPolicy = SunsetPolicy{
    DeprecationNoticeDays:   90,  // 3 months notice
    SunsetGracePeriodDays:   180, // 6 months after deprecation
    MinSupportedVersions:    2,   // Always support at least 2 versions
}

// CalculateSunsetDate determines when a version should be sunset
func (p SunsetPolicy) CalculateSunsetDate(announcementDate time.Time) time.Time {
    return announcementDate.AddDate(0, 0, p.DeprecationNoticeDays+p.SunsetGracePeriodDays)
}

// ShouldWarn returns true if version is approaching sunset
func (p SunsetPolicy) ShouldWarn(sunsetDate time.Time) bool {
    warningThreshold := sunsetDate.AddDate(0, 0, -30) // 30 days before
    return time.Now().After(warningThreshold)
}
```

### Migration Path

#### Phase 1: Infrastructure Setup (Week 1-2)

1. **Create version middleware** and routing structure
2. **Duplicate current handlers** to `api/handlers/v1/`
3. **Set up version routing** with v1 as current
4. **Test that existing clients work** with /api/v1/ URLs

#### Phase 2: Dual Version Support (Week 3-4)

1. **Create v2 handlers** with desired changes
2. **Update documentation** with version differences
3. **Add deprecation headers** to v1 responses
4. **Notify API consumers** about v2 availability

#### Phase 3: Default Redirect (Week 5-6)

1. **Make v2 the current version**
2. **Add redirects** from unversioned to /api/v2/
3. **Monitor for client issues**
4. **Update SDKs and examples** to use v2

#### Phase 4: v1 Deprecation (6+ months later)

1. **Add sunset date header** to v1 responses
2. **Send deprecation notices** to API consumers
3. **Monitor v1 usage** - should decline over time
4. **Eventually remove v1** after sunset date

### Why This Is Better

| Aspect | Before | After |
|--------|--------|-------|
| **Breaking Changes** | Impossible without breaking all clients | Smooth migration path |
| **Client Control** | Forced to accept changes immediately | Choose when to upgrade |
| **API Evolution** | Stagnation or chaos | Continuous improvement |
| **Communication** | No visibility into changes | Deprecation headers, sunset dates |
| **Cacheability** | Unclear | Version in URL = cache-friendly |
| **Testing** | One version to test | Can test new version before switching |

#### Business Benefits

1. **Customer Retention**: Clients not forced to upgrade on your schedule
2. **Faster Innovation**: Can ship breaking changes without fear
3. **Clear Communication**: Deprecation headers give clients time to plan
4. **Professional Image**: Mature API management

#### Technical Benefits

1. **Parallel Versions**: Can run v1 and v2 simultaneously
2. **Gradual Migration**: Clients upgrade at their own pace
3. **Rollback Safety**: Can revert to previous version instantly
4. **A/B Testing**: Can test new versions with subset of clients

---

## VULNERABILITY 19: Hardcoded Timeouts

### Overview

| Attribute | Value |
|-----------|-------|
| **Severity** | LOW |
| **Affected Files** | `services/*.go` |
| **CWE** | CWE-1088: Synchronous Access of Remote Resource without Timeout |
| **CVSS 3.1** | 3.1 (Low) |

### Current Vulnerable Code

```go
// services/user.go - VULNERABLE
func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    // Hardcoded timeout - not adaptive, not configurable
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    return s.db.QueryRowContext(ctx, "SELECT * FROM users WHERE id = $1", id).Scan(&user)
}
```

### Deep Root Cause Analysis

#### The Timeout Problem

Hardcoded timeouts create multiple issues:

```
┌─────────────────────────────────────────────────────────────┐
│              Hardcoded Timeout Issues                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. ENVIRONMENT MISMATCH                                     │
│     Dev:  30s timeout, query takes 100ms  → 29s wasted       │
│     Prod: 30s timeout, query takes 5s   → acceptable         │
│     Prod: 30s timeout, query takes 35s  → cascade failure    │
│                                                              │
│  2. NO ADAPTATION                                            │
│     Peak traffic: Queries slow down, timeouts don't adjust   │
│     Network issues: Timeouts too short for recovery         │
│     Maintenance: Timeouts don't account for planned work     │
│                                                              │
│  3. CASCADE FAILURES                                         │
│     Service A calls Service B with 30s timeout              │
│     Service B calls DB with 30s timeout                     │
│     DB is slow → Both timeouts fire → Double resource use   │
│                                                              │
│  4. NO OBSERVABILITY                                         │
│     Can't tune without recompiling                          │
│     No metrics on timeout frequency                           │
│     No correlation between timeout and latency              │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

#### Timeout Propagation Chain

```
HTTP Request (60s timeout from client)
    ↓
API Handler (no timeout)
    ↓
UserService.GetUser(30s timeout) ← HARDCODED
    ↓
Database Query (30s timeout) ← HARDCODED
    ↓
Network Call to Auth Service (30s timeout) ← HARDCODED

Problems:
1. Total timeout = 90s (30+30+30) > client timeout (60s)
2. Client gives up, but server keeps working
3. Resources wasted on abandoned requests
4. No way to cancel downstream work
```

#### Adaptive Timeout Benefits

```
Latency History (last 1000 requests):
P50: 50ms
P95: 200ms  ← Use this + margin for timeout
P99: 500ms

Adaptive timeout = P95 * 3 = 600ms

Benefits:
- Fast failure when truly slow
- No false timeouts during normal operation
- Adjusts to changing conditions
- Different per endpoint based on history
```

### The Ultimate Solution

#### Adaptive Timeout Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           Adaptive Timeout System                            │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. LATENCY TRACKING                                         │
│     ├── Record every request duration                        │
│     ├── Maintain P50, P95, P99 statistics                    │
│     └── Rolling window (last 1000 requests)                  │
│                                                              │
│  2. ADAPTIVE CALCULATION                                     │
│     ├── Base timeout = P95 latency * multiplier (3x)          │
│     ├── Minimum bound (never below 100ms)                    │
│     ├── Maximum bound (never above 30s)                      │
│     └── Adjust based on circuit breaker state                │
│                                                              │
│  3. CONTEXT PROPAGATION                                      │
│     ├── Pass deadline through context chain                  │
│     ├── Child timeouts = parent deadline - margin            │
│     └── Respect cancellation signals                         │
│                                                              │
│  4. ENVIRONMENT CONFIGURATION                                │
│     ├── Development: Longer timeouts, more forgiving        │
│     ├── Production: Tighter timeouts based on history        │
│     └── Testing: Deterministic timeouts                      │
│                                                              │
│  5. CIRCUIT BREAKER INTEGRATION                              │
│     ├── Open circuit = immediate failure                     │
│     ├── Half-open = reduced timeout                          │
│     └── Closed = normal adaptive timeout                     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Concrete Implementation

#### Step 1: Latency Tracker

```go
// pkg/timeout/latency_tracker.go
package timeout

import (
    "container/ring"
    "sync"
    "time"
)

// LatencyTracker maintains P50, P95, P99 statistics
type LatencyTracker struct {
    mu       sync.RWMutex
    samples  *ring.Ring // Circular buffer of durations
    capacity int
}

// NewLatencyTracker creates a tracker with specified capacity
func NewLatencyTracker(capacity int) *LatencyTracker {
    return &LatencyTracker{
        samples:  ring.New(capacity),
        capacity: capacity,
    }
}

// Record adds a new latency sample
func (t *LatencyTracker) Record(d time.Duration) {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    t.samples.Value = d
    t.samples = t.samples.Next()
}

// Percentile returns the specified percentile (0-100)
func (t *LatencyTracker) Percentile(p float64) time.Duration {
    t.mu.RLock()
    defer t.mu.RUnlock()
    
    // Collect all values
    var values []time.Duration
    t.samples.Do(func(v interface{}) {
        if v != nil {
            values = append(values, v.(time.Duration))
        }
    })
    
    if len(values) == 0 {
        return 0
    }
    
    // Sort and get percentile
    sort.Slice(values, func(i, j int) bool {
        return values[i] < values[j]
    })
    
    index := int(float64(len(values)-1) * p / 100.0)
    return values[index]
}

// P50, P95, P99 convenience methods
func (t *LatencyTracker) P50() time.Duration { return t.Percentile(50) }
func (t *LatencyTracker) P95() time.Duration { return t.Percentile(95) }
func (t *LatencyTracker) P99() time.Duration { return t.Percentile(99) }
```

#### Step 2: Adaptive Timeout Calculator

```go
// pkg/timeout/adaptive.go
package timeout

import (
    "math"
    "time"
)

// AdaptiveTimeoutConfig defines timeout calculation parameters
type AdaptiveTimeoutConfig struct {
    // Multiplier for P95 to get timeout (e.g., 3.0 = 3x P95)
    Multiplier float64
    
    // Minimum timeout regardless of latency
    MinTimeout time.Duration
    
    // Maximum timeout cap
    MaxTimeout time.Duration
    
    // Margin subtracted from parent deadline for child contexts
    ParentMargin time.Duration
}

// DefaultConfig for production use
var DefaultConfig = AdaptiveTimeoutConfig{
    Multiplier:   3.0,
    MinTimeout:   100 * time.Millisecond,
    MaxTimeout:   30 * time.Second,
    ParentMargin: 50 * time.Millisecond,
}

// DevelopmentConfig with longer timeouts
var DevelopmentConfig = AdaptiveTimeoutConfig{
    Multiplier:   5.0, // More forgiving
    MinTimeout:   500 * time.Millisecond,
    MaxTimeout:   60 * time.Second,
    ParentMargin: 100 * time.Millisecond,
}

// Calculator computes adaptive timeouts
type Calculator struct {
    config  AdaptiveTimeoutConfig
    tracker *LatencyTracker
}

// NewCalculator creates an adaptive timeout calculator
func NewCalculator(config AdaptiveTimeoutConfig, tracker *LatencyTracker) *Calculator {
    return &Calculator{
        config:  config,
        tracker: tracker,
    }
}

// Calculate returns the adaptive timeout based on history
func (c *Calculator) Calculate() time.Duration {
    p95 := c.tracker.P95()
    
    if p95 == 0 {
        // No history yet, use conservative default
        return c.config.MaxTimeout / 2
    }
    
    // Calculate: P95 * multiplier
    timeout := time.Duration(float64(p95) * c.config.Multiplier)
    
    // Apply bounds
    if timeout < c.config.MinTimeout {
        timeout = c.config.MinTimeout
    }
    if timeout > c.config.MaxTimeout {
        timeout = c.config.MaxTimeout
    }
    
    return timeout
}

// CalculateWithParent adjusts timeout based on parent context deadline
func (c *Calculator) CalculateWithParent(parentCtx context.Context) (context.Context, context.CancelFunc) {
    // Get parent deadline if exists
    if deadline, ok := parentCtx.Deadline(); ok {
        remaining := time.Until(deadline) - c.config.ParentMargin
        adaptive := c.Calculate()
        
        // Use the shorter of adaptive or parent remaining
        if remaining < adaptive && remaining > 0 {
            return context.WithTimeout(parentCtx, remaining)
        }
    }
    
    return context.WithTimeout(parentCtx, c.Calculate())
}
```

#### Step 3: Timeout Manager per Service

```go
// pkg/timeout/manager.go
package timeout

import (
    "context"
    "sync"
    "time"
)

// Manager handles timeouts for a specific service/endpoint
type Manager struct {
    mu          sync.RWMutex
    calculators map[string]*Calculator
    configs     map[string]AdaptiveTimeoutConfig
    trackers    map[string]*LatencyTracker
}

// NewManager creates a timeout manager
func NewManager() *Manager {
    return &Manager{
        calculators: make(map[string]*Calculator),
        configs:     make(map[string]AdaptiveTimeoutConfig),
        trackers:    make(map[string]*LatencyTracker),
    }
}

// RegisterEndpoint initializes tracking for an endpoint
func (m *Manager) RegisterEndpoint(name string, config AdaptiveTimeoutConfig) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    tracker := NewLatencyTracker(1000)
    calculator := NewCalculator(config, tracker)
    
    m.trackers[name] = tracker
    m.calculators[name] = calculator
    m.configs[name] = config
}

// WithTimeout returns a context with adaptive timeout for the endpoint
func (m *Manager) WithTimeout(parentCtx context.Context, endpoint string) (context.Context, context.CancelFunc, *Calculator) {
    m.mu.RLock()
    calc, exists := m.calculators[endpoint]
    m.mu.RUnlock()
    
    if !exists {
        // Fallback to default if not registered
        ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
        return ctx, cancel, nil
    }
    
    ctx, cancel := calc.CalculateWithParent(parentCtx)
    return ctx, cancel, calc
}

// RecordLatency records the actual latency for adaptation
func (m *Manager) RecordLatency(endpoint string, d time.Duration) {
    m.mu.RLock()
    tracker, exists := m.trackers[endpoint]
    m.mu.RUnlock()
    
    if exists {
        tracker.Record(d)
    }
}

// GetStats returns current latency statistics for an endpoint
func (m *Manager) GetStats(endpoint string) (p50, p95, p99, currentTimeout time.Duration, ok bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    tracker, exists := m.trackers[endpoint]
    if !exists {
        return 0, 0, 0, 0, false
    }
    
    calc := m.calculators[endpoint]
    return tracker.P50(), tracker.P95(), tracker.P99(), calc.Calculate(), true
}
```

#### Step 4: Service Integration

```go
// services/user.go
package services

import (
    "context"
    "time"
    
    "myapp/pkg/timeout"
)

type UserService struct {
    db             *sql.DB
    timeoutManager *timeout.Manager
}

// NewUserService creates a user service with timeout management
func NewUserService(db *sql.DB, tm *timeout.Manager) *UserService {
    s := &UserService{
        db:             db,
        timeoutManager: tm,
    }
    
    // Register endpoints with appropriate configs
    tm.RegisterEndpoint("user.get", timeout.DefaultConfig)
    tm.RegisterEndpoint("user.list", timeout.DefaultConfig)
    tm.RegisterEndpoint("user.create", timeout.DefaultConfig)
    tm.RegisterEndpoint("user.update", timeout.DefaultConfig)
    
    return s
}

// GetUser retrieves a user with adaptive timeout
func (s *UserService) GetUser(ctx context.Context, id int) (*User, error) {
    start := time.Now()
    endpoint := "user.get"
    
    // Get adaptive timeout context
    ctx, cancel, calc := s.timeoutManager.WithTimeout(ctx, endpoint)
    defer cancel()
    
    // Log the timeout being used
    if calc != nil {
        timeout := calc.Calculate()
        log.Debug().
            Str("endpoint", endpoint).
            Dur("timeout", timeout).
            Msg("Using adaptive timeout")
    }
    
    var user User
    err := s.db.QueryRowContext(ctx, 
        "SELECT id, email, full_name FROM users WHERE id = $1", id).
        Scan(&user.ID, &user.Email, &user.FullName)
    
    // Record actual latency for future adaptation
    elapsed := time.Since(start)
    s.timeoutManager.RecordLatency(endpoint, elapsed)
    
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return nil, fmt.Errorf("user.get timeout after %v: %w", elapsed, err)
        }
        return nil, err
    }
    
    return &user, nil
}

// GetUsers retrieves multiple users with different timeout profile
func (s *UserService) GetUsers(ctx context.Context, limit int) ([]User, error) {
    start := time.Now()
    endpoint := "user.list"
    
    ctx, cancel, _ := s.timeoutManager.WithTimeout(ctx, endpoint)
    defer cancel()
    
    rows, err := s.db.QueryContext(ctx, 
        "SELECT id, email, full_name FROM users LIMIT $1", limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var users []User
    for rows.Next() {
        var user User
        if err := rows.Scan(&user.ID, &user.Email, &user.FullName); err != nil {
            return nil, err
        }
        users = append(users, user)
    }
    
    // Record latency
    s.timeoutManager.RecordLatency(endpoint, time.Since(start))
    
    return users, nil
}
```

#### Step 5: Environment-Based Configuration

```go
// config/timeout.go
package config

import (
    "os"
    "myapp/pkg/timeout"
)

// LoadTimeoutConfig returns config based on environment
func LoadTimeoutConfig() timeout.AdaptiveTimeoutConfig {
    env := os.Getenv("APP_ENV")
    
    switch env {
    case "development", "dev":
        return timeout.DevelopmentConfig
    case "testing", "test":
        // Deterministic timeouts for tests
        return timeout.AdaptiveTimeoutConfig{
            Multiplier:   1.0,
            MinTimeout:   5 * time.Second,
            MaxTimeout:   5 * time.Second,
            ParentMargin: 100 * time.Millisecond,
        }
    case "production", "prod":
        return timeout.DefaultConfig
    default:
        return timeout.DevelopmentConfig
    }
}
```

#### Step 6: Monitoring and Metrics

```go
// pkg/timeout/metrics.go
package timeout

import (
    "github.com/prometheus/client_golang/prometheus"
)

var (
    latencyHistogram = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request latency",
            Buckets: prometheus.DefBuckets,
        },
        []string{"endpoint"},
    )
    
    timeoutGauge = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "adaptive_timeout_seconds",
            Help: "Current adaptive timeout value",
        },
        []string{"endpoint"},
    )
    
    timeoutCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "timeout_events_total",
            Help: "Total timeout events",
        },
        []string{"endpoint", "reason"},
    )
)

func init() {
    prometheus.MustRegister(latencyHistogram)
    prometheus.MustRegister(timeoutGauge)
    prometheus.MustRegister(timeoutCounter)
}

// RecordMetrics updates Prometheus metrics
func (m *Manager) RecordMetrics(endpoint string) {
    p50, p95, p99, current, ok := m.GetStats(endpoint)
    if !ok {
        return
    }
    
    // Update gauges
    timeoutGauge.WithLabelValues(endpoint).Set(current.Seconds())
    
    // Log statistics
    log.Debug().
        Str("endpoint", endpoint).
        Dur("p50", p50).
        Dur("p95", p95).
        Dur("p99", p99).
        Dur("current_timeout", current).
        Msg("Timeout statistics")
}
```

### Migration Path

#### Phase 1: Infrastructure (Week 1)

1. **Create timeout package** with tracker and calculator
2. **Add metrics collection** for latency tracking
3. **Create timeout manager** singleton
4. **Test with one endpoint** to validate approach

#### Phase 2: Gradual Rollout (Week 2-3)

1. **Update service layer** to use timeout manager
2. **Register all endpoints** with appropriate configs
3. **Monitor latency statistics** - should see P95 values
4. **Tune multiplier** based on false timeout rate

#### Phase 3: Context Propagation (Week 4)

1. **Update HTTP handlers** to pass context through
2. **Ensure cancellation propagates** to all downstream calls
3. **Add parent margin** to prevent cascade timeouts
4. **Test with slow dependencies** to verify behavior

#### Phase 4: Production Tuning (Ongoing)

1. **Monitor timeout frequency** - should be < 0.1%
2. **Adjust multipliers** per endpoint based on behavior
3. **Add alerting** for endpoints with increasing P95
4. **Document timeout behavior** for operators

### Why This Is Better

| Aspect | Before | After |
|--------|--------|-------|
| **Adaptability** | Fixed 30s regardless of conditions | Adjusts to actual latency |
| **Resource Usage** | Resources wasted on slow requests | Fast failure frees resources |
| **Cascade Prevention** | Multiple timeouts stack | Context propagation respects parent deadline |
| **Observability** | No visibility into timeout effectiveness | P50/P95/P99 metrics per endpoint |
| **Configuration** | Requires recompile to change | Environment-based config |
| **False Timeouts** | Fixed rate | Minimized through adaptation |

#### Performance Improvements

1. **Faster Failure**: Slow requests fail quickly, freeing resources
2. **Better Resource Usage**: No wasted work on abandoned requests
3. **Cascade Prevention**: Parent deadlines respected by children
4. **Self-Tuning**: Automatically adjusts to changing conditions

#### Operational Improvements

1. **Metrics-Driven**: P95 latency visible per endpoint
2. **Configurable**: Different settings per environment
3. **Predictable**: Timeout behavior based on actual history
4. **Debuggable**: Clear logs showing timeout decisions

---

## VULNERABILITY 20: Missing Request Size Limits

### Overview

| Attribute | Value |
|-----------|-------|
| **Severity** | LOW |
| **Affected File** | `cmd/server.go` |
| **CWE** | CWE-770: Allocation of Resources Without Limits or Throttling |
| **CVSS 3.1** | 3.7 (Low) |

### Current Vulnerable Code

```go
// cmd/server.go - VULNERABLE
func main() {
    app := fiber.New()
    
    // Fiber accepts unlimited body size by default
    // No limits on:
    // - Request body size
    // - Multipart form size
    // - File upload size
    // - Number of files
    
    app.Post("/api/users", CreateUser)
    app.Post("/uploads/avatar", UploadAvatar)
}
```

### Deep Root Cause Analysis

#### The Resource Exhaustion Attack

Without size limits, attackers can exhaust server resources:

```
Attack Scenario: Memory Exhaustion

Attacker sends: POST /api/users
Content-Length: 1000000000 (1GB)
Body: [1GB of JSON data]

Server behavior:
1. Reads entire body into memory
2. Attempts to parse as JSON
3. Memory usage spikes
4. Other requests starved
5. OOM killer may terminate process

Result: Denial of Service for all users
```

```
Attack Scenario: Disk Exhaustion

Attacker sends: POST /uploads/avatar (1000 concurrent requests)
Content-Type: multipart/form-data
Each upload: 100MB file

Server behavior:
1. Stores each file to temp directory
2. 1000 * 100MB = 100GB disk usage
3. Disk fills up
4. Logs can't be written
5. Database operations fail
6. Server crashes

Result: Complete service outage
```

#### Size Limit Categories

```
┌─────────────────────────────────────────────────────────────┐
│              Request Size Limit Categories                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. REQUEST BODY                                            │
│     - JSON API payloads                                      │
│     - Typical: 10KB - 10MB                                 │
│     - Prevents: JSON bomb attacks, massive object attacks    │
│                                                              │
│  2. MULTIPART FORM                                           │
│     - Total form size including all files                    │
│     - Typical: 32MB - 100MB                                  │
│     - Prevents: Form field overflow, combined file attacks   │
│                                                              │
│  3. INDIVIDUAL FILE                                          │
│     - Single uploaded file                                   │
│     - Typical: 5MB - 50MB                                    │
│     - Prevents: Individual huge file uploads                 │
│                                                              │
│  4. FILE COUNT                                               │
│     - Number of files in single request                      │
│     - Typical: 1 - 10                                        │
│     - Prevents: Death by a thousand cuts (many small files)  │
│                                                              │
│  5. HEADER SIZE                                              │
│     - Total HTTP header size                                 │
│     - Typical: 8KB - 64KB                                    │
│     - Prevents: Header overflow attacks                      │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### The Ultimate Solution

#### Multi-Layer Size Limit Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           Request Size Limit Defense Layers                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  LAYER 1: REVERSE PROXY (Nginx/CloudFlare)                   │
│  ├── client_max_body_size 10M;                               │
│  └── Early rejection, no load on application                 │
│                                                              │
│  LAYER 2: APPLICATION SERVER (Fiber)                         │
│  ├── BodyLimit: 10MB global                                  │
│  ├── Concurrency: 1000 max connections                       │
│  └── ReadTimeout: 10s                                        │
│                                                              │
│  LAYER 3: MIDDLEWARE VALIDATION                              │
│  ├── Content-Length check before reading                   │
│  ├── Per-endpoint limits                                     │
│  └── Streaming for large requests                          │
│                                                              │
│  LAYER 4: HANDLER VALIDATION                                 │
│  ├── File count limits                                       │
│  ├── Individual file size limits                             │
│  └── File type validation                                    │
│                                                              │
│  LAYER 5: RESOURCE MONITORING                                │
│  ├── Memory usage alerts                                     │
│  ├── Disk space monitoring                                   │
│  └── Automatic circuit breaker on resource exhaustion       │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Concrete Implementation

#### Step 1: Global Server Limits

```go
// cmd/server.go
package main

import (
    "github.com/gofiber/fiber/v3"
    "myapp/api/middleware"
)

func main() {
    app := fiber.New(fiber.Config{
        // Global body size limit (safety net)
        BodyLimit: 10 * 1024 * 1024, // 10MB
        
        // Connection limits
        Concurrency: 1000, // Max concurrent connections
        
        // Timeouts
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  120 * time.Second,
        
        // Header limits
        ReadBufferSize:  8 * 1024,  // 8KB
        WriteBufferSize: 8 * 1024,  // 8KB
        
        // Disable keep-alive for large requests
        DisableKeepalive: false,
    })
    
    // ... rest of setup
}
```

#### Step 2: Size Limit Middleware

```go
// api/middleware/size_limits.go
package middleware

import (
    "fmt"
    "net/http"
    "strconv"

    "github.com/gofiber/fiber/v3"
)

// SizeLimitConfig defines size limits for different request types
type SizeLimitConfig struct {
    // Maximum request body size (bytes)
    MaxBodySize int64
    
    // Maximum multipart form size (bytes)
    MaxMultipartSize int64
    
    // Maximum individual file size (bytes)
    MaxFileSize int64
    
    // Maximum number of files in multipart
    MaxFileCount int
    
    // Check Content-Length before reading body
    CheckContentLength bool
    
    // Allow streaming for large requests (don't buffer entirely)
    EnableStreaming bool
}

// DefaultSizeLimits for API endpoints
var DefaultSizeLimits = SizeLimitConfig{
    MaxBodySize:        10 * 1024 * 1024,  // 10MB
    MaxMultipartSize:   32 * 1024 * 1024,  // 32MB
    MaxFileSize:        5 * 1024 * 1024,   // 5MB
    MaxFileCount:       5,
    CheckContentLength: true,
    EnableStreaming:    true,
}

// StrictSizeLimits for small payload endpoints
var StrictSizeLimits = SizeLimitConfig{
    MaxBodySize:        10 * 1024,        // 10KB
    MaxMultipartSize:   0,                // No multipart
    MaxFileSize:        0,                // No files
    MaxFileCount:       0,
    CheckContentLength: true,
    EnableStreaming:    false,
}

// SizeLimitMiddleware enforces request size limits
func SizeLimitMiddleware(config SizeLimitConfig) fiber.Handler {
    return func(c *fiber.Ctx) error {
        contentType := c.Get("Content-Type")
        contentLength := c.Get("Content-Length")
        
        // Check 1: Content-Length validation (early rejection)
        if config.CheckContentLength && contentLength != "" {
            size, err := strconv.ParseInt(contentLength, 10, 64)
            if err != nil {
                return c.Status(http.StatusBadRequest).JSON(fiber.Map{
                    "error": "Invalid Content-Length header",
                    "code":  "INVALID_CONTENT_LENGTH",
                })
            }
            
            // Check against appropriate limit
            var maxSize int64
            if isMultipart(contentType) {
                maxSize = config.MaxMultipartSize
            } else {
                maxSize = config.MaxBodySize
            }
            
            if maxSize > 0 && size > maxSize {
                return c.Status(http.StatusRequestEntityTooLarge).JSON(fiber.Map{
                    "error": "Request body too large",
                    "code":  "PAYLOAD_TOO_LARGE",
                    "max_size": maxSize,
                    "received": size,
                })
            }
        }
        
        // Check 2: Multipart-specific validation
        if isMultipart(contentType) {
            if config.MaxMultipartSize == 0 {
                return c.Status(http.StatusUnsupportedMediaType).JSON(fiber.Map{
                    "error": "Multipart uploads not allowed for this endpoint",
                    "code":  "MULTIPART_NOT_ALLOWED",
                })
            }
            
            // Store limits in context for handler
            c.Locals("max_file_size", config.MaxFileSize)
            c.Locals("max_file_count", config.MaxFileCount)
        }
        
        return c.Next()
    }
}

func isMultipart(contentType string) bool {
    return len(contentType) > 19 && contentType[:19] == "multipart/form-data"
}
```

#### Step 3: Per-Endpoint Size Configuration

```go
// api/routes.go
package api

import (
    "github.com/gofiber/fiber/v3"
    "myapp/api/middleware"
)

func SetupRoutes(app *fiber.App) {
    // Small payload endpoints - strict limits
    api := app.Group("/api", middleware.SizeLimitMiddleware(middleware.StrictSizeLimits))
    {
        // Login - tiny payload
        api.Post("/auth/login", LoginHandler)
        
        // User creation - small payload
        api.Post("/users", CreateUserHandler)
        
        // User update - small payload
        api.Put("/users/:id", UpdateUserHandler)
    }
    
    // Medium payload endpoints
    mediumConfig := middleware.SizeLimitConfig{
        MaxBodySize:        100 * 1024,     // 100KB
        MaxMultipartSize:   0,
        MaxFileSize:        0,
        MaxFileCount:       0,
        CheckContentLength: true,
        EnableStreaming:    false,
    }
    medium := app.Group("/api/medium", middleware.SizeLimitMiddleware(mediumConfig))
    {
        medium.Post("/bulk-update", BulkUpdateHandler)
        medium.Post("/reports", CreateReportHandler)
    }
    
    // File upload endpoints - larger limits
    upload := app.Group("/uploads", middleware.SizeLimitMiddleware(middleware.DefaultSizeLimits))
    {
        upload.Post("/avatar", UploadAvatarHandler)
        upload.Post("/documents", UploadDocumentHandler)
    }
    
    // Bulk import - very large but controlled
    bulkConfig := middleware.SizeLimitConfig{
        MaxBodySize:        50 * 1024 * 1024, // 50MB
        MaxMultipartSize:   0,
        MaxFileSize:        0,
        MaxFileCount:       0,
        CheckContentLength: true,
        EnableStreaming:    true, // Stream, don't buffer
    }
    app.Post("/api/bulk-import", 
        middleware.SizeLimitMiddleware(bulkConfig),
        BulkImportHandler,
    )
}
```

#### Step 4: Streaming File Upload Handler

```go
// api/handlers/upload.go
package handlers

import (
    "fmt"
    "io"
    "net/http"

    "github.com/gofiber/fiber/v3"
)

// UploadAvatarHandler handles avatar uploads with size limits
func UploadAvatarHandler(c *fiber.Ctx) error {
    // Get limits from context (set by middleware)
    maxFileSize := c.Locals("max_file_size").(int64)
    maxFileCount := c.Locals("max_file_count").(int)
    
    // Get multipart form
    form, err := c.MultipartForm()
    if err != nil {
        return c.Status(http.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid multipart form",
            "code":  "INVALID_FORM",
        })
    }
    
    // Check file count
    files := form.File["avatar"]
    if len(files) == 0 {
        return c.Status(http.StatusBadRequest).JSON(fiber.Map{
            "error": "No file uploaded",
            "code":  "NO_FILE",
        })
    }
    if len(files) > maxFileCount {
        return c.Status(http.StatusRequestEntityTooLarge).JSON(fiber.Map{
            "error": "Too many files",
            "code":  "TOO_MANY_FILES",
            "max_files": maxFileCount,
            "received": len(files),
        })
    }
    
    // Process each file
    for _, file := range files {
        // Check individual file size
        if file.Size > maxFileSize {
            return c.Status(http.StatusRequestEntityTooLarge).JSON(fiber.Map{
                "error": "File too large",
                "code":  "FILE_TOO_LARGE",
                "max_size": maxFileSize,
                "file_size": file.Size,
                "filename": file.Filename,
            })
        }
        
        // Open file with size-limited reader
        src, err := file.Open()
        if err != nil {
            return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to read file",
                "code":  "FILE_READ_ERROR",
            })
        }
        defer src.Close()
        
        // Use LimitReader to enforce size during streaming
        limitedReader := io.LimitReader(src, maxFileSize+1)
        
        // Process file (save to storage, etc.)
        if err := processAvatar(limitedReader, file.Filename, file.Size); err != nil {
            return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to process file",
                "code":  "FILE_PROCESS_ERROR",
            })
        }
    }
    
    return c.JSON(fiber.Map{
        "success": true,
        "files_processed": len(files),
    })
}

func processAvatar(reader io.Reader, filename string, size int64) error {
    // Read with limit to prevent memory exhaustion
    data, err := io.ReadAll(reader)
    if err != nil {
        return err
    }
    
    // Check if limit was exceeded
    if int64(len(data)) > size {
        return fmt.Errorf("file size exceeded during read")
    }
    
    // Validate image format
    if !isValidImage(data) {
        return fmt.Errorf("invalid image format")
    }
    
    // Save to storage
    return saveToStorage(data, filename)
}
```

#### Step 5: JSON Bomb Protection

```go
// api/middleware/json_limits.go
package middleware

import (
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/gofiber/fiber/v3"
)

// JSONLimitConfig prevents JSON bomb attacks
type JSONLimitConfig struct {
    // Maximum nesting depth
    MaxDepth int
    
    // Maximum number of keys
    MaxKeys int
    
    // Maximum string length
    MaxStringLength int
    
    // Maximum array length
    MaxArrayLength int
}

var DefaultJSONLimits = JSONLimitConfig{
    MaxDepth:        10,
    MaxKeys:         100,
    MaxStringLength: 10000,
    MaxArrayLength:  1000,
}

// SafeJSONParser wraps standard parser with limits
type SafeJSONParser struct {
    config JSONLimitConfig
}

func (p *SafeJSONParser) Parse(data []byte, v interface{}) error {
    // First, check raw size
    if len(data) > 10*1024*1024 { // 10MB raw limit
        return fmt.Errorf("JSON payload too large: %d bytes", len(data))
    }
    
    // Use decoder with DisallowUnknownFields for extra safety
    decoder := json.NewDecoder(nil)
    decoder.DisallowUnknownFields()
    
    // Validate structure before full parsing
    var raw interface{}
    if err := json.Unmarshal(data, &raw); err != nil {
        return err
    }
    
    // Check limits recursively
    if err := p.validateValue(raw, 0); err != nil {
        return err
    }
    
    // Safe to parse into target
    return json.Unmarshal(data, v)
}

func (p *SafeJSONParser) validateValue(v interface{}, depth int) error {
    if depth > p.config.MaxDepth {
        return fmt.Errorf("JSON nesting too deep: %d > %d", depth, p.config.MaxDepth)
    }
    
    switch val := v.(type) {
    case map[string]interface{}:
        if len(val) > p.config.MaxKeys {
            return fmt.Errorf("too many keys: %d > %d", len(val), p.config.MaxKeys)
        }
        for _, v := range val {
            if err := p.validateValue(v, depth+1); err != nil {
                return err
            }
        }
    case []interface{}:
        if len(val) > p.config.MaxArrayLength {
            return fmt.Errorf("array too long: %d > %d", len(val), p.config.MaxArrayLength)
        }
        for _, v := range val {
            if err := p.validateValue(v, depth+1); err != nil {
                return err
            }
        }
    case string:
        if len(val) > p.config.MaxStringLength {
            return fmt.Errorf("string too long: %d > %d", len(val), p.config.MaxStringLength)
        }
    case float64:
        // Numbers are fine
    case bool:
        // Booleans are fine
    case nil:
        // Null is fine
    }
    
    return nil
}

// JSONLimitMiddleware prevents JSON bomb attacks
func JSONLimitMiddleware(config JSONLimitConfig) fiber.Handler {
    parser := &SafeJSONParser{config: config}
    
    return func(c *fiber.Ctx) error {
        // Only process JSON requests
        if c.Get("Content-Type") != "application/json" {
            return c.Next()
        }
        
        // Get body
        body := c.Body()
        
        // Validate with safe parser
        var dummy interface{}
        if err := parser.Parse(body, &dummy); err != nil {
            return c.Status(http.StatusBadRequest).JSON(fiber.Map{
                "error": "Invalid JSON structure",
                "code":  "JSON_VALIDATION_ERROR",
                "details": err.Error(),
            })
        }
        
        return c.Next()
    }
}
```

#### Step 6: Resource Monitoring

```go
// internal/monitoring/resources.go
package monitoring

import (
    "runtime"
    "time"
)

// ResourceMonitor tracks resource usage
type ResourceMonitor struct {
    maxMemoryMB    int
    maxDiskUsageGB int
    alertChannel   chan Alert
}

type Alert struct {
    Type    string
    Message string
    Value   float64
    Limit   float64
}

// CheckResources monitors memory and disk
func (m *ResourceMonitor) CheckResources() {
    // Check memory
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    
    memMB := float64(memStats.Alloc) / 1024 / 1024
    if memMB > float64(m.maxMemoryMB)*0.8 {
        m.alertChannel <- Alert{
            Type:    "memory",
            Message: "High memory usage detected",
            Value:   memMB,
            Limit:   float64(m.maxMemoryMB),
        }
    }
    
    // Check disk (simplified - would use actual disk check)
    // ...
}

// StartMonitoring begins periodic resource checks
func (m *ResourceMonitor) StartMonitoring() {
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for range ticker.C {
            m.CheckResources()
        }
    }()
}
```

### Migration Path

#### Phase 1: Assessment (Week 1)

1. **Audit all endpoints** for expected payload sizes
2. **Identify file upload endpoints**
3. **Document current traffic patterns**
4. **Set baseline limits** based on P99 payload sizes

#### Phase 2: Global Limits (Week 2)

1. **Add Fiber BodyLimit** (10MB default)
2. **Monitor for 413 errors** - indicates legitimate large requests
3. **Adjust global limit** if needed
4. **Document limit in API docs**

#### Phase 3: Per-Endpoint Limits (Week 3)

1. **Add size limit middleware**
2. **Configure strict limits** for small endpoints
3. **Configure larger limits** for upload endpoints
4. **Add streaming** for bulk endpoints

#### Phase 4: Validation (Week 4)

1. **Load testing** with oversized payloads
2. **Verify 413 responses** are returned correctly
3. **Check resource usage** under attack simulation
4. **Tune limits** based on results

### Why This Is Better

| Aspect | Before | After |
|--------|--------|-------|
| **DoS Protection** | Vulnerable to memory/disk exhaustion | Multi-layer defense |
| **Resource Usage** | Unpredictable spikes | Controlled, bounded usage |
| **Error Handling** | OOM crashes, unclear errors | Clear 413 responses |
| **Flexibility** | One-size-fits-all | Per-endpoint configuration |
| **Streaming** | Everything buffered | Large requests streamed |
| **Monitoring** | No visibility | Resource usage tracked |

#### Security Improvements

1. **DoS Prevention**: Memory and disk exhaustion attacks blocked
2. **JSON Bomb Protection**: Deep nesting and large structures rejected
3. **Early Rejection**: Content-Length checked before body read
4. **Resource Isolation**: One large request can't affect others

#### Operational Improvements

1. **Predictable Resources**: Memory/disk usage bounded
2. **Clear Errors**: 413 status with helpful messages
3. **Per-Endpoint Control**: Different limits for different use cases
4. **Monitoring**: Alerts on high resource usage

---

## VULNERABILITY 21: Log Injection via User Input

### Overview

| Attribute | Value |
|-----------|-------|
| **Severity** | LOW |
| **Affected Files** | `api/*.go` |
| **CWE** | CWE-117: Improper Output Neutralization for Logs |
| **CVSS 3.1** | 3.1 (Low) |

### Current Vulnerable Code

```go
// api/auth.go - VULNERABLE
func Login(c *fiber.Ctx) error {
    username := c.FormValue("username")
    
    // Direct user input in logs - INJECTION VULNERABILITY
    log.Info().
        Str("username", username).
        Str("ip", c.IP()).
        Msg("Login attempt")
    
    // ...
}
```

### Deep Root Cause Analysis

#### The Log Injection Attack

Log injection occurs when user input containing special characters is written directly to logs:

```
Attack Scenario 1: Fake Log Entries

Attacker sends: username = "admin\n[INFO] User admin logged in successfully"

Log output:
[INFO] Login attempt username=admin
[INFO] User admin logged in successfully ip=192.168.1.1

Result: Fake success log entry appears legitimate
```

```
Attack Scenario 2: Log Corruption

Attacker sends: username = "admin\r\n\r\n[ERROR] Database connection failed"

Log output becomes corrupted, real errors hidden
```

```
Attack Scenario 3: Log File Injection

Attacker sends: username = "admin\n<script>alert('xss')</script>"

If logs are viewed in web interface, XSS attack possible
```

```
Attack Scenario 4: SIEM/Log Aggregation Bypass

Attacker sends: username = "admin\tfield=value\tanother=bad"

Structured log parsers may interpret injected tabs as field separators
```

#### Why This Is Dangerous

```
┌─────────────────────────────────────────────────────────────┐
│              Log Injection Consequences                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  1. FORENSIC CORRUPTION                                      │
│     - Fake entries confuse incident investigation            │
│     - Real attacks hidden among injected noise               │
│     - Timeline reconstruction becomes impossible           │
│                                                              │
│  2. COMPLIANCE VIOLATIONS                                    │
│     - Audit logs must be tamper-evident                      │
│     - Injected entries violate integrity requirements        │
│     - SOC2, ISO27001, PCI-DSS non-compliance                 │
│                                                              │
│  3. SECURITY MONITORING BYPASS                               │
│     - SIEM rules may match injected patterns                 │
│     - False positives desensitize operators                  │
│     - Real alerts lost in noise                              │
│                                                              │
│  4. LEGAL/EVIDENTIARY ISSUES                                 │
│     - Logs may be inadmissible in court                      │
│     - Cannot prove what actually happened                    │
│     - Chain of custody broken                                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### The Ultimate Solution

#### Defense in Depth Architecture

```
┌─────────────────────────────────────────────────────────────┐
│           Log Injection Defense Layers                       │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  LAYER 1: INPUT SANITIZATION                                 │
│  ├── Remove control characters (0x00-0x1F, 0x7F)            │
│  ├── Limit length (1000 chars max)                          │
│  ├── Normalize whitespace                                   │
│  └── Validate against allowlist                            │
│                                                              │
│  LAYER 2: STRUCTURED LOGGING                                 │
│  ├── Use zerolog/json logging (not text)                    │
│  ├── Field names controlled by application                  │
│  ├── Values properly escaped by library                     │
│  └── No string concatenation in log messages               │
│                                                              │
│  LAYER 3: OUTPUT ENCODING                                    │
│  ├── JSON encoding escapes special characters               │
│  ├── Newlines become \n                                     │
│  ├── Quotes become \"                                       │
│  └── Control chars become \uXXXX                            │
│                                                              │
│  LAYER 4: CENTRALIZED VALIDATION                             │
│  ├── All user input through SafeString() function           │
│  ├── Consistent sanitization across codebase                │
│  └── Auditable, testable sanitization logic                 │
│                                                              │
│  LAYER 5: SIEM INTEGRATION                                   │
│  ├── Structured format for machine parsing                  │
│  ├── Integrity checksums on log entries                     │
│  └── Tamper detection in log aggregation                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Concrete Implementation

#### Step 1: SafeString Sanitization

```go
// pkg/sanitize/safestring.go
package sanitize

import (
    "strings"
    "unicode"
)

// SafeString sanitizes user input for safe logging
type SafeString string

// MaxSafeLength prevents log bloat
const MaxSafeLength = 1000

// NewSafeString creates a sanitized string from user input
func NewSafeString(input string) SafeString {
    if input == "" {
        return ""
    }
    
    // Step 1: Limit length
    if len(input) > MaxSafeLength {
        input = input[:MaxSafeLength] + "[TRUNCATED]"
    }
    
    // Step 2: Remove control characters
    // Keep only printable ASCII and common Unicode
    var result strings.Builder
    for _, r := range input {
        if isSafeRune(r) {
            result.WriteRune(r)
        } else {
            // Replace unsafe chars with replacement character
            result.WriteRune('')
        }
    }
    
    // Step 3: Normalize whitespace
    sanitized := result.String()
    sanitized = normalizeWhitespace(sanitized)
    
    return SafeString(sanitized)
}

// isSafeRune determines if a character is safe for logging
func isSafeRune(r rune) bool {
    // Allow:
    // - Printable ASCII (0x20-0x7E)
    // - Common Unicode letters and numbers
    // - Basic punctuation
    
    switch {
    case r >= 0x20 && r <= 0x7E:
        // Printable ASCII
        return true
    case unicode.IsLetter(r) || unicode.IsNumber(r):
        // Unicode letters and numbers
        return true
    case unicode.IsSpace(r) && r != '\n' && r != '\r' && r != '\t':
        // Non-ASCII spaces (be careful)
        return true
    default:
        return false
    }
}

// normalizeWhitespace collapses multiple spaces and removes dangerous chars
func normalizeWhitespace(s string) string {
    // Replace various whitespace with single space
    s = strings.ReplaceAll(s, "\n", " ")
    s = strings.ReplaceAll(s, "\r", " ")
    s = strings.ReplaceAll(s, "\t", " ")
    s = strings.ReplaceAll(s, "\f", " ")
    s = strings.ReplaceAll(s, "\v", " ")
    s = strings.ReplaceAll(s, "\b", "")
    s = strings.ReplaceAll(s, "\x00", "") // Null byte
    
    // Collapse multiple spaces
    for strings.Contains(s, "  ") {
        s = strings.ReplaceAll(s, "  ", " ")
    }
    
    return strings.TrimSpace(s)
}

// String returns the sanitized string value
func (s SafeString) String() string {
    return string(s)
}

// IsEmpty checks if the sanitized string is empty
func (s SafeString) IsEmpty() bool {
    return len(s) == 0
}

// Truncated returns true if the string was truncated
func (s SafeString) Truncated() bool {
    return strings.HasSuffix(string(s), "[TRUNCATED]")
}
```

#### Step 2: Structured Logging Integration

```go
// pkg/log/safelogger.go
package log

import (
    "github.com/rs/zerolog"
    "myapp/pkg/sanitize"
)

// SafeLogger wraps zerolog with automatic sanitization
type SafeLogger struct {
    logger zerolog.Logger
}

// NewSafeLogger creates a logger with sanitization
func NewSafeLogger(logger zerolog.Logger) *SafeLogger {
    return &SafeLogger{logger: logger}
}

// Str logs a string field with automatic sanitization
func (l *SafeLogger) Str(key string, value string) *SafeLogger {
    safe := sanitize.NewSafeString(value)
    l.logger = l.logger.With().Str(key, safe.String()).Logger()
    return l
}

// Strs logs a slice of strings with sanitization
func (l *SafeLogger) Strs(key string, values []string) *SafeLogger {
    safeValues := make([]string, len(values))
    for i, v := range values {
        safeValues[i] = sanitize.NewSafeString(v).String()
    }
    l.logger = l.logger.With().Strs(key, safeValues).Logger()
    return l
}

// UserInput logs a user input field (explicitly marked)
func (l *SafeLogger) UserInput(key string, value string) *SafeLogger {
    safe := sanitize.NewSafeString(value)
    l.logger = l.logger.With().
        Str(key, safe.String()).
        Bool(key+"_sanitized", true).
        Logger()
    return l
}

// Msg logs the message (also sanitized)
func (l *SafeLogger) Msg(message string) {
    safe := sanitize.NewSafeString(message)
    l.logger.Info().Msg(safe.String())
}

// Error logs an error
func (l *SafeLogger) Error(err error) *SafeLogger {
    if err != nil {
        l.logger = l.logger.With().Err(err).Logger()
    }
    return l
}

// Interface logs arbitrary data (use carefully)
func (l *SafeLogger) Interface(key string, v interface{}) *SafeLogger {
    // For interfaces, we rely on JSON encoding
    // which handles escaping automatically
    l.logger = l.logger.With().Interface(key, v).Logger()
    return l
}
```

#### Step 3: API Handler Updates

```go
// api/auth.go
package api

import (
    "github.com/gofiber/fiber/v3"
    "github.com/rs/zerolog"
    
    "myapp/pkg/log"
    "myapp/pkg/sanitize"
)

func Login(c *fiber.Ctx, logger zerolog.Logger) error {
    // Get raw user input
    username := c.FormValue("username")
    password := c.FormValue("password") // Never log passwords!
    ip := c.IP()
    userAgent := c.Get("User-Agent")
    
    // Create safe logger
    safeLog := log.NewSafeLogger(logger)
    
    // Log with automatic sanitization
    safeLog.
        UserInput("username", username). // Marked as user input
        Str("ip_address", ip).           // IP is safe but still sanitized
        UserInput("user_agent", userAgent).
        Str("endpoint", "/api/auth/login").
        Msg("Login attempt")
    
    // Alternative: Direct sanitization
    logger.Info().
        Str("username", sanitize.NewSafeString(username).String()).
        Str("ip", ip).
        Str("user_agent", sanitize.NewSafeString(userAgent).String()).
        Msg("Login attempt")
    
    // ... rest of login logic
}

// Example with more complex logging
func CreateUser(c *fiber.Ctx, logger zerolog.Logger) error {
    // Multiple user inputs
    email := c.FormValue("email")
    fullName := c.FormValue("full_name")
    bio := c.FormValue("bio")
    
    // Sanitize all inputs
    safeEmail := sanitize.NewSafeString(email)
    safeName := sanitize.NewSafeString(fullName)
    safeBio := sanitize.NewSafeString(bio)
    
    // Check if any were truncated
    if safeEmail.Truncated() || safeName.Truncated() || safeBio.Truncated() {
        logger.Warn().
            Str("email", safeEmail.String()).
            Bool("truncated", true).
            Msg("User input truncated during logging")
    }
    
    logger.Info().
        Str("email", safeEmail.String()).
        Str("full_name", safeName.String()).
        Str("bio_length", fmt.Sprintf("%d", len(safeBio.String()))).
        Str("ip", c.IP()).
        Msg("User creation attempt")
    
    // ... rest of handler
}
```

#### Step 4: Centralized Log Validation

```go
// pkg/log/validator.go
package log

import (
    "encoding/json"
    "regexp"
    "strings"
)

// LogValidator checks log entries for injection attempts
type LogValidator struct {
    // Patterns that indicate injection
    dangerousPatterns []*regexp.Regexp
}

// NewLogValidator creates a validator with default patterns
func NewLogValidator() *LogValidator {
    return &LogValidator{
        dangerousPatterns: []*regexp.Regexp{
            regexp.MustCompile(`\n\[(INFO|WARN|ERROR|DEBUG)\]`), // Fake log levels
            regexp.MustCompile(`\r\n`),                           // CRLF injection
            regexp.MustCompile(`[\x00-\x08\x0B-\x0C\x0E-\x1F\x7F]`), // Control chars
        },
    }
}

// ValidateEntry checks a log entry for injection
func (v *LogValidator) ValidateEntry(entry map[string]interface{}) (bool, []string) {
    var issues []string
    
    for key, value := range entry {
        strValue, ok := value.(string)
        if !ok {
            continue
        }
        
        // Check for dangerous patterns
        for _, pattern := range v.dangerousPatterns {
            if pattern.MatchString(strValue) {
                issues = append(issues, fmt.Sprintf(
                    "Field '%s' contains dangerous pattern: %s",
                    key, pattern.String(),
                ))
            }
        }
        
        // Check for unescaped newlines
        if strings.Contains(strValue, "\n") || strings.Contains(strValue, "\r") {
            issues = append(issues, fmt.Sprintf(
                "Field '%s' contains unescaped newline",
                key,
            ))
        }
    }
    
    return len(issues) == 0, issues
}

// SanitizeAndValidate combines sanitization with validation
func (v *LogValidator) SanitizeAndValidate(data map[string]string) (map[string]string, error) {
    sanitized := make(map[string]string)
    
    for key, value := range data {
        safe := sanitize.NewSafeString(value)
        sanitized[key] = safe.String()
    }
    
    // Convert to interface{} for validation
    entry := make(map[string]interface{})
    for k, v := range sanitized {
        entry[k] = v
    }
    
    valid, issues := v.ValidateEntry(entry)
    if !valid {
        return nil, fmt.Errorf("log validation failed: %v", issues)
    }
    
    return sanitized, nil
}
```

#### Step 5: SIEM-Compatible Output

```go
// pkg/log/siem.go
package log

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "time"
)

// SIEMEntry represents a log entry for SIEM systems
type SIEMEntry struct {
    Timestamp   string                 `json:"@timestamp"`
    Level       string                 `json:"level"`
    Message     string                 `json:"message"`
    Service     string                 `json:"service"`
    Environment string                 `json:"environment"`
    Fields      map[string]string      `json:"fields"`
    Checksum    string                 `json:"_checksum"` // Integrity check
}

// SIEMFormatter formats logs for SIEM consumption
type SIEMFormatter struct {
    service     string
    environment string
}

// Format creates a SIEM-compatible log entry
func (f *SIEMFormatter) Format(level, message string, fields map[string]string) ([]byte, error) {
    // Sanitize all fields
    safeFields := make(map[string]string)
    for k, v := range fields {
        safeFields[k] = sanitize.NewSafeString(v).String()
    }
    
    entry := SIEMEntry{
        Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
        Level:       level,
        Message:     sanitize.NewSafeString(message).String(),
        Service:     f.service,
        Environment: f.environment,
        Fields:      safeFields,
    }
    
    // Calculate checksum for integrity
    entry.Checksum = f.calculateChecksum(entry)
    
    return json.Marshal(entry)
}

func (f *SIEMFormatter) calculateChecksum(entry SIEMEntry) string {
    // Create checksum of all fields except checksum itself
    data, _ := json.Marshal(struct {
        TS      string            `json:"@timestamp"`
        Level   string            `json:"level"`
        Msg     string            `json:"message"`
        Service string            `json:"service"`
        Env     string            `json:"environment"`
        Fields  map[string]string `json:"fields"`
    }{
        TS:      entry.Timestamp,
        Level:   entry.Level,
        Msg:     entry.Message,
        Service: entry.Service,
        Env:     entry.Environment,
        Fields:  entry.Fields,
    })
    
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:])
}

// VerifyChecksum validates entry integrity
func (f *SIEMFormatter) VerifyChecksum(entry SIEMEntry) bool {
    expected := f.calculateChecksum(entry)
    return expected == entry.Checksum
}
```

#### Step 6: Middleware for Automatic Sanitization

```go
// api/middleware/logging.go
package middleware

import (
    "github.com/gofiber/fiber/v3"
    "github.com/rs/zerolog"
    "myapp/pkg/sanitize"
)

// SafeLoggingMiddleware sanitizes all user inputs before logging
type SafeLoggingMiddleware struct {
    logger zerolog.Logger
}

// NewSafeLoggingMiddleware creates middleware with sanitization
func NewSafeLoggingMiddleware(logger zerolog.Logger) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Log request with sanitized fields
        logger.Info().
            Str("method", c.Method()).
            Str("path", sanitize.NewSafeString(c.Path()).String()).
            Str("ip", c.IP()).
            Str("user_agent", sanitize.NewSafeString(c.Get("User-Agent")).String()).
            Msg("Request started")
        
        // Store sanitized values in context for handlers
        c.Locals("sanitized_path", sanitize.NewSafeString(c.Path()))
        c.Locals("sanitized_query", sanitizeQuery(c.Queries()))
        
        err := c.Next()
        
        // Log response with sanitized status
        logger.Info().
            Int("status", c.Response().StatusCode()).
            Str("path", sanitize.NewSafeString(c.Path()).String()).
            Msg("Request completed")
        
        return err
    }
}

func sanitizeQuery(queries map[string]string) map[string]string {
    sanitized := make(map[string]string)
    for k, v := range queries {
        sanitized[sanitize.NewSafeString(k).String()] = sanitize.NewSafeString(v).String()
    }
    return sanitized
}
```

### Migration Path

#### Phase 1: Audit (Week 1)

1. **Find all log statements** with user input:
   ```bash
   grep -r "log\." api/ | grep -E "(FormValue|Query|Param|Body)"
   ```

2. **Identify injection vulnerabilities**
3. **Document all user input sources**
4. **Create sanitization package**

#### Phase 2: Infrastructure (Week 2)

1. **Implement SafeString** sanitization
2. **Create SafeLogger** wrapper
3. **Add log validation** for SIEM compatibility
4. **Write tests** for injection patterns

#### Phase 3: Handler Updates (Week 3)

1. **Update authentication handlers** (highest risk)
2. **Update user input handlers**
3. **Update API endpoints** with form data
4. **Verify no raw user input** in logs

#### Phase 4: Validation (Week 4)

1. **Test with injection payloads**:
   ```
   username: admin\n[INFO] Fake entry
   username: admin\r\n\r\n[ERROR] Corruption
   ```

2. **Verify logs are clean** in log aggregation
3. **Check SIEM parsing** works correctly
4. **Document sanitization** for future developers

### Why This Is Better

| Aspect | Before | After |
|--------|--------|-------|
| **Security** | Vulnerable to log injection | All user input sanitized |
| **Forensics** | Logs can be corrupted | Tamper-evident, trustworthy |
| **Compliance** | Violates audit requirements | Meets SOC2, ISO27001, PCI-DSS |
| **SIEM Integration** | Parsing errors, false positives | Structured, validated output |
| **Debugging** | May see injected noise | Clean, accurate logs |
| **Maintenance** | Ad-hoc sanitization | Centralized, consistent |

#### Security Improvements

1. **Injection Prevention**: Control characters removed, newlines escaped
2. **Forensic Integrity**: Logs cannot be corrupted by attackers
3. **Audit Compliance**: Meets regulatory requirements for log integrity
4. **SIEM Reliability**: Structured format prevents parsing errors

#### Operational Improvements

1. **Clean Logs**: No fake entries or corruption
2. **Accurate Monitoring**: Real alerts not lost in noise
3. **Centralized Sanitization**: One place to update rules
4. **Audit Trail**: Checksums verify log integrity

---

## Summary

This document provides comprehensive solutions for 6 LOW severity security vulnerabilities:

| # | Vulnerability | Key Solution |
|---|---------------|--------------|
| 16 | Verbose Error Messages | Error taxonomy + Public/Internal separation |
| 17 | Missing Content-Type Validation | Strict whitelist + 415 responses |
| 18 | Missing API Versioning | URL versioning + deprecation headers |
| 19 | Hardcoded Timeouts | Adaptive timeouts based on P95 latency |
| 20 | Missing Request Size Limits | Multi-layer limits + streaming |
| 21 | Log Injection | SafeString sanitization + structured logging |

All solutions follow defense-in-depth principles and provide concrete, production-ready implementations.
