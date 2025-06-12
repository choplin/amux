package ui

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      OutputFormat
		wantError bool
	}{
		{
			name:  "empty string defaults to pretty",
			input: "",
			want:  FormatPretty,
		},
		{
			name:  "pretty format",
			input: "pretty",
			want:  FormatPretty,
		},
		{
			name:  "json format",
			input: "json",
			want:  FormatJSON,
		},
		{
			name:      "invalid format",
			input:     "xml",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFormat(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ParseFormat() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONFormatter_Output(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	formatter := NewJSONFormatter()

	// Test data
	testData := map[string]string{
		"name":    "test",
		"version": "1.0.0",
	}

	// Write output
	err := formatter.Output(testData)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}

	// Restore stdout and read output
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify JSON output
	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if result["name"] != "test" || result["version"] != "1.0.0" {
		t.Errorf("Unexpected JSON output: %v", result)
	}
}

func TestJSONFormatter_IsJSON(t *testing.T) {
	jsonFormatter := NewJSONFormatter()
	if !jsonFormatter.IsJSON() {
		t.Error("JSONFormatter.IsJSON() should return true")
	}

	prettyFormatter := NewPrettyFormatter()
	if prettyFormatter.IsJSON() {
		t.Error("PrettyFormatter.IsJSON() should return false")
	}
}

func TestSetGlobalFormatter(t *testing.T) {
	// Save original formatter
	original := GlobalFormatter
	defer func() { GlobalFormatter = original }()

	// Test setting JSON formatter
	err := SetGlobalFormatter(FormatJSON)
	if err != nil {
		t.Fatalf("SetGlobalFormatter(FormatJSON) error = %v", err)
	}
	if !GlobalFormatter.IsJSON() {
		t.Error("GlobalFormatter should be JSON formatter")
	}

	// Test setting pretty formatter
	err = SetGlobalFormatter(FormatPretty)
	if err != nil {
		t.Fatalf("SetGlobalFormatter(FormatPretty) error = %v", err)
	}
	if GlobalFormatter.IsJSON() {
		t.Error("GlobalFormatter should be pretty formatter")
	}
}

func TestWithFormatter(t *testing.T) {
	// Save original formatter
	original := GlobalFormatter
	defer func() { GlobalFormatter = original }()

	// Set initial formatter to pretty
	GlobalFormatter = NewPrettyFormatter()

	// Use WithFormatter to temporarily switch to JSON
	executed := false
	err := WithFormatter(FormatJSON, func() error {
		executed = true
		if !GlobalFormatter.IsJSON() {
			t.Error("GlobalFormatter should be JSON within function")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithFormatter() error = %v", err)
	}

	if !executed {
		t.Error("Function was not executed")
	}

	// Verify formatter was restored
	if GlobalFormatter.IsJSON() {
		t.Error("GlobalFormatter should be restored to pretty formatter")
	}
}
