package models

import (
	"time"
)

type PortAssignment struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	InboundID            uint      `json:"inbound_id" gorm:"uniqueIndex;not null"`
	UserListenPort       int       `json:"user_listen_port" gorm:"index;not null"`
	UserListenAddr       string    `json:"user_listen_addr" gorm:"default:'0.0.0.0'"`
	BackendPort          int       `json:"backend_port" gorm:"not null"`
	CoreType             string    `json:"core_type" gorm:"index;not null"`
	UseHAProxy           bool      `json:"use_haproxy" gorm:"default:true"`
	SNIMatch             string    `json:"sni_match,omitempty"`
	PathMatch            string    `json:"path_match,omitempty"`
	SendProxyProtocol    bool      `json:"send_proxy_protocol" gorm:"default:false"`
	ProxyProtocolVersion int       `json:"proxy_protocol_version,omitempty"`
	IsActive             bool      `json:"is_active" gorm:"default:true"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (PortAssignment) TableName() string {
	return "port_assignments"
}
