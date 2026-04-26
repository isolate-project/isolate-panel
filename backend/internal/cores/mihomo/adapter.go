package mihomo

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/stats"
)

func init() {
	cores.RegisterCore("mihomo", func() cores.CoreAdapter { return &Adapter{} })
}

type Adapter struct {
	APIKey  string
	coreCfg *cores.CoreConfig
}

func NewAdapter() *Adapter { return &Adapter{} }

func (a *Adapter) SetCoreConfig(cfg *cores.CoreConfig) {
	a.coreCfg = cfg
}

func (a *Adapter) ConfigFilename() string { return "config.yaml" }

func (a *Adapter) GenerateConfig(ctx *cores.ConfigContext, coreID uint) (any, error) {
	return GenerateConfig(ctx, coreID)
}

func (a *Adapter) ValidateConfig(config any) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for mihomo: expected *mihomo.Config")
	}
	return ValidateConfig(cfg)
}

func (a *Adapter) WriteConfig(config any, path string) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for mihomo: expected *mihomo.Config")
	}
	return WriteConfig(cfg, path)
}

func (a *Adapter) GetHealthCheckEndpoint() string {
	if a.coreCfg != nil {
		return "http://" + a.coreCfg.MihomoAPIAddr() + "/version"
	}
	return "http://127.0.0.1:9091/version"
}

func (a *Adapter) CheckHealth(ctx context.Context, timeout time.Duration) error {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	addr := "127.0.0.1:9091"
	if a.coreCfg != nil {
		addr = a.coreCfg.MihomoAPIAddr()
	}
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, "http://"+addr+"/version", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("mihomo API not reachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mihomo API returned status %d", resp.StatusCode)
	}
	return nil
}

func (a *Adapter) GetDefaultLogPaths() (string, string) {
	return "/var/log/supervisor/mihomo_access.log", "/var/log/supervisor/mihomo_error.log"
}

func (a *Adapter) CreateStatsClient(endpoint string) (cores.StatsClient, error) {
	client := NewStatsClient(endpoint, a.APIKey)
	return &mihomoStatsClientWrapper{client: client}, nil
}

type mihomoStatsClientWrapper struct {
	client *StatsClient
}

func (w *mihomoStatsClientWrapper) GetTrafficStats(ctx context.Context) ([]stats.TrafficSample, error) {
	return w.client.GetTrafficStats(ctx, 0)
}

func (w *mihomoStatsClientWrapper) GetActiveConnections(ctx context.Context) ([]stats.ConnectionInfo, error) {
	return w.client.GetActiveConnections(ctx, 0)
}

func (w *mihomoStatsClientWrapper) Close() error {
	return w.client.Close()
}

func (w *mihomoStatsClientWrapper) CloseConnection(ctx context.Context, connectionID string) error {
	return w.client.CloseConnection(ctx, 0, connectionID)
}

func (a *Adapter) SupportedProtocols() []string {
	return []string{
		"shadowsocks", "shadowsocksr", "vmess", "vless", "trojan",
		"tuic_v4", "tuic_v5", "hysteria2", "hysteria", "snell", "mieru", "sudoku", "trusttunnel", "redirect", "mixed",
	}
}

func (a *Adapter) DisplayName() string {
	return "Mihomo"
}

func (a *Adapter) WriteConfigToDir(config any, configDir string, coreName string) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for mihomo: expected *mihomo.Config")
	}
	path := configDir + "/" + coreName + "/" + a.ConfigFilename()
	return WriteConfig(cfg, path)
}

func (a *Adapter) NewStatsClient(config cores.StatsClientConfig) cores.StatsClient {
	client := NewStatsClient(config.ClashBaseURL, config.APIKey)
	return &mihomoStatsClientWrapper{client: client}
}

func (a *Adapter) HotReloadInfo() (cores.HotReloadMethod, string, string) {
	addr := "127.0.0.1:9091"
	if a.coreCfg != nil {
		addr = a.coreCfg.MihomoAPIAddr()
	}
	return cores.HotReloadAPI, "", fmt.Sprintf("http://%s/configs?force=true", addr)
}

func (a *Adapter) SupportsHotReload() bool {
	return true
}

func (a *Adapter) ReloadConfig(ctx context.Context) error {
	if a.APIKey == "" {
		return fmt.Errorf("mihomo API key is not configured — cannot reload config securely")
	}
	addr := "127.0.0.1:9091"
	if a.coreCfg != nil {
		addr = a.coreCfg.MihomoAPIAddr()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		fmt.Sprintf("http://%s/configs", addr),
		strings.NewReader(`{"path": ""}`))
	if err != nil {
		return fmt.Errorf("failed to create reload request: %w", err)
	}
	if a.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.APIKey)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("mihomo reload request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("mihomo reload returned status %d", resp.StatusCode)
	}
	return nil
}
