package local

// Options contains configuration specific to the local runtime
type Options struct {
	// Shell specifies the shell to use for command execution
	// If empty, defaults to user's shell or /bin/sh
	Shell string
}

// IsRuntimeOptions implements the runtime.RuntimeOptions interface
func (o Options) IsRuntimeOptions() {}
