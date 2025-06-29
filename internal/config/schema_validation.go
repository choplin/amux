package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"

	"github.com/aki/amux/internal/task"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

//go:embed schemas/config.schema.json
var configSchema []byte

// schemaCompiler is cached to avoid recompiling the schema on every validation
var schemaCompiler *jsonschema.Schema

// compileSchema compiles the embedded JSON schema
func compileSchema() (*jsonschema.Schema, error) {
	if schemaCompiler != nil {
		return schemaCompiler, nil
	}

	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020

	// Add the schema resource
	if err := compiler.AddResource("config.schema.json", bytes.NewReader(configSchema)); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	// Compile the schema
	schema, err := compiler.Compile("config.schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	schemaCompiler = schema
	return schema, nil
}

// ValidateYAML validates YAML content against the JSON schema
func ValidateYAML(yamlContent []byte) error {
	// Compile schema
	schema, err := compileSchema()
	if err != nil {
		return err
	}

	// Parse YAML to generic interface for validation
	var data interface{}
	if err := yaml.Unmarshal(yamlContent, &data); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate against schema
	if err := schema.Validate(data); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	return nil
}

// LoadWithValidation loads and validates configuration
func LoadWithValidation(path string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("configuration file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Validate YAML against schema
	if err := ValidateYAML(data); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Unmarshal to Config struct
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate tasks if present
	if len(cfg.Tasks) > 0 {
		validator := task.NewValidator()
		if err := validator.ValidateTaskList(cfg.Tasks); err != nil {
			return nil, fmt.Errorf("task validation failed: %w", err)
		}
	}

	return &cfg, nil
}

// ValidateFile validates a configuration file without loading it
func ValidateFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("configuration file not found: %s", path)
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return ValidateYAML(data)
}
