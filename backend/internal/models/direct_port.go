package models

import (
	"time"
)

type DirectPort struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	InboundID     uint      `json:"inbound_id" gorm:"uniqueIndex;not null"`
	ListenPort    int       `json:"listen_port" gorm:"index;not null"`
	ListenAddress string    `json:"listen_address" gorm:"default:'0.0.0.0'"`
	CoreType      string    `json:"core_type" gorm:"index;not null"`
	BackendPort   int       `json:"backend_port" gorm:"not null"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at"`
}

func (DirectPort) TableName() string {
	return "direct_ports"
}
