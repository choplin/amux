package runtime

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRuntime is a mock implementation of Runtime for testing
type mockRuntime struct {
	name string
}

func (m *mockRuntime) Type() string {
	return m.name
}

func (m *mockRuntime) Execute(ctx context.Context, spec ExecutionSpec) (Process, error) {
	return &mockProcess{}, nil
}

func (m *mockRuntime) Find(ctx context.Context, id string) (Process, error) {
	return nil, ErrProcessNotFound
}

func (m *mockRuntime) List(ctx context.Context) ([]Process, error) {
	return nil, nil
}

func (m *mockRuntime) Validate() error {
	return nil
}

// mockProcess is a mock implementation of Process
type mockProcess struct{}

func (m *mockProcess) ID() string                     { return "mock-id" }
func (m *mockProcess) State() ProcessState            { return StateRunning }
func (m *mockProcess) Wait(ctx context.Context) error { return nil }
func (m *mockProcess) Stop(ctx context.Context) error { return nil }
func (m *mockProcess) Kill(ctx context.Context) error { return nil }
func (m *mockProcess) Output() (io.Reader, io.Reader) { return nil, nil }
func (m *mockProcess) ExitCode() (int, error)         { return 0, nil }
func (m *mockProcess) StartTime() time.Time           { return time.Now() }
func (m *mockProcess) Metadata() Metadata             { return nil }

// mockOptions is a mock implementation of RuntimeOptions
type mockOptions struct {
	value string
}

func (m mockOptions) IsRuntimeOptions() {}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name        string
		runtimeName string
		runtime     Runtime
		options     RuntimeOptions
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid registration",
			runtimeName: "test",
			runtime:     &mockRuntime{name: "test"},
			options:     mockOptions{value: "default"},
			wantErr:     false,
		},
		{
			name:        "empty name",
			runtimeName: "",
			runtime:     &mockRuntime{name: "test"},
			wantErr:     true,
			errContains: "name cannot be empty",
		},
		{
			name:        "nil runtime",
			runtimeName: "test",
			runtime:     nil,
			wantErr:     true,
			errContains: "runtime cannot be nil",
		},
		{
			name:        "duplicate registration",
			runtimeName: "duplicate",
			runtime:     &mockRuntime{name: "duplicate"},
			wantErr:     false,
		},
		{
			name:        "duplicate registration error",
			runtimeName: "duplicate",
			runtime:     &mockRuntime{name: "duplicate2"},
			wantErr:     true,
			errContains: "already registered",
		},
		{
			name:        "nil options allowed",
			runtimeName: "no-opts",
			runtime:     &mockRuntime{name: "no-opts"},
			options:     nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Register(tt.runtimeName, tt.runtime, tt.options)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	// Register some runtimes
	rt1 := &mockRuntime{name: "runtime1"}
	rt2 := &mockRuntime{name: "runtime2"}
	require.NoError(t, r.Register("runtime1", rt1, nil))
	require.NoError(t, r.Register("runtime2", rt2, nil))

	tests := []struct {
		name    string
		runtime string
		want    Runtime
		wantErr bool
	}{
		{
			name:    "existing runtime 1",
			runtime: "runtime1",
			want:    rt1,
			wantErr: false,
		},
		{
			name:    "existing runtime 2",
			runtime: "runtime2",
			want:    rt2,
			wantErr: false,
		},
		{
			name:    "non-existent runtime",
			runtime: "unknown",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Get(tt.runtime)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRegistry_GetDefaultOptions(t *testing.T) {
	r := NewRegistry()

	// Register runtimes with and without options
	opts1 := mockOptions{value: "opts1"}
	opts2 := mockOptions{value: "opts2"}
	require.NoError(t, r.Register("with-opts1", &mockRuntime{}, opts1))
	require.NoError(t, r.Register("with-opts2", &mockRuntime{}, opts2))
	require.NoError(t, r.Register("no-opts", &mockRuntime{}, nil))

	tests := []struct {
		name    string
		runtime string
		want    RuntimeOptions
		wantErr bool
	}{
		{
			name:    "runtime with options 1",
			runtime: "with-opts1",
			want:    opts1,
			wantErr: false,
		},
		{
			name:    "runtime with options 2",
			runtime: "with-opts2",
			want:    opts2,
			wantErr: false,
		},
		{
			name:    "runtime without options",
			runtime: "no-opts",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "non-existent runtime",
			runtime: "unknown",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.GetDefaultOptions(tt.runtime)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	// Initially empty
	names := r.List()
	assert.Empty(t, names)

	// Register some runtimes
	require.NoError(t, r.Register("runtime1", &mockRuntime{}, nil))
	require.NoError(t, r.Register("runtime2", &mockRuntime{}, nil))
	require.NoError(t, r.Register("runtime3", &mockRuntime{}, nil))

	// Should list all
	names = r.List()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "runtime1")
	assert.Contains(t, names, "runtime2")
	assert.Contains(t, names, "runtime3")
}

func TestRegistry_Has(t *testing.T) {
	r := NewRegistry()

	// Register a runtime
	require.NoError(t, r.Register("exists", &mockRuntime{}, nil))

	assert.True(t, r.Has("exists"))
	assert.False(t, r.Has("not-exists"))
}

func TestRegistry_Clear(t *testing.T) {
	r := NewRegistry()

	// Register some runtimes
	require.NoError(t, r.Register("runtime1", &mockRuntime{}, mockOptions{value: "opt1"}))
	require.NoError(t, r.Register("runtime2", &mockRuntime{}, mockOptions{value: "opt2"}))

	// Verify they exist
	assert.Len(t, r.List(), 2)
	assert.True(t, r.Has("runtime1"))
	assert.True(t, r.Has("runtime2"))

	// Clear
	r.Clear()

	// Should be empty
	assert.Empty(t, r.List())
	assert.False(t, r.Has("runtime1"))
	assert.False(t, r.Has("runtime2"))

	// Should not be able to get options
	_, err := r.GetDefaultOptions("runtime1")
	assert.Error(t, err)
}

func TestRegistry_Concurrent(t *testing.T) {
	r := NewRegistry()

	// Register initial runtime
	require.NoError(t, r.Register("base", &mockRuntime{}, nil))

	// Concurrent operations
	done := make(chan bool)
	errors := make(chan error, 100)

	// Writers
	for i := 0; i < 10; i++ {
		go func(id int) {
			rt := &mockRuntime{name: fmt.Sprintf("runtime%d", id)}
			if err := r.Register(fmt.Sprintf("runtime%d", id), rt, nil); err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Readers
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = r.Get("base")
			_ = r.List()
			_ = r.Has("base")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}

	// Verify all runtimes were registered
	names := r.List()
	assert.Contains(t, names, "base")
	for i := 0; i < 10; i++ {
		assert.Contains(t, names, fmt.Sprintf("runtime%d", i))
	}
}

func TestDefaultRegistry(t *testing.T) {
	// Clear any existing registrations
	defaultRegistry.Clear()

	// Test Register
	rt := &mockRuntime{name: "default-test"}
	opts := mockOptions{value: "default-opts"}
	err := Register("test", rt, opts)
	require.NoError(t, err)

	// Test Get
	got, err := Get("test")
	require.NoError(t, err)
	assert.Equal(t, rt, got)

	// Test List
	names := List()
	assert.Contains(t, names, "test")

	// Test GetDefaultOptions
	gotOpts, err := GetDefaultOptions("test")
	require.NoError(t, err)
	assert.Equal(t, opts, gotOpts)

	// Clean up
	defaultRegistry.Clear()
}
