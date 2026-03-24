package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Security SecurityConfig `mapstructure:"security"`
	Cores    CoresConfig    `mapstructure:"cores"`
}

type AppConfig struct {
	Name       string `mapstructure:"name"`
	Env        string `mapstructure:"env"`
	Port       int    `mapstructure:"port"`
	Host       string `mapstructure:"host"`
	AdminEmail string `mapstructure:"admin_email"`
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

type CoresConfig struct {
	ConfigDir     string `mapstructure:"config_dir"`
	SupervisorURL string `mapstructure:"supervisor_url"`
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

	// Override with specific environment variables
	if jwtSecret := v.GetString("JWT_SECRET"); jwtSecret != "" {
		config.JWT.Secret = jwtSecret
	}
	if dbPath := v.GetString("DATABASE_PATH"); dbPath != "" {
		config.Database.Path = dbPath
	}
	if port := v.GetInt("PORT"); port != 0 {
		config.App.Port = port
	}
	if appEnv := v.GetString("APP_ENV"); appEnv != "" {
		config.App.Env = appEnv
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
	if c.JWT.Secret == "" || c.JWT.Secret == "change-this-in-production-use-env-var" {
		return fmt.Errorf("JWT secret must be set via JWT_SECRET environment variable")
	}
	if c.JWT.AccessTokenTTL <= 0 {
		return fmt.Errorf("invalid access token TTL: %d", c.JWT.AccessTokenTTL)
	}
	if c.JWT.RefreshTokenTTL <= 0 {
		return fmt.Errorf("invalid refresh token TTL: %d", c.JWT.RefreshTokenTTL)
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
