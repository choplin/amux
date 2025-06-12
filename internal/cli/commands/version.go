package commands

import (
	"runtime"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/spf13/cobra"
)

// Version information - these will be set at build time
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  "Display detailed version information about amux",
	Run: func(cmd *cobra.Command, args []string) {
		ui.OutputLine("amux version %s", Version)
		ui.OutputLine("  Git commit: %s", GitCommit)
		ui.OutputLine("  Build date: %s", BuildDate)
		ui.OutputLine("  Go version: %s", runtime.Version())
		ui.OutputLine("  OS/Arch:    %s/%s", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
