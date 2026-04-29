package dto

// ListResponse is a generic paginated list response
type ListResponse[T any] struct {
	Data     []T   `json:"data"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Pages    int64 `json:"pages"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Code    string                 `json:"code,omitempty"`
	TraceID string                 `json:"trace_id,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// SuccessResponse represents a generic success response with a message
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message"`
}
