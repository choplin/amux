package task

import (
	"fmt"
	"os"
	"strings"
)

// ParseCommand parses a command template with variable substitution
func ParseCommand(template string, vars map[string]string) ([]string, error) {
	// Simple implementation - split by spaces and substitute variables
	// TODO: Implement proper shell-like parsing with quotes

	// Expand environment variables
	expanded := os.Expand(template, func(key string) string {
		if val, ok := vars[key]; ok {
			return val
		}
		return os.Getenv(key)
	})

	// Split into arguments (simple implementation)
	parts := strings.Fields(expanded)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	return parts, nil
}
