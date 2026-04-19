package haproxy

import (
	"time"
)

type GlobalConfig struct {
	LogTarget   string
	MaxConn     int
	StatsSocket string
}

type DefaultsConfig struct {
	Mode           string
	TimeoutConnect time.Duration
	TimeoutClient  time.Duration
	TimeoutServer  time.Duration
}

type RouteRule struct {
	Name        string
	Type        string
	Match       string
	BackendName string
	BackendPort int
	Priority    int
}

type FrontendConfig struct {
	Name               string
	BindAddress        string
	BindPort           int
	Mode               string
	TLSEnabled         bool
	TLSCertPath        string
	SNIInspectionDelay time.Duration
	Routes             []RouteRule
	DefaultBackend     string
}

type BackendServer struct {
	Name          string
	Address       string
	Port          int
	SendProxyV2   bool
	CheckInterval time.Duration
}

type BackendConfig struct {
	Name             string
	Mode             string
	Servers          []BackendServer
	UseXForwardedFor bool
	TimeoutTunnel    time.Duration
}

type HAProxyConfiguration struct {
	Global    GlobalConfig
	Defaults  DefaultsConfig
	Frontends []FrontendConfig
	Backends  []BackendConfig
}

// TemplateData represents the data structure expected by the HAProxy template
type TemplateData struct {
	PortGroups    map[int]PortGroupData
	Backends      []BackendData
	StatsPassword string
}

// PortGroupData represents a group of backends sharing the same user listen port
type PortGroupData struct {
	Port     int
	Mode     string
	HasTLS   bool
	HasSNI   bool
	HasPath  bool
	Backends []BackendData
}

// BackendData represents a single backend configuration for the template
type BackendData struct {
	Name              string
	BackendName       string
	BackendPort       int
	CoreType          string
	Mode              string
	SNIMatch          string
	PathMatch         string
	ServerName        string
	SendProxyProtocol bool
	UseXForwardedFor  bool
}
