package mihomo

import (
	"context"
	"fmt"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/stats"
)

func init() {
	cores.RegisterCore("mihomo", func() cores.CoreAdapter { return &Adapter{} })
}

type Adapter struct {
	APIKey string
}

func NewAdapter() *Adapter { return &Adapter{} }

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
	return "http://127.0.0.1:9091/version"
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
		"ss", "ssr", "vmess", "vless", "trojan",
		"tuic", "hysteria2", "snell",
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
