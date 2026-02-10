package core

import (
	"fmt"
	"sync"
)

// Registry manages registered providers and orchestrators
type Registry struct {
	mu            sync.RWMutex
	providers     map[ProviderType]Provider
	orchestrators map[OrchestratorType]Orchestrator
}

// NewRegistry creates a new registry instance
func NewRegistry() *Registry {
	return &Registry{
		providers:     make(map[ProviderType]Provider),
		orchestrators: make(map[OrchestratorType]Orchestrator),
	}
}

// RegisterProvider registers a new provider
func (r *Registry) RegisterProvider(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.Name()] = provider
}

// RegisterOrchestrator registers a new orchestrator
func (r *Registry) RegisterOrchestrator(orchestrator Orchestrator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orchestrators[orchestrator.Name()] = orchestrator
}

// GetProvider retrieves a provider by name
func (r *Registry) GetProvider(name ProviderType) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}
	return provider, nil
}

// GetOrchestrator retrieves an orchestrator by name
func (r *Registry) GetOrchestrator(name OrchestratorType) (Orchestrator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	orchestrator, ok := r.orchestrators[name]
	if !ok {
		return nil, fmt.Errorf("orchestrator '%s' not found", name)
	}
	return orchestrator, nil
}

// ListProviders returns all registered provider names
func (r *Registry) ListProviders() []ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]ProviderType, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// ListOrchestrators returns all registered orchestrator names
func (r *Registry) ListOrchestrators() []OrchestratorType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]OrchestratorType, 0, len(r.orchestrators))
	for name := range r.orchestrators {
		names = append(names, name)
	}
	return names
}

// GlobalRegistry is the global registry instance
var GlobalRegistry = NewRegistry()
