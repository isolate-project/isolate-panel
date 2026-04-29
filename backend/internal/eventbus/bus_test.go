package eventbus

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

func init() {
	logger.Init(&logger.Config{
		Level:  "debug",
		Format: "console",
		Output: "stdout",
	})
}

func TestNewEventBus(t *testing.T) {
	bus := NewEventBus[string](logger.Log)
	defer bus.Close()

	if bus == nil {
		t.Fatal("expected non-nil EventBus")
	}

	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", bus.SubscriberCount())
	}
}

func TestSubscribeAndPublish(t *testing.T) {
	bus := NewEventBus[string](logger.Log)
	defer bus.Close()

	received := make(chan string, 1)
	handler := func(event string) error {
		received <- event
		return nil
	}

	id := bus.Subscribe(handler)
	if id == 0 {
		t.Error("expected non-zero subscription ID")
	}

	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	bus.Publish("test message")

	select {
	case msg := <-received:
		if msg != "test message" {
			t.Errorf("expected 'test message', got %s", msg)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewEventBus[int](logger.Log)
	defer bus.Close()

	received1 := make(chan int, 1)
	received2 := make(chan int, 1)

	bus.Subscribe(func(event int) error {
		received1 <- event
		return nil
	})

	bus.Subscribe(func(event int) error {
		received2 <- event
		return nil
	})

	if bus.SubscriberCount() != 2 {
		t.Errorf("expected 2 subscribers, got %d", bus.SubscriberCount())
	}

	bus.Publish(42)

	select {
	case val := <-received1:
		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for subscriber 1")
	}

	select {
	case val := <-received2:
		if val != 42 {
			t.Errorf("expected 42, got %d", val)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for subscriber 2")
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewEventBus[string](logger.Log)
	defer bus.Close()

	received := make(chan string, 1)
	id := bus.Subscribe(func(event string) error {
		received <- event
		return nil
	})

	if bus.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount())
	}

	if !bus.Unsubscribe(id) {
		t.Error("expected Unsubscribe to return true")
	}

	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", bus.SubscriberCount())
	}

	if bus.Unsubscribe(id) {
		t.Error("expected Unsubscribe to return false for non-existent ID")
	}

	bus.Publish("should not receive")

	select {
	case <-received:
		t.Error("should not receive event after unsubscribe")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestPublishSync(t *testing.T) {
	bus := NewEventBus[string](logger.Log)
	defer bus.Close()

	received := make(chan string, 2)

	bus.Subscribe(func(event string) error {
		received <- event + "-1"
		return nil
	})

	bus.Subscribe(func(event string) error {
		received <- event + "-2"
		return nil
	})

	err := bus.PublishSync("sync message")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var results []string
	for i := 0; i < 2; i++ {
		select {
		case msg := <-received:
			results = append(results, msg)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for sync event")
		}
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestPublishSyncWithError(t *testing.T) {
	bus := NewEventBus[string](logger.Log)
	defer bus.Close()

	testErr := errors.New("handler error")

	bus.Subscribe(func(event string) error {
		return nil
	})

	bus.Subscribe(func(event string) error {
		return testErr
	})

	err := bus.PublishSync("test")
	if err == nil {
		t.Error("expected error from handler")
	}

	if !errors.Is(err, testErr) {
		t.Errorf("expected testErr, got %v", err)
	}
}

func TestPanicRecovery(t *testing.T) {
	bus := NewEventBus[string](logger.Log)
	defer bus.Close()

	received := make(chan string, 1)

	bus.Subscribe(func(event string) error {
		panic("intentional panic")
	})

	bus.Subscribe(func(event string) error {
		received <- event
		return nil
	})

	bus.Publish("panic test")

	select {
	case msg := <-received:
		if msg != "panic test" {
			t.Errorf("expected 'panic test', got %s", msg)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for event from non-panicking handler")
	}
}

func TestPanicRecoverySync(t *testing.T) {
	bus := NewEventBus[string](logger.Log)
	defer bus.Close()

	bus.Subscribe(func(event string) error {
		panic("intentional panic")
	})

	err := bus.PublishSync("panic test")
	if err == nil {
		t.Error("expected error from panicking handler")
	}

	if !errors.Is(err, ErrHandlerPanicked) {
		t.Errorf("expected ErrHandlerPanicked, got %v", err)
	}
}

func TestConcurrentSubscribePublish(t *testing.T) {
	bus := NewEventBus[int](logger.Log)
	defer bus.Close()

	var subWg sync.WaitGroup
	var pubWg sync.WaitGroup
	var handlerWg sync.WaitGroup
	received := make(chan int, 100)

	for i := 0; i < 10; i++ {
		subWg.Add(1)
		go func() {
			defer subWg.Done()
			bus.Subscribe(func(event int) error {
				received <- event
				handlerWg.Done()
				return nil
			})
		}()
	}

	subWg.Wait()

	if bus.SubscriberCount() != 10 {
		t.Errorf("expected 10 subscribers, got %d", bus.SubscriberCount())
	}

	handlerWg.Add(100)
	for i := 0; i < 10; i++ {
		pubWg.Add(1)
		go func(val int) {
			defer pubWg.Done()
			bus.Publish(val)
		}(i)
	}

	done := make(chan struct{})
	go func() {
		pubWg.Wait()
		handlerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for concurrent operations")
	}

	close(received)
	count := 0
	for range received {
		count++
	}

	if count != 100 {
		t.Errorf("expected 100 events, got %d", count)
	}
}

func TestClose(t *testing.T) {
	bus := NewEventBus[string](logger.Log)

	for i := 0; i < 5; i++ {
		bus.Subscribe(func(event string) error {
			return nil
		})
	}

	if bus.SubscriberCount() != 5 {
		t.Errorf("expected 5 subscribers, got %d", bus.SubscriberCount())
	}

	bus.Close()

	if bus.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after close, got %d", bus.SubscriberCount())
	}

	bus.Close()
}

func TestNilHandlerPanic(t *testing.T) {
	bus := NewEventBus[string](logger.Log)
	defer bus.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil handler")
		}
	}()

	bus.Subscribe(nil)
}

func TestDomainEvents(t *testing.T) {
	userBus := NewEventBus[UserCreatedEvent](logger.Log)
	defer userBus.Close()

	received := make(chan UserCreatedEvent, 1)

	userBus.Subscribe(func(event UserCreatedEvent) error {
		received <- event
		return nil
	})

	event := UserCreatedEvent{
		User: models.User{
			ID:       1,
			Username: "testuser",
			UUID:     "test-uuid",
		},
		CreatedBy: 1,
		Timestamp: time.Now(),
	}

	userBus.Publish(event)

	select {
	case receivedEvent := <-received:
		if receivedEvent.User.ID != 1 {
			t.Errorf("expected user ID 1, got %d", receivedEvent.User.ID)
		}
		if receivedEvent.User.Username != "testuser" {
			t.Errorf("expected username 'testuser', got %s", receivedEvent.User.Username)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for domain event")
	}
}

func TestMultipleEventTypes(t *testing.T) {
	userBus := NewEventBus[UserCreatedEvent](logger.Log)
	coreBus := NewEventBus[CoreStartedEvent](logger.Log)
	defer userBus.Close()
	defer coreBus.Close()

	userReceived := make(chan UserCreatedEvent, 1)
	coreReceived := make(chan CoreStartedEvent, 1)

	userBus.Subscribe(func(event UserCreatedEvent) error {
		userReceived <- event
		return nil
	})

	coreBus.Subscribe(func(event CoreStartedEvent) error {
		coreReceived <- event
		return nil
	})

	userBus.Publish(UserCreatedEvent{User: models.User{ID: 1}})
	coreBus.Publish(CoreStartedEvent{CoreID: 1, CoreName: "xray"})

	select {
	case userEvent := <-userReceived:
		if userEvent.User.ID != 1 {
			t.Errorf("expected user ID 1, got %d", userEvent.User.ID)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for user event")
	}

	select {
	case coreEvent := <-coreReceived:
		if coreEvent.CoreName != "xray" {
			t.Errorf("expected core name 'xray', got %s", coreEvent.CoreName)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for core event")
	}
}
