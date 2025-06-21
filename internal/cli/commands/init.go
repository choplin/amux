// Package commands provides CLI command implementations for amux.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/git"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Amux in the current project",
	Long:  "Initialize Amux configuration in the current project directory",
	RunE:  runInit,
}

var forceInit bool

func init() {
	initCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "Force initialization, overwriting existing configuration")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if it's a git repository
	gitOps := git.NewOperations(cwd)
	if !gitOps.IsGitRepository() {
		return fmt.Errorf("not a git repository. Amux requires a git repository")
	}

	// Create configuration manager
	configManager := config.NewManager(cwd)

	// Check if already initialized
	if configManager.IsInitialized() && !forceInit {
		return fmt.Errorf("amux already initialized. Use --force to reinitialize")
	}

	// Create default configuration
	cfg := config.DefaultConfig()

	// Save configuration
	if err := configManager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Create workspaces directory
	workspacesDir := configManager.GetWorkspacesDir()
	if err := os.MkdirAll(workspacesDir, 0o755); err != nil {
		return fmt.Errorf("failed to create workspaces directory: %w", err)
	}

	// Ask user if they want to add .amux to .gitignore
	if shouldUpdateGitignore(cwd) {
		if ui.ConfirmWithDefault("Add .amux/ to .gitignore", false) {
			if err := addToGitignore(cwd); err != nil {
				ui.Warning("Failed to update .gitignore: %v", err)
			} else {
				ui.OutputLine("Added .amux/ to .gitignore")
			}
		}
	}

	ui.Success("Amux initialized successfully in %s", cwd)
	ui.PrintKeyValue("Configuration", filepath.Join(config.AmuxDir, config.ConfigFile))
	ui.OutputLine("\nRun 'amux mcp' to start the MCP server")

	return nil
}

func shouldUpdateGitignore(projectRoot string) bool {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	// Read existing .gitignore
	content := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		content = string(data)
	}

	// Check if .amux is already ignored
	return !strings.Contains(content, ".amux")
}

func addToGitignore(projectRoot string) error {
	gitignorePath := filepath.Join(projectRoot, ".gitignore")

	// Read existing .gitignore
	content := ""
	if data, err := os.ReadFile(gitignorePath); err == nil {
		content = string(data)
	}

	// Check if .amux is already ignored
	if strings.Contains(content, ".amux") {
		return nil
	}

	// Append .amux entries
	entries := "\n# Amux\n.amux/\n"

	file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = file.WriteString(entries)
	return err
}
