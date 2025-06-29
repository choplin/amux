package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
	"github.com/spf13/cobra"
)

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration in your editor",
	Long:  "Launch your default editor to edit the amux configuration file. The configuration will be validated after editing.",
	Example: `  # Edit configuration using $EDITOR
  amux config edit

  # Edit with a specific editor
  EDITOR=nano amux config edit`,
	RunE: runConfigEdit,
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	configPath := filepath.Join(projectRoot, config.AmuxDir, config.ConfigFile)

	// Ensure config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		ui.OutputLine("Configuration file not found. Creating default configuration...")
		mgr := config.NewManager(projectRoot)
		defaultCfg := config.DefaultConfig()
		if err := mgr.Save(defaultCfg); err != nil {
			return fmt.Errorf("failed to create default configuration: %w", err)
		}
	}

	// Find editor
	editor := findEditor()
	if editor == "" {
		return fmt.Errorf("no editor found. Please set the EDITOR environment variable")
	}

	ui.OutputLine("Opening configuration in %s...", editor)

	// Launch editor
	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}

	// Validate configuration after editing
	ui.OutputLine("Validating configuration...")
	mgr := config.NewManager(projectRoot)
	if _, err := mgr.Load(); err != nil {
		ui.Error("Configuration validation failed: %v", err)
		ui.OutputLine("Please fix the errors and try again.")
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	ui.Success("Configuration is valid!")
	return nil
}

// findEditor detects the editor to use in order of preference:
// 1. EDITOR environment variable
// 2. VISUAL environment variable
// 3. Common editors based on OS
func findEditor() string {
	// Check environment variables
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}

	// Try common editors
	var editors []string
	switch runtime.GOOS {
	case "darwin":
		// On macOS, try VS Code, Sublime, TextMate, then common Unix editors
		editors = []string{"code", "subl", "mate", "vim", "nano", "vi"}
	case "windows":
		editors = []string{"notepad", "notepad++"}
	default:
		// Linux and other Unix-like systems
		editors = []string{"vim", "nano", "vi"}
	}

	for _, editor := range editors {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	return ""
}
