package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// GeoService manages GeoIP/GeoSite databases and rules
type GeoService struct {
	db        *gorm.DB
	geoDir    string
	stopCh    chan struct{}
	stopOnce  sync.Once
	startOnce sync.Once
}

// GeoDatabase represents a GeoIP/GeoSite database
type GeoDatabase struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"` // "geoip" or "geosite"
	LastUpdate time.Time `json:"last_update"`
	Size       int64     `json:"size"`
}

// GeoCountry represents a country in GeoIP database
type GeoCountry struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// GeoCategory represents a category in GeoSite database
type GeoCategory struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Domains     []string `json:"domains,omitempty"`
}

// NewGeoService creates a new Geo service
func NewGeoService(db *gorm.DB, geoDir string) *GeoService {
	return &GeoService{
		db:     db,
		geoDir: geoDir,
	}
}

// Initialize creates the Geo directory if it doesn't exist
func (s *GeoService) Initialize() error {
	return os.MkdirAll(s.geoDir, 0755)
}

// GetGeoDatabases returns list of available Geo databases
func (s *GeoService) GetGeoDatabases() ([]GeoDatabase, error) {
	databases := make([]GeoDatabase, 0)

	// Check for GeoIP files
	files := []string{
		"geoip.dat",
		"geoip.db",
		"geosite.dat",
		"geosite.db",
		"Country.mmdb",
	}

	for _, file := range files {
		path := filepath.Join(s.geoDir, file)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		dbType := "geoip"
		if file[:7] == "geosite" {
			dbType = "geosite"
		}

		databases = append(databases, GeoDatabase{
			Name:       file,
			Type:       dbType,
			LastUpdate: info.ModTime(),
			Size:       info.Size(),
		})
	}

	return databases, nil
}

// DownloadGeoIP downloads GeoIP database
func (s *GeoService) DownloadGeoIP() error {
	// Use Loyalsoldier/v2ray-rules-dat for GeoIP
	url := "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat"
	return s.downloadFile(url, "geoip.dat")
}

// DownloadGeoSite downloads GeoSite database
func (s *GeoService) DownloadGeoSite() error {
	// Use Loyalsoldier/v2ray-rules-dat for GeoSite
	url := "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat"
	return s.downloadFile(url, "geosite.dat")
}

// DownloadCountryMMDB downloads MaxMind Country database
func (s *GeoService) DownloadCountryMMDB() error {
	// Use MaxMind GeoLite2 Country database (free)
	url := "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-Country.mmdb"
	return s.downloadFile(url, "Country.mmdb")
}

// isPrivateURL validates that a URL is not pointing to private/internal addresses
func isPrivateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if ip == nil {
		// It's a hostname, not an IP - resolve it
		ips, err := net.LookupIP(host)
		if err != nil {
			return nil // Can't resolve, let it fail naturally
		}
		for _, resolvedIP := range ips {
			if isPrivateIP(resolvedIP) {
				return fmt.Errorf("hostname resolves to private IP: %s", resolvedIP)
			}
		}
		return nil
	}
	if isPrivateIP(ip) {
		return fmt.Errorf("private IP not allowed: %s", ip)
	}
	return nil
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		{mustParseCIDR("169.254.0.0/16")},
		{mustParseCIDR("127.0.0.0/8")},
		{mustParseCIDR("::1/128")},
		{mustParseCIDR("fc00::/7")},
	}
	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}
	return false
}

// mustParseCIDR parses a CIDR string and panics on error
func mustParseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return network
}

// downloadFile downloads a file from URL with atomic write (tmp → rename)
func (s *GeoService) downloadFile(url, filename string) error {
	if err := isPrivateURL(url); err != nil {
		return fmt.Errorf("URL validation failed: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	//nolint:gosec // G107: url is securely constructed from reliable config and validated
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil // already up-to-date
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s", resp.Status)
	}

	// Atomic write: download to tmp, then rename
	tmpPath := filepath.Join(s.geoDir, filename+".tmp")
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close file %s: %w", tmpPath, err)
	}

	// Rename to final path
	finalPath := filepath.Join(s.geoDir, filename)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}

// GetCountries returns list of countries for GeoIP rules
func (s *GeoService) GetCountries() ([]GeoCountry, error) {
	// Common countries for Lite mode
	return []GeoCountry{
		{Code: "US", Name: "United States"},
		{Code: "CN", Name: "China"},
		{Code: "RU", Name: "Russia"},
		{Code: "DE", Name: "Germany"},
		{Code: "FR", Name: "France"},
		{Code: "GB", Name: "United Kingdom"},
		{Code: "NL", Name: "Netherlands"},
		{Code: "JP", Name: "Japan"},
		{Code: "KR", Name: "South Korea"},
		{Code: "SG", Name: "Singapore"},
	}, nil
}

