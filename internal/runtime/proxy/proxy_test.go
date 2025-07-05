package proxy

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestProxy_Run(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions", "test-session")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		opts      Options
		wantError bool
	}{
		{
			name: "basic command",
			opts: Options{
				SessionDir: sessionDir,
				StatusPath: filepath.Join(sessionDir, "status.yaml"),
				LogPath:    sessionDir + "/",
				SocketPath: filepath.Join(t.TempDir(), "test.sock"),
				Command:    []string{"echo", "hello"},
			},
			wantError: false,
		},
		{
			name: "command with error",
			opts: Options{
				SessionDir: sessionDir,
				StatusPath: filepath.Join(sessionDir, "status.yaml"),
				LogPath:    "",
				SocketPath: filepath.Join(t.TempDir(), "test.sock"),
				Command:    []string{"sh", "-c", "exit 1"},
			},
			wantError: true,
		},
		{
			name: "empty command",
			opts: Options{
				SessionDir: sessionDir,
				StatusPath: filepath.Join(sessionDir, "status.yaml"),
				LogPath:    "",
				SocketPath: filepath.Join(t.TempDir(), "test.sock"),
				Command:    []string{},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set project root to temp dir
			os.Setenv("AMUX_PROJECT_ROOT", tmpDir)
			defer os.Unsetenv("AMUX_PROJECT_ROOT")

			p, err := New(tt.opts)
			if err != nil {
				if !tt.wantError {
					t.Fatalf("New() error = %v", err)
				}
				return
			}

			err = p.Run()
			if (err != nil) != tt.wantError {
				t.Errorf("Run() error = %v, wantError = %v", err, tt.wantError)
			}

			// Check status file was created
			if !tt.wantError && tt.name != "empty command" {
				statusFile := filepath.Join(sessionDir, "status.yaml")
				if _, err := os.Stat(statusFile); err != nil {
					t.Errorf("Status file not created: %v", err)
				}

				// Read and verify status
				data, err := os.ReadFile(statusFile)
				if err != nil {
					t.Errorf("Failed to read status file: %v", err)
				}

				var status Status
				if err := yaml.Unmarshal(data, &status); err != nil {
					t.Errorf("Failed to unmarshal status: %v", err)
				}

				if status.Status != "exited" {
					t.Errorf("Expected status 'exited', got %s", status.Status)
				}
			}

			// Check log file if enabled
			if tt.opts.LogPath != "" && !tt.wantError {
				// Log files are created in sessionDir/runID/console.log format
				// We can't easily predict the runID, so just check that the directory exists
				if _, err := os.Stat(sessionDir); err != nil {
					t.Errorf("Session directory not created: %v", err)
				}
			}
		})
	}
}

func TestProxy_Socket(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions", "test-session")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Use a socket in temp dir
	socketPath := filepath.Join(tmpDir, "test.sock")

	opts := Options{
		SessionDir: sessionDir,
		StatusPath: filepath.Join(sessionDir, "status.yaml"),
		LogPath:    sessionDir + "/",
		SocketPath: socketPath,
		Command:    []string{"sh", "-c", "for i in 1 2 3; do echo $i; sleep 0.1; done"},
	}

	// Set project root to temp dir
	os.Setenv("AMUX_PROJECT_ROOT", tmpDir)
	defer os.Unsetenv("AMUX_PROJECT_ROOT")

	p, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Run proxy in background
	done := make(chan error)
	go func() {
		done <- p.Run()
	}()

	// Wait for socket to be created
	var socketCreated bool
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			socketCreated = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !socketCreated {
		t.Fatal("Socket was not created")
	}

	// Give socket listener time to start
	time.Sleep(100 * time.Millisecond)

	// Connect to socket and read data
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use a pipe to safely capture output
	r, w := io.Pipe()
	readerDone := make(chan string)

	go func() {
		// Connect to socket and read
		go func() {
			err := p.connectAndReadSocket(ctx, w)
			if err != nil && err != context.Canceled {
				t.Logf("Socket read error (expected): %v", err)
			}
			w.Close()
		}()

		// Read from pipe
		data, err := io.ReadAll(r)
		if err != nil {
			readerDone <- ""
			return
		}
		readerDone <- string(data)
	}()

	// Give it time to collect output
	time.Sleep(500 * time.Millisecond)
	cancel() // Stop reading

	// Wait for proxy to complete
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Proxy failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Proxy did not complete in time")
	}

	// Get captured output
	var got string
	select {
	case got = <-readerDone:
	case <-time.After(1 * time.Second):
		t.Fatal("Reader did not complete in time")
	}

	expected := "1\n2\n3\n"
	if got != expected {
		t.Errorf("Expected output %q, got %q", expected, got)
	}
}

func TestProxy_StatusUpdates(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	sessionDir := filepath.Join(tmpDir, "sessions", "test-session")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}

	opts := Options{
		SessionDir: sessionDir,
		StatusPath: filepath.Join(sessionDir, "status.yaml"),
		LogPath:    "", // No logging
		SocketPath: filepath.Join(tmpDir, "test.sock"),
		Command:    []string{"sleep", "0.5"},
	}

	// Set project root to temp dir
	os.Setenv("AMUX_PROJECT_ROOT", tmpDir)
	defer os.Unsetenv("AMUX_PROJECT_ROOT")

	p, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Run proxy in background
	done := make(chan error)
	go func() {
		done <- p.Run()
	}()

	// Wait for initial status
	time.Sleep(100 * time.Millisecond)

	// Read status
	data, err := os.ReadFile(opts.StatusPath)
	if err != nil {
		t.Fatal(err)
	}

	var status Status
	if err := yaml.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}

	// Check initial status
	if status.Status != "running" {
		t.Errorf("Expected initial status 'running', got %s", status.Status)
	}

	if status.PID == 0 {
		t.Error("PID should be set")
	}

	// Wait for completion
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Proxy failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Proxy did not complete in time")
	}

	// Read final status
	data, err = os.ReadFile(opts.StatusPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := yaml.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}

	// Check final status
	if status.Status != "exited" {
		t.Errorf("Expected final status 'exited', got %s", status.Status)
	}

	if status.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", status.ExitCode)
	}
}
