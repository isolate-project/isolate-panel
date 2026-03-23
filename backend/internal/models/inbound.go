package models

import (
	"time"
)

type Inbound struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	Name          string `gorm:"not null" json:"name"`
	Protocol      string `gorm:"not null" json:"protocol"`
	CoreID        uint   `gorm:"not null" json:"core_id"`
	ListenAddress string `gorm:"default:'0.0.0.0'" json:"listen_address"`
	Port          int    `gorm:"not null" json:"port"`
	ConfigJSON    string `gorm:"type:text" json:"config_json"`

	// TLS/REALITY
	TLSEnabled        bool   `gorm:"default:false" json:"tls_enabled"`
	TLSCertID         *uint  `json:"tls_cert_id"`
	RealityEnabled    bool   `gorm:"default:false" json:"reality_enabled"`
	RealityConfigJSON string `gorm:"type:text" json:"reality_config_json"`

	// Status
	IsEnabled bool `gorm:"default:true" json:"is_enabled"`

	// Metadata
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Core *Core `gorm:"foreignKey:CoreID" json:"core,omitempty"`
}

func (Inbound) TableName() string {
	return "inbounds"
}

type Outbound struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"not null" json:"name"`
	Protocol   string    `gorm:"not null" json:"protocol"`
	CoreID     uint      `gorm:"not null" json:"core_id"`
	ConfigJSON string    `gorm:"type:text;not null" json:"config_json"`
	Priority   int       `gorm:"default:0" json:"priority"`
	IsEnabled  bool      `gorm:"default:true" json:"is_enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relations
	Core *Core `gorm:"foreignKey:CoreID" json:"core,omitempty"`
}

func (Outbound) TableName() string {
	return "outbounds"
}
