package xray

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vovk4morkovk4/isolate-panel/internal/stats"
	"github.com/xtls/xray-core/app/proxyman/command"
	statscommand "github.com/xtls/xray-core/app/stats/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/vmess"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// StatsClient provides access to Xray stats and user management via gRPC
type StatsClient struct {
	conn          *grpc.ClientConn
	handlerClient command.HandlerServiceClient
	statsClient   statscommand.StatsServiceClient
	address       string
}

// NewStatsClient creates a new Xray stats client
func NewStatsClient(address string) (*StatsClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Xray gRPC: %w", err)
	}

	return &StatsClient{
		conn:          conn,
		handlerClient: command.NewHandlerServiceClient(conn),
		statsClient:   statscommand.NewStatsServiceClient(conn),
		address:       address,
	}, nil
}

// Close closes the gRPC connection
func (c *StatsClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetTrafficStats retrieves traffic statistics for all users
func (c *StatsClient) GetTrafficStats(ctx context.Context, coreID uint) ([]stats.TrafficSample, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Query all user stats using pattern matching
	// Xray stats format: user>>>email>>>traffic>>>uplink/downlink
	req := &statscommand.QueryStatsRequest{
		Pattern: "user>>>",
		Reset_:  false,
	}

	resp, err := c.statsClient.QueryStats(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}

	// Parse stats and group by user email
	userStats := make(map[string]*struct {
		Upload   uint64
		Download uint64
	})

	for _, stat := range resp.Stat {
		// Parse stat name: user>>>email>>>traffic>>>uplink/downlink
		parts := strings.Split(stat.Name, ">>>")
		if len(parts) < 4 {
			continue
		}

		email := parts[1]
		direction := parts[3]

		if _, exists := userStats[email]; !exists {
			userStats[email] = &struct {
				Upload   uint64
				Download uint64
			}{}
		}

		if direction == "uplink" {
			userStats[email].Upload = uint64(stat.Value)
		} else if direction == "downlink" {
			userStats[email].Download = uint64(stat.Value)
		}
	}

	// Convert to TrafficSample slice
	// Email format: user_<user_id>_<inbound_id> (convention used in config generation)
	samples := make([]stats.TrafficSample, 0, len(userStats))
	for email, s := range userStats {
		var userID, inboundID uint

		// Parse email format: user_<user_id>_<inbound_id>
		parts := strings.Split(email, "_")
		if len(parts) >= 3 {
			if id, err := strconv.ParseUint(parts[1], 10, 32); err == nil {
				userID = uint(id)
			}
			if id, err := strconv.ParseUint(parts[2], 10, 32); err == nil {
				inboundID = uint(id)
			}
		}

		samples = append(samples, stats.TrafficSample{
			UserID:    userID,
			InboundID: inboundID,
			CoreID:    coreID,
			Upload:    s.Upload,
			Download:  s.Download,
			Timestamp: time.Now(),
		})
	}

	return samples, nil
}

// GetActiveConnections retrieves active connections (placeholder)
// Xray doesn't provide per-connection info via gRPC API
func (c *StatsClient) GetActiveConnections(ctx context.Context, coreID uint) ([]stats.ConnectionInfo, error) {
	// Xray doesn't expose individual connection details via gRPC
	// Return empty slice - connections will be tracked at network level
	return []stats.ConnectionInfo{}, nil
}

// CloseConnection closes a specific connection (not supported by Xray gRPC)
func (c *StatsClient) CloseConnection(ctx context.Context, coreID uint, connectionID string) error {
	return fmt.Errorf("Xray doesn't support closing individual connections via gRPC")
}

// RemoveUser removes a user from an inbound without restarting Xray
func (c *StatsClient) RemoveUser(ctx context.Context, inboundTag string, userUUID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Xray uses email field for user identification
	email := fmt.Sprintf("user_%s", userUUID)

	req := &command.AlterInboundRequest{
		Tag: inboundTag,
		Operation: serial.ToTypedMessage(&command.RemoveUserOperation{
			Email: email,
		}),
	}

	_, err := c.handlerClient.AlterInbound(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}

	return nil
}

// AddUser adds a user to an inbound without restarting Xray
func (c *StatsClient) AddUser(ctx context.Context, inboundTag string, userUUID string, protocolType string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	email := fmt.Sprintf("user_%s", userUUID)

	// Create user based on protocol
	var user *protocol.User
	switch protocolType {
	case "vmess":
		user = &protocol.User{
			Level: 0,
			Email: email,
			Account: serial.ToTypedMessage(&vmess.Account{
				Id: userUUID,
			}),
		}
	default:
		return fmt.Errorf("unsupported protocol: %s", protocolType)
	}

	req := &command.AlterInboundRequest{
		Tag: inboundTag,
		Operation: serial.ToTypedMessage(&command.AddUserOperation{
			User: user,
		}),
	}

	_, err := c.handlerClient.AlterInbound(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	return nil
}
