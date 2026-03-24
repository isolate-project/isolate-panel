package mihomo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/vovk4morkovk4/isolate-panel/internal/stats"
)

// StatsClient provides access to Mihomo stats via External Controller API
// Mihomo (Clash.Meta) exposes a REST API for statistics and connection management
type StatsClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// MihomoTrafficResponse represents the traffic response from Mihomo API
type MihomoTrafficResponse struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// MihomoConnectionsResponse represents active connections from Mihomo API
type MihomoConnectionsResponse struct {
	DownloadTotal int64              `json:"downloadTotal"`
	UploadTotal   int64              `json:"uploadTotal"`
	Connections   []MihomoConnection `json:"connections"`
}

// MihomoConnection represents a single active connection
type MihomoConnection struct {
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
	UID             int    `json:"uid"`
	Process         string `json:"process"`
	ProcessPath     string `json:"processPath"`
}

// NewStatsClient creates a new Mihomo stats client
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
// Mihomo API doesn't provide per-user stats directly
func (c *StatsClient) GetTrafficStats(ctx context.Context, coreID uint) ([]stats.TrafficSample, error) {
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

	var traffic MihomoTrafficResponse
	if err := json.NewDecoder(resp.Body).Decode(&traffic); err != nil {
		return nil, fmt.Errorf("failed to decode traffic response: %w", err)
	}

	// Mihomo doesn't provide per-user stats via API
	return []stats.TrafficSample{}, nil
}

// GetActiveConnections retrieves active connections from Mihomo
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

	var connectionsResp MihomoConnectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&connectionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode connections response: %w", err)
	}

	// Convert to our ConnectionInfo format
	result := make([]stats.ConnectionInfo, 0, len(connectionsResp.Connections))
	for _, conn := range connectionsResp.Connections {
		sourcePort, _ := strconv.Atoi(conn.Metadata.SourcePort)
		destPort, _ := strconv.Atoi(conn.Metadata.DestinationPort)

		userID := uint(0)

		result = append(result, stats.ConnectionInfo{
			UserID:          userID,
			InboundID:       0,
			CoreID:          coreID,
			CoreName:        "mihomo",
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

// RemoveUser removes a user from Mihomo
func (c *StatsClient) RemoveUser(ctx context.Context, inboundTag string, userUUID string) error {
	return fmt.Errorf("mihomo requires graceful reload for user removal")
}

// AddUser adds a user to Mihomo
func (c *StatsClient) AddUser(ctx context.Context, inboundTag string, userUUID string, protocolType string) error {
	return fmt.Errorf("mihomo requires graceful reload for user addition")
}
