package eventbus

import (
	"sync"

	"github.com/isolate-project/isolate-panel/internal/logger"
)

// Registry holds all typed event buses for the application.
// This provides a centralized location for event bus access while maintaining type safety.
type Registry struct {
	UserCreated   *EventBus[UserCreatedEvent]
	UserUpdated   *EventBus[UserUpdatedEvent]
	UserDeleted   *EventBus[UserDeletedEvent]
	CoreStarted   *EventBus[CoreStartedEvent]
	CoreStopped   *EventBus[CoreStoppedEvent]
	CoreRestarted *EventBus[CoreRestartedEvent]
	InboundCreated *EventBus[InboundCreatedEvent]
	InboundDeleted *EventBus[InboundDeletedEvent]
	BackupCreated  *EventBus[BackupCreatedEvent]
	AdminLogin     *EventBus[AdminLoginEvent]
	AdminAction    *EventBus[AdminActionEvent]
}

// NewRegistry creates a new event bus registry with all typed buses initialized.
func NewRegistry() *Registry {
	log := logger.WithComponent("eventbus")

	return &Registry{
		UserCreated:    NewEventBus[UserCreatedEvent](log),
		UserUpdated:    NewEventBus[UserUpdatedEvent](log),
		UserDeleted:    NewEventBus[UserDeletedEvent](log),
		CoreStarted:    NewEventBus[CoreStartedEvent](log),
		CoreStopped:    NewEventBus[CoreStoppedEvent](log),
		CoreRestarted:  NewEventBus[CoreRestartedEvent](log),
		InboundCreated: NewEventBus[InboundCreatedEvent](log),
		InboundDeleted: NewEventBus[InboundDeletedEvent](log),
		BackupCreated:  NewEventBus[BackupCreatedEvent](log),
		AdminLogin:     NewEventBus[AdminLoginEvent](log),
		AdminAction:    NewEventBus[AdminActionEvent](log),
	}
}

// Close shuts down all event buses in the registry.
func (r *Registry) Close() {
	r.UserCreated.Close()
	r.UserUpdated.Close()
	r.UserDeleted.Close()
	r.CoreStarted.Close()
	r.CoreStopped.Close()
	r.CoreRestarted.Close()
	r.InboundCreated.Close()
	r.InboundDeleted.Close()
	r.BackupCreated.Close()
	r.AdminLogin.Close()
	r.AdminAction.Close()
}

var (
	// defaultRegistry is the global event bus registry instance.
	defaultRegistry *Registry
	// defaultRegistryOnce ensures the global registry is initialized only once.
	defaultRegistryOnce sync.Once
)

// GetDefaultRegistry returns the global event bus registry, initializing it if necessary.
// This is safe for concurrent use.
func GetDefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}
