package haproxy

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

type Manager struct {
	configPath   string
	socketPath   string
	templatePath string
	generator    *Generator
	db           *gorm.DB
}

func NewManager(configPath, socketPath, templatePath string, statsPassword string, db *gorm.DB) (*Manager, error) {
	generator, err := NewGenerator(templatePath, statsPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create generator: %w", err)
	}

	return &Manager{
		configPath:   configPath,
		socketPath:   socketPath,
		templatePath: templatePath,
		generator:    generator,
		db:           db,
	}, nil
}

func (m *Manager) GenerateConfig() (string, error) {
	var assignments []models.PortAssignment
	if err := m.db.Where("is_active = ?", true).Find(&assignments).Error; err != nil {
		return "", fmt.Errorf("failed to fetch assignments: %w", err)
	}

	return m.generator.Generate(assignments)
}

func (m *Manager) ValidateConfig(cfg string) error {
	tmpFile, err := os.CreateTemp("", "haproxy-config-*.cfg")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	tmpFile.Close()

	cmd := exec.Command("haproxy", "-c", "-f", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("config validation failed: %s", string(output))
	}

	return nil
}

func (m *Manager) WriteConfig(cfg string) error {
	if err := os.WriteFile(m.configPath, []byte(cfg), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func (m *Manager) Reload() error {
	if _, err := os.Stat(m.socketPath); os.IsNotExist(err) {
		return fmt.Errorf("haproxy socket not found: %s", m.socketPath)
	}

	conn, err := net.Dial("unix", m.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to haproxy socket: %w", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("reload\n")); err != nil {
		return fmt.Errorf("failed to send reload command: %w", err)
	}

	time.Sleep(100 * time.Millisecond)
	return nil
}

func (m *Manager) Apply() error {
	cfg, err := m.GenerateConfig()
	if err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	if err := m.ValidateConfig(cfg); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	if err := m.WriteConfig(cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := m.Reload(); err != nil {
		return fmt.Errorf("failed to reload haproxy: %w", err)
	}

	return nil
}
