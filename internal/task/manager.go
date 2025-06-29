package task

import (
	"fmt"
	"sync"
)

// Manager manages tasks and their lifecycle
type Manager struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

// NewManager creates a new task manager
func NewManager() *Manager {
	return &Manager{
		tasks: make(map[string]*Task),
	}
}

// LoadTasks loads tasks into the manager
func (m *Manager) LoadTasks(tasks []*Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing tasks
	m.tasks = make(map[string]*Task)

	// Load and validate each task
	for _, task := range tasks {
		if err := task.Validate(); err != nil {
			return fmt.Errorf("invalid task %s: %w", task.Name, err)
		}

		if _, exists := m.tasks[task.Name]; exists {
			return fmt.Errorf("duplicate task name: %s", task.Name)
		}

		m.tasks[task.Name] = task
	}

	// Validate dependencies
	for _, task := range m.tasks {
		for _, dep := range task.DependsOn {
			if _, exists := m.tasks[dep]; !exists {
				return fmt.Errorf("task %s depends on non-existent task: %s", task.Name, dep)
			}
		}
	}

	return nil
}

// GetTask retrieves a task by name
func (m *Manager) GetTask(name string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[name]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", name)
	}

	return task, nil
}

// ListTasks returns all tasks
func (m *Manager) ListTasks() []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetTaskNames returns all task names
func (m *Manager) GetTaskNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.tasks))
	for name := range m.tasks {
		names = append(names, name)
	}

	return names
}

// HasTask checks if a task exists
func (m *Manager) HasTask(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.tasks[name]
	return exists
}
