package eventbus

import (
	"time"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// UserCreatedEvent is fired when a new user is created
type UserCreatedEvent struct {
	User      models.User
	CreatedBy uint
	Timestamp time.Time
}

// UserUpdatedEvent is fired when a user is modified
type UserUpdatedEvent struct {
	User         models.User
	UpdatedBy    uint
	PreviousData *models.User
	Timestamp    time.Time
}

// UserDeletedEvent is fired when a user is deleted
type UserDeletedEvent struct {
	UserID      uint
	Username    string
	DeletedBy   uint
	Timestamp   time.Time
	InboundIDs  []uint
}

// CoreStartedEvent is fired when a core process starts
type CoreStartedEvent struct {
	CoreID    uint
	CoreName  string
	PID       int
	Timestamp time.Time
}

// CoreStoppedEvent is fired when a core process stops
type CoreStoppedEvent struct {
	CoreID    uint
	CoreName  string
	Reason    string
	Timestamp time.Time
}

// CoreRestartedEvent is fired when a core is restarted
type CoreRestartedEvent struct {
	CoreID    uint
	CoreName  string
	OldPID    int
	NewPID    int
	Timestamp time.Time
}

// InboundCreatedEvent is fired when a new inbound is created
type InboundCreatedEvent struct {
	Inbound   models.Inbound
	CreatedBy uint
	Timestamp time.Time
}

// InboundDeletedEvent is fired when an inbound is deleted
type InboundDeletedEvent struct {
	InboundID uint
	CoreID    uint
	Port      int
	Protocol  string
	DeletedBy uint
	Timestamp time.Time
}

// BackupCreatedEvent is fired when a backup is completed
type BackupCreatedEvent struct {
	Backup    models.Backup
	CreatedBy uint
	Timestamp time.Time
}

// AdminLoginEvent is fired when an admin logs in (for audit logging)
type AdminLoginEvent struct {
	AdminID   uint
	Username  string
	IPAddress string
	Success   bool
	Reason    string
	Timestamp time.Time
}

// AdminActionEvent is fired for significant admin actions (for audit logging)
type AdminActionEvent struct {
	AdminID    uint
	Username   string
	Action     string
	Resource   string
	ResourceID *uint
	Details    string
	IPAddress  string
	Timestamp  time.Time
}
