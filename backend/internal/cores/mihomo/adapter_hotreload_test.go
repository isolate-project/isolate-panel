package mihomo

import (
	"context"
	"testing"
)

func TestMihomoAdapter_SupportsHotReload(t *testing.T) {
	adapter := NewAdapter()
	if !adapter.SupportsHotReload() {
		t.Error("Mihomo adapter should support hot-reload")
	}
}

func TestMihomoAdapter_ReloadConfig_NotImplemented(t *testing.T) {
	adapter := NewAdapter()
	ctx := context.Background()
	err := adapter.ReloadConfig(ctx)
	if err == nil {
		t.Error("Mihomo ReloadConfig should return error (not yet implemented)")
	}
}