package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App           AppConfig           `mapstructure:"app"`
	Database      DatabaseConfig      `mapstructure:"database"`
	JWT           JWTConfig           `mapstructure:"jwt"`
	Logging       LoggingConfig       `mapstructure:"logging"`
	Security      SecurityConfig      `mapstructure:"security"`
	Cores         CoresConfig         `mapstructure:"cores"`
	Data          DataConfig          `mapstructure:"data"`
	Notifications NotificationsConfig `mapstructure:"notifications"`
	Traffic       TrafficConfig       `mapstructure:"traffic"`
	Subscription  SubscriptionConfig  `mapstructure:"subscription"`
	HAProxy       HAProxyConfig       `mapstructure:"haproxy"`
}

type AppConfig struct {
	Name       string `mapstructure:"name"`
	Env        string `mapstructure:"env"`
	Port       int    `mapstructure:"port"`
	Host       string `mapstructure:"host"`
	AdminEmail string `mapstructure:"admin_email"`
	PanelURL   string `mapstructure:"panel_url"`
	BodyLimit  int    `mapstructure:"body_limit"` // Request body limit in KB (default: 2048 = 2MB)
}

type DatabaseConfig struct {
	Path         string `mapstructure:"path"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type JWTConfig struct {
	Secret          string `mapstructure:"secret"`
	AccessTokenTTL  int    `mapstructure:"access_token_ttl"`  // seconds
	RefreshTokenTTL int    `mapstructure:"refresh_token_ttl"` // seconds
}

type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	FilePath   string `mapstructure:"file_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type SecurityConfig struct {
	Argon2 Argon2Config `mapstructure:"argon2"`
}

type Argon2Config struct {
	Time       uint32 `mapstructure:"time"`
	Memory     uint32 `mapstructure:"memory"`
	Threads    uint8  `mapstructure:"threads"`
	KeyLength  uint32 `mapstructure:"key_length"`
	SaltLength uint32 `mapstructure:"salt_length"`
}

type DataConfig struct {
	DataDir   string `mapstructure:"data_dir"`
	WarpDir   string `mapstructure:"warp_dir"`
	GeoDir    string `mapstructure:"geo_dir"`
	BackupDir string `mapstructure:"backup_dir"`
	CertDir   string `mapstructure:"cert_dir"`
}

type NotificationsConfig struct {
	WebhookURL     string `mapstructure:"webhook_url"`
	WebhookSecret  string `mapstructure:"webhook_secret"`
	TelegramToken  string `mapstructure:"telegram_token"`
	TelegramChatID string `mapstructure:"telegram_chat_id"`
}

type TrafficConfig struct {
	CollectInterval int `mapstructure:"collect_interval"` // seconds
	ConnInterval    int `mapstructure:"conn_interval"`    // seconds
}

type SubscriptionConfig struct {
	Enabled   bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	Host      string `mapstructure:"host" json:"host" yaml:"host"`
	Port      int    `mapstructure:"port" json:"port" yaml:"port"`
	AutoTLS   bool   `mapstructure:"auto_tls" json:"auto_tls" yaml:"auto_tls"`
	AllowHTTP bool   `mapstructure:"allow_http" json:"allow_http" yaml:"allow_http"`
}

type HAProxyConfig struct {
	StatsPassword string `mapstructure:"stats_password"`
}