// GetCategories returns list of categories for GeoSite rules
func (s *GeoService) GetCategories() ([]GeoCategory, error) {
	// Common categories for Lite mode
	return []GeoCategory{
		{
			Name:        "google",
			Description: "Google services (google.com, youtube.com, gmail.com)",
		},
		{
			Name:        "netflix",
			Description: "Netflix streaming service",
		},
		{
			Name:        "telegram",
			Description: "Telegram messenger",
		},
		{
			Name:        "facebook",
			Description: "Facebook and related services",
		},
		{
			Name:        "twitter",
			Description: "Twitter / X",
		},
		{
			Name:        "instagram",
			Description: "Instagram",
		},
		{
			Name:        "linkedin",
			Description: "LinkedIn",
		},
		{
			Name:        "apple",
			Description: "Apple services",
		},
		{
			Name:        "microsoft",
			Description: "Microsoft services",
		},
		{
			Name:        "github",
			Description: "GitHub",
		},
		{
			Name:        "cloudflare",
			Description: "Cloudflare services",
		},
		{
			Name:        "openai",
			Description: "OpenAI services (ChatGPT, API)",
		},
	}, nil
}

// GetGeoRulesForCore returns all enabled Geo rules for a core
func (s *GeoService) GetGeoRulesForCore(coreID uint) ([]models.GeoRule, error) {
	var rules []models.GeoRule
	err := s.db.Where("core_id = ? AND is_enabled = ?", coreID, true).
		Order("priority DESC").
		Find(&rules).Error
	return rules, err
}

// CreateGeoRule creates a new Geo rule
func (s *GeoService) CreateGeoRule(rule *models.GeoRule) error {
	// Check for duplicates
	var existing models.GeoRule
	err := s.db.Where(
		"core_id = ? AND type = ? AND code = ? AND action = ?",
		rule.CoreID, rule.Type, rule.Code, rule.Action,
	).First(&existing).Error

	if err == nil {
		return fmt.Errorf("duplicate rule: %s/%s -> %s already exists", rule.Type, rule.Code, rule.Action)
	}

	return s.db.Create(rule).Error
}

// UpdateGeoRule updates an existing Geo rule
func (s *GeoService) UpdateGeoRule(rule *models.GeoRule) error {
	return s.db.Save(rule).Error
}

// DeleteGeoRule deletes a Geo rule
func (s *GeoService) DeleteGeoRule(id uint, coreID uint) error {
	return s.db.Where("id = ? AND core_id = ?", id, coreID).Delete(&models.GeoRule{}).Error
}

// ToggleGeoRule enables/disables a Geo rule
func (s *GeoService) ToggleGeoRule(id uint, coreID uint, enabled bool) error {
	return s.db.Model(&models.GeoRule{}).
		Where("id = ? AND core_id = ?", id, coreID).
		Update("is_enabled", enabled).Error
}

// DB returns the database instance
func (s *GeoService) DB() *gorm.DB {
	return s.db
}

// downloadFileConditional downloads if modified (using If-Modified-Since)
func (s *GeoService) downloadFileConditional(url, filename string) (bool, error) {
	if err := isPrivateURL(url); err != nil {
		return false, fmt.Errorf("URL validation failed: %w", err)
	}
	path := filepath.Join(s.geoDir, filename)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	// Check existing file for If-Modified-Since
	if info, err := os.Stat(path); err == nil {
		req.Header.Set("If-Modified-Since", info.ModTime().UTC().Format(http.TimeFormat))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return false, nil // already up-to-date
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to download: %s", resp.Status)
	}

	// Atomic: write to tmp then rename
	tmpPath := path + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return false, err
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return false, err
	}
	file.Close()

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return false, err
	}

	return true, nil
}

// UpdateAllDatabases downloads all geo databases if they are outdated
func (s *GeoService) UpdateAllDatabases() error {
	databases := map[string]string{
		"geoip.dat":    "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat",
		"geosite.dat":  "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat",
		"Country.mmdb": "https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-Country.mmdb",
	}

	var lastErr error
	for filename, url := range databases {
		updated, err := s.downloadFileConditional(url, filename)
		if err != nil {
			log.Printf("[GEO] Failed to update %s: %v", filename, err)
			lastErr = err
			continue
		}
		if updated {
			log.Printf("[GEO] Updated %s", filename)
		} else {
			log.Printf("[GEO] %s is up-to-date", filename)
		}
	}

	return lastErr
}

// StartAutoUpdate starts a background goroutine to periodically update geo databases
func (s *GeoService) StartAutoUpdate(interval time.Duration) {
	s.startOnce.Do(func() {
		s.stopCh = make(chan struct{})
		s.stopOnce = sync.Once{}

		// Initial async download if files don't exist
		go func() {
			if _, err := os.Stat(filepath.Join(s.geoDir, "geoip.dat")); os.IsNotExist(err) {
				log.Printf("[GEO] Geo databases not found, downloading...")
				if err := s.UpdateAllDatabases(); err != nil {
					log.Printf("[GEO] Initial download failed: %v", err)
				}
			}
		}()

		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := s.UpdateAllDatabases(); err != nil {
						log.Printf("[GEO] Auto-update failed: %v", err)
					}
				case <-s.stopCh:
					return
				}
			}
		}()
	})
}

// StopAutoUpdate stops the background auto-update goroutine
func (s *GeoService) StopAutoUpdate() {
	s.stopOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
	})
}
