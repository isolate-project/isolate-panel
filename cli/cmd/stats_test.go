package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vovk4morkovk4/isolate-panel/cli/pkg"
)

func TestStatsOverview(t *testing.T) {
	mock := &MockClient{
		GetFunc: func(path string, result interface{}) error {
			if path == "/api/traffic/overview" {
				res := result.(*struct {
					Data map[string]interface{} `json:"data"`
				})
				res.Data = map[string]interface{}{
					"total_up":   float64(1024),
					"total_down": float64(2048),
				}
			}
			return nil
		},
	}

	pkg.DefaultClient = mock
	statsFormat = "json" // reset for current execution

	cmd := StatsCmd()
	cmd.SetArgs([]string{"--format=json"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 API call, got %d", len(mock.Calls))
	}

	var out map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if out["total_up"] != float64(1024) {
		t.Errorf("expected total_up 1024, got %v", out["total_up"])
	}
}
