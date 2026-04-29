// Package eventbus provides a generics-based, thread-safe event bus for decoupled
// communication between application modules. It supports both async (fire-and-forget)
// and sync (wait for handlers) publishing modes, with panic recovery and zerolog integration.
package eventbus

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
)

var (
	// ErrHandlerPanicked is returned when a handler panics during execution
	ErrHandlerPanicked = errors.New("handler panicked")
	// ErrNoSubscribers is returned when publishing to a bus with no subscribers
	ErrNoSubscribers = errors.New("no subscribers for event type")
)

// SubscriptionID uniquely identifies a subscription for unsubscription
type SubscriptionID uint64

var globalSubIDCounter atomic.Uint64

func nextSubscriptionID() SubscriptionID {
	return SubscriptionID(globalSubIDCounter.Add(1))
}

// Handler is a function that handles events of type T
type Handler[T any] func(event T) error

// subscription represents a single subscription to an event type
type subscription[T any] struct {
	id      SubscriptionID
	handler Handler[T]
}

// EventBus is a generics-based event bus that handles events of type T.
// It is safe for concurrent use by multiple goroutines.
type EventBus[T any] struct {
	mu            sync.RWMutex
	subscriptions map[SubscriptionID]*subscription[T]
	logger        zerolog.Logger
}

// NewEventBus creates a new EventBus for events of type T with the provided logger.
func NewEventBus[T any](logger zerolog.Logger) *EventBus[T] {
	return &EventBus[T]{
		subscriptions: make(map[SubscriptionID]*subscription[T]),
		logger:        logger.With().Str("component", "eventbus").Logger(),
	}
}

// Subscribe registers a handler for events of type T and returns a subscription ID.
// The handler will be called for all future events until Unsubscribe is called.
func (b *EventBus[T]) Subscribe(handler Handler[T]) SubscriptionID {
	if handler == nil {
		panic("eventbus: nil handler")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	id := nextSubscriptionID()
	b.subscriptions[id] = &subscription[T]{
		id:      id,
		handler: handler,
	}

	b.logger.Debug().
		Str("event_type", fmt.Sprintf("%T", *new(T))).
		Uint64("subscription_id", uint64(id)).
		Int("subscriber_count", len(b.subscriptions)).
		Msg("new subscription")

	return id
}

// Unsubscribe removes a subscription by its ID. Returns true if the subscription was found and removed.
func (b *EventBus[T]) Unsubscribe(id SubscriptionID) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscriptions[id]; !exists {
		return false
	}

	delete(b.subscriptions, id)

	b.logger.Debug().
		Str("event_type", fmt.Sprintf("%T", *new(T))).
		Uint64("subscription_id", uint64(id)).
		Int("subscriber_count", len(b.subscriptions)).
		Msg("subscription removed")

	return true
}

// Publish publishes an event asynchronously (fire-and-forget).
// Handlers execute in separate goroutines with panic recovery.
func (b *EventBus[T]) Publish(event T) {
	b.mu.RLock()
	subs := make([]*subscription[T], 0, len(b.subscriptions))
	for _, sub := range b.subscriptions {
		subs = append(subs, sub)
	}
	b.mu.RUnlock()

	if len(subs) == 0 {
		b.logger.Debug().
			Str("event_type", fmt.Sprintf("%T", event)).
			Msg("no subscribers for event")
		return
	}

	for _, sub := range subs {
		go func(s *subscription[T]) {
			b.executeHandler(s, event)
		}(sub)
	}
}

// PublishSync publishes an event synchronously, waiting for all handlers to complete.
// Returns the first error encountered, or nil if all handlers succeed.
func (b *EventBus[T]) PublishSync(event T) error {
	b.mu.RLock()
	subs := make([]*subscription[T], 0, len(b.subscriptions))
	for _, sub := range b.subscriptions {
		subs = append(subs, sub)
	}
	b.mu.RUnlock()

	if len(subs) == 0 {
		b.logger.Debug().
			Str("event_type", fmt.Sprintf("%T", event)).
			Msg("no subscribers for sync event")
		return nil
	}

	var firstErr error
	for _, sub := range subs {
		if err := b.executeHandlerSync(sub, event); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// executeHandler executes a handler with panic recovery (async version)
func (b *EventBus[T]) executeHandler(sub *subscription[T], event T) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error().
				Str("event_type", fmt.Sprintf("%T", event)).
				Uint64("subscription_id", uint64(sub.id)).
				Interface("panic", r).
				Msg("handler panicked")
		}
	}()

	if err := sub.handler(event); err != nil {
		b.logger.Error().
			Str("event_type", fmt.Sprintf("%T", event)).
			Uint64("subscription_id", uint64(sub.id)).
			Err(err).
			Msg("handler returned error")
	}
}

// executeHandlerSync executes a handler with panic recovery (sync version)
func (b *EventBus[T]) executeHandlerSync(sub *subscription[T], event T) (err error) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error().
				Str("event_type", fmt.Sprintf("%T", event)).
				Uint64("subscription_id", uint64(sub.id)).
				Interface("panic", r).
				Msg("handler panicked")
			err = fmt.Errorf("%w: %v", ErrHandlerPanicked, r)
		}
	}()

	return sub.handler(event)
}

// SubscriberCount returns the current number of subscribers
func (b *EventBus[T]) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscriptions)
}

// Close removes all subscriptions. Safe to call multiple times.
func (b *EventBus[T]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	count := len(b.subscriptions)
	for id := range b.subscriptions {
		delete(b.subscriptions, id)
	}

	if count > 0 {
		b.logger.Debug().
			Str("event_type", fmt.Sprintf("%T", *new(T))).
			Int("removed_count", count).
			Msg("event bus closed")
	}
}
