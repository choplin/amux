package tmux

import (
	"testing"
)

func TestMockAdapter_CreateSessionWithOptions(t *testing.T) {
	adapter := NewMockAdapter()

	tests := []struct {
		name    string
		opts    CreateSessionOptions
		wantErr bool
	}{
		{
			name: "basic session",
			opts: CreateSessionOptions{
				SessionName: "test-session",
				WorkDir:     "/tmp/test",
			},
			wantErr: false,
		},
		{
			name: "session with window name",
			opts: CreateSessionOptions{
				SessionName: "test-session-window",
				WorkDir:     "/tmp/test",
				WindowName:  "dev",
			},
			wantErr: false,
		},
		{
			name: "session with all options",
			opts: CreateSessionOptions{
				SessionName: "test-session-full",
				WorkDir:     "/tmp/test",
				WindowName:  "workspace",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.CreateSessionWithOptions(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSessionWithOptions() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify session was created
				if !adapter.SessionExists(tt.opts.SessionName) {
					t.Errorf("Expected session %s to exist", tt.opts.SessionName)
				}

				// Verify session properties
				sessions := adapter.GetSessions()
				session, exists := sessions[tt.opts.SessionName]
				if !exists {
					t.Fatalf("Session %s not found in sessions map", tt.opts.SessionName)
				}

				if session.windowName != tt.opts.WindowName {
					t.Errorf("Expected window name %s, got %s", tt.opts.WindowName, session.windowName)
				}

				if session.workDir != tt.opts.WorkDir {
					t.Errorf("Expected work dir %s, got %s", tt.opts.WorkDir, session.workDir)
				}
			}
		})
	}
}

func TestMockAdapter_CreateSessionBackwardCompatibility(t *testing.T) {
	adapter := NewMockAdapter()

	// Test that CreateSession still works
	err := adapter.CreateSession("test-legacy", "/tmp/legacy")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if !adapter.SessionExists("test-legacy") {
		t.Error("Expected legacy session to exist")
	}

	sessions := adapter.GetSessions()
	session, exists := sessions["test-legacy"]
	if !exists {
		t.Fatal("Legacy session not found")
	}

	if session.workDir != "/tmp/legacy" {
		t.Errorf("Expected work dir /tmp/legacy, got %s", session.workDir)
	}

	// windowName should be empty for legacy sessions

	if session.windowName != "" {
		t.Errorf("Expected empty window name, got %s", session.windowName)
	}
}
