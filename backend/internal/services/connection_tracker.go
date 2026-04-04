package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/isolate-project/isolate-panel/internal/cores/mihomo"
	"github.com/isolate-project/isolate-panel/internal/cores/singbox"
	"github.com/isolate-project/isolate-panel/internal/cores/xray"
	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// ConnectionTracker tracks active user connections
type ConnectionTracker struct {
	db            *gorm.DB
	interval      time.Duration
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
	xrayClient    *xray.StatsClient
	singboxClient *singbox.StatsClient
	mihomoClient  *mihomo.StatsClient

	// In-memory connection cache (for real-time tracking)
	connections map[string]*models.ActiveConnection // connection_id -> connection
}

// NewConnectionTracker creates a new connection tracker
func NewConnectionTracker(
	db *gorm.DB,
	interval time.Duration,
	xrayAddr, singboxAddr, mihomoAddr string,
	singboxAPIKey, mihomoAPIKey string,
) *ConnectionTracker {
	if interval == 0 {
		interval = 10 * time.Second // Default: 10 seconds for real-time feel
	}

	ct := &ConnectionTracker{
		db:          db,
		interval:    interval,
		stopChan:    make(chan struct{}),
		connections: make(map[string]*models.ActiveConnection),
	}

	// Initialize Xray client
	if xrayAddr != "" {
		client, err := xray.NewStatsClient(xrayAddr)
		if err == nil {
			ct.xrayClient = client
		}
	}

	// Initialize Sing-box client
	if singboxAddr != "" {
		ct.singboxClient = singbox.NewStatsClient(singboxAddr, singboxAPIKey)
	}

	// Initialize Mihomo client
	if mihomoAddr != "" {
		ct.mihomoClient = mihomo.NewStatsClient(mihomoAddr, mihomoAPIKey)
	}

	return ct
}

// Start starts the connection tracking loop
func (ct *ConnectionTracker) Start() {
	ct.wg.Add(1)
	go func() {
		defer ct.wg.Done()
		ticker := time.NewTicker(ct.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ct.updateConnections()
			case <-ct.stopChan:
				return
			}
		}
	}()
}

// Stop stops the connection tracking loop
func (ct *ConnectionTracker) Stop() {
	close(ct.stopChan)
	ct.wg.Wait()

	// Close clients
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if ct.xrayClient != nil {
		ct.xrayClient.Close()
	}
}

// updateConnections updates the connection list from all cores
func (ct *ConnectionTracker) updateConnections() {
	ctx := context.Background()
	now := time.Now()

	// Get all running cores
	var cores []models.Core
	if err := ct.db.Where("is_running = ?", true).Find(&cores).Error; err != nil {
		return
	}

	// Clear old connections from cache
	ct.mu.Lock()
	ct.connections = make(map[string]*models.ActiveConnection)
	ct.mu.Unlock()

	// Collect connections from all cores
	var allConnections []models.ActiveConnection
	for _, core := range cores {
		conns, err := ct.getCoreConnections(ctx, core)
		if err != nil {
			continue
		}
		allConnections = append(allConnections, conns...)
	}

	// Upsert connections to database and cache
	ct.mu.Lock()
	for i := range allConnections {
		conn := &allConnections[i]
		conn.LastActivity = now

		var existing models.ActiveConnection
		ct.db.Where("core_id = ? AND user_id = ? AND source_ip = ? AND source_port = ?",
			conn.CoreID, conn.UserID, conn.SourceIP, conn.SourcePort).
			Assign(models.ActiveConnection{
				LastActivity: now,
				Upload:       conn.Upload,
				Download:     conn.Download,
			}).
			FirstOrCreate(&existing)

		conn.ID = existing.ID
		if existing.StartedAt.IsZero() {
			existing.StartedAt = now
			ct.db.Save(&existing)
		}

		key := ct.connectionKey(conn.CoreID, conn.UserID, conn.ID)
		ct.connections[key] = conn
	}
	ct.mu.Unlock()

	// Cleanup stale connections (not seen in last 2 minutes)
	ct.CleanupStaleConnections(2 * time.Minute)
}

