package mailbox

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Manager handles mailbox operations for agent sessions
type Manager struct {
	amuxDir string
}

// NewManager creates a new mailbox manager
func NewManager(amuxDir string) *Manager {
	return &Manager{
		amuxDir: amuxDir,
	}
}

// GetMailboxPath returns the path to a session's mailbox directory
func (m *Manager) GetMailboxPath(sessionID string) string {
	return filepath.Join(m.amuxDir, "mailbox", sessionID)
}

// Initialize creates the mailbox directory structure for a session
func (m *Manager) Initialize(sessionID string) error {
	mailboxPath := m.GetMailboxPath(sessionID)

	// Create mailbox directory
	if err := os.MkdirAll(mailboxPath, 0o755); err != nil {
		return fmt.Errorf("failed to create mailbox directory: %w", err)
	}

	// Create in and out directories
	inPath := filepath.Join(mailboxPath, string(DirectionIn))
	outPath := filepath.Join(mailboxPath, string(DirectionOut))

	if err := os.MkdirAll(inPath, 0o755); err != nil {
		return fmt.Errorf("failed to create in directory: %w", err)
	}

	if err := os.MkdirAll(outPath, 0o755); err != nil {
		return fmt.Errorf("failed to create out directory: %w", err)
	}

	// Create initial context.md file
	contextPath := filepath.Join(mailboxPath, "context.md")
	contextContent := fmt.Sprintf(`# Session Context

Session ID: %s
Created: %s

## Purpose
This file provides context for the agent session. You can update this file to provide ongoing context about the work being done.

## Communication
- Messages TO you are in the 'in/' directory
- Messages FROM you should be written to the 'out/' directory
- All messages use the format: {unix-timestamp}-{descriptive-name}.md

## Example
To send a message, create a file like:
out/1704921050-status-update.md
`, sessionID, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(contextPath, []byte(contextContent), 0o644); err != nil {
		return fmt.Errorf("failed to create context file: %w", err)
	}

	return nil
}

// SendMessage creates a new message in the mailbox
func (m *Manager) SendMessage(sessionID, name, content string) error {
	mailboxPath := m.GetMailboxPath(sessionID)
	inPath := filepath.Join(mailboxPath, string(DirectionIn))

	// Ensure the directory exists
	if err := os.MkdirAll(inPath, 0o755); err != nil {
		return fmt.Errorf("failed to create in directory: %w", err)
	}

	// Create filename with timestamp
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d-%s.md", timestamp, sanitizeFilename(name))
	filePath := filepath.Join(inPath, filename)

	// Write the message
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// ListMessages retrieves messages from the mailbox
func (m *Manager) ListMessages(opts Options) ([]Message, error) {
	mailboxPath := m.GetMailboxPath(opts.SessionID)

	var messages []Message

	// If no specific direction, check both
	directions := []Direction{DirectionIn, DirectionOut}
	if opts.Direction != "" {
		directions = []Direction{opts.Direction}
	}

	for _, dir := range directions {
		dirPath := filepath.Join(mailboxPath, string(dir))

		// Skip if directory doesn't exist
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s directory: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			// Parse timestamp from filename
			parts := strings.SplitN(entry.Name(), "-", 2)
			if len(parts) != 2 {
				continue
			}

			timestamp, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				continue
			}

			name := strings.TrimSuffix(parts[1], ".md")

			messages = append(messages, Message{
				Timestamp: time.Unix(timestamp, 0),
				Name:      name,
				Direction: dir,
				Path:      filepath.Join(dirPath, entry.Name()),
			})
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.After(messages[j].Timestamp)
	})

	// Apply limit if specified
	if opts.Limit > 0 && len(messages) > opts.Limit {
		messages = messages[:opts.Limit]
	}

	return messages, nil
}

// ReadMessage reads the content of a message
func (m *Manager) ReadMessage(message Message) (string, error) {
	content, err := os.ReadFile(message.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read message: %w", err)
	}
	return string(content), nil
}

// Clean removes the mailbox directory for a session
func (m *Manager) Clean(sessionID string) error {
	mailboxPath := m.GetMailboxPath(sessionID)
	if err := os.RemoveAll(mailboxPath); err != nil {
		return fmt.Errorf("failed to remove mailbox: %w", err)
	}
	return nil
}

// sanitizeFilename removes or replaces characters that are problematic in filenames
func sanitizeFilename(name string) string {
	// Replace spaces with hyphens
	name = strings.ReplaceAll(name, " ", "-")

	// Remove or replace other problematic characters
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "-",
		"?", "-",
		"\"", "-",
		"<", "-",
		">", "-",
		"|", "-",
		"\n", "-",
		"\r", "-",
	)
	name = replacer.Replace(name)

	// Remove leading/trailing hyphens and dots
	name = strings.Trim(name, "-.")

	// Limit length
	if len(name) > 50 {
		name = name[:50]
	}

	// Default if empty
	if name == "" {
		name = "message"
	}

	return name
}
