// Package mailbox provides file-based communication for agent sessions.
package mailbox

import (
	"time"
)

// Direction represents the direction of a message
type Direction string

const (
	// DirectionIn represents messages TO the agent
	DirectionIn Direction = "in"
	// DirectionOut represents messages FROM the agent
	DirectionOut Direction = "out"
)

// Message represents a mailbox message
type Message struct {
	// Timestamp when the message was created
	Timestamp time.Time
	// Name is the filename without timestamp prefix
	Name string
	// Direction indicates if this is an incoming or outgoing message
	Direction Direction
	// Path is the full path to the message file
	Path string
}

// Options contains options for mailbox operations
type Options struct {
	// SessionID is the session this mailbox belongs to
	SessionID string
	// Direction to filter messages (optional)
	Direction Direction
	// Limit the number of messages to retrieve (0 = all)
	Limit int
}
