package stats

import (
	"testing"
	"time"
)

func TestConnectionInfo(t *testing.T) {
	now := time.Now()
	conn := ConnectionInfo{
		ConnectionID: "123",
		UserID:       1,
		CoreID:       2,
		StartedAt:    now,
		Upload:       1024,
		Download:     2048,
	}

	if conn.ConnectionID != "123" {
		t.Errorf("Expected ConnectionID 123, got %s", conn.ConnectionID)
	}
	if conn.Upload != 1024 {
		t.Errorf("Expected Upload 1024, got %d", conn.Upload)
	}
}

func TestTrafficSample(t *testing.T) {
	now := time.Now()
	sample := TrafficSample{
		UserID:    1,
		CoreID:    2,
		Upload:    100,
		Download:  200,
		Timestamp: now,
	}

	if sample.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", sample.UserID)
	}
}
