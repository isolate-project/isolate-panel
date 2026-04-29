package repository

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Domain errors for repository layer
var (
	// ErrNotFound indicates that the requested resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists indicates that a resource with the same unique identifier already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput indicates that the provided input is invalid
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates that the operation is not authorized
	ErrUnauthorized = errors.New("unauthorized")

	// ErrDatabase indicates a generic database error
	ErrDatabase = errors.New("database error")

	// ErrTransaction indicates a transaction error
	ErrTransaction = errors.New("transaction error")
)

// WrapError wraps a GORM or other low-level error into a domain error
func WrapError(err error) error {
	if err == nil {
		return nil
	}

	// Handle GORM specific errors
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrAlreadyExists
	}

	if errors.Is(err, gorm.ErrInvalidData) {
		return ErrInvalidInput
	}

	if errors.Is(err, gorm.ErrInvalidTransaction) {
		return ErrTransaction
	}

	// Return the original error wrapped with context
	return fmt.Errorf("%w: %w", ErrDatabase, err)
}

// IsNotFound checks if the error is ErrNotFound
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsAlreadyExists checks if the error is ErrAlreadyExists
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// IsInvalidInput checks if the error is ErrInvalidInput
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// IsUnauthorized checks if the error is ErrUnauthorized
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}
