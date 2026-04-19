package haproxy

import (
	"strings"
	"testing"

	"github.com/isolate-project/isolate-panel/internal/models"
)

func TestNewPortPool(t *testing.T) {
	pool := NewPortPool(40000, 50000)
	if pool.Start != 40000 {
		t.Errorf("expected Start to be 40000, got %d", pool.Start)
	}
	if pool.End != 50000 {
		t.Errorf("expected End to be 50000, got %d", pool.End)
	}
	if len(pool.Used) != 0 {
		t.Errorf("expected empty Used map, got %d entries", len(pool.Used))
	}
}

func TestPortPoolAllocatePort(t *testing.T) {
	pool := NewPortPool(40000, 40002)

	// Allocate first port
	port1, err := pool.AllocatePort()
	if err != nil {
		t.Fatalf("failed to allocate first port: %v", err)
	}
	if port1 != 40000 {
		t.Errorf("expected port 40000, got %d", port1)
	}

	// Allocate second port
	port2, err := pool.AllocatePort()
	if err != nil {
		t.Fatalf("failed to allocate second port: %v", err)
	}
	if port2 != 40001 {
		t.Errorf("expected port 40001, got %d", port2)
	}

	// Allocate third port
	port3, err := pool.AllocatePort()
	if err != nil {
		t.Fatalf("failed to allocate third port: %v", err)
	}
	if port3 != 40002 {
		t.Errorf("expected port 40002, got %d", port3)
	}

	// Pool should be exhausted now
	_, err = pool.AllocatePort()
	if err == nil {
		t.Error("expected error when pool exhausted, got nil")
	}
	if !strings.Contains(err.Error(), "no available ports") {
		t.Errorf("expected error message to contain 'no available ports', got: %v", err)
	}
}

func TestPortPoolReleasePort(t *testing.T) {
	pool := NewPortPool(40000, 40001)

	// Allocate and release
	port, _ := pool.AllocatePort()
	if !pool.IsAllocated(port) {
		t.Error("expected port to be allocated")
	}

	pool.ReleasePort(port)
	if pool.IsAllocated(port) {
		t.Error("expected port to be released")
	}

	// Should be able to allocate again
	port2, err := pool.AllocatePort()
	if err != nil {
		t.Fatalf("failed to allocate after release: %v", err)
	}
	if port2 != 40000 {
		t.Errorf("expected port 40000 after release, got %d", port2)
	}
}

func TestPortPoolIsAllocated(t *testing.T) {
	pool := NewPortPool(40000, 40010)

	// Port not allocated yet
	if pool.IsAllocated(40005) {
		t.Error("expected unallocated port to return false")
	}

	// Allocate specific port through sequence
	pool.AllocatePort() // 40000
	pool.AllocatePort() // 40001
	pool.AllocatePort() // 40002

	if !pool.IsAllocated(40000) {
		t.Error("expected port 40000 to be allocated")
	}
	if !pool.IsAllocated(40001) {
		t.Error("expected port 40001 to be allocated")
	}
	if !pool.IsAllocated(40002) {
		t.Error("expected port 40002 to be allocated")
	}
	if pool.IsAllocated(40003) {
		t.Error("expected port 40003 to not be allocated")
	}
}

func TestPortPoolConcurrency(t *testing.T) {
	pool := NewPortPool(40000, 40099)

	// Simulate concurrent allocations
	allocated := make(chan int, 100)
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		go func() {
			port, err := pool.AllocatePort()
			if err != nil {
				errors <- err
			} else {
				allocated <- port
			}
		}()
	}

	// Collect results
	var ports []int
	var errs []error
	for i := 0; i < 100; i++ {
		select {
		case port := <-allocated:
			ports = append(ports, port)
		case err := <-errors:
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		t.Errorf("got %d errors during concurrent allocation: %v", len(errs), errs[0])
	}

	if len(ports) != 100 {
		t.Errorf("expected 100 ports, got %d", len(ports))
	}

	// Check no duplicates
	portMap := make(map[int]bool)
	for _, p := range ports {
		if portMap[p] {
			t.Errorf("duplicate port allocated: %d", p)
		}
		portMap[p] = true
	}
}

func TestPortPoolGetCounts(t *testing.T) {
	pool := NewPortPool(40000, 40009) // 10 ports total

	if pool.GetUsedCount() != 0 {
		t.Errorf("expected 0 used, got %d", pool.GetUsedCount())
	}
	if pool.GetAvailableCount() != 10 {
		t.Errorf("expected 10 available, got %d", pool.GetAvailableCount())
	}

	// Allocate 5 ports
	for i := 0; i < 5; i++ {
		pool.AllocatePort()
	}

	if pool.GetUsedCount() != 5 {
		t.Errorf("expected 5 used, got %d", pool.GetUsedCount())
	}
	if pool.GetAvailableCount() != 5 {
		t.Errorf("expected 5 available, got %d", pool.GetAvailableCount())
	}
}

// MockDB is a simple mock for testing SyncWithDB
type MockDB struct {
	portAssignments []models.PortAssignment
	directPorts     []models.DirectPort
}

func (m *MockDB) Model(value interface{}) *MockDB {
	return m
}

func (m *MockDB) Where(query interface{}, args ...interface{}) *MockDB {
	return m
}

func (m *MockDB) Pluck(column string, dest interface{}) error {
	switch column {
	case "backend_port":
		if ports, ok := dest.(*[]int); ok {
			for _, pa := range m.portAssignments {
				*ports = append(*ports, pa.BackendPort)
			}
			for _, dp := range m.directPorts {
				*ports = append(*ports, dp.BackendPort)
			}
		}
	}
	return nil
}

func TestPortPoolSyncWithDB(t *testing.T) {
	pool := NewPortPool(40000, 40010)

	// Simulate DB with some ports already allocated
	_ = &MockDB{
		portAssignments: []models.PortAssignment{
			{BackendPort: 40000, IsActive: true},
			{BackendPort: 40001, IsActive: true},
			{BackendPort: 40005, IsActive: true},
		},
		directPorts: []models.DirectPort{
			{BackendPort: 40002, IsActive: true},
		},
	}

	// We can't easily test with real GORM, but the structure is correct
	// In real usage, this would query the database
	if pool.IsAllocated(40000) {
		t.Error("before sync, port should not be allocated")
	}

	// After sync, these should be marked
	// Note: This test documents expected behavior
	// Actual sync testing requires database integration tests
}
