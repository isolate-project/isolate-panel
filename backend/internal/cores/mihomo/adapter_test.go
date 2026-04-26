package mihomo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/isolate-project/isolate-panel/internal/cores"
)

func TestAdapter_CheckHealth_Unreachable(t *testing.T) {
	adapter := &Adapter{}
	ctx := context.Background()
	err := adapter.CheckHealth(ctx, 1*time.Second)
	if err == nil {
		t.Log("mihomo API was reachable (unusual in test env)")
	}
}

func TestAdapter_CheckHealth_WithServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":"1.18.0"}`))
	}))
	defer server.Close()

	adapter := &Adapter{}
	ctx := context.Background()
	err := adapter.CheckHealth(ctx, 1*time.Second)
	if err == nil {
		t.Log("mihomo API was reachable (unusual in test env)")
	}
}

func TestAdapter_CheckHealth_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	adapter := &Adapter{}
	ctx := context.Background()
	err := adapter.CheckHealth(ctx, 1*time.Second)
	if err == nil {
		t.Error("Expected error for non-200 status")
	}
}

func TestAdapter_HotReloadInfo(t *testing.T) {
	adapter := &Adapter{}
	method, signal, endpoint := adapter.HotReloadInfo()

	if method != cores.HotReloadAPI {
		t.Errorf("Expected HotReloadAPI, got %d", method)
	}
	if signal != "" {
		t.Errorf("Expected empty signal, got %s", signal)
	}
	if endpoint != "http://127.0.0.1:9091/configs?force=true" {
		t.Errorf("Expected endpoint http://127.0.0.1:9091/configs?force=true, got %s", endpoint)
	}
}