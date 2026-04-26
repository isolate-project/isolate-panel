package api

type CreateInboundDTO struct {
	Name              string `json:"name"`
	Protocol          string `json:"protocol" validate:"required"`
	CoreID            uint   `json:"core_id" validate:"required"`
	ListenAddress     string `json:"listen_address"`
	Port              int    `json:"port" validate:"required,min=1,max=65535"`
	ConfigJSON        string `json:"config_json" validate:"required"`
	TLSEnabled        bool   `json:"tls_enabled"`
	TLSCertID         *uint  `json:"tls_cert_id"`
	RealityEnabled    bool   `json:"reality_enabled"`
	RealityConfigJSON string `json:"reality_config_json"`
	IsEnabled         bool   `json:"is_enabled"`
}

type UpdateInboundDTO struct {
	Name              *string `json:"name"`
	Protocol          *string `json:"protocol"`
	ListenAddress     *string `json:"listen_address"`
	Port              *int    `json:"port" validate:"omitempty,min=1,max=65535"`
	ConfigJSON        *string `json:"config_json"`
	TLSEnabled        *bool   `json:"tls_enabled"`
	TLSCertID         *uint   `json:"tls_cert_id"`
	RealityEnabled    *bool   `json:"reality_enabled"`
	RealityConfigJSON *string `json:"reality_config_json"`
	IsEnabled         *bool   `json:"is_enabled"`
}

func (dto *UpdateInboundDTO) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	if dto.Name != nil {
		m["name"] = *dto.Name
	}
	if dto.Protocol != nil {
		m["protocol"] = *dto.Protocol
	}
	if dto.ListenAddress != nil {
		m["listen_address"] = *dto.ListenAddress
	}
	if dto.Port != nil {
		m["port"] = *dto.Port
	}
	if dto.ConfigJSON != nil {
		m["config_json"] = *dto.ConfigJSON
	}
	if dto.TLSEnabled != nil {
		m["tls_enabled"] = *dto.TLSEnabled
	}
	if dto.TLSCertID != nil {
		m["tls_cert_id"] = *dto.TLSCertID
	}
	if dto.RealityEnabled != nil {
		m["reality_enabled"] = *dto.RealityEnabled
	}
	if dto.RealityConfigJSON != nil {
		m["reality_config_json"] = *dto.RealityConfigJSON
	}
	if dto.IsEnabled != nil {
		m["is_enabled"] = *dto.IsEnabled
	}
	return m
}

type UpdateOutboundDTO struct {
	Name       *string `json:"name"`
	Protocol   *string `json:"protocol"`
	ConfigJSON *string `json:"config_json"`
	Priority   *int    `json:"priority" validate:"omitempty,min=0"`
	IsEnabled  *bool   `json:"is_enabled"`
}

func (dto *UpdateOutboundDTO) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	if dto.Name != nil {
		m["name"] = *dto.Name
	}
	if dto.Protocol != nil {
		m["protocol"] = *dto.Protocol
	}
	if dto.ConfigJSON != nil {
		m["config_json"] = *dto.ConfigJSON
	}
	if dto.Priority != nil {
		m["priority"] = *dto.Priority
	}
	if dto.IsEnabled != nil {
		m["is_enabled"] = *dto.IsEnabled
	}
	return m
}

type CreateOutboundDTO struct {
	Name       string `json:"name"`
	Protocol   string `json:"protocol" validate:"required"`
	ConfigJSON string `json:"config_json" validate:"required"`
	Priority   int    `json:"priority"`
	IsEnabled  bool   `json:"is_enabled"`
	CoreID     uint   `json:"core_id" validate:"required"`
}

