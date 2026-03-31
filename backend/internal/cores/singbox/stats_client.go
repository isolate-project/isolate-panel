package singbox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/isolate-project/isolate-panel/internal/stats"
)

// StatsClient provides access to Sing-box stats via Clash API
// Sing-box exposes a Clash-compatible API when experimental.api is enabled
type StatsClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// ClashTrafficResponse represents the traffic response from Clash API
type ClashTrafficResponse struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// ClashConnectionsResponse represents active connections from Clash API
type ClashConnectionsResponse struct {
	DownloadTotal int64             `json:"downloadTotal"`
	UploadTotal   int64             `json:"uploadTotal"`
	Connections   []ClashConnection `json:"connections"`
}

// ClashConnection represents a single active connection
type ClashConnection struct {
	ID          string    `json:"id"`
	Metadata    ConnMeta  `json:"metadata"`
	Upload      int64     `json:"upload"`
	Download    int64     `json:"download"`
	Start       time.Time `json:"start"`
	Chains      []string  `json:"chains"`
	Rule        string    `json:"rule"`
	RulePayload string    `json:"rulePayload"`
	RemoteDest  string    `json:"remoteDestination"`
}

// ConnMeta represents connection metadata
type ConnMeta struct {
	Network         string `json:"network"`
	Type            string `json:"type"`
	SourceIP        string `json:"sourceIP"`
	DestinationIP   string `json:"destinationIP"`
	SourcePort      string `json:"sourcePort"`
	DestinationPort string `json:"destinationPort"`
	Host            string `json:"host"`
	DNSMode         string `json:"dnsMode"`
	SpecialProxy    string `json:"specialProxy"`
	SpecialRules    string `json:"specialRules"`
}

// NewStatsClient creates a new Sing-box stats client
func NewStatsClient(baseURL, apiKey string) *StatsClient {
	return &StatsClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// Close closes any resources (no-op for HTTP client)
func (c *StatsClient) Close() error {
	return nil
}

// GetTrafficStats retrieves traffic statistics for all users
// Sing-box Clash API doesn't provide per-user stats directly
// We need to aggregate from connection data
func (c *StatsClient) GetTrafficStats(ctx context.Context, coreID uint) ([]stats.TrafficSample, error) {
	// Get traffic overview (total uplink/downlink)
	trafficURL := fmt.Sprintf("%s/traffic", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", trafficURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get traffic stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var traffic ClashTrafficResponse
	if err := json.NewDecoder(resp.Body).Decode(&traffic); err != nil {
		return nil, fmt.Errorf("failed to decode traffic response: %w", err)
	}

	// Sing-box doesn't provide per-user stats via Clash API
	// Return empty slice as we can't map to users without additional data
	return []stats.TrafficSample{}, nil
}

// GetActiveConnections retrieves active connections from Sing-box
func (c *StatsClient) GetActiveConnections(ctx context.Context, coreID uint) ([]stats.ConnectionInfo, error) {
	connectionsURL := fmt.Sprintf("%s/connections", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", connectionsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get connections: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var connectionsResp ClashConnectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&connectionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode connections response: %w", err)
	}

	// Convert to our ConnectionInfo format
	result := make([]stats.ConnectionInfo, 0, len(connectionsResp.Connections))
	for _, conn := range connectionsResp.Connections {
		// Parse source port
		sourcePort, _ := strconv.Atoi(conn.Metadata.SourcePort)
		destPort, _ := strconv.Atoi(conn.Metadata.DestinationPort)

		// Try to extract user info from connection chains or metadata
		// Sing-box doesn't directly expose user ID in connections
		userID := uint(0)

		result = append(result, stats.ConnectionInfo{
			UserID:          userID,
			InboundID:       0,
			CoreID:          coreID,
			CoreName:        "singbox",
			SourceIP:        conn.Metadata.SourceIP,
			SourcePort:      sourcePort,
			DestinationIP:   conn.Metadata.DestinationIP,
			DestinationPort: destPort,
			StartedAt:       conn.Start,
			LastActivity:    time.Now(),
			Upload:          uint64(conn.Upload),
			Download:        uint64(conn.Download),
			ConnectionID:    conn.ID,
		})
	}

	return result, nil
}

// CloseConnection closes a specific connection by ID
func (c *StatsClient) CloseConnection(ctx context.Context, coreID uint, connectionID string) error {
	url := fmt.Sprintf("%s/connections/%s", c.baseURL, connectionID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

// RemoveUser removes a user from Sing-box
// Sing-box requires config reload for user changes
func (c *StatsClient) RemoveUser(ctx context.Context, inboundTag string, userUUID string) error {
	return fmt.Errorf("sing-box requires graceful reload for user removal")
}

// AddUser adds a user to Sing-box
// Sing-box requires config reload for user changes
func (c *StatsClient) AddUser(ctx context.Context, inboundTag string, userUUID string, protocolType string) error {
	return fmt.Errorf("sing-box requires graceful reload for user addition")
}
