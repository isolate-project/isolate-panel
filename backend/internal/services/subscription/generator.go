package subscription

import (
	"fmt"
	"sync"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// UserSubscriptionData holds the data needed to generate a subscription
type UserSubscriptionData struct {
	User     models.User
	Inbounds []models.Inbound
	Filter   *SubscriptionFilter
}

// Generator defines the interface for subscription format generators
type Generator interface {
	Name() string
	Generate(data *UserSubscriptionData) (string, error)
}

// Registry manages subscription format generators
type Registry struct {
	generators map[string]Generator
	mu         sync.RWMutex
}

// NewRegistry creates a new generator registry
func NewRegistry() *Registry {
	return &Registry{
		generators: make(map[string]Generator),
	}
}

// Register adds a generator to the registry
func (r *Registry) Register(g Generator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.generators[g.Name()] = g
}

// Generate generates a subscription using the specified format
func (r *Registry) Generate(format string, data *UserSubscriptionData) (string, error) {
	r.mu.RLock()
	g, ok := r.generators[format]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("unknown subscription format: %s", format)
	}

	return g.Generate(data)
}

// Names returns a list of all registered generator names
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.generators))
	for name := range r.generators {
		names = append(names, name)
	}
	return names
}

// Has checks if a generator is registered for the given format
func (r *Registry) Has(format string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.generators[format]
	return ok
}