// getCoreConnections retrieves connections from a core using the appropriate provider
func (ct *ConnectionTracker) getCoreConnections(ctx context.Context, core models.Core) ([]models.ActiveConnection, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var connInfos []ConnectionInfo
	var err error

	switch core.Name {
	case "xray":
		if ct.xrayClient == nil {
			return []models.ActiveConnection{}, fmt.Errorf("xray client not initialized")
		}
		connInfos, err = ct.xrayClient.GetActiveConnections(ctx, core.ID)

	case "singbox":
		if ct.singboxClient == nil {
			return []models.ActiveConnection{}, fmt.Errorf("singbox client not initialized")
		}
		connInfos, err = ct.singboxClient.GetActiveConnections(ctx, core.ID)

	case "mihomo":
		if ct.mihomoClient == nil {
			return []models.ActiveConnection{}, fmt.Errorf("mihomo client not initialized")
		}
		connInfos, err = ct.mihomoClient.GetActiveConnections(ctx, core.ID)

	default:
		return []models.ActiveConnection{}, fmt.Errorf("unknown core: %s", core.Name)
	}

	if err != nil {
		return []models.ActiveConnection{}, err
	}

	// Convert ConnectionInfo to models.ActiveConnection
	result := make([]models.ActiveConnection, 0, len(connInfos))
	for _, info := range connInfos {
		conn := models.ActiveConnection{
			UserID:          info.UserID,
			InboundID:       info.InboundID,
			CoreID:          info.CoreID,
			CoreName:        info.CoreName,
			SourceIP:        info.SourceIP,
			SourcePort:      info.SourcePort,
			DestinationIP:   info.DestinationIP,
			DestinationPort: info.DestinationPort,
			StartedAt:       info.StartedAt,
			LastActivity:    info.LastActivity,
			Upload:          info.Upload,
			Download:        info.Download,
		}
		result = append(result, conn)
	}

	return result, nil
}

// connectionKey generates a unique key for a connection
func (ct *ConnectionTracker) connectionKey(coreID, userID, connID uint) string {
	return fmt.Sprintf("%d-%d-%d", coreID, userID, connID)
}

// AddConnection adds a new active connection (manual addition)
func (ct *ConnectionTracker) AddConnection(conn *models.ActiveConnection) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	conn.StartedAt = time.Now()
	conn.LastActivity = time.Now()

	return ct.db.Create(conn).Error
}

// RemoveConnection removes a connection
func (ct *ConnectionTracker) RemoveConnection(connID uint) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	return ct.db.Delete(&models.ActiveConnection{}, connID).Error
}

// GetUserConnections gets all active connections for a user
func (ct *ConnectionTracker) GetUserConnections(userID uint) ([]models.ActiveConnection, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var connections []models.ActiveConnection
	err := ct.db.Where("user_id = ?", userID).Find(&connections).Error
	return connections, err
}

// GetActiveConnectionsCount gets total active connections count
func (ct *ConnectionTracker) GetActiveConnectionsCount() (int64, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var count int64
	err := ct.db.Model(&models.ActiveConnection{}).Count(&count).Error
	return count, err
}

// CleanupStaleConnections removes connections that haven't been active recently
func (ct *ConnectionTracker) CleanupStaleConnections(threshold time.Duration) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	cutoff := time.Now().Add(-threshold)
	return ct.db.Where("last_activity < ?", cutoff).Delete(&models.ActiveConnection{}).Error
}

// CloseUserConnection closes a specific user connection by core and connection ID
func (ct *ConnectionTracker) CloseUserConnection(ctx context.Context, coreName string, coreID uint, connectionID string) error {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	switch coreName {
	case "xray":
		if ct.xrayClient == nil {
			return fmt.Errorf("xray client not initialized")
		}
		return ct.xrayClient.CloseConnection(ctx, coreID, connectionID)

	case "singbox":
		if ct.singboxClient == nil {
			return fmt.Errorf("singbox client not initialized")
		}
		return ct.singboxClient.CloseConnection(ctx, coreID, connectionID)

	case "mihomo":
		if ct.mihomoClient == nil {
			return fmt.Errorf("mihomo client not initialized")
		}
		return ct.mihomoClient.CloseConnection(ctx, coreID, connectionID)

	default:
		return fmt.Errorf("unknown core: %s", coreName)
	}
}

// RemoveUserFromCore removes a user from a specific core
func (ct *ConnectionTracker) RemoveUserFromCore(ctx context.Context, coreName string, inboundTag string, userUUID string) error {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	switch coreName {
	case "xray":
		if ct.xrayClient == nil {
			return fmt.Errorf("xray client not initialized")
		}
		return ct.xrayClient.RemoveUser(ctx, inboundTag, userUUID)

	case "singbox":
		if ct.singboxClient == nil {
			return fmt.Errorf("singbox client not initialized")
		}
		return ct.singboxClient.RemoveUser(ctx, inboundTag, userUUID)

	case "mihomo":
		if ct.mihomoClient == nil {
			return fmt.Errorf("mihomo client not initialized")
		}
		return ct.mihomoClient.RemoveUser(ctx, inboundTag, userUUID)

	default:
		return fmt.Errorf("unknown core: %s", coreName)
	}
}
