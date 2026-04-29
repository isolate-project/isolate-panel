package protocol

// registryAdapter adapts the global registry functions to interface methods.
type registryAdapter struct{}

// NewRegistryAdapter creates a new Registry instance that wraps the global functions.
func NewRegistryAdapter() Registry {
	return &registryAdapter{}
}

// List returns summaries of all registered protocols.
func (r *registryAdapter) List() []ProtocolSummary {
	return GetAllProtocols()
}

// ListByCore returns all protocols that support a given core.
func (r *registryAdapter) ListByCore(core string) []ProtocolSummary {
	return GetProtocolsByCore(core)
}

// ListByCoreAndDirection returns protocols filtered by core and direction.
func (r *registryAdapter) ListByCoreAndDirection(core, direction string) []ProtocolSummary {
	return GetProtocolsByCoreAndDirection(core, direction)
}

// GetSchema returns the schema for a specific protocol.
func (r *registryAdapter) GetSchema(name string) (*ProtocolSchema, bool) {
	return GetProtocolSchema(name)
}

// SupportsCore checks if a protocol supports a given core.
func (r *registryAdapter) SupportsCore(protocol, core string) bool {
	return ValidateProtocolForCore(protocol, core)
}

// ValidateConfig validates a protocol configuration JSON.
func (r *registryAdapter) ValidateConfig(protocol, configJSON string) error {
	return ValidateConfigJSON(protocol, configJSON)
}

// DefaultWidget returns the default UI widget for a given parameter type.
func (r *registryAdapter) DefaultWidget(paramType ParameterType) string {
	return DefaultWidget(paramType)
}

// AutoGenerate calls the appropriate generator function by name.
func (r *registryAdapter) AutoGenerate(funcName string) (interface{}, error) {
	return AutoGenerate(funcName)
}
