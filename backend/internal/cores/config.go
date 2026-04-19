package cores

import (
	"fmt"
	"os"
	"strconv"
)

// CoreConfig holds configurable ports and paths for proxy cores (defaults preserve existing behaviour).
type CoreConfig struct {
	APIPort       int    `mapstructure:"api_port" json:"api_port" yaml:"api_port"`               // default: 10085
	LogDirectory  string `mapstructure:"log_directory" json:"log_directory" yaml:"log_directory"` // default: /var/log/supervisor
	ClashAPIPort  int    `mapstructure:"clash_api_port" json:"clash_api_port" yaml:"clash_api_port"`     // default: 9090
	MihomoAPIPort int    `mapstructure:"mihomo_api_port" json:"mihomo_api_port" yaml:"mihomo_api_port"` // default: 9091
	V2RayAPIPort  int    `mapstructure:"v2ray_api_port" json:"v2ray_api_port" yaml:"v2ray_api_port"`   // default: 10086
}

// ApplyDefaults fills zero-value fields with production defaults.
func (c *CoreConfig) ApplyDefaults() {
	if c.APIPort == 0 {
		c.APIPort = 10085
	}
	if c.LogDirectory == "" {
		c.LogDirectory = "/var/log/supervisor"
	}
	if c.ClashAPIPort == 0 {
		c.ClashAPIPort = 9090
	}
	if c.MihomoAPIPort == 0 {
		c.MihomoAPIPort = 9091
	}
	if c.V2RayAPIPort == 0 {
		c.V2RayAPIPort = 10086
	}
}

// XrayAPIAddr returns the full Xray gRPC API address string.
func (c *CoreConfig) XrayAPIAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", c.APIPort)
}

// ClashAPIAddr returns the full Sing-box Clash API address string.
func (c *CoreConfig) ClashAPIAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", c.ClashAPIPort)
}

// MihomoAPIAddr returns the full Mihomo external-controller address string.
func (c *CoreConfig) MihomoAPIAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", c.MihomoAPIPort)
}

// V2RayAPIAddr returns the full Sing-box V2Ray API address string.
func (c *CoreConfig) V2RayAPIAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", c.V2RayAPIPort)
}

// XrayAccessLog returns the Xray access log path.
func (c *CoreConfig) XrayAccessLog() string {
	return c.LogDirectory + "/xray_access.log"
}

// XrayErrorLog returns the Xray error log path.
func (c *CoreConfig) XrayErrorLog() string {
	return c.LogDirectory + "/xray_error.log"
}

// SingboxLogPath returns the Sing-box log file path.
func (c *CoreConfig) SingboxLogPath() string {
	return c.LogDirectory + "/singbox.log"
}

// CoreConfigFromEnv builds a CoreConfig from ISOLATE_CORE_* environment
// variables, falling back to defaults for unset values.
func CoreConfigFromEnv() *CoreConfig {
	cfg := &CoreConfig{}
	if v := envInt("ISOLATE_CORE_API_PORT"); v != 0 {
		cfg.APIPort = v
	}
	if v := os.Getenv("ISOLATE_CORE_LOG_DIRECTORY"); v != "" {
		cfg.LogDirectory = v
	}
	if v := envInt("ISOLATE_CORE_CLASH_API_PORT"); v != 0 {
		cfg.ClashAPIPort = v
	}
	if v := envInt("ISOLATE_CORE_MIHOMO_API_PORT"); v != 0 {
		cfg.MihomoAPIPort = v
	}
	if v := envInt("ISOLATE_CORE_V2RAY_API_PORT"); v != 0 {
		cfg.V2RayAPIPort = v
	}
	cfg.ApplyDefaults()
	return cfg
}

// envInt reads an integer from the environment variable named key.
// Returns 0 on missing or invalid value.
func envInt(key string) int {
	s := os.Getenv(key)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
