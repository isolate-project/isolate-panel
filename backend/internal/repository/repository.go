package repository

import (
	"context"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// UserRepository defines the contract for user data access
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *models.User) error

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id uint) (*models.User, error)

	// GetByUUID retrieves a user by UUID
	GetByUUID(ctx context.Context, uuid string) (*models.User, error)

	// GetByToken retrieves a user by subscription token
	GetByToken(ctx context.Context, token string) (*models.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *models.User) error

	// Delete soft-deletes a user by ID
	Delete(ctx context.Context, id uint) error

	// List retrieves users with pagination and optional filtering
	List(ctx context.Context, offset, limit int, search, status string) ([]models.User, int64, error)

	// Search searches users by username or email
	Search(ctx context.Context, query string, offset, limit int) ([]models.User, int64, error)

	// UpdateTrafficUsed updates the traffic used for a user
	UpdateTrafficUsed(ctx context.Context, id uint, bytes int64) error

	// UpdateOnlineStatus updates the online status of a user
	UpdateOnlineStatus(ctx context.Context, id uint, isOnline bool) error

	// UpdateLastConnected updates the last connected timestamp
	UpdateLastConnected(ctx context.Context, id uint) error
}

// InboundRepository defines the contract for inbound data access
type InboundRepository interface {
	// Create creates a new inbound
	Create(ctx context.Context, inbound *models.Inbound) error

	// GetByID retrieves an inbound by ID
	GetByID(ctx context.Context, id uint) (*models.Inbound, error)

	// Update updates an existing inbound
	Update(ctx context.Context, inbound *models.Inbound) error

	// Delete deletes an inbound by ID
	Delete(ctx context.Context, id uint) error

	// List retrieves all inbounds with optional filtering
	List(ctx context.Context, coreID *uint, isEnabled *bool) ([]models.Inbound, error)

	// ListPaginated retrieves inbounds with pagination
	ListPaginated(ctx context.Context, coreID *uint, isEnabled *bool, offset, limit int) ([]models.Inbound, int64, error)

	// GetByCore retrieves inbounds by core ID
	GetByCore(ctx context.Context, coreID uint) ([]models.Inbound, error)

	// GetByCoreName retrieves inbounds by core name
	GetByCoreName(ctx context.Context, coreName string) ([]models.Inbound, error)

	// GetByUser retrieves inbounds assigned to a user
	GetByUser(ctx context.Context, userID uint) ([]models.Inbound, error)

	// AssignToUser assigns an inbound to a user
	AssignToUser(ctx context.Context, userID, inboundID uint) error

	// UnassignFromUser removes an inbound assignment from a user
	UnassignFromUser(ctx context.Context, userID, inboundID uint) error

	// GetUsers retrieves users assigned to an inbound
	GetUsers(ctx context.Context, inboundID uint) ([]models.User, error)

	// BulkAssign performs bulk assignment/unassignment of users to an inbound
	BulkAssign(ctx context.Context, inboundID uint, addUserIDs, removeUserIDs []uint) (added, removed int, err error)
}

// OutboundRepository defines the contract for outbound data access
type OutboundRepository interface {
	// Create creates a new outbound
	Create(ctx context.Context, outbound *models.Outbound) error

	// GetByID retrieves an outbound by ID
	GetByID(ctx context.Context, id uint) (*models.Outbound, error)

	// Update updates an existing outbound
	Update(ctx context.Context, outbound *models.Outbound) error

	// Delete deletes an outbound by ID
	Delete(ctx context.Context, id uint) error

	// List retrieves all outbounds with optional filtering
	List(ctx context.Context, coreID *uint, protocolFilter string) ([]models.Outbound, error)

	// GetByCore retrieves outbounds by core ID
	GetByCore(ctx context.Context, coreID uint) ([]models.Outbound, error)
}

// CoreRepository defines the contract for core data access
type CoreRepository interface {
	// Create creates a new core
	Create(ctx context.Context, core *models.Core) error

	// GetByID retrieves a core by ID
	GetByID(ctx context.Context, id uint) (*models.Core, error)

	// GetByName retrieves a core by name
	GetByName(ctx context.Context, name string) (*models.Core, error)

	// Update updates an existing core
	Update(ctx context.Context, core *models.Core) error

	// Delete deletes a core by ID
	Delete(ctx context.Context, id uint) error

	// List retrieves all cores
	List(ctx context.Context) ([]models.Core, error)

	// UpdateStatus updates the running status of a core
	UpdateStatus(ctx context.Context, id uint, isRunning bool, pid *int) error

	// UpdateHealth updates the health status of a core
	UpdateHealth(ctx context.Context, id uint, healthStatus, lastError string) error

	// UpdateUptime updates the uptime statistics
	UpdateUptime(ctx context.Context, id uint, uptimeSeconds int, restartCount int) error
}

// AdminRepository defines the contract for admin data access
type AdminRepository interface {
	// Create creates a new admin
	Create(ctx context.Context, admin *models.Admin) error

	// GetByID retrieves an admin by ID
	GetByID(ctx context.Context, id uint) (*models.Admin, error)

	// GetByUsername retrieves an admin by username
	GetByUsername(ctx context.Context, username string) (*models.Admin, error)

	// Update updates an existing admin
	Update(ctx context.Context, admin *models.Admin) error

	// Delete deletes an admin by ID
	Delete(ctx context.Context, id uint) error

	// List retrieves all admins
	List(ctx context.Context) ([]models.Admin, error)

	// UpdateLastLogin updates the last login timestamp
	UpdateLastLogin(ctx context.Context, id uint) error

	// UpdatePassword updates the password hash
	UpdatePassword(ctx context.Context, id uint, passwordHash string) error

	// UpdateTOTPSecret updates the TOTP secret
	UpdateTOTPSecret(ctx context.Context, id uint, secret string) error

	// UpdatePermissions updates the admin permissions
	UpdatePermissions(ctx context.Context, id uint, permissions uint64) error
}

// BackupRepository defines the contract for backup data access
type BackupRepository interface {
	// Create creates a new backup record
	Create(ctx context.Context, backup *models.Backup) error

	// GetByID retrieves a backup by ID
	GetByID(ctx context.Context, id uint) (*models.Backup, error)

	// Update updates an existing backup
	Update(ctx context.Context, backup *models.Backup) error

	// Delete deletes a backup by ID
	Delete(ctx context.Context, id uint) error

	// List retrieves all backups with pagination
	List(ctx context.Context, offset, limit int) ([]models.Backup, int64, error)

	// ListByStatus retrieves backups by status
	ListByStatus(ctx context.Context, status models.BackupStatus) ([]models.Backup, error)

	// UpdateStatus updates the status of a backup
	UpdateStatus(ctx context.Context, id uint, status models.BackupStatus, errorMessage string) error

	// UpdateCompletion updates the completion details
	UpdateCompletion(ctx context.Context, id uint, fileSizeBytes int64, checksumSHA256 string, durationMs int) error
}
