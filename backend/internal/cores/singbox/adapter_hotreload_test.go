package singbox

import (
	"context"
	"testing"
)

func TestSingboxAdapter_SupportsHotReload(t *testing.T) {
	adapter := NewAdapter()
	if !adapter.SupportsHotReload() {
		t.Error("Sing-box adapter should support hot-reload")
	}
}

func TestSingboxAdapter_ReloadConfig_NotImplemented(t *testing.T) {
	adapter := NewAdapter()
	ctx := context.Background()
	err := adapter.ReloadConfig(ctx)
	if err == nil {
		t.Error("Sing-box ReloadConfig should return error (not yet implemented)")
	}
}