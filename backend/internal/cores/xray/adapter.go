package xray

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/isolate-project/isolate-panel/internal/cores"
	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/stats"
)

func init() {
	cores.RegisterCore("xray", func() cores.CoreAdapter { return &Adapter{} })
}

type Adapter struct {
	coreCfg *cores.CoreConfig
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
		return fmt.Errorf("invalid config type for xray: expected *xray.Config")
	}
	return ValidateConfig(cfg)
}

func (a *Adapter) WriteConfig(config any, path string) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for xray: expected *xray.Config")
	}
	return WriteConfig(cfg, path)
}

func (a *Adapter) GetHealthCheckEndpoint() string {
	if a.coreCfg != nil {
		return "tcp://" + a.coreCfg.XrayAPIAddr()
	}
	return "tcp://127.0.0.1:10085"
}

func (a *Adapter) CheckHealth(ctx context.Context, timeout time.Duration) error {
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var d net.Dialer
	addr := "127.0.0.1:10085"
	if a.coreCfg != nil {
		addr = a.coreCfg.XrayAPIAddr()
	}
	conn, err := d.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("xray gRPC API not reachable: %w", err)
	}
	conn.Close()
	return nil
}

func (a *Adapter) GetDefaultLogPaths() (string, string) {
	return "/var/log/supervisor/xray_access.log", "/var/log/supervisor/xray_error.log"
}

func (a *Adapter) CreateStatsClient(endpoint string) (cores.StatsClient, error) {
	client, err := NewStatsClient(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create xray stats client: %w", err)
	}
	return &xrayStatsClientWrapper{client: client}, nil
}

type xrayStatsClientWrapper struct {
	client *StatsClient
}

func (w *xrayStatsClientWrapper) GetTrafficStats(ctx context.Context) ([]stats.TrafficSample, error) {
	return w.client.GetTrafficStats(ctx, 0)
}

func (w *xrayStatsClientWrapper) GetActiveConnections(ctx context.Context) ([]stats.ConnectionInfo, error) {
	return w.client.GetActiveConnections(ctx, 0)
}

func (w *xrayStatsClientWrapper) Close() error {
	return w.client.Close()
}

func (w *xrayStatsClientWrapper) CloseConnection(ctx context.Context, connectionID string) error {
	return fmt.Errorf("Xray doesn't support closing individual connections via gRPC")
}

func (a *Adapter) SupportedProtocols() []string {
	return []string{
		"vmess", "vless", "trojan", "shadowsocks",
		"hysteria2", "http", "socks5", "xhttp", "tun",
	}
}

func (a *Adapter) DisplayName() string {
	return "Xray"
}

func (a *Adapter) WriteConfigToDir(config any, configDir string, coreName string) error {
	cfg, ok := config.(*Config)
	if !ok {
		return fmt.Errorf("invalid config type for xray: expected *xray.Config")
	}
	path := configDir + "/" + coreName + "/" + a.ConfigFilename()
	return WriteConfig(cfg, path)
}

func (a *Adapter) NewStatsClient(config cores.StatsClientConfig) cores.StatsClient {
	client, err := NewStatsClient(config.GRPCAddress)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("Xray gRPC initial connect failed (will retry)")
	}
	return &xrayStatsClientWrapper{client: client}
}

func (a *Adapter) HotReloadInfo() (cores.HotReloadMethod, string, string) {
	return cores.HotReloadNone, "", ""
}

	func (a *Adapter) SupportsHotReload() bool {
	return false
}

func (a *Adapter) ReloadConfig(ctx context.Context) error {
	return fmt.Errorf("xray hot-reload via gRPC not yet implemented")
}
