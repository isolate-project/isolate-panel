package xray

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/isolate-project/isolate-panel/internal/cores"
)

func TestAdapter_CheckHealth_Unreachable(t *testing.T) {
	adapter := &Adapter{}
	ctx := context.Background()
	err := adapter.CheckHealth(ctx, 1*time.Second)
	if err == nil {
		t.Log("xray gRPC API was reachable (unusual in test env)")
	}
}

func TestAdapter_CheckHealth_WithListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:10085")
	if err != nil {
		t.Skipf("Cannot start test listener on 127.0.0.1:10085: %v", err)
	}
	defer listener.Close()

	adapter := &Adapter{}
	ctx := context.Background()
	err = adapter.CheckHealth(ctx, 1*time.Second)
	if err != nil {
		t.Errorf("Expected no error with listener running, got: %v", err)
	}
}

func TestAdapter_HotReloadInfo(t *testing.T) {
	adapter := &Adapter{}
	method, signal, endpoint := adapter.HotReloadInfo()

	if method != cores.HotReloadNone {
		t.Errorf("Expected HotReloadNone, got %d", method)
	}
	if signal != "" {
		t.Errorf("Expected empty signal, got %s", signal)
	}
	if endpoint != "" {
		t.Errorf("Expected empty endpoint, got %s", endpoint)
	}
}