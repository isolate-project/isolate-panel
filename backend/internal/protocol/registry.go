package protocol

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
	Min          *int          `json:"min,omitempty"`
	Max          *int          `json:"max,omitempty"`
	DependsOn    []Dependency  `json:"depends_on,omitempty"`
	Group        string        `json:"group,omitempty"` // UI grouping: "basic", "transport", "tls", "advanced"
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
	Protocol    string               `json:"protocol"`
	Label       string               `json:"label"`
	Description string               `json:"description"`
	Core        []string             `json:"core"`
	Direction   string               `json:"direction"` // "inbound", "outbound", "both"
	RequiresTLS bool                 `json:"requires_tls"`
	Parameters  map[string]Parameter `json:"parameters"`
	Transport   []string             `json:"transport,omitempty"`
	Category    string               `json:"category"` // "proxy", "tunnel", "utility"
}

// ProtocolSummary is a lightweight representation for listing protocols
type ProtocolSummary struct {
	Protocol    string   `json:"protocol"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Core        []string `json:"core"`
	Direction   string   `json:"direction"`
	RequiresTLS bool     `json:"requires_tls"`
	Category    string   `json:"category"`
}

// registry holds all registered protocol schemas
var registry = make(map[string]*ProtocolSchema)

// Register adds a protocol schema to the registry
func Register(schema *ProtocolSchema) {
	registry[schema.Protocol] = schema
}

// GetProtocolSchema returns the schema for a specific protocol
func GetProtocolSchema(name string) (*ProtocolSchema, bool) {
	schema, ok := registry[name]
	return schema, ok
}

// GetAllProtocols returns summaries of all registered protocols
func GetAllProtocols() []ProtocolSummary {
	summaries := make([]ProtocolSummary, 0, len(registry))
	for _, schema := range registry {
		summaries = append(summaries, ProtocolSummary{
			Protocol:    schema.Protocol,
			Label:       schema.Label,
			Description: schema.Description,
			Core:        schema.Core,
			Direction:   schema.Direction,
			RequiresTLS: schema.RequiresTLS,
			Category:    schema.Category,
		})
	}
	return summaries
}

// GetProtocolsByCore returns all protocols that support a given core
func GetProtocolsByCore(coreName string) []ProtocolSummary {
	var result []ProtocolSummary
	for _, schema := range registry {
		for _, c := range schema.Core {
			if c == coreName {
				result = append(result, ProtocolSummary{
					Protocol:    schema.Protocol,
					Label:       schema.Label,
					Description: schema.Description,
					Core:        schema.Core,
					Direction:   schema.Direction,
					RequiresTLS: schema.RequiresTLS,
					Category:    schema.Category,
				})
				break
			}
		}
	}
	return result
}

// GetProtocolsByCoreAndDirection returns protocols filtered by core and direction
func GetProtocolsByCoreAndDirection(coreName, direction string) []ProtocolSummary {
	var result []ProtocolSummary
	for _, schema := range registry {
		if direction != "" && schema.Direction != direction && schema.Direction != "both" {
			continue
		}
		for _, c := range schema.Core {
			if c == coreName {
				result = append(result, ProtocolSummary{
					Protocol:    schema.Protocol,
					Label:       schema.Label,
					Description: schema.Description,
					Core:        schema.Core,
					Direction:   schema.Direction,
					RequiresTLS: schema.RequiresTLS,
					Category:    schema.Category,
				})
				break
			}
		}
	}
	return result
}

// ValidateProtocolForCore checks if a protocol is valid for a given core
func ValidateProtocolForCore(protocolName, coreName string) bool {
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
