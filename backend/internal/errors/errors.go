package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode is a machine-readable error identifier.
type ErrorCode string

const (
	CodeInvalidInput     ErrorCode = "INVALID_INPUT"
	CodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	CodeForbidden        ErrorCode = "FORBIDDEN"
	CodeNotFound         ErrorCode = "NOT_FOUND"
	CodeConflict         ErrorCode = "CONFLICT"
	CodeInternal         ErrorCode = "INTERNAL_ERROR"
	CodeRateLimited      ErrorCode = "RATE_LIMITED"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// ServiceError is a domain error with code, status, and safe message.
type ServiceError struct {
	Code     ErrorCode
	Status   int
	Message  string
	Internal string
	Cause    error
	TraceID  string
}

func (e *ServiceError) Error() string {
	if e.Internal != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ServiceError) Unwrap() error {
	return e.Cause
}

func IsServiceError(err error) (*ServiceError, bool) {
	var se *ServiceError
	if errors.As(err, &se) {
		return se, true
	}
	return nil, false
}

func Validation(field, detail string) *ServiceError {
	return &ServiceError{
		Code:     CodeInvalidInput,
		Status:   http.StatusBadRequest,
		Message:  fmt.Sprintf("Validation failed for field '%s'", field),
		Internal: detail,
	}
}

func Validationf(field string, format string, args ...interface{}) *ServiceError {
	return Validation(field, fmt.Sprintf(format, args...))
}

func Auth(detail string) *ServiceError {
	return &ServiceError{
		Code:     CodeUnauthorized,
		Status:   http.StatusUnauthorized,
		Message:  "Authentication failed",
		Internal: detail,
	}
}

func Authf(format string, args ...interface{}) *ServiceError {
	return Auth(fmt.Sprintf(format, args...))
}

func Forbidden(resource string) *ServiceError {
	return &ServiceError{
		Code:     CodeForbidden,
		Status:   http.StatusForbidden,
		Message:  fmt.Sprintf("Access denied to '%s'", resource),
		Internal: "insufficient permissions",
	}
}

func NotFound(resource string, id interface{}) *ServiceError {
	return &ServiceError{
		Code:     CodeNotFound,
		Status:   http.StatusNotFound,
		Message:  fmt.Sprintf("%s not found", resource),
		Internal: fmt.Sprintf("id=%v", id),
	}
}

func Conflict(resource, detail string) *ServiceError {
	return &ServiceError{
		Code:     CodeConflict,
		Status:   http.StatusConflict,
		Message:  fmt.Sprintf("Conflict with '%s'", resource),
		Internal: detail,
	}
}

func Internal(cause error) *ServiceError {
	return &ServiceError{
		Code:     CodeInternal,
		Status:   http.StatusInternalServerError,
		Message:  "An internal error occurred",
		Internal: cause.Error(),
		Cause:    cause,
	}
}

func RateLimit(retryAfter int) *ServiceError {
	return &ServiceError{
		Code:     CodeRateLimited,
		Status:   http.StatusTooManyRequests,
		Message:  fmt.Sprintf("Rate limit exceeded. Retry after %d seconds", retryAfter),
		Internal: fmt.Sprintf("retry_after=%d", retryAfter),
	}
}

func Unavailable(service string) *ServiceError {
	return &ServiceError{
		Code:     CodeServiceUnavailable,
		Status:   http.StatusServiceUnavailable,
		Message:  fmt.Sprintf("Service '%s' is temporarily unavailable", service),
		Internal: "dependency failure",
	}
}

func Wrap(err error, context string) error {
	if err == nil {
		return nil
	}
	if se, ok := IsServiceError(err); ok {
		if se.Internal == "" {
			se.Internal = context
		} else {
			se.Internal = context + ": " + se.Internal
		}
		return se
	}
	return &ServiceError{
		Code:     CodeInternal,
		Status:   http.StatusInternalServerError,
		Message:  "An internal error occurred",
		Internal: context + ": " + err.Error(),
		Cause:    err,
	}
}

func Wrapf(err error, format string, args ...interface{}) error {
	return Wrap(err, fmt.Sprintf(format, args...))
}

func HTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if se, ok := IsServiceError(err); ok {
		return se.Status
	}
	return http.StatusInternalServerError
}

func SafeMessage(err error) string {
	if err == nil {
		return ""
	}
	if se, ok := IsServiceError(err); ok {
		return se.Message
	}
	return "An unexpected error occurred"
}

func SafeResponse(err error, traceID string) map[string]interface{} {
	if err == nil {
		return nil
	}
	resp := map[string]interface{}{
		"error":    SafeMessage(err),
		"trace_id": traceID,
	}
	if se, ok := IsServiceError(err); ok {
		resp["code"] = se.Code
	}
	return resp
}
