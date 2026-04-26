package protocol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConfigJSON_EmptyString(t *testing.T) {
	err := ValidateConfigJSON("vless", "")
	assert.NoError(t, err)
}

func TestValidateConfigJSON_InvalidJSON(t *testing.T) {
	err := ValidateConfigJSON("vless", "{invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON")
}

func TestValidateConfigJSON_UnknownProtocol(t *testing.T) {
	err := ValidateConfigJSON("nonexistent_proto_xyz", `{"key": "val"}`)
	assert.NoError(t, err)
}

func TestValidateConfigJSON_UnknownField(t *testing.T) {
	schema := &ProtocolSchema{
		Protocol:   "test_unknown_field",
		Direction:  "inbound",
		Core:       []string{"xray"},
		Parameters: map[string]Parameter{
			"method": {Name: "method", Type: TypeString, Label: "Method"},
		},
	}
	Register(schema)

	err := ValidateConfigJSON("test_unknown_field", `{"future_field": "value", "method": "aes"}`)
	assert.NoError(t, err)

	Unregister("test_unknown_field")
}

func TestValidateConfigJSON_ValidStringField(t *testing.T) {
	schema := &ProtocolSchema{
		Protocol:  "test_valid_str",
		Direction: "inbound",
		Core:      []string{"xray"},
		Parameters: map[string]Parameter{
			"method": {Name: "method", Type: TypeString, Label: "Method"},
		},
	}
	Register(schema)

	err := ValidateConfigJSON("test_valid_str", `{"method": "aes-256-gcm"}`)
	assert.NoError(t, err)

	Unregister("test_valid_str")
}

func TestValidateConfigJSON_InvalidFieldType(t *testing.T) {
	schema := &ProtocolSchema{
		Protocol:  "test_invalid_type",
		Direction: "inbound",
		Core:      []string{"xray"},
		Parameters: map[string]Parameter{
			"method": {Name: "method", Type: TypeString, Label: "Method"},
		},
	}
	Register(schema)

	err := ValidateConfigJSON("test_invalid_type", `{"method": 123}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")

	Unregister("test_invalid_type")
}

func TestValidateConfigJSON_SelectField(t *testing.T) {
	schema := &ProtocolSchema{
		Protocol:  "test_select",
		Direction: "inbound",
		Core:      []string{"xray"},
		Parameters: map[string]Parameter{
			"flow": {Name: "flow", Type: TypeSelect, Label: "Flow", Options: []string{"xtls-rprx-vision", ""}},
		},
	}
	Register(schema)

	err := ValidateConfigJSON("test_select", `{"flow": "xtls-rprx-vision"}`)
	assert.NoError(t, err)

	err = ValidateConfigJSON("test_select", `{"flow": "invalid-value"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of")

	err = ValidateConfigJSON("test_select", `{"flow": ""}`)
	assert.NoError(t, err)

	Unregister("test_select")
}

func TestValidateConfigJSON_BooleanField(t *testing.T) {
	schema := &ProtocolSchema{
		Protocol:  "test_bool",
		Direction: "inbound",
		Core:      []string{"xray"},
		Parameters: map[string]Parameter{
			"tls": {Name: "tls", Type: TypeBoolean, Label: "TLS"},
		},
	}
	Register(schema)

	err := ValidateConfigJSON("test_bool", `{"tls": true}`)
	assert.NoError(t, err)

	err = ValidateConfigJSON("test_bool", `{"tls": "yes"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a boolean")

	Unregister("test_bool")
}

func TestValidateConfigJSON_IntegerField(t *testing.T) {
	minVal := 1
	maxVal := 65535
	schema := &ProtocolSchema{
		Protocol:  "test_int",
		Direction: "inbound",
		Core:      []string{"xray"},
		Parameters: map[string]Parameter{
			"port": {Name: "port", Type: TypeInteger, Label: "Port", Min: &minVal, Max: &maxVal},
		},
	}
	Register(schema)

	err := ValidateConfigJSON("test_int", `{"port": 443}`)
	assert.NoError(t, err)

	err = ValidateConfigJSON("test_int", `{"port": 0}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "below minimum")

	err = ValidateConfigJSON("test_int", `{"port": 99999}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "above maximum")

	err = ValidateConfigJSON("test_int", `{"port": "not-a-number"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a number")

	Unregister("test_int")
}

func TestValidateConfigJSON_RequiredField(t *testing.T) {
	schema := &ProtocolSchema{
		Protocol:  "test_required",
		Direction: "inbound",
		Core:      []string{"xray"},
		Parameters: map[string]Parameter{
			"password": {Name: "password", Type: TypeString, Label: "Password", Required: true},
		},
	}
	Register(schema)

	err := ValidateConfigJSON("test_required", `{}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required field 'password' is missing")

	err = ValidateConfigJSON("test_required", `{"password": "secret"}`)
	assert.NoError(t, err)

	Unregister("test_required")
}

func TestValidateConfigJSON_ArrayField(t *testing.T) {
	schema := &ProtocolSchema{
		Protocol:  "test_array",
		Direction: "inbound",
		Core:      []string{"xray"},
		Parameters: map[string]Parameter{
			"tags": {Name: "tags", Type: TypeArray, Label: "Tags"},
		},
	}
	Register(schema)

	err := ValidateConfigJSON("test_array", `{"tags": ["a", "b"]}`)
	assert.NoError(t, err)

	err = ValidateConfigJSON("test_array", `{"tags": "a,b"}`)
	assert.NoError(t, err)

	err = ValidateConfigJSON("test_array", `{"tags": 123}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be an array")

	Unregister("test_array")
}

func TestValidateConfigJSON_MultipleErrors(t *testing.T) {
	schema := &ProtocolSchema{
		Protocol:  "test_multi_err",
		Direction: "inbound",
		Core:      []string{"xray"},
		Parameters: map[string]Parameter{
			"port":   {Name: "port", Type: TypeInteger, Label: "Port"},
			"method": {Name: "method", Type: TypeString, Label: "Method"},
		},
	}
	Register(schema)

	err := ValidateConfigJSON("test_multi_err", `{"port": "not-int", "method": 123}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a number")
	assert.Contains(t, err.Error(), "must be a string")

	Unregister("test_multi_err")
}
