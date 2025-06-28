package runtime

import (
	"fmt"
	"sync"
)

// Registry manages runtime implementations
type Registry struct {
	mu       sync.RWMutex
	runtimes map[string]Runtime
	defaults map[string]RuntimeOptions
}

// defaultRegistry is the global registry instance
var defaultRegistry = &Registry{
	runtimes: make(map[string]Runtime),
	defaults: make(map[string]RuntimeOptions),
}

// Register adds a runtime to the default registry
func Register(name string, runtime Runtime, defaultOpts RuntimeOptions) error {
	return defaultRegistry.Register(name, runtime, defaultOpts)
}

// Get retrieves a runtime from the default registry
func Get(name string) (Runtime, error) {
	return defaultRegistry.Get(name)
}

// List returns all registered runtime names from the default registry
func List() []string {
	return defaultRegistry.List()
}

// GetDefaultOptions returns the default options for a runtime from the default registry
func GetDefaultOptions(name string) (RuntimeOptions, error) {
	return defaultRegistry.GetDefaultOptions(name)
}

// NewRegistry creates a new registry instance
func NewRegistry() *Registry {
	return &Registry{
		runtimes: make(map[string]Runtime),
		defaults: make(map[string]RuntimeOptions),
	}
}

// Register adds a runtime to the registry
func (r *Registry) Register(name string, runtime Runtime, defaultOpts RuntimeOptions) error {
	if name == "" {
		return fmt.Errorf("runtime name cannot be empty")
	}
	if runtime == nil {
		return fmt.Errorf("runtime cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.runtimes[name]; exists {
		return fmt.Errorf("runtime %q already registered", name)
	}

	r.runtimes[name] = runtime
	if defaultOpts != nil {
		r.defaults[name] = defaultOpts
	}

	return nil
}

// Get retrieves a runtime by name
func (r *Registry) Get(name string) (Runtime, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	runtime, exists := r.runtimes[name]
	if !exists {
		return nil, fmt.Errorf("runtime %q not found", name)
	}

	return runtime, nil
}

// GetDefaultOptions returns the default options for a runtime
func (r *Registry) GetDefaultOptions(name string) (RuntimeOptions, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	opts, exists := r.defaults[name]
	if !exists {
		return nil, fmt.Errorf("no default options for runtime %q", name)
	}

	return opts, nil
}

// List returns all registered runtime names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.runtimes))
	for name := range r.runtimes {
		names = append(names, name)
	}

	return names
}

// Has checks if a runtime is registered
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.runtimes[name]
	return exists
}

// Clear removes all registered runtimes
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.runtimes = make(map[string]Runtime)
	r.defaults = make(map[string]RuntimeOptions)
}
