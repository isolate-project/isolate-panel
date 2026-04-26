package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidateConfigJSON validates a ConfigJSON string against the protocol schema
// for the given protocol name. Unknown fields are allowed (forward compatibility).
// Returns nil if the config is valid or if no schema is found.
func ValidateConfigJSON(protocol, configJSON string) error {
	if configJSON == "" {
		return nil
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	schema, ok := Get(protocol)
	if !ok {
		return nil
	}

	return validateAgainstSchema(cfg, schema)
}

func validateAgainstSchema(cfg map[string]interface{}, schema *ProtocolSchema) error {
	var errors []string

	for key, value := range cfg {
		param, known := schema.Parameters[key]
		if !known {
			continue
		}

		if err := validateParameterValue(key, value, param); err != nil {
			errors = append(errors, err.Error())
		}
	}

	for name, param := range schema.Parameters {
		if param.Required {
			if _, exists := cfg[name]; !exists {
				errors = append(errors, fmt.Sprintf("required field '%s' is missing", name))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("config validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

func validateParameterValue(name string, value interface{}, param Parameter) error {
	switch param.Type {
	case TypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string, got %T", name, value)
		}
		if len(param.Options) > 0 {
			strVal := value.(string)
			if strVal == "" {
				return nil
			}
			valid := false
			for _, opt := range param.Options {
				if strVal == opt {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("field '%s' value '%s' must be one of: %s", name, strVal, strings.Join(param.Options, ", "))
			}
		}
	case TypeInteger:
		switch v := value.(type) {
		case float64:
			if param.Min != nil && int(v) < *param.Min {
				return fmt.Errorf("field '%s' value %v is below minimum %d", name, v, *param.Min)
			}
			if param.Max != nil && int(v) > *param.Max {
				return fmt.Errorf("field '%s' value %v is above maximum %d", name, v, *param.Max)
			}
		default:
			return fmt.Errorf("field '%s' must be a number, got %T", name, value)
		}
	case TypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field '%s' must be a boolean, got %T", name, value)
		}
	case TypeSelect:
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("field '%s' must be a string, got %T", name, value)
		}
		if strVal == "" {
			return nil
		}
		if len(param.Options) > 0 {
			valid := false
			for _, opt := range param.Options {
				if strVal == opt {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("field '%s' value '%s' must be one of: %s", name, strVal, strings.Join(param.Options, ", "))
			}
		}
	case TypeArray:
		switch value.(type) {
		case []interface{}:
		case string:
		default:
			return fmt.Errorf("field '%s' must be an array, got %T", name, value)
		}
	case TypeObject:
		switch value.(type) {
		case map[string]interface{}:
		default:
			return fmt.Errorf("field '%s' must be an object, got %T", name, value)
		}
	case TypeUUID:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field '%s' must be a string, got %T", name, value)
		}
	}

	return nil
}
