package dto

import (
	"time"
)

// CreateInboundRequest represents the request body for creating a new inbound
type CreateInboundRequest struct {
	Name              string `json:"name" validate:"required,min=1,max=255"`
	Protocol          string `json:"protocol" validate:"required,min=1,max=50"`
	CoreID            uint   `json:"core_id" validate:"required"`
	ListenAddress     string `json:"listen_address" validate:"omitempty,ip"`
	Port              int    `json:"port" validate:"required,min=1,max=65535"`
	ConfigJSON        string `json:"config_json" validate:"required"`
	TLSEnabled        bool   `json:"tls_enabled"`
	TLSCertID         *uint  `json:"tls_cert_id,omitempty"`
	RealityEnabled    bool   `json:"reality_enabled"`
	RealityConfigJSON string `json:"reality_config_json,omitempty"`
	IsEnabled         bool   `json:"is_enabled"`
}

// UpdateInboundRequest represents the request body for updating an inbound
type UpdateInboundRequest struct {
	Name              *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Protocol          *string `json:"protocol,omitempty" validate:"omitempty,min=1,max=50"`
	ListenAddress     *string `json:"listen_address,omitempty" validate:"omitempty,ip"`
	Port              *int    `json:"port,omitempty" validate:"omitempty,min=1,max=65535"`
	ConfigJSON        *string `json:"config_json,omitempty"`
	TLSEnabled        *bool   `json:"tls_enabled,omitempty"`
	TLSCertID         *uint   `json:"tls_cert_id,omitempty"`
	RealityEnabled    *bool   `json:"reality_enabled,omitempty"`
	RealityConfigJSON *string `json:"reality_config_json,omitempty"`
	IsEnabled         *bool   `json:"is_enabled,omitempty"`
}

// InboundResponse represents the response for an inbound
type InboundResponse struct {
	ID                uint      `json:"id"`
	Name              string    `json:"name"`
	Protocol          string    `json:"protocol"`
	CoreID            uint      `json:"core_id"`
	ListenAddress     string    `json:"listen_address"`
	Port              int       `json:"port"`
	ConfigJSON        string    `json:"config_json,omitempty"`
	TLSEnabled        bool      `json:"tls_enabled"`
	TLSCertID         *uint     `json:"tls_cert_id,omitempty"`
	RealityEnabled    bool      `json:"reality_enabled"`
	RealityConfigJSON string    `json:"reality_config_json,omitempty"`
	IsEnabled         bool      `json:"is_enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ToMap converts UpdateInboundRequest to a map for partial updates
func (r *UpdateInboundRequest) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	if r.Name != nil {
		m["name"] = *r.Name
	}
	if r.Protocol != nil {
		m["protocol"] = *r.Protocol
	}
	if r.ListenAddress != nil {
		m["listen_address"] = *r.ListenAddress
	}
	if r.Port != nil {
		m["port"] = *r.Port
	}
	if r.ConfigJSON != nil {
		m["config_json"] = *r.ConfigJSON
	}
	if r.TLSEnabled != nil {
		m["tls_enabled"] = *r.TLSEnabled
	}
	if r.TLSCertID != nil {
		m["tls_cert_id"] = *r.TLSCertID
	}
	if r.RealityEnabled != nil {
		m["reality_enabled"] = *r.RealityEnabled
	}
	if r.RealityConfigJSON != nil {
		m["reality_config_json"] = *r.RealityConfigJSON
	}
	if r.IsEnabled != nil {
		m["is_enabled"] = *r.IsEnabled
	}
	return m
}
