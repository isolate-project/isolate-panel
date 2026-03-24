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
	reservedPorts map[int]string // port -> reason
}

// NewPortManager creates a new port manager
func NewPortManager(db *gorm.DB) *PortManager {
	return &PortManager{
		db: db,
		reservedPorts: map[int]string{
			22:    "SSH",
			80:    "HTTP (system)",
			8080:  "Panel HTTP",
			9090:  "Sing-box API",
			10085: "Xray API",
			9091:  "Mihomo API",
		},
	}
}

// IsPortAvailable checks if a port is available globally (across all cores)
// Allows any port from 1-65535 except reserved ports and conflicts
func (pm *PortManager) IsPortAvailable(port int, excludeInboundID *uint) (bool, string, error) {
	// Check if port is in valid range (1-65535)
	if port < 1 || port > 65535 {
		return false, "Port must be between 1 and 65535", nil
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

// AllocatePort finds and returns the next available port
// Prefers high ports (40000-65535) to avoid common conflicts
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

	// Try random high ports first (40000-65535) for better compatibility
	for i := 0; i < 100; i++ {
		port := 40000 + rand.Intn(25536)
		if !usedSet[port] {
			return port, nil
		}
	}

	// Fallback: try all ports from 1024-65535
	for port := 1024; port <= 65535; port++ {
		if !usedSet[port] {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports")
}

// ValidatePort validates a port number
func (pm *PortManager) ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
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
