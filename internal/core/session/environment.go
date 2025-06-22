package session

// GetAMUXEnvironment returns the standard AMUX environment variables for a session
func GetAMUXEnvironment(s Session) map[string]string {
	return map[string]string{
		"AMUX_WORKSPACE_ID":   s.WorkspaceID(),
		"AMUX_WORKSPACE_PATH": s.WorkspacePath(),
		"AMUX_SESSION_ID":     s.ID(),
		"AMUX_AGENT_ID":       s.AgentID(),
	}
}

// MergeEnvironment merges multiple environment maps, with later maps overriding earlier ones
func MergeEnvironment(envs ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, env := range envs {
		for k, v := range env {
			result[k] = v
		}
	}
	return result
}
