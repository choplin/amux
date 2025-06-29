// Package config provides runtime configuration loading functionality.
package config

// RuntimeConfig represents configuration for custom runtimes
type RuntimeConfig struct {
	Runtimes map[string]RuntimeDefinition `yaml:"runtimes"`
}

// RuntimeDefinition defines a custom runtime
type RuntimeDefinition struct {
	// Type specifies which built-in runtime to extend
	Type string `yaml:"type"`

	// DefaultOptions provides default options for this runtime
	DefaultOptions map[string]interface{} `yaml:"defaultOptions,omitempty"`

	// Description provides a human-readable description
	Description string `yaml:"description,omitempty"`
}
