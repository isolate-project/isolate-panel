package services

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/vovk4morkovk4/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// GeoService manages GeoIP/GeoSite databases and rules
type GeoService struct {
	db     *gorm.DB
	geoDir string
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

// downloadFile downloads a file from URL
func (s *GeoService) downloadFile(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s", resp.Status)
	}

	path := filepath.Join(s.geoDir, filename)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// UpdateAllDatabases downloads all Geo databases
func (s *GeoService) UpdateAllDatabases() error {
	if err := s.DownloadGeoIP(); err != nil {
		return fmt.Errorf("failed to download GeoIP: %w", err)
	}

	if err := s.DownloadGeoSite(); err != nil {
		return fmt.Errorf("failed to download GeoSite: %w", err)
	}

	if err := s.DownloadCountryMMDB(); err != nil {
		return fmt.Errorf("failed to download Country MMDB: %w", err)
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
