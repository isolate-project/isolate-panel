package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/vovk4morkovk4/isolate-panel/cli/pkg"
)

func TestUserList(t *testing.T) {
	mock := &MockClient{
		GetFunc: func(path string, result interface{}) error {
			if path == "/api/users" {
				res := result.(*struct {
					Data []map[string]interface{} `json:"data"`
				})
				res.Data = []map[string]interface{}{
					{"id": float64(1), "username": "admin", "is_active": true},
					{"id": float64(2), "username": "testuser", "is_active": false},
				}
			}
			return nil
		},
	}
	
	// Inject mock
	pkg.DefaultClient = mock
	userFormat = "json" // Force output format to json for easy assert

	cmd := UserCmd()
	cmd.SetArgs([]string{"list", "--format=json"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Errorf("expected 1 API call, got %d", len(mock.Calls))
	}
	if mock.Calls[0].Path != "/api/users" {
		t.Errorf("expected path /api/users, got %s", mock.Calls[0].Path)
	}

	var out []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if len(out) != 2 {
		t.Errorf("expected 2 users, got %d", len(out))
	}
	if out[0]["username"] != "admin" {
		t.Errorf("expected admin, got %v", out[0]["username"])
	}
}

func TestUserCreate(t *testing.T) {
	mock := &MockClient{
		PostFunc: func(path string, body interface{}, result interface{}) error {
			if path == "/api/users" {
				res := result.(*map[string]interface{})
				*res = map[string]interface{}{
					"id": float64(3),
					"username": "newuser",
				}
			}
			return nil
		},
	}
	
	pkg.DefaultClient = mock
	userFormat = "json"

	cmd := UserCmd()
	cmd.SetArgs([]string{"create", "newuser", "--email=test@test.com"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 API call, got %d", len(mock.Calls))
	}
	
	reqBody := mock.Calls[0].Body.(map[string]interface{})
	if reqBody["username"] != "newuser" {
		t.Errorf("expected username 'newuser', got %v", reqBody["username"])
	}
	if reqBody["email"] != "test@test.com" {
		t.Errorf("expected email 'test@test.com', got %v", reqBody["email"])
	}
}
