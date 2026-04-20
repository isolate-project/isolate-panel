package services

import (
	"github.com/isolate-project/isolate-panel/internal/models"
)

// UserServiceInterface defines the contract for user service operations
type UserServiceInterface interface {
	CreateUser(req *CreateUserRequest, adminID uint) (*models.User, error)
	GetUser(id uint) (*models.User, error)
	ListUsers(page, pageSize int, search, status string) ([]models.User, int64, error)
	UpdateUser(id uint, req *UpdateUserRequest) (*models.User, error)
	DeleteUser(id uint) error
	RegenerateCredentials(id uint) (*models.User, error)
	GetUserInbounds(userID uint) ([]models.Inbound, error)
}

// InboundServiceInterface defines the contract for inbound service operations
type InboundServiceInterface interface {
	CreateInbound(inbound *models.Inbound) error
	GetInbound(id uint) (*models.Inbound, error)
	ListInbounds(coreID *uint, isEnabled *bool) ([]models.Inbound, error)
	ListInboundsPaginated(coreID *uint, isEnabled *bool, page, pageSize int) ([]models.Inbound, int64, error)
	UpdateInbound(id uint, updates map[string]interface{}) (*models.Inbound, error)
	DeleteInbound(id uint) error
	GetInboundsByCore(coreID uint) ([]models.Inbound, error)
	GetInboundsByCoreName(coreName string) ([]models.Inbound, error)
	GetInboundsByUser(userID uint) ([]models.Inbound, error)
	AssignInboundToUser(userID uint, inboundID uint) error
	UnassignInboundFromUser(userID uint, inboundID uint) error
	GetInboundUsers(inboundID uint) ([]models.User, error)
	BulkAssignUsers(inboundID uint, addUserIDs, removeUserIDs []uint) (int, int, error)
	ValidateInboundConfig(protocol string, configJSON string) error
}

// OutboundServiceInterface defines the contract for outbound service operations
type OutboundServiceInterface interface {
	CreateOutbound(outbound *models.Outbound) error
	GetOutbound(id uint) (*models.Outbound, error)
	ListOutbounds(coreID *uint, protocolFilter string) ([]models.Outbound, error)
	UpdateOutbound(id uint, updates map[string]interface{}) (*models.Outbound, error)
	DeleteOutbound(id uint) error
	GetOutboundsByCore(coreID uint) ([]models.Outbound, error)
}

// CoreServiceInterface defines the contract for core lifecycle operations
type CoreServiceInterface interface {
	StartCore(coreName string) error
	StopCore(coreName string) error
	RestartCore(coreName string) error
	GetCoreStatus(coreName string) (map[string]interface{}, error)
	ListCores() ([]models.Core, error)
	GetCore(coreName string) (*models.Core, error)
}

// SubscriptionServiceInterface defines the contract for subscription operations
type SubscriptionServiceInterface interface {
	GetAutoDetectSubscription(token string) (string, error)
	GetClashSubscription(token string) (string, error)
	GetSingboxSubscription(token string) (string, error)
	GetIsolateSubscription(token string) (string, error)
	GetQRCode(token string) (string, error)
	GetUserShortURL(userID uint) (string, error)
	RegenerateToken(userID uint) error
	GetAccessStats(userID uint) (map[string]interface{}, error)
	InvalidateUserCache(userID uint)
}