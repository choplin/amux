// Package init provides runtime initialization functions
package init

import (
	"fmt"
	"os"

	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/runtime/config"
	"github.com/aki/amux/internal/runtime/local"
	"github.com/aki/amux/internal/runtime/tmux"
)

// RegisterDefaults registers the default runtime implementations
func RegisterDefaults() error {
	// Check if already registered to avoid duplicate registration
	if len(runtime.List()) > 0 {
		// Runtimes already registered
		return nil
	}

	// Register local runtime
	localRT := local.New()
	if err := runtime.Register("local", localRT, local.Options{
		InheritEnv:    true,
		CaptureOutput: false,
	}); err != nil {
		return fmt.Errorf("failed to register local runtime: %w", err)
	}

	// Register tmux runtime if available
	tmuxRT, err := tmux.New("")
	if err == nil && tmuxRT.Validate() == nil {
		if err := runtime.Register("tmux", tmuxRT, tmux.Options{
			WindowName:    "amux",
			CaptureOutput: true,
			OutputHistory: 10000,
		}); err != nil {
			return fmt.Errorf("failed to register tmux runtime: %w", err)
		}
	}

	return nil
}

// Config represents runtime configuration
type Config struct {
	Type    string                 `yaml:"type" json:"type"`
	Options map[string]interface{} `yaml:"options" json:"options"`
}

// CreateRuntime creates a runtime from configuration
func CreateRuntime(cfg Config) (runtime.Runtime, error) {
	switch cfg.Type {
	case "local":
		return createLocal(cfg.Options)
	case "tmux":
		return createTmux(cfg.Options)
	default:
		return nil, fmt.Errorf("unknown runtime type: %s", cfg.Type)
	}
}

// createLocal creates a local runtime from options
func createLocal(options map[string]interface{}) (runtime.Runtime, error) {
	// Local runtime doesn't need configuration
	return local.New(), nil
}

// createTmux creates a tmux runtime from options
func createTmux(options map[string]interface{}) (runtime.Runtime, error) {
	baseDir := ""
	if bd, ok := options["base_dir"].(string); ok {
		baseDir = bd
	}

	return tmux.New(baseDir)
}

// CreateFromType creates a runtime instance by type name
func CreateFromType(runtimeType string) (runtime.Runtime, error) {
	// Check if already registered
	if rt, err := runtime.Get(runtimeType); err == nil {
		return rt, nil
	}

	// Otherwise create new instance
	return CreateRuntime(Config{Type: runtimeType})
}

// RegisterCustomRuntimes loads and registers custom runtimes from configuration files
func RegisterCustomRuntimes(projectDir string) error {
	homeDir, _ := os.UserHomeDir()
	loader := config.NewLoader(homeDir, projectDir)

	cfg, err := loader.Load()
	if err != nil {
		return err
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	// Register each custom runtime
	for name, def := range cfg.Runtimes {
		// Get the base runtime
		baseRuntime, err := runtime.Get(def.Type)
		if err != nil {
			continue // Skip if base runtime not found
		}

		// For now, custom runtimes are just aliases with default options
		// In the future, we could create wrapper runtimes with custom behavior
		if err := runtime.Register(name, baseRuntime, nil); err != nil {
			// Ignore registration errors (e.g., already registered)
			continue
		}
	}

	return nil
}
