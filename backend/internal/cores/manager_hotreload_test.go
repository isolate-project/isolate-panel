package cores

import (
	"context"
	"testing"
)

func TestCoreManager_ReloadCore_NoAdapter(t *testing.T) {
	cm := &CoreManager{}
	err := cm.ReloadCore(context.Background(), "nonexistent")
	if err == nil {
		t.Error("ReloadCore should return error for unknown core")
	}
	if err.Error() != "no adapter for core nonexistent: unknown core type: nonexistent" {
		t.Errorf("Expected 'no adapter for core nonexistent: unknown core type: nonexistent', got '%s'", err.Error())
	}
}