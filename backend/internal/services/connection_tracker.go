package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// ConnectionTracker tracks active user connections
type ConnectionTracker struct {
	db       *gorm.DB
	interval time.Duration
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex

	// In-memory connection cache (for real-time tracking)
	connections map[string]*models.ActiveConnection // connection_id -> connection
}

// NewConnectionTracker creates a new connection tracker
func NewConnectionTracker(db *gorm.DB, interval time.Duration) *ConnectionTracker {
	if interval == 0 {
		interval = 10 * time.Second // Default: 10 seconds for real-time feel
	}

	return &ConnectionTracker{
		db:          db,
		interval:    interval,
		stopChan:    make(chan struct{}),
		connections: make(map[string]*models.ActiveConnection),
	}
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

	for _, core := range cores {
		// TODO: Get connections from each core via their API
		// For now, get connections from database
		ct.loadConnectionsFromDB(ctx, core.ID)
	}

	// Update last activity timestamp
	ct.mu.RLock()
	for _, conn := range ct.connections {
		conn.LastActivity = now
		ct.db.Save(conn)
	}
	ct.mu.RUnlock()
}

// loadConnectionsFromDB loads connections from database (placeholder)
func (ct *ConnectionTracker) loadConnectionsFromDB(ctx context.Context, coreID uint) {
	var connections []models.ActiveConnection
	if err := ct.db.Where("core_id = ?", coreID).Find(&connections).Error; err != nil {
		return
	}

	ct.mu.Lock()
	for i := range connections {
		key := ct.connectionKey(coreID, connections[i].UserID, connections[i].ID)
		ct.connections[key] = &connections[i]
	}
	ct.mu.Unlock()
}

// connectionKey generates a unique key for a connection
func (ct *ConnectionTracker) connectionKey(coreID, userID, connID uint) string {
	return fmt.Sprintf("%d-%d-%d", coreID, userID, connID)
}

// AddConnection adds a new active connection
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
