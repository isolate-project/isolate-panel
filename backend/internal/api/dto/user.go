package dto

// CreateUserRequest represents the request body for creating a new user
type CreateUserRequest struct {
	Username         string `json:"username" validate:"required,min=3,max=50"`
	Email            string `json:"email" validate:"email"`
	Password         string `json:"password" validate:"min=8"`
	TrafficLimitBytes *int64 `json:"traffic_limit_bytes"`
	ExpiryDays       *int   `json:"expiry_days"`
	InboundIDs       []uint `json:"inbound_ids"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Username         *string `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	Email            *string `json:"email,omitempty" validate:"omitempty,email"`
	Password         *string `json:"password,omitempty" validate:"omitempty,min=8"`
	IsActive         *bool   `json:"is_active,omitempty"`
	TrafficLimitBytes *int64  `json:"traffic_limit_bytes,omitempty"`
	ExpiryDays       *int    `json:"expiry_days,omitempty"`
	InboundIDs       []uint  `json:"inbound_ids,omitempty"`
}

// UserResponse represents the response for a user (without sensitive credentials)
type UserResponse struct {
	ID                uint    `json:"id"`
	Username          string  `json:"username"`
	Email             string  `json:"email"`
	UUID              string  `json:"uuid"`
	Token             *string `json:"token,omitempty"`
	SubscriptionToken string  `json:"subscription_token"`
	TrafficLimitBytes *int64  `json:"traffic_limit_bytes,omitempty"`
	TrafficUsedBytes  int64   `json:"traffic_used_bytes"`
	ExpiryDate        *string `json:"expiry_date,omitempty"`
	IsActive          bool    `json:"is_active"`
	IsOnline          bool    `json:"is_online"`
	CreatedAt         string  `json:"created_at"`
}

// CreateUserResponse includes sensitive credentials — returned only on Create/Regenerate
type CreateUserResponse struct {
	UserResponse
	Password string `json:"password"`
}
