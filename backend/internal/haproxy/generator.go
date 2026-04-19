package haproxy

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/isolate-project/isolate-panel/internal/models"
)

type Generator struct {
	tmpl          *template.Template
	statsPassword string
}

func NewGenerator(templatePath string, statsPassword string) (*Generator, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	return &Generator{tmpl: tmpl, statsPassword: statsPassword}, nil
}

func (g *Generator) Generate(assignments []models.PortAssignment) (string, error) {
	// Group assignments by user listen port
	portGroups := make(map[int][]models.PortAssignment)
	for _, a := range assignments {
		if !a.UseHAProxy {
			continue
		}
		portGroups[a.UserListenPort] = append(portGroups[a.UserListenPort], a)
	}

	// Build port group data for template
	templatePortGroups := make(map[int]PortGroupData)
	for port, group := range portGroups {
		groupData := PortGroupData{
			Port:     port,
			Backends: make([]BackendData, 0, len(group)),
		}

		// Determine group properties
		hasTLS := false
		hasSNI := false
		hasPath := false

		for _, a := range group {
			backendData := BackendData{
				Name:        fmt.Sprintf("inbound_%d", a.InboundID),
				BackendName: fmt.Sprintf("bk_%s_%d", a.CoreType, a.BackendPort),
				BackendPort: a.BackendPort,
				CoreType:    a.CoreType,
				SNIMatch:    a.SNIMatch,
				PathMatch:   a.PathMatch,
				ServerName:  fmt.Sprintf("%s_%d", a.CoreType, a.BackendPort),
			}

			if a.SNIMatch != "" {
				hasSNI = true
				hasTLS = true
			}
			if a.PathMatch != "" {
				hasPath = true
				hasTLS = true
			}

			groupData.Backends = append(groupData.Backends, backendData)
		}

		// Determine mode
		if hasPath {
			groupData.Mode = "http"
		} else {
			groupData.Mode = "tcp"
		}

		groupData.HasTLS = hasTLS
		groupData.HasSNI = hasSNI
		groupData.HasPath = hasPath

		templatePortGroups[port] = groupData
	}

	// Build backend data for template
	templateBackends := make([]BackendData, 0, len(assignments))
	for _, a := range assignments {
		if !a.UseHAProxy {
			continue
		}

		useProxyV2 := a.SendProxyProtocol && a.CoreType == "xray"
		useXFF := a.CoreType == "singbox" || a.CoreType == "mihomo"

		mode := "tcp"
		if a.PathMatch != "" {
			mode = "http"
		}

		backendData := BackendData{
			Name:              fmt.Sprintf("inbound_%d", a.InboundID),
			BackendName:       fmt.Sprintf("bk_%s_%d", a.CoreType, a.BackendPort),
			BackendPort:       a.BackendPort,
			CoreType:          a.CoreType,
			Mode:              mode,
			ServerName:        fmt.Sprintf("%s_%d", a.CoreType, a.BackendPort),
			SendProxyProtocol: useProxyV2,
			UseXForwardedFor:  useXFF,
		}

		templateBackends = append(templateBackends, backendData)
	}

	// Create template data structure
	templateData := TemplateData{
		PortGroups:    templatePortGroups,
		Backends:      templateBackends,
		StatsPassword: g.statsPassword,
	}

	var buf bytes.Buffer
	if err := g.tmpl.Execute(&buf, templateData); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
