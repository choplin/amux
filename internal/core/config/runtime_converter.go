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
	// First check if we have explicit runtime options
	if opts := a.GetRuntimeOptions(); opts != nil {
		// If it's already a RuntimeOptions, return as-is
		if runtimeOpts, ok := opts.(runtime.RuntimeOptions); ok {
			return runtimeOpts
		}

		// Handle TmuxParams conversion for backward compatibility
		if a.Type == AgentTypeTmux {
			if tmuxParams, ok := opts.(*TmuxParams); ok {
				return convertTmuxParamsToOptions(tmuxParams)
			}
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

// convertTmuxParamsToOptions converts legacy TmuxParams to runtime TmuxOptions
func convertTmuxParamsToOptions(params *TmuxParams) tmux.TmuxOptions {
	opts := tmux.TmuxOptions{
		WindowName:    params.WindowName,
		CaptureOutput: true, // Default to capturing output
	}

	// Generate session name if needed
	if params.WindowName != "" {
		opts.SessionName = "amux-" + params.WindowName
	}

	// Map detached/autoAttach behavior
	// Note: These concepts don't directly map to the new runtime model
	// as attachment is handled separately from execution

	return opts
}
