package stats

import (
	"context"
	"time"
)

// TrafficSample represents a single traffic measurement
type TrafficSample struct {
	UserID    uint
	InboundID uint
	CoreID    uint
	Upload    uint64
	Download  uint64
	Timestamp time.Time
}

// ConnectionInfo represents an active connection
type ConnectionInfo struct {
	UserID          uint
	InboundID       uint
	CoreID          uint
	CoreName        string
	SourceIP        string
	SourcePort      int
	DestinationIP   string
	DestinationPort int
	StartedAt       time.Time
	LastActivity    time.Time
	Upload          uint64
	Download        uint64
	ConnectionID    string // Core-specific connection identifier
}

// Provider defines the interface for collecting stats from different cores
type Provider interface {
	// GetTrafficStats retrieves cumulative traffic statistics
	GetTrafficStats(ctx context.Context, coreID uint) ([]TrafficSample, error)

	// GetActiveConnections retrieves list of active connections
	GetActiveConnections(ctx context.Context, coreID uint) ([]ConnectionInfo, error)

	// CloseConnection closes a specific connection
	CloseConnection(ctx context.Context, coreID uint, connectionID string) error

	// RemoveUser removes a user from the core (for quota enforcement)
	RemoveUser(ctx context.Context, coreID uint, userUUID string) error

	// Close closes the client connection
	Close() error
}
