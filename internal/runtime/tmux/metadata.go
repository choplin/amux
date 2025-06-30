package tmux

import (
	"github.com/aki/amux/internal/runtime"
)

// Metadata represents tmux runtime specific metadata
type Metadata struct {
	// SessionName is the tmux session name
	SessionName string `json:"session_name" yaml:"session_name"`

	// WindowName is the tmux window name
	WindowName string `json:"window_name" yaml:"window_name"`

	// PaneID is the tmux pane ID (e.g., "%0")
	PaneID string `json:"pane_id,omitempty" yaml:"pane_id,omitempty"`
}

// ToMap converts the metadata to a map for serialization
func (m *Metadata) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"session_name": m.SessionName,
		"window_name":  m.WindowName,
	}

	if m.PaneID != "" {
		result["pane_id"] = m.PaneID
	}

	return result
}

// RuntimeType returns the runtime type this metadata belongs to
func (m *Metadata) RuntimeType() string {
	return "tmux"
}

// MetadataFromMap creates TmuxMetadata from a map
func MetadataFromMap(data map[string]interface{}) (*Metadata, error) {
	m := &Metadata{}

	if sessionName, ok := data["session_name"].(string); ok {
		m.SessionName = sessionName
	}

	if windowName, ok := data["window_name"].(string); ok {
		m.WindowName = windowName
	}

	if paneID, ok := data["pane_id"].(string); ok {
		m.PaneID = paneID
	}

	return m, nil
}

// compile-time check
var _ runtime.Metadata = (*Metadata)(nil)
