package xray

import (
	"context"
	"testing"
)

func TestXrayAdapter_SupportsHotReload(t *testing.T) {
	adapter := NewAdapter()
	if adapter.SupportsHotReload() {
		t.Error("Xray adapter should NOT support hot-reload (falls back to restart)")
	}
}

func TestXrayAdapter_ReloadConfig_NotImplemented(t *testing.T) {
	adapter := NewAdapter()
	ctx := context.Background()
	err := adapter.ReloadConfig(ctx)
	if err == nil {
		t.Error("Xray ReloadConfig should return error (not yet implemented)")
	}
}