package models

import "time"

// Setting represents an application setting
type Setting struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Key         string    `json:"key" gorm:"uniqueIndex:idx_settings_key;size:100;not null"`
	Value       string    `json:"value" gorm:"type:text"`
	ValueType   string    `json:"value_type" gorm:"size:20;default:'string'"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName returns the table name for Setting model
func (Setting) TableName() string {
	return "settings"
}
