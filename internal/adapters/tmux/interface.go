package tmux

// CreateSessionOptions contains options for creating a tmux session
type CreateSessionOptions struct {
	SessionName string
	WorkDir     string
	Shell       string            // Optional: custom shell to use (e.g., /bin/zsh)
	WindowName  string            // Optional: custom window name
	Environment map[string]string // Optional: environment variables to set
}

// Adapter defines the interface for tmux operations
type Adapter interface {
	IsAvailable() bool
	CreateSession(sessionName, workDir string) error
	CreateSessionWithOptions(opts CreateSessionOptions) error
	SessionExists(sessionName string) bool
	KillSession(sessionName string) error
	SendKeys(sessionName, keys string) error
	CapturePane(sessionName string) (string, error)
	AttachSession(sessionName string) error
	ListSessions() ([]string, error)
	GetSessionPID(sessionName string) (int, error)
	SetEnvironment(sessionName string, env map[string]string) error
	ResizeWindow(sessionName string, width, height int) error
	CapturePaneWithOptions(sessionName string, lines int) (string, error)
	IsPaneDead(sessionName string) (bool, error)
}
