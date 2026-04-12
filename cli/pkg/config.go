package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents the CLI configuration
type Config struct {
	CurrentProfile string             `json:"current_profile"`
	Profiles       map[string]Profile `json:"profiles"`
}

// Profile represents a single panel profile
type Profile struct {
	PanelURL       string `json:"panel_url"`
	Username       string `json:"username,omitempty"`
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	TokenExpiresAt string `json:"token_expires_at"`
}

// IsTokenExpired checks if the access token is expired
func (p *Profile) IsTokenExpired() bool {
	if p.TokenExpiresAt == "" {
		return true
	}

	expiresAt, err := time.Parse(time.RFC3339, p.TokenExpiresAt)
	if err != nil {
		return true
	}

	return time.Now().After(expiresAt)
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	if envPath := os.Getenv("ISOLATE_PANEL_CONFIG"); envPath != "" {
		return envPath
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".isolate-panel.json"
	}
	return filepath.Join(homeDir, ".isolate-panel", "config.json")
}

// LoadConfig loads the configuration from file
func LoadConfig() (*Config, error) {
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config
			return &Config{
				CurrentProfile: "default",
				Profiles:       make(map[string]Profile),
			}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure profiles map is initialized
	if config.Profiles == nil {
		config.Profiles = make(map[string]Profile)
	}

	return &config, nil
}

// SaveConfig saves the configuration to file
func (c *Config) SaveConfig() error {
	path := ConfigPath()

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCurrentProfile returns the current profile
func (c *Config) GetCurrentProfile() (*Profile, error) {
	profile, ok := c.Profiles[c.CurrentProfile]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found", c.CurrentProfile)
	}
	return &profile, nil
}

// SetProfile sets a profile
func (c *Config) SetProfile(name string, profile Profile) {
	c.Profiles[name] = profile
}

// SetCurrentProfile sets the current profile
func (c *Config) SetCurrentProfile(name string) {
	c.CurrentProfile = name
}

// ListProfiles returns a list of profile names
func (c *Config) ListProfiles() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

// DeleteProfile deletes a profile
func (c *Config) DeleteProfile(name string) {
	delete(c.Profiles, name)
	if c.CurrentProfile == name {
		c.CurrentProfile = "default"
	}
}

// RefreshToken updates the tokens for the current profile
func (c *Config) RefreshToken(accessToken, refreshToken string, expiresAt time.Time) error {
	profile, err := c.GetCurrentProfile()
	if err != nil {
		return err
	}

	profile.AccessToken = accessToken
	profile.RefreshToken = refreshToken
	profile.TokenExpiresAt = expiresAt.Format(time.RFC3339)

	c.Profiles[c.CurrentProfile] = *profile
	return c.SaveConfig()
}
