package session

import (
	"context"
)

// Store provides persistent storage for sessions
type Store interface {
	// Save persists a session
	Save(ctx context.Context, session *Session) error

	// Load retrieves a session by ID
	Load(ctx context.Context, id string) (*Session, error)

	// List returns all sessions, optionally filtered by workspace
	List(ctx context.Context, workspaceID string) ([]*Session, error)

	// Remove deletes a session
	Remove(ctx context.Context, id string) error

	// GetLogs retrieves logs for a session
	GetLogs(ctx context.Context, id string) (LogReader, error)
}
