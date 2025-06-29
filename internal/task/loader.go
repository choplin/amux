package task

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadFromYAML loads tasks from a YAML file
func LoadFromYAML(path string) ([]*Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	var tasks []*Task
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse task file: %w", err)
	}

	return tasks, nil
}

// LoadFromDirectory loads all task files from a directory
func LoadFromDirectory(dir string) ([]*Task, error) {
	var allTasks []*Task

	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, return empty list
			return allTasks, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	// Walk through all YAML files in the directory
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process YAML files
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		// Load tasks from file
		tasks, err := LoadFromYAML(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		allTasks = append(allTasks, tasks...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return allTasks, nil
}
