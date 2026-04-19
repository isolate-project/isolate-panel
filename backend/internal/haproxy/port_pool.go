package haproxy

import (
	"fmt"
	"sync"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

type PortPool struct {
	Start int
	End   int
	Used  map[int]bool
	mu    sync.RWMutex
}

func NewPortPool(start, end int) *PortPool {
	return &PortPool{
		Start: start,
		End:   end,
		Used:  make(map[int]bool),
	}
}

func (p *PortPool) AllocatePort() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for port := p.Start; port <= p.End; port++ {
		if !p.Used[port] {
			p.Used[port] = true
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in pool %d-%d", p.Start, p.End)
}

func (p *PortPool) ReleasePort(port int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.Used, port)
}

func (p *PortPool) IsAllocated(port int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Used[port]
}

func (p *PortPool) SyncWithDB(db *gorm.DB) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var backendPorts []int
	if err := db.Model(&models.PortAssignment{}).
		Where("is_active = ?", true).
		Pluck("backend_port", &backendPorts).Error; err != nil {
		return fmt.Errorf("failed to query backend ports: %w", err)
	}

	for _, port := range backendPorts {
		if port >= p.Start && port <= p.End {
			p.Used[port] = true
		}
	}

	var directPorts []int
	if err := db.Model(&models.DirectPort{}).
		Where("is_active = ?", true).
		Pluck("backend_port", &directPorts).Error; err != nil {
		return fmt.Errorf("failed to query direct backend ports: %w", err)
	}

	for _, port := range directPorts {
		if port >= p.Start && port <= p.End {
			p.Used[port] = true
		}
	}

	return nil
}

func (p *PortPool) GetUsedCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.Used)
}

func (p *PortPool) GetAvailableCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	total := p.End - p.Start + 1
	return total - len(p.Used)
}