type CoresConfig struct {
	ConfigDir     string `mapstructure:"config_dir"`
	SupervisorURL string `mapstructure:"supervisor_url"`
	// gRPC/HTTP API addresses for stats collection
	XrayAPIAddr    string `mapstructure:"xray_api_addr"`    // e.g., "127.0.0.1:10085"
	SingboxAPIAddr string `mapstructure:"singbox_api_addr"` // e.g., "127.0.0.1:9090"
	MihomoAPIAddr  string `mapstructure:"mihomo_api_addr"`  // e.g., "127.0.0.1:9091"
	// API keys for Sing-box and Mihomo (Clash-compatible API)
	SingboxAPIKey string `mapstructure:"singbox_api_key"`
	MihomoAPIKey  string `mapstructure:"mihomo_api_key"`
	// Configurable ports and paths (override hardcoded defaults)
	APIPort       int    `mapstructure:"api_port"`        // Xray gRPC API port (default: 10085)
	LogDirectory  string `mapstructure:"log_directory"`   // Core log directory (default: /var/log/supervisor)
	ClashAPIPort  int    `mapstructure:"clash_api_port"`  // Sing-box Clash API port (default: 9090)
	MihomoAPIPort int    `mapstructure:"mihomo_api_port"` // Mihomo API port (default: 9091)
	V2RayAPIPort  int    `mapstructure:"v2ray_api_port"` // Sing-box V2Ray API port (default: 10086)
}

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Environment variables override
	v.SetEnvPrefix("ISOLATE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with specific environment variables (supporting both prefixed and non-prefixed)
	if jwtSecret := v.GetString("JWT_SECRET"); jwtSecret != "" {
		config.JWT.Secret = jwtSecret
	} else if val := os.Getenv("JWT_SECRET"); val != "" {
		config.JWT.Secret = val
	}

	if dbPath := v.GetString("DATABASE_PATH"); dbPath != "" {
		config.Database.Path = dbPath
	} else if val := os.Getenv("DATABASE_PATH"); val != "" {
		config.Database.Path = val
	}

	if port := v.GetInt("PORT"); port != 0 {
		config.App.Port = port
	} else if val := os.Getenv("PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			config.App.Port = p
		}
	}

	if appEnv := v.GetString("APP_ENV"); appEnv != "" {
		config.App.Env = appEnv
	} else if val := os.Getenv("APP_ENV"); val != "" {
		config.App.Env = val
	}

	if logLevel := v.GetString("LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	} else if val := os.Getenv("LOG_LEVEL"); val != "" {
		config.Logging.Level = val
	}

	if panelURL := v.GetString("APP_PANEL_URL"); panelURL != "" {
		config.App.PanelURL = panelURL
	} else if val := os.Getenv("APP_PANEL_URL"); val != "" {
		config.App.PanelURL = val
	}

	if bodyLimit := v.GetInt("APP_BODY_LIMIT"); bodyLimit > 0 {
		config.App.BodyLimit = bodyLimit
	} else if val := os.Getenv("APP_BODY_LIMIT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			config.App.BodyLimit = n
		}
	}

	// Apply default body limit if not set
	if config.App.BodyLimit == 0 {
		config.App.BodyLimit = 2048 // 2MB default
	}

	// Core API keys from environment (without ISOLATE_ prefix for backwards compat)
	if key := v.GetString("CORES_SINGBOX_API_KEY"); key != "" {
		config.Cores.SingboxAPIKey = key
	}
	if key := v.GetString("CORES_MIHOMO_API_KEY"); key != "" {
		config.Cores.MihomoAPIKey = key
	}

	// Core ports/paths from environment (ISOLATE_CORE_* prefix)
	if val := os.Getenv("ISOLATE_CORE_API_PORT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			config.Cores.APIPort = n
		}
	}
	if val := os.Getenv("ISOLATE_CORE_LOG_DIRECTORY"); val != "" {
		config.Cores.LogDirectory = val
	}
	if val := os.Getenv("ISOLATE_CORE_CLASH_API_PORT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			config.Cores.ClashAPIPort = n
		}
	}
	if val := os.Getenv("ISOLATE_CORE_MIHOMO_API_PORT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			config.Cores.MihomoAPIPort = n
		}
	}
	if val := os.Getenv("ISOLATE_CORE_V2RAY_API_PORT"); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			config.Cores.V2RayAPIPort = n
		}
	}

	// HAProxy stats password from environment
	if password := v.GetString("HAPROXY_STATS_PASSWORD"); password != "" {
		config.HAProxy.StatsPassword = password
	} else if val := os.Getenv("HAPROXY_STATS_PASSWORD"); val != "" {
		config.HAProxy.StatsPassword = val
	}

	// Subscription configuration from environment
	if host := v.GetString("SUBSCRIPTION_HOST"); host != "" {
		config.Subscription.Host = host
	} else if val := os.Getenv("ISOLATE_SUBSCRIPTION_HOST"); val != "" {
		config.Subscription.Host = val
	}
	if allowHTTP := v.GetString("SUBSCRIPTION_ALLOW_HTTP"); allowHTTP != "" {
		if b, err := strconv.ParseBool(allowHTTP); err == nil {
			config.Subscription.AllowHTTP = b
		}
	} else if val := os.Getenv("ISOLATE_SUBSCRIPTION_ALLOW_HTTP"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			config.Subscription.AllowHTTP = b
		}
	}

	// Apply defaults for subscription configuration
	if config.Subscription.Host == "" {
		config.Subscription.Host = "127.0.0.1"
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.App.Port <= 0 || c.App.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.App.Port)
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database path is required")
	}
	if c.JWT.Secret == "" || c.JWT.Secret == "change-this-in-production-use-env-var" || c.JWT.Secret == "change-this-in-production-use-a-strong-random-secret" {
		log.Printf("WARNING: JWT secret not properly configured - auto-generation should handle this in Docker")
	}
	if c.JWT.AccessTokenTTL <= 0 {
		return fmt.Errorf("invalid access token TTL: %d", c.JWT.AccessTokenTTL)
	}
	if c.JWT.RefreshTokenTTL <= 0 {
		return fmt.Errorf("invalid refresh token TTL: %d", c.JWT.RefreshTokenTTL)
	}

	// Validate log level
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	return nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}
