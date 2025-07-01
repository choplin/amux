package session

import (
	"fmt"
	"os"

	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/session"
	"github.com/aki/amux/internal/task"
)

// setupManagers finds the project root and creates necessary managers
func setupManagers() (*config.Manager, session.Manager, error) {
	// FindProjectRoot searches up from current directory
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		cwd, _ := os.Getwd()
		return nil, nil, fmt.Errorf("not in an amux project (searched from %s up to /). Run 'amux init' to create a project", cwd)
	}

	configMgr := config.NewManager(projectRoot)
	if !configMgr.IsInitialized() {
		return nil, nil, fmt.Errorf("amux not initialized. Run 'amux init' first")
	}

	sessionMgr := getSessionManager(configMgr)
	return configMgr, sessionMgr, nil
}

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
