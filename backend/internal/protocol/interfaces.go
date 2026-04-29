package protocol

// Lister is for listing and filtering protocols.
type Lister interface {
	List() []ProtocolSummary
	ListByCore(core string) []ProtocolSummary
	ListByCoreAndDirection(core, direction string) []ProtocolSummary
}

// SchemaProvider is for retrieving protocol schemas.
type SchemaProvider interface {
	GetSchema(name string) (*ProtocolSchema, bool)
	SupportsCore(protocol, core string) bool
}

// Validator is for validating protocol configurations.
type Validator interface {
	ValidateConfig(protocol, configJSON string) error
	SupportsCore(protocol, core string) bool
}

// UIGenerator is for UI-related protocol helpers.
type UIGenerator interface {
	DefaultWidget(paramType ParameterType) string
	AutoGenerate(funcName string) (interface{}, error)
}

// Registry combines all consumer interfaces.
type Registry interface {
	Lister
	SchemaProvider
	Validator
	UIGenerator
}
