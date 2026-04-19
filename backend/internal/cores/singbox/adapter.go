package singbox

import (
	"context"
	"fmt"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/stats"
)

func init() {
	cores.RegisterCore("singbox", func() cores.CoreAdapter { return &Adapter{} })
}

type Adapter struct {
	APIKey       string
	V2RayAPIAddr string
}

func NewAdapter() *Adapter { return &Adapter{} }

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
	return "http://127.0.0.1:9090/version"
}

func (a *Adapter) GetDefaultLogPaths() (string, string) {
	return "/var/log/supervisor/singbox.log", "/var/log/supervisor/singbox.log"
}

func (a *Adapter) CreateStatsClient(endpoint string) (cores.StatsClient, error) {
	client := NewStatsClient(a.V2RayAPIAddr, endpoint, a.APIKey)
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
		"vless-reality", "trojan-reality", "tuic", "hysteria2",
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
	client := NewStatsClient(config.GRPCAddress, config.ClashBaseURL, config.APIKey)
	return &singboxStatsClientWrapper{client: client}
}
