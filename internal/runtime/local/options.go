package local

// Options contains configuration specific to the local runtime
type Options struct {
	// Shell specifies the shell to use for command execution
	// If empty, defaults to user's shell or /bin/sh
	Shell string

	// Detach indicates whether the process should run in the background
	// If true, the process will be detached from the parent and run independently
	// If false (default), the process runs in the foreground and waits for completion
	Detach bool
}

// IsRuntimeOptions implements the runtime.RuntimeOptions interface
func (o Options) IsRuntimeOptions() {}
