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
<<<<<<< Updated upstream
	// Get traffic overview (total uplink/downlink)
	trafficURL := fmt.Sprintf("%s/traffic", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", trafficURL, nil)
=======
	if c.grpcAddr == "" {
		// Fallback to Clash API when gRPC is not available
		logger.Log.Warn().Msg("sing-box: v2ray_api gRPC not configured, falling back to Clash API for traffic stats (per-user stats unavailable)")
		return c.getTrafficStatsFromClashAPI(ctx, coreID)
	}

	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req := &statscommand.QueryStatsRequest{
		Pattern: "user>>>",
		Reset_:  false,
	}

	resp, err := c.statsClient.QueryStats(ctx, req)
>>>>>>> Stashed changes
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

<<<<<<< Updated upstream
	var traffic ClashTrafficResponse
	if err := json.NewDecoder(resp.Body).Decode(&traffic); err != nil {
		return nil, fmt.Errorf("failed to decode traffic response: %w", err)
	}

	// Sing-box doesn't provide per-user stats via Clash API
	// Return empty slice as we can't map to users without additional data
	return []stats.TrafficSample{}, nil
=======
	return samples, nil
}

// getTrafficStatsFromClashAPI retrieves traffic stats from Clash API /connections endpoint
// This is a fallback when gRPC is not available. Returns aggregate traffic as a single sample.
func (c *StatsClient) getTrafficStatsFromClashAPI(ctx context.Context, coreID uint) ([]stats.TrafficSample, error) {
	connectionsURL := fmt.Sprintf("%s/connections", c.clashBaseURL)
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

	// Aggregate total traffic from all connections
	var totalUpload, totalDownload uint64
	for _, conn := range connectionsResp.Connections {
		totalUpload += uint64(conn.Upload)
		totalDownload += uint64(conn.Download)
	}

	// Return a single aggregate sample with userID=0 (will be skipped by TrafficCollector)
	// This at least provides traffic data even if per-user resolution is unavailable
	return []stats.TrafficSample{
		{
			UserID:    0,
			InboundID: 0,
			CoreID:    coreID,
			Upload:    totalUpload,
			Download:  totalDownload,
			Timestamp: time.Now(),
		},
	}, nil
}

func parseEmailToIDs(email string) (userID uint, inboundID uint) {
	if !strings.HasPrefix(email, "user_") {
		return 0, 0
	}
	parts := strings.SplitN(strings.TrimPrefix(email, "user_"), "::", 2)
	id, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, 0
	}
	userID = uint(id)
	if len(parts) >= 2 {
		if ibID, err := strconv.ParseUint(parts[1], 10, 32); err == nil {
			inboundID = uint(ibID)
		}
	}
	return userID, inboundID
>>>>>>> Stashed changes
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
		inboundID := uint(0)

		// Try to extract userID from rule field or metadata
		// Rule format may contain user information like "user_<id>::<inboundID>"
		if conn.Rule != "" {
			if strings.HasPrefix(conn.Rule, "user_") {
				userID, inboundID = parseEmailToIDs(conn.Rule)
			}
		}

		// TODO: Enhance userID resolution by checking metadata.host against user patterns
		// or by maintaining a mapping of sourceIP+sourcePort to userID from recent traffic samples

		result = append(result, stats.ConnectionInfo{
			UserID:          userID,
			InboundID:       inboundID,
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
