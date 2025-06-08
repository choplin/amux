package commands

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "agentcave",
	Short: "Private development caves for AI agents",
	Long: `AgentCave provides isolated git worktree-based environments where AI agents 
can work independently without context mixing.`,
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(workspaceCmd)
	rootCmd.AddCommand(serveCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}