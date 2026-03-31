package mihomo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatsClient_GetTrafficStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/traffic" {
			t.Errorf("expected /traffic, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"up": 5120, "down": 10240}`))
	}))
	defer server.Close()

	client := NewStatsClient(server.URL, "test-key")
	defer client.Close()

	stats, err := client.GetTrafficStats(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetTrafficStats() error = %v", err)
	}

	// Mihomo returns empty slice (no per-user stats)
	if len(stats) != 0 {
		t.Errorf("expected empty slice, got length %d", len(stats))
	}
}

func TestStatsClient_GetTrafficStats_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewStatsClient(server.URL, "")
	_, err := client.GetTrafficStats(context.Background(), 1)
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

func TestStatsClient_GetTrafficStats_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	client := NewStatsClient(server.URL, "")
	_, err := client.GetTrafficStats(context.Background(), 1)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestStatsClient_GetTrafficStats_ConnectionRefused(t *testing.T) {
	client := NewStatsClient("http://127.0.0.1:1", "")
	_, err := client.GetTrafficStats(context.Background(), 1)
	if err == nil {
		t.Error("expected error for connection refused, got nil")
	}
}

func TestStatsClient_GetActiveConnections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/connections" {
			t.Errorf("expected /connections, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"downloadTotal": 200,
			"uploadTotal": 100,
			"connections": [
				{
					"id": "mihomo-conn-1",
					"upload": 50,
					"download": 100,
					"start": "2026-01-15T10:30:00Z",
					"chains": ["DIRECT"],
					"rule": "MATCH",
					"metadata": {
						"network": "tcp",
						"sourceIP": "10.0.0.5",
						"sourcePort": "54321",
						"destinationIP": "1.1.1.1",
						"destinationPort": "443",
						"host": "example.com"
					}
				},
				{
					"id": "mihomo-conn-2",
					"upload": 25,
					"download": 75,
					"start": "2026-01-15T10:35:00Z",
					"metadata": {
						"sourceIP": "10.0.0.6",
						"sourcePort": "54322",
						"destinationIP": "8.8.4.4",
						"destinationPort": "80"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewStatsClient(server.URL, "test-key")

	conns, err := client.GetActiveConnections(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetActiveConnections() error = %v", err)
	}

	if len(conns) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(conns))
	}

	// Verify first connection
	c1 := conns[0]
	if c1.ConnectionID != "mihomo-conn-1" {
		t.Errorf("expected connection ID mihomo-conn-1, got %s", c1.ConnectionID)
	}
	if c1.Upload != 50 {
		t.Errorf("expected upload 50, got %d", c1.Upload)
	}
	if c1.Download != 100 {
		t.Errorf("expected download 100, got %d", c1.Download)
	}
	if c1.SourceIP != "10.0.0.5" {
		t.Errorf("expected sourceIP 10.0.0.5, got %s", c1.SourceIP)
	}
	if c1.SourcePort != 54321 {
		t.Errorf("expected sourcePort 54321, got %d", c1.SourcePort)
	}
	if c1.DestinationIP != "1.1.1.1" {
		t.Errorf("expected destIP 1.1.1.1, got %s", c1.DestinationIP)
	}
	if c1.DestinationPort != 443 {
		t.Errorf("expected destPort 443, got %d", c1.DestinationPort)
	}
	if c1.CoreID != 2 {
		t.Errorf("expected coreID 2, got %d", c1.CoreID)
	}
	if c1.CoreName != "mihomo" {
		t.Errorf("expected coreName mihomo, got %s", c1.CoreName)
	}

	// Verify second connection
	if conns[1].ConnectionID != "mihomo-conn-2" {
		t.Errorf("expected connection ID mihomo-conn-2, got %s", conns[1].ConnectionID)
	}
}

func TestStatsClient_GetActiveConnections_EmptyList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"downloadTotal": 0, "uploadTotal": 0, "connections": []}`))
	}))
	defer server.Close()

	client := NewStatsClient(server.URL, "")
	conns, err := client.GetActiveConnections(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetActiveConnections() error = %v", err)
	}
	if len(conns) != 0 {
		t.Errorf("expected 0 connections, got %d", len(conns))
	}
}

func TestStatsClient_CloseConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/connections/conn-123" {
			t.Errorf("expected /connections/conn-123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewStatsClient(server.URL, "test-key")
	err := client.CloseConnection(context.Background(), 1, "conn-123")
	if err != nil {
		t.Errorf("CloseConnection() error = %v", err)
	}
}

func TestStatsClient_CloseConnection_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewStatsClient(server.URL, "")
	err := client.CloseConnection(context.Background(), 1, "nonexistent")
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestStatsClient_RemoveUser_ReturnsError(t *testing.T) {
	client := NewStatsClient("http://localhost", "")
	err := client.RemoveUser(context.Background(), "tag", "uuid")
	if err == nil {
		t.Error("expected error for RemoveUser (requires reload), got nil")
	}
}

func TestStatsClient_NoAuthHeader_WhenKeyEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "" {
			t.Errorf("expected no auth header, got %s", auth)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"up": 0, "down": 0}`))
	}))
	defer server.Close()

	client := NewStatsClient(server.URL, "")
	_, err := client.GetTrafficStats(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStatsClient_Close(t *testing.T) {
	client := NewStatsClient("http://localhost", "")
	err := client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
