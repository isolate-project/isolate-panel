package services

import (
	"context"

	"github.com/isolate-project/isolate-panel/internal/stats"
)

// TrafficSample is an alias for stats.TrafficSample for backward compatibility
type TrafficSample = stats.TrafficSample

// ConnectionInfo is an alias for stats.ConnectionInfo for backward compatibility
type ConnectionInfo = stats.ConnectionInfo

// CoreStatsProvider defines the interface for collecting stats from different cores
type CoreStatsProvider interface {
	// GetTrafficStats retrieves cumulative traffic statistics
	GetTrafficStats(ctx context.Context, coreID uint) ([]TrafficSample, error)

	// GetActiveConnections retrieves list of active connections
	GetActiveConnections(ctx context.Context, coreID uint) ([]ConnectionInfo, error)

	// CloseConnection closes a specific connection
	CloseConnection(ctx context.Context, coreID uint, connectionID string) error

	// RemoveUser removes a user from the core (for quota enforcement)
	RemoveUser(ctx context.Context, coreID uint, userUUID string) error
}
