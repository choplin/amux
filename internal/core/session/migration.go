package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// LegacyStatusState represents the old StatusState structure for migration
type LegacyStatusState struct {
	Status          string    `yaml:"status"`
	StatusChangedAt time.Time `yaml:"statusChangedAt"`
	LastOutputHash  uint32    `yaml:"lastOutputHash,omitempty"`
	LastOutputTime  time.Time `yaml:"lastOutputTime,omitempty"`
	LastStatusCheck time.Time `yaml:"lastStatusCheck,omitempty"`
}

// LegacyInfo represents the old Info structure with StatusState
type LegacyInfo struct {
	ID            string            `yaml:"id"`
	Type          Type              `yaml:"type"`
	WorkspaceID   string            `yaml:"workspace_id"`
	AgentID       string            `yaml:"agent_id"`
	StatusState   LegacyStatusState `yaml:"statusState"`
	Command       string            `yaml:"command"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	InitialPrompt string            `yaml:"initial_prompt,omitempty"`
	PID           int               `yaml:"pid,omitempty"`
	TmuxSession   string            `yaml:"tmux_session,omitempty"`
	CreatedAt     time.Time         `yaml:"created_at"`
	StartedAt     *time.Time        `yaml:"started_at,omitempty"`
	StoppedAt     *time.Time        `yaml:"stopped_at,omitempty"`
	Error         string            `yaml:"error,omitempty"`
	StoragePath   string            `yaml:"storage_path,omitempty"`
	StateDir      string            `yaml:"state_dir,omitempty"`
	Name          string            `yaml:"name,omitempty"`
	Description   string            `yaml:"description,omitempty"`
}

// MigrateSessionInfo attempts to migrate old session format to new format
func MigrateSessionInfo(sessionFile string) error {
	// Try to read as legacy format first
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return err
	}

	// First, try to unmarshal as current format to see if migration is needed
	var currentInfo Info
	if err := yaml.Unmarshal(data, &currentInfo); err == nil {
		// Check if it's already in new format (has ActivityTracking)
		if currentInfo.ActivityTracking.LastOutputTime.IsZero() && currentInfo.ActivityTracking.LastOutputHash == 0 {
			// Might be old format, let's check
			var legacy LegacyInfo
			if err := yaml.Unmarshal(data, &legacy); err == nil && !legacy.StatusState.StatusChangedAt.IsZero() {
				// It's old format, migrate it
				return migrateFile(sessionFile, &legacy)
			}
		}
		// Already in new format or empty, no migration needed
		return nil
	}

	// Failed to unmarshal, try as legacy
	var legacy LegacyInfo
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		// Can't parse as either format
		return err
	}

	return migrateFile(sessionFile, &legacy)
}

func migrateFile(sessionFile string, legacy *LegacyInfo) error {
	// Convert to new format
	info := &Info{
		ID:          legacy.ID,
		Type:        legacy.Type,
		WorkspaceID: legacy.WorkspaceID,
		AgentID:     legacy.AgentID,
		ActivityTracking: ActivityTracking{
			LastOutputHash:  legacy.StatusState.LastOutputHash,
			LastOutputTime:  legacy.StatusState.LastOutputTime,
			LastStatusCheck: legacy.StatusState.LastStatusCheck,
		},
		Command:       legacy.Command,
		Environment:   legacy.Environment,
		InitialPrompt: legacy.InitialPrompt,
		PID:           legacy.PID,
		TmuxSession:   legacy.TmuxSession,
		CreatedAt:     legacy.CreatedAt,
		StartedAt:     legacy.StartedAt,
		StoppedAt:     legacy.StoppedAt,
		Error:         legacy.Error,
		StoragePath:   legacy.StoragePath,
		StateDir:      legacy.StateDir,
		Name:          legacy.Name,
		Description:   legacy.Description,
	}

	// If StateDir is not set, construct it from session ID
	if info.StateDir == "" && info.StoragePath != "" {
		// Extract session ID from storage path
		// StoragePath is typically: {sessionsDir}/{sessionID}/storage
		sessionDir := filepath.Dir(info.StoragePath)
		info.StateDir = filepath.Join(sessionDir, "state")
	}

	// Migrate state to state.json if needed
	if info.StateDir != "" && legacy.StatusState.Status != "" {
		stateFile := filepath.Join(info.StateDir, "state.json")
		if _, err := os.Stat(stateFile); os.IsNotExist(err) {
			// Create state directory if needed
			if err := os.MkdirAll(info.StateDir, 0o755); err == nil {
				// Write state file
				stateData := map[string]interface{}{
					"status":            legacy.StatusState.Status,
					"status_changed_at": legacy.StatusState.StatusChangedAt,
				}
				if data, err := json.MarshalIndent(stateData, "", "  "); err == nil {
					_ = os.WriteFile(stateFile, data, 0o644)
				}
			}
		}
	}

	// Write back in new format
	data, err := yaml.Marshal(info)
	if err != nil {
		return err
	}

	// Write to temporary file first
	tmpFile := sessionFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpFile, sessionFile)
}