type UpdateNotificationSettingsDTO struct {
	WebhookEnabled      bool   `json:"webhook_enabled"`
	WebhookURL          string `json:"webhook_url" validate:"omitempty,url"`
	WebhookSecret       string `json:"webhook_secret"`
	TelegramEnabled     bool   `json:"telegram_enabled"`
	TelegramBotToken    string `json:"telegram_bot_token"`
	TelegramChatID      string `json:"telegram_chat_id"`
	NotifyQuotaExceeded bool   `json:"notify_quota_exceeded"`
	NotifyExpiryWarning bool   `json:"notify_expiry_warning"`
	NotifyCertRenewed   bool   `json:"notify_cert_renewed"`
	NotifyCoreError     bool   `json:"notify_core_error"`
	NotifyFailedLogin   bool   `json:"notify_failed_login"`
	NotifyUserCreated   bool   `json:"notify_user_created"`
	NotifyUserDeleted   bool   `json:"notify_user_deleted"`
}

type CreateProviderDTO struct {
	Name         string `json:"name"`
	CoreID       uint   `json:"core_id"`
	ProviderType string `json:"provider_type"`
	SubType      string `json:"sub_type"`
	ConfigJSON   string `json:"config_json"`
	Priority     int    `json:"priority"`
	IsEnabled    bool   `json:"is_enabled"`
}

type UpdateProviderDTO struct {
	Name         *string `json:"name"`
	ProviderType *string `json:"provider_type"`
	SubType      *string `json:"sub_type"`
	ConfigJSON   *string `json:"config_json"`
	Priority     *int    `json:"priority"`
	IsEnabled    *bool   `json:"is_enabled"`
}

func (dto *UpdateProviderDTO) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	if dto.Name != nil {
		m["name"] = *dto.Name
	}
	if dto.ProviderType != nil {
		m["provider_type"] = *dto.ProviderType
	}
	if dto.SubType != nil {
		m["sub_type"] = *dto.SubType
	}
	if dto.ConfigJSON != nil {
		m["config_json"] = *dto.ConfigJSON
	}
	if dto.Priority != nil {
		m["priority"] = *dto.Priority
	}
	if dto.IsEnabled != nil {
		m["is_enabled"] = *dto.IsEnabled
	}
	return m
}

type UpdateWarpRouteDTO struct {
	ResourceType  *string `json:"resource_type"`
	ResourceValue *string `json:"resource_value"`
	Description   *string `json:"description"`
	Priority      *int    `json:"priority" validate:"omitempty,min=1,max=100"`
}

func (dto *UpdateWarpRouteDTO) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	if dto.ResourceType != nil {
		m["resource_type"] = *dto.ResourceType
	}
	if dto.ResourceValue != nil {
		m["resource_value"] = *dto.ResourceValue
	}
	if dto.Description != nil {
		m["description"] = *dto.Description
	}
	if dto.Priority != nil {
		m["priority"] = *dto.Priority
	}
	return m
}

type UpdateGeoRuleDTO struct {
	Type        *string `json:"type"`
	Code        *string `json:"code"`
	Action      *string `json:"action"`
	Priority    *int    `json:"priority" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description"`
}

func (dto *UpdateGeoRuleDTO) ToMap() map[string]interface{} {
	m := make(map[string]interface{})
	if dto.Type != nil {
		m["type"] = *dto.Type
	}
	if dto.Code != nil {
		m["code"] = *dto.Code
	}
	if dto.Action != nil {
		m["action"] = *dto.Action
	}
	if dto.Priority != nil {
		m["priority"] = *dto.Priority
	}
	if dto.Description != nil {
		m["description"] = *dto.Description
	}
	return m
}

type CheckPortRequestDTO struct {
	Port      int    `json:"port" validate:"required,min=1,max=65535"`
	Listen    string `json:"listen"`
	Protocol  string `json:"protocol" validate:"required"`
	Transport string `json:"transport"`
	CoreType  string `json:"core_type" validate:"required"`
}

type PortConflictItemDTO struct {
	InboundID  uint   `json:"inbound_id"`
	Name       string `json:"name"`
	Protocol   string `json:"protocol"`
	Transport  string `json:"transport,omitempty"`
	CanShare   bool   `json:"can_share"`
}

type PortConflictDTO struct {
	IsAvailable bool                  `json:"is_available"`
	Severity    string                `json:"severity"`
	Message     string                `json:"message"`
	Action      string                `json:"action"`
	Conflicts   []PortConflictItemDTO `json:"conflicts,omitempty"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=12,max=128,alphanum_special"`
}
