package leak_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain verifies no goroutine leaks across all tests in this package
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// TestNoGoroutineLeaks is a simple test to ensure no goroutine leaks
func TestNoGoroutineLeaks(t *testing.T) {
	// This test will fail if there are any goroutine leaks
	defer goleak.VerifyNone(t)

	// Add test code here that uses goroutines
	// The defer above will verify they're cleaned up
}
