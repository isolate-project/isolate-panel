package cores

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/isolate-project/isolate-panel/internal/stats"
)

// HotReloadMethod represents the method used for hot-reloading a core
type HotReloadMethod int

const (
	// HotReloadNone indicates the core does not support hot-reload and requires full restart
	HotReloadNone HotReloadMethod = iota
	// HotReloadSignal indicates the core supports hot-reload via signal (e.g., SIGUSR1)
	HotReloadSignal
	// HotReloadAPI indicates the core supports hot-reload via REST API
	HotReloadAPI
)

// CoreAdapter provides a unified interface for all proxy core operations.
// Each proxy core (xray, singbox, mihomo) implements this adapter,
// allowing callers to treat all cores uniformly without switch statements.
// Adding a new core requires only: 1 adapter file + 1 RegisterCore call.
type CoreAdapter interface {
	ConfigFilename() string
	GenerateConfig(ctx *ConfigContext, coreID uint) (any, error)
	ValidateConfig(config any) error
	WriteConfig(config any, path string) error
	GetHealthCheckEndpoint() string
	// CheckHealth verifies the core process is healthy and responsive
	CheckHealth(ctx context.Context, timeout time.Duration) error
	// SupportsHotReload returns true if the core supports hot-reload without full restart
	SupportsHotReload() bool
	// ReloadConfig triggers a config reload for the core (if supported)
	ReloadConfig(ctx context.Context) error
	GetDefaultLogPaths() (access string, errorLog string)
	CreateStatsClient(endpoint string) (StatsClient, error)
	// SupportedProtocols returns the list of protocols this core supports
	SupportedProtocols() []string
	// DisplayName returns the human-readable core name
	DisplayName() string
	// WriteConfigToDir writes the generated config to disk with configDir and coreName
	WriteConfigToDir(config any, configDir string, coreName string) error
	// NewStatsClient creates a stats client with configuration
	NewStatsClient(config StatsClientConfig) StatsClient
	// HotReloadInfo returns information about how this core supports hot-reload
	// Returns: (method, signal, endpoint)
	//   - method: HotReloadNone, HotReloadSignal, or HotReloadAPI
	//   - signal: signal name for HotReloadSignal (e.g., "USR1")
	//   - endpoint: API endpoint for HotReloadAPI (e.g., "http://127.0.0.1:9091/configs")
	HotReloadInfo() (method HotReloadMethod, signal string, endpoint string)
}

// StatsClient provides a unified interface for collecting runtime statistics
// from any proxy core. The caller already knows which core the client belongs to.
type StatsClient interface {
	GetTrafficStats(ctx context.Context) ([]stats.TrafficSample, error)
	GetActiveConnections(ctx context.Context) ([]stats.ConnectionInfo, error)
	Close() error
	// CloseConnection closes a specific connection by ID
	CloseConnection(ctx context.Context, connectionID string) error
}

// StatsClientConfig holds configuration for creating a stats client
type StatsClientConfig struct {
	GRPCAddress  string
	ClashBaseURL string
	APIKey       string
}

type CoreFactory func() CoreAdapter

var (
	coreFactories = make(map[string]CoreFactory)
	registryOnce  sync.Once
)

// RegisterCore adds a CoreAdapter factory to the registry.
// Called by each core sub-package in its init() function.
func RegisterCore(name string, factory CoreFactory) {
	if _, exists := coreFactories[name]; exists {
		log.Warn().Str("core", name).Msg("Core adapter already registered, overwriting")
	}
	coreFactories[name] = factory
}

// GetCoreAdapter returns the CoreAdapter for the given core name.
func GetCoreAdapter(name string) (CoreAdapter, error) {
	registryOnce.Do(func() {})
	factory, ok := coreFactories[name]
	if !ok {
		return nil, fmt.Errorf("unknown core type: %s", name)
	}
	return factory(), nil
}

// RegisteredCores returns the names of all registered cores.
func RegisteredCores() []string {
	registryOnce.Do(func() {})
	names := make([]string, 0, len(coreFactories))
	for name := range coreFactories {
		names = append(names, name)
	}
	return names
}
