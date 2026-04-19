package haproxy

import (
	"os"
	"testing"
	"text/template"
)

func TestTemplateParsing(t *testing.T) {
	tmplPath := "templates/haproxy.cfg.tmpl"
	content, err := os.ReadFile(tmplPath)
	if err != nil {
		t.Fatalf("Failed to read template file: %v", err)
	}

	_, err = template.New("haproxy.cfg").Parse(string(content))
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}
}

func TestTemplateStructure(t *testing.T) {
	tmplPath := "templates/haproxy.cfg.tmpl"
	content, err := os.ReadFile(tmplPath)
	if err != nil {
		t.Fatalf("Failed to read template file: %v", err)
	}

	contentStr := string(content)

	requiredSections := []string{
		"global",
		"defaults",
		"frontend ft_user_",
		"backend bk_",
		"backend bk_default_web",
		"stats socket /run/haproxy/admin.sock",
		"maxconn 4096",
		"mode tcp",
		"timeout connect 5s",
		"timeout client 30s",
		"timeout server 30s",
		"bind :",
		"v4v6 tfo",
		"tcp-request inspect-delay 5s",
		"acl is_",
		"use_backend",
		"server",
	}

	for _, section := range requiredSections {
		if !contains(contentStr, section) {
			t.Errorf("Template missing required section: %s", section)
		}
	}
}

func TestTemplateVariables(t *testing.T) {
	tmplPath := "templates/haproxy.cfg.tmpl"
	content, err := os.ReadFile(tmplPath)
	if err != nil {
		t.Fatalf("Failed to read template file: %v", err)
	}

	contentStr := string(content)

	expectedVars := []string{
		"{{range $port, $group := .PortGroups}}",
		"{{range .Backends}}",
		"{{$port}}",
		"{{$group.Backends | len}}",
		"{{$group.Mode}}",
		"{{.Name}}",
		"{{.BackendName}}",
		"{{$backend.SNIMatch}}",
		"{{$backend.PathMatch}}",
		"{{.CoreType}}",
		"{{.BackendPort}}",
		"{{.Mode}}",
		"{{if .SendProxyProtocol}}",
		"{{if .UseXForwardedFor}}",
		"{{.ServerName}}",
		"{{.StatsPassword}}",
	}

	for _, v := range expectedVars {
		if !contains(contentStr, v) {
			t.Errorf("Template missing expected variable: %s", v)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
