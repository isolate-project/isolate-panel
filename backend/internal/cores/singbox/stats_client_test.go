package singbox

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatsClient_GetTrafficStats(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/traffic" {
			t.Errorf("expected /traffic, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"up": 1024, "down": 2048}`))
	}))
	defer server.Close()

	client := NewStatsClient(server.URL, "test-key")
	defer client.Close()

	stats, err := client.GetTrafficStats(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetTrafficStats() error = %v", err)
	}

	// Always returns empty slice currently because it doesn't support per-user stats
	if len(stats) != 0 {
		t.Errorf("expected empty slice, got length %d", len(stats))
	}
}

func TestStatsClient_GetActiveConnections(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/connections" {
			t.Errorf("expected /connections, got %s", r.URL.Path)
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"downloadTotal": 100,
			"uploadTotal": 50,
			"connections": [
				{
					"id": "conn1",
					"upload": 10,
					"download": 20,
					"start": "2026-01-01T00:00:00Z",
					"metadata": {
						"sourceIP": "192.168.1.1",
						"sourcePort": "12345",
						"destinationIP": "8.8.8.8",
						"destinationPort": "443"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewStatsClient(server.URL, "test-key")

	conns, err := client.GetActiveConnections(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetActiveConnections() error = %v", err)
	}

	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}

	if conns[0].ConnectionID != "conn1" {
		t.Errorf("expected connection ID conn1, got %s", conns[0].ConnectionID)
	}
	if conns[0].Upload != 10 {
		t.Errorf("expected upload 10, got %d", conns[0].Upload)
	}
	if conns[0].SourceIP != "192.168.1.1" {
		t.Errorf("expected sourceIP 192.168.1.1, got %s", conns[0].SourceIP)
	}
}
