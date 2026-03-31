package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/isolate-project/isolate-panel/cli/pkg"
)

func TestCoreList(t *testing.T) {
	mock := &MockClient{
		GetFunc: func(path string, result interface{}) error {
			if path == "/api/cores" {
				res := result.(*struct {
					Data []map[string]interface{} `json:"data"`
				})
				res.Data = []map[string]interface{}{
					{"name": "singbox", "status": "running", "uptime": "12m", "version": "1.8.0"},
					{"name": "xray", "status": "stopped", "uptime": "0s", "version": "1.7.5"},
				}
			}
			return nil
		},
	}
	
	pkg.DefaultClient = mock
	coreFormat = "json"

	cmd := CoreCmd()
	cmd.SetArgs([]string{"list", "--format=json"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 API call, got %d", len(mock.Calls))
	}
	if mock.Calls[0].Path != "/api/cores" {
		t.Errorf("expected path /api/cores, got %s", mock.Calls[0].Path)
	}

	var out []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if len(out) != 2 {
		t.Errorf("expected 2 cores, got %d", len(out))
	}
	if out[0]["status"] != "running" {
		t.Errorf("expected running, got %v", out[0]["status"])
	}
}

func TestCoreStart(t *testing.T) {
	mock := &MockClient{
		PostFunc: func(path string, body interface{}, result interface{}) error {
			if path == "/api/cores/xray/start" {
				res := result.(*map[string]interface{})
				*res = map[string]interface{}{
					"status": "started",
				}
			}
			return nil
		},
	}
	
	pkg.DefaultClient = mock
	coreFormat = "table"

	cmd := CoreCmd()
	cmd.SetArgs([]string{"start", "xray"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 API call, got %d", len(mock.Calls))
	}
	
	if mock.Calls[0].Path != "/api/cores/xray/start" {
		t.Errorf("expected path /api/cores/xray/start, got %s", mock.Calls[0].Path)
	}
}
