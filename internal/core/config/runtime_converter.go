package config

import (
	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/runtime/tmux"
)

// ToExecutionSpec converts an Agent configuration to a runtime ExecutionSpec
func (a *Agent) ToExecutionSpec() runtime.ExecutionSpec {
	spec := runtime.ExecutionSpec{
		Command:     a.GetCommand(),
		Environment: a.Environment,
		WorkingDir:  a.WorkingDir,
		Options:     a.convertRuntimeOptions(),
	}

	return spec
}

// convertRuntimeOptions converts agent params to runtime options
func (a *Agent) convertRuntimeOptions() runtime.RuntimeOptions {
	// Check if we have explicit runtime options
	if opts := a.GetRuntimeOptions(); opts != nil {
		// If it's already a RuntimeOptions, return as-is
		if runtimeOpts, ok := opts.(runtime.RuntimeOptions); ok {
			return runtimeOpts
		}
	}

	// Return default options based on runtime type
	switch a.GetRuntimeType() {
	case "tmux":
		return tmux.TmuxOptions{}
	default:
		return nil
	}
}
