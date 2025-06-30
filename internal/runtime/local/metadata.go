package local

import (
	"github.com/aki/amux/internal/runtime"
)

// Metadata represents local runtime specific metadata
type Metadata struct {
	// PID is the actual OS process ID
	PID int `json:"pid" yaml:"pid"`

	// PGID is the process group ID
	PGID int `json:"pgid,omitempty" yaml:"pgid,omitempty"`

	// Detached indicates if the process was started in detached mode
	Detached bool `json:"detached" yaml:"detached"`
}

// ToMap converts the metadata to a map for serialization
func (m *Metadata) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"pid":      m.PID,
		"detached": m.Detached,
	}

	if m.PGID != 0 {
		result["pgid"] = m.PGID
	}

	return result
}

// RuntimeType returns the runtime type this metadata belongs to
func (m *Metadata) RuntimeType() string {
	return "local"
}

// MetadataFromMap creates LocalMetadata from a map
func MetadataFromMap(data map[string]interface{}) (*Metadata, error) {
	m := &Metadata{}

	if pid, ok := data["pid"].(float64); ok {
		m.PID = int(pid)
	}

	if pgid, ok := data["pgid"].(float64); ok {
		m.PGID = int(pgid)
	}

	if detached, ok := data["detached"].(bool); ok {
		m.Detached = detached
	}

	return m, nil
}

// compile-time check
var _ runtime.Metadata = (*Metadata)(nil)
