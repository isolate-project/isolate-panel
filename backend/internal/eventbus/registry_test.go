package eventbus

import (
	"sync"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	defer reg.Close()

	if reg.UserCreated == nil {
		t.Error("expected UserCreated bus to be initialized")
	}
	if reg.UserUpdated == nil {
		t.Error("expected UserUpdated bus to be initialized")
	}
	if reg.UserDeleted == nil {
		t.Error("expected UserDeleted bus to be initialized")
	}
	if reg.CoreStarted == nil {
		t.Error("expected CoreStarted bus to be initialized")
	}
	if reg.CoreStopped == nil {
		t.Error("expected CoreStopped bus to be initialized")
	}
	if reg.CoreRestarted == nil {
		t.Error("expected CoreRestarted bus to be initialized")
	}
	if reg.InboundCreated == nil {
		t.Error("expected InboundCreated bus to be initialized")
	}
	if reg.InboundDeleted == nil {
		t.Error("expected InboundDeleted bus to be initialized")
	}
	if reg.BackupCreated == nil {
		t.Error("expected BackupCreated bus to be initialized")
	}
	if reg.AdminLogin == nil {
		t.Error("expected AdminLogin bus to be initialized")
	}
	if reg.AdminAction == nil {
		t.Error("expected AdminAction bus to be initialized")
	}
}

func TestRegistryClose(t *testing.T) {
	reg := NewRegistry()

	reg.UserCreated.Subscribe(func(event UserCreatedEvent) error { return nil })
	reg.CoreStarted.Subscribe(func(event CoreStartedEvent) error { return nil })

	if reg.UserCreated.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber on UserCreated, got %d", reg.UserCreated.SubscriberCount())
	}

	reg.Close()

	if reg.UserCreated.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after close, got %d", reg.UserCreated.SubscriberCount())
	}
	if reg.CoreStarted.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after close, got %d", reg.CoreStarted.SubscriberCount())
	}
}

func TestGetDefaultRegistry(t *testing.T) {
	defaultRegistry = nil
	defaultRegistryOnce = sync.Once{}

	reg1 := GetDefaultRegistry()
	if reg1 == nil {
		t.Fatal("expected non-nil registry")
	}

	reg2 := GetDefaultRegistry()
	if reg1 != reg2 {
		t.Error("expected GetDefaultRegistry to return the same instance")
	}

	reg1.Close()
	defaultRegistry = nil
	defaultRegistryOnce = sync.Once{}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	defaultRegistry = nil
	defaultRegistryOnce = sync.Once{}

	var wg sync.WaitGroup
	registries := make(chan *Registry, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			registries <- GetDefaultRegistry()
		}()
	}

	wg.Wait()
	close(registries)

	var first *Registry
	for reg := range registries {
		if first == nil {
			first = reg
		} else if reg != first {
			t.Error("concurrent GetDefaultRegistry calls returned different instances")
		}
	}

	first.Close()
	defaultRegistry = nil
	defaultRegistryOnce = sync.Once{}
}

func TestRegistryIntegration(t *testing.T) {
	reg := NewRegistry()
	defer reg.Close()

	userReceived := make(chan UserCreatedEvent, 1)
	coreReceived := make(chan CoreStartedEvent, 1)

	reg.UserCreated.Subscribe(func(event UserCreatedEvent) error {
		userReceived <- event
		return nil
	})

	reg.CoreStarted.Subscribe(func(event CoreStartedEvent) error {
		coreReceived <- event
		return nil
	})

	reg.UserCreated.Publish(UserCreatedEvent{})
	reg.CoreStarted.Publish(CoreStartedEvent{CoreName: "xray"})
}
