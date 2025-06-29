package session

import (
	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/session"
	"github.com/aki/amux/internal/task"
)

// getSessionManager creates a session manager for the project
func getSessionManager(configMgr *config.Manager) session.Manager {
	// Get runtimes
	runtimes := make(map[string]runtime.Runtime)
	for _, name := range runtime.List() {
		if rt, err := runtime.Get(name); err == nil {
			runtimes[name] = rt
		}
	}

	// Create task manager
	taskMgr := task.NewManager()
	// TODO: Load tasks from config

	// Create session store
	store := session.NewFileStore(configMgr.GetAmuxDir())

	// Create session manager
	return session.NewManager(store, runtimes, taskMgr)
}
