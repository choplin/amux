package ui

import (
	"encoding/json"
	"fmt"
	"os"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	// FormatPretty represents human-readable output format
	FormatPretty OutputFormat = "pretty"
	// FormatJSON represents JSON output format
	FormatJSON OutputFormat = "json"
)

// ParseFormat converts a string to OutputFormat
func ParseFormat(s string) (OutputFormat, error) {
	switch s {
	case "pretty", "":
		return FormatPretty, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", s)
	}
}

// Formatter is the interface for output formatting
type Formatter interface {
	// Output formats and displays any data
	Output(data interface{}) error

	// OutputError formats and displays an error
	OutputError(err error) error

	// IsJSON returns true if this formatter outputs JSON
	IsJSON() bool
}

// prettyFormatter implements Formatter for human-readable output
type prettyFormatter struct{}

// NewPrettyFormatter creates a new pretty formatter
func NewPrettyFormatter() Formatter {
	return &prettyFormatter{}
}

func (f *prettyFormatter) Output(data interface{}) error {
	// For pretty format, we expect data to be already formatted
	// This is just a passthrough that prints the data
	if str, ok := data.(string); ok {
		fmt.Print(str)
		return nil
	}

	// For other types, use default formatting
	fmt.Println(data)
	return nil
}

func (f *prettyFormatter) OutputError(err error) error {
	// Pretty formatter outputs errors to stderr with formatting
	fmt.Fprintf(os.Stderr, "%s %s\n", ErrorIcon, ErrorStyle.Render(err.Error()))
	return nil
}

func (f *prettyFormatter) IsJSON() bool {
	return false
}

// jsonFormatter implements Formatter for JSON output
type jsonFormatter struct {
	encoder *json.Encoder
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() Formatter {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return &jsonFormatter{encoder: encoder}
}

func (f *jsonFormatter) Output(data interface{}) error {
	return f.encoder.Encode(data)
}

func (f *jsonFormatter) OutputError(err error) error {
	// For JSON format, we still output errors to stderr as plain text
	// This maintains compatibility with scripts that expect errors on stderr
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	return nil
}

func (f *jsonFormatter) IsJSON() bool {
	return true
}

// GlobalFormatter is the global formatter instance
var GlobalFormatter Formatter = NewPrettyFormatter()

// SetGlobalFormatter sets the global formatter
func SetGlobalFormatter(format OutputFormat) error {
	switch format {
	case FormatPretty:
		GlobalFormatter = NewPrettyFormatter()
	case FormatJSON:
		GlobalFormatter = NewJSONFormatter()
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	return nil
}

// WithFormatter temporarily sets a formatter for a function execution
func WithFormatter(format OutputFormat, fn func() error) error {
	oldFormatter := GlobalFormatter
	defer func() { GlobalFormatter = oldFormatter }()

	if err := SetGlobalFormatter(format); err != nil {
		return err
	}

	return fn()
}
