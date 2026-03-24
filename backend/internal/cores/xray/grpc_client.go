package xray

import (
	"context"
	"fmt"
)

// GRPCClient is a placeholder for Xray gRPC client
// Full implementation requires complex dependencies
// For now, we use graceful reload for all operations
type GRPCClient struct {
	address string
}

// NewGRPCClient creates a new Xray gRPC client (placeholder)
func NewGRPCClient(address string) (*GRPCClient, error) {
	return &GRPCClient{
		address: address,
	}, nil
}

// Close closes the gRPC connection
func (c *GRPCClient) Close() error {
	return nil
}

// GetStats retrieves a specific stat from Xray (placeholder)
func (c *GRPCClient) GetStats(ctx context.Context, name string) (uint64, error) {
	return 0, fmt.Errorf("not implemented - use stats API via core manager")
}

// QueryStats queries multiple stats with a pattern (placeholder)
func (c *GRPCClient) QueryStats(ctx context.Context, pattern string) (map[string]uint64, error) {
	return nil, fmt.Errorf("not implemented - use stats API via core manager")
}

// RemoveUser removes a user from an inbound (placeholder)
// Actual implementation requires graceful reload
func (c *GRPCClient) RemoveUser(ctx context.Context, tag string, userUUID string) error {
	return fmt.Errorf("not implemented - use graceful reload instead")
}

// AddUser adds a user to an inbound (placeholder)
func (c *GRPCClient) AddUser(ctx context.Context, tag string, userUUID string) error {
	return fmt.Errorf("not implemented - use graceful reload instead")
}
