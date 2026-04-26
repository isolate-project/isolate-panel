package singbox

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/stats"
)

func init() {
	cores.RegisterCore("singbox", func() cores.CoreAdapter { return &Adapter{} })
}

type Adapter struct {
	APIKey       string
	V2RayAPIAddr string
	coreCfg      *cores.CoreConfig
}

func NewAdapter() *Adapter { return &Adapter{} }

func (a *Adapter) SetCoreConfig(cfg *cores.CoreConfig) {
	a.coreCfg = cfg
}

func (a *Adapter) ConfigFilename() string { return "config.json" }

func (a *Adapter) GenerateConfig(ctx *cores.ConfigContext, coreID uint) (any, error) {
	return GenerateConfig(ctx, coreID)
}

func (a *Adapter) ValidateConfig(config any) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for singbox: expected *singbox.Config")
	}
	return ValidateConfig(cfg)
}

func (a *Adapter) WriteConfig(config any, path string) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for singbox: expected *singbox.Config")
	}
	return WriteConfig(cfg, path)
}

func (a *Adapter) GetHealthCheckEndpoint() string {
	if a.coreCfg != nil {
		return "http://" + a.coreCfg.ClashAPIAddr() + "/version"
	}
	return "http://127.0.0.1:9090/version"
}

func (a *Adapter) CheckHealth(ctx context.Context, timeout time.Duration) error {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	addr := "127.0.0.1:9090"
	if a.coreCfg != nil {
		addr = a.coreCfg.ClashAPIAddr()
	}
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, "http://"+addr+"/version", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("singbox API not reachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("singbox API returned status %d", resp.StatusCode)
	}
	return nil
}

func (a *Adapter) GetDefaultLogPaths() (string, string) {
	return "/var/log/supervisor/singbox.log", "/var/log/supervisor/singbox.log"
}

func (a *Adapter) CreateStatsClient(endpoint string) (cores.StatsClient, error) {
	client := NewStatsClient(endpoint, a.APIKey)
	return &singboxStatsClientWrapper{client: client}, nil
}

type singboxStatsClientWrapper struct {
	client *StatsClient
}

func (w *singboxStatsClientWrapper) GetTrafficStats(ctx context.Context) ([]stats.TrafficSample, error) {
	return w.client.GetTrafficStats(ctx, 0)
}

func (w *singboxStatsClientWrapper) GetActiveConnections(ctx context.Context) ([]stats.ConnectionInfo, error) {
	return w.client.GetActiveConnections(ctx, 0)
}

func (w *singboxStatsClientWrapper) Close() error {
	return w.client.Close()
}

func (w *singboxStatsClientWrapper) CloseConnection(ctx context.Context, connectionID string) error {
	return w.client.CloseConnection(ctx, 0, connectionID)
}

func (a *Adapter) SupportedProtocols() []string {
	return []string{
		"vmess", "vless", "trojan", "shadowsocks",
		"tuic_v4", "tuic_v5", "hysteria2", "hysteria",
		"anytls", "naive", "mixed", "redirect",
	}
}

func (a *Adapter) DisplayName() string {
	return "Sing-box"
}

func (a *Adapter) WriteConfigToDir(config any, configDir string, coreName string) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for singbox: expected *singbox.Config")
	}
	path := configDir + "/" + coreName + "/" + a.ConfigFilename()
	return WriteConfig(cfg, path)
}

func (a *Adapter) NewStatsClient(config cores.StatsClientConfig) cores.StatsClient {
	client := NewStatsClient(config.ClashBaseURL, config.APIKey)
	return &singboxStatsClientWrapper{client: client}
}

func (a *Adapter) HotReloadInfo() (cores.HotReloadMethod, string, string) {
	return cores.HotReloadSignal, "USR1", ""
}

func (a *Adapter) SupportsHotReload() bool {
	return true
}

	func (a *Adapter) ReloadConfig(ctx context.Context) error {
	return fmt.Errorf("sing-box reload should use SIGUSR1 via supervisor (SignalProcess), not adapter direct reload")
}
