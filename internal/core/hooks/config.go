// Package hooks provides functionality for managing and executing workspace lifecycle hooks
package hooks

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// HooksConfigFile is the name of the hooks configuration file
	HooksConfigFile = "hooks.yaml"
	// TrustFile is the name of the trust information file
	TrustFile = ".hooks-trust.yaml"
)

// LoadConfig loads hooks configuration from the given directory
func LoadConfig(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, HooksConfigFile)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &Config{
				Hooks: make(map[string][]Hook),
			}, nil
		}
		return nil, fmt.Errorf("failed to read hooks config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse hooks config: %w", err)
	}

	// Set defaults
	for event, hooks := range config.Hooks {
		for i := range hooks {
			if hooks[i].OnError == "" {
				hooks[i].OnError = ErrorStrategyWarn
			}
			if hooks[i].Timeout == "" {
				hooks[i].Timeout = "5m"
			}
		}
		config.Hooks[event] = hooks
	}

	return &config, nil
}

// SaveConfig saves hooks configuration to the given directory
func SaveConfig(configDir string, config *Config) error {
	configPath := filepath.Join(configDir, HooksConfigFile)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal hooks config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write hooks config: %w", err)
	}

	return nil
}

// CalculateConfigHash calculates SHA256 hash of the configuration
func CalculateConfigHash(config *Config) (string, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config for hashing: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// LoadTrustInfo loads trust information from the given directory
func LoadTrustInfo(configDir string) (*TrustInfo, error) {
	trustPath := filepath.Join(configDir, TrustFile)

	data, err := os.ReadFile(trustPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read trust info: %w", err)
	}

	var trust TrustInfo
	if err := yaml.Unmarshal(data, &trust); err != nil {
		return nil, fmt.Errorf("failed to parse trust info: %w", err)
	}

	return &trust, nil
}

// SaveTrustInfo saves trust information to the given directory
func SaveTrustInfo(configDir string, trust *TrustInfo) error {
	trustPath := filepath.Join(configDir, TrustFile)

	data, err := yaml.Marshal(trust)
	if err != nil {
		return fmt.Errorf("failed to marshal trust info: %w", err)
	}

	if err := os.WriteFile(trustPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write trust info: %w", err)
	}

	return nil
}

// IsTrusted checks if the current configuration is trusted
func IsTrusted(configDir string, config *Config) (bool, error) {
	trust, err := LoadTrustInfo(configDir)
	if err != nil {
		return false, err
	}

	if trust == nil {
		return false, nil
	}

	currentHash, err := CalculateConfigHash(config)
	if err != nil {
		return false, err
	}

	return trust.Hash == currentHash, nil
}

// GetHooksForEvent returns hooks configured for a specific event
func (c *Config) GetHooksForEvent(event Event) []Hook {
	if c.Hooks == nil {
		return nil
	}
	return c.Hooks[string(event)]
}
