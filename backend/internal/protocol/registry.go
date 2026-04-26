package protocol

import "sync"

// ParameterType defines the type of a protocol parameter
type ParameterType string

const (
	TypeString  ParameterType = "string"
	TypeInteger ParameterType = "integer"
	TypeBoolean ParameterType = "boolean"
	TypeSelect  ParameterType = "select"
	TypeUUID    ParameterType = "uuid"
	TypeArray   ParameterType = "array"
	TypeObject  ParameterType = "object"
)

// Parameter describes a single protocol configuration parameter
type Parameter struct {
	Name         string        `json:"name"`
	Label        string        `json:"label"`
	Type         ParameterType `json:"type"`
	Required     bool          `json:"required"`
	Default      interface{}   `json:"default,omitempty"`
	AutoGenerate bool          `json:"auto_generate"`
	AutoGenFunc  string        `json:"auto_gen_func,omitempty"`
	Options      []string      `json:"options,omitempty"`
	Description  string        `json:"description,omitempty"`
	Example      string        `json:"example,omitempty"`
	Placeholder  string        `json:"placeholder,omitempty"`
	Min          *int          `json:"min,omitempty"`
	Max          *int          `json:"max,omitempty"`
	DependsOn    []Dependency  `json:"depends_on,omitempty"`
	Group        string        `json:"group,omitempty"` // UI grouping: "basic", "transport", "tls", "advanced"
	Widget       string        `json:"widget,omitempty"` // UI widget: "input", "select", "textarea", "checkbox", "slider", "password", "tags"
}

// Dependency defines a conditional dependency between parameters
type Dependency struct {
	Field     string      `json:"field"`
	Value     interface{} `json:"value"`
	Condition string      `json:"condition"` // "equals", "not_equals", "in", "not_in"
}

// TransportConfig describes available transport options for a protocol
type TransportConfig struct {
	Name       string      `json:"name"`
	Label      string      `json:"label"`
	Parameters []Parameter `json:"parameters"`
}

// ProtocolSchema defines the full schema for a protocol
type ProtocolSchema struct {
	Protocol          string               `json:"protocol"`
	Label             string               `json:"label"`
	Description       string               `json:"description"`
	Core              []string             `json:"core"`
	Direction         string               `json:"direction"` // "inbound", "outbound", "both"
	RequiresTLS       bool                 `json:"requires_tls"`
	Parameters        map[string]Parameter `json:"parameters"`
	Transport         []string             `json:"transport,omitempty"`
	Category          string               `json:"category"` // "proxy", "tunnel", "utility"
	Deprecated        bool                 `json:"deprecated,omitempty"`
	DeprecationNotice string               `json:"deprecation_notice,omitempty"`
}

// ProtocolSummary is a lightweight representation for listing protocols
type ProtocolSummary struct {
	Protocol          string   `json:"protocol"`
	Label             string   `json:"label"`
	Description       string   `json:"description"`
	Core              []string `json:"core"`
	Direction         string   `json:"direction"`
	RequiresTLS       bool     `json:"requires_tls"`
	Category          string   `json:"category"`
	Deprecated        bool     `json:"deprecated,omitempty"`
	DeprecationNotice string   `json:"deprecation_notice,omitempty"`
}

// registry holds all registered protocol schemas
var (
	registry   = make(map[string]*ProtocolSchema)
	registryMu sync.RWMutex
)

// Register adds a protocol schema to the registry
func Register(schema *ProtocolSchema) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[schema.Protocol] = schema
}

// Get returns the schema for a specific protocol
func Get(protocol string) (*ProtocolSchema, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	schema, ok := registry[protocol]
	return schema, ok
}

// Unregister removes a protocol schema from the registry
func Unregister(protocol string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(registry, protocol)
}

// ListAll returns all registered protocol schemas
func ListAll() []*ProtocolSchema {
	registryMu.RLock()
	defer registryMu.RUnlock()
	schemas := make([]*ProtocolSchema, 0, len(registry))
	for _, schema := range registry {
		schemas = append(schemas, schema)
	}
	return schemas
}

// GetProtocolSchema returns the schema for a specific protocol (deprecated, use Get)
func GetProtocolSchema(name string) (*ProtocolSchema, bool) {
	return Get(name)
}

// GetAllProtocols returns summaries of all registered protocols
func GetAllProtocols() []ProtocolSummary {
	registryMu.RLock()
	defer registryMu.RUnlock()
	summaries := make([]ProtocolSummary, 0, len(registry))
	for _, schema := range registry {
		summaries = append(summaries, ProtocolSummary{
			Protocol:          schema.Protocol,
			Label:             schema.Label,
			Description:       schema.Description,
			Core:              schema.Core,
			Direction:         schema.Direction,
			RequiresTLS:       schema.RequiresTLS,
			Category:          schema.Category,
			Deprecated:        schema.Deprecated,
			DeprecationNotice: schema.DeprecationNotice,
		})
	}
	return summaries
}

// GetProtocolsByCore returns all protocols that support a given core
func GetProtocolsByCore(coreName string) []ProtocolSummary {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var result []ProtocolSummary
	for _, schema := range registry {
		for _, c := range schema.Core {
			if c == coreName {
				result = append(result, ProtocolSummary{
					Protocol:          schema.Protocol,
					Label:             schema.Label,
					Description:       schema.Description,
					Core:              schema.Core,
					Direction:         schema.Direction,
					RequiresTLS:       schema.RequiresTLS,
					Category:          schema.Category,
					Deprecated:        schema.Deprecated,
					DeprecationNotice: schema.DeprecationNotice,
				})
				break
			}
		}
	}
	return result
}

// GetProtocolsByCoreAndDirection returns protocols filtered by core and direction
func GetProtocolsByCoreAndDirection(coreName, direction string) []ProtocolSummary {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var result []ProtocolSummary
	for _, schema := range registry {
		if direction != "" && schema.Direction != direction && schema.Direction != "both" {
			continue
		}
		for _, c := range schema.Core {
			if c == coreName {
				result = append(result, ProtocolSummary{
					Protocol:          schema.Protocol,
					Label:             schema.Label,
					Description:       schema.Description,
					Core:              schema.Core,
					Direction:         schema.Direction,
					RequiresTLS:       schema.RequiresTLS,
					Category:          schema.Category,
					Deprecated:        schema.Deprecated,
					DeprecationNotice: schema.DeprecationNotice,
				})
				break
			}
		}
	}
	return result
}

// ValidateProtocolForCore checks if a protocol is valid for a given core
func ValidateProtocolForCore(protocolName, coreName string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	schema, ok := registry[protocolName]
	if !ok {
		return false
	}
	for _, c := range schema.Core {
		if c == coreName {
			return true
		}
	}
	return false
}

// intPtr is a helper to create *int values for Min/Max
func intPtr(v int) *int {
	return &v
}

// DefaultWidget returns the default UI widget for a given parameter type
func DefaultWidget(t ParameterType) string {
	switch t {
	case TypeString:
		return "input"
	case TypeInteger:
		return "number"
	case TypeBoolean:
		return "checkbox"
	case TypeSelect:
		return "select"
	case TypeUUID:
		return "input"
	case TypeArray:
		return "tags"
	case TypeObject:
		return "textarea"
	default:
		return "input"
	}
}
