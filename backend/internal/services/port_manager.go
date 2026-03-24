package services

import (
	"fmt"
	"math/rand"
	"sync"

	"gorm.io/gorm"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
)

// PortManager handles port validation, conflict detection, and auto-allocation
type PortManager struct {
	db            *gorm.DB
	mu            sync.Mutex
	minPort       int
	maxPort       int
	reservedPorts map[int]string // port -> reason
}

// NewPortManager creates a new port manager
func NewPortManager(db *gorm.DB) *PortManager {
	return &PortManager{
		db:      db,
		minPort: 10000,
		maxPort: 60000,
		reservedPorts: map[int]string{
			22:    "SSH",
			53:    "DNS",
			80:    "HTTP",
			443:   "HTTPS",
			8080:  "Panel HTTP",
			8443:  "Panel HTTPS",
			9090:  "Sing-box API",
			10085: "Xray API",
			9097:  "Mihomo API",
		},
	}
}

// IsPortAvailable checks if a port is available globally (across all cores)
func (pm *PortManager) IsPortAvailable(port int, excludeInboundID *uint) (bool, string, error) {
	// Check if port is in valid range
	if port < 1024 || port > 65535 {
		return false, "Port must be between 1024 and 65535", nil
	}

	// Check reserved ports
	if reason, reserved := pm.reservedPorts[port]; reserved {
		return false, fmt.Sprintf("Port %d is reserved for %s", port, reason), nil
	}

	// Check against all inbounds (global, not per-core)
	var existing models.Inbound
	query := pm.db.Where("port = ?", port)
	if excludeInboundID != nil {
		query = query.Where("id != ?", *excludeInboundID)
	}

	err := query.First(&existing).Error
	if err == nil {
		return false, fmt.Sprintf("Port %d is already in use by inbound '%s'", port, existing.Name), nil
	}

	return true, "", nil
}

// AllocatePort finds and returns the next available port in the configured range
func (pm *PortManager) AllocatePort() (int, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Get all used ports
	var usedPorts []int
	if err := pm.db.Model(&models.Inbound{}).Pluck("port", &usedPorts).Error; err != nil {
		return 0, fmt.Errorf("failed to query used ports: %w", err)
	}

	usedSet := make(map[int]bool)
	for _, p := range usedPorts {
		usedSet[p] = true
	}

	// Add reserved ports to used set
	for p := range pm.reservedPorts {
		usedSet[p] = true
	}

	// Try random ports first for better distribution
	for i := 0; i < 100; i++ {
		port := pm.minPort + rand.Intn(pm.maxPort-pm.minPort+1)
		if !usedSet[port] {
			return port, nil
		}
	}

	// Fallback: linear scan
	for port := pm.minPort; port <= pm.maxPort; port++ {
		if !usedSet[port] {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", pm.minPort, pm.maxPort)
}

// ValidatePort validates a port number
func (pm *PortManager) ValidatePort(port int) error {
	if port < 1024 || port > 65535 {
		return fmt.Errorf("port must be between 1024 and 65535")
	}
	if reason, reserved := pm.reservedPorts[port]; reserved {
		return fmt.Errorf("port %d is reserved for %s", port, reason)
	}
	return nil
}

// GetUsedPorts returns all ports currently in use
func (pm *PortManager) GetUsedPorts() ([]int, error) {
	var ports []int
	if err := pm.db.Model(&models.Inbound{}).Pluck("port", &ports).Error; err != nil {
		return nil, fmt.Errorf("failed to query used ports: %w", err)
	}
	return ports, nil
}

// GetReservedPorts returns the list of reserved ports
func (pm *PortManager) GetReservedPorts() map[int]string {
	return pm.reservedPorts
}
