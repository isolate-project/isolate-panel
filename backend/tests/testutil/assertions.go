package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// AssertErrorIs checks if error is of expected type or message
func AssertErrorIs(t *testing.T, err error, expectedMessage string) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected error containing %q, got nil", expectedMessage)
		return
	}
	if !assert.Contains(t, err.Error(), expectedMessage) {
		t.Errorf("Expected error containing %q, got %q", expectedMessage, err.Error())
	}
}

// AssertNoError checks if error is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	assert.NoError(t, err)
}

// AssertError checks if error is not nil
func AssertError(t *testing.T, err error) {
	t.Helper()
	assert.Error(t, err)
}

// AssertRecordNotFound checks if error is gorm.ErrRecordNotFound
func AssertRecordNotFound(t *testing.T, err error) {
	t.Helper()
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// AssertEqual compares two values
func AssertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	assert.Equal(t, expected, actual)
}

// AssertNotEmpty checks if value is not empty
func AssertNotEmpty(t *testing.T, value interface{}) {
	t.Helper()
	assert.NotEmpty(t, value)
}

// AssertTrue checks if value is true
func AssertTrue(t *testing.T, value bool) {
	t.Helper()
	assert.True(t, value)
}

// AssertFalse checks if value is false
func AssertFalse(t *testing.T, value bool) {
	t.Helper()
	assert.False(t, value)
}

// AssertNil checks if value is nil
func AssertNil(t *testing.T, value interface{}) {
	t.Helper()
	assert.Nil(t, value)
}

// AssertNotNil checks if value is not nil
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()
	assert.NotNil(t, value)
}

// AssertLen checks if slice/map has expected length
func AssertLen(t *testing.T, value interface{}, length int) {
	t.Helper()
	assert.Len(t, value, length)
}

// AssertGreater checks if value is greater than expected
func AssertGreater[T int | int64 | int32 | float64](t *testing.T, value, expected T) {
	t.Helper()
	assert.Greater(t, value, expected)
}

// AssertZero checks if value is zero
func AssertZero(t *testing.T, value interface{}) {
	t.Helper()
	assert.Zero(t, value)
}

// AssertNotZero checks if value is not zero
func AssertNotZero(t *testing.T, value interface{}) {
	t.Helper()
	assert.NotZero(t, value)
}
