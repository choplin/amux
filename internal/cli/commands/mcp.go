package commands

import (
	"context"

	"fmt"

	"os"

	"os/signal"

	"path/filepath"

	"syscall"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"

	"github.com/aki/amux/internal/core/config"

	"github.com/aki/amux/internal/core/git"

	"github.com/aki/amux/internal/mcp"
)

var mcpCmd = &cobra.Command{

	Use: "mcp",

	Short: "Start the MCP server",

	Long: "Start the Model Context Protocol server for AI agent integration",

	RunE: runMCP,
}

var (
	serveTransport string

	servePort int

	serveAuthType string

	serveAuthToken string

	serveAuthUser string

	serveAuthPass string

	rootDir string
)

func init() {

	mcpCmd.Flags().StringVar(&rootDir, "root-dir", "", "Project root directory (required if not in amux project)")

	mcpCmd.Flags().StringVarP(&serveTransport, "transport", "t", "", "Transport type (stdio, http, https)")

	mcpCmd.Flags().IntVarP(&servePort, "port", "p", 3000, "Port for HTTP/HTTPS transport")

	mcpCmd.Flags().StringVar(&serveAuthType, "auth", "", "Authentication type (none, bearer, basic)")

	mcpCmd.Flags().StringVar(&serveAuthToken, "auth-token", "", "Bearer token for authentication")

	mcpCmd.Flags().StringVar(&serveAuthUser, "auth-user", "", "Username for basic authentication")

	mcpCmd.Flags().StringVar(&serveAuthPass, "auth-pass", "", "Password for basic authentication")

}

func runMCP(cmd *cobra.Command, args []string) error {

	// Determine project root
	var projectRoot string
	var err error

	if rootDir != "" {
		// Use explicitly provided root directory
		absPath, err := filepath.Abs(rootDir)
		if err != nil {
			return fmt.Errorf("invalid root directory: %w", err)
		}
		projectRoot = absPath

		// Validate it's a git repository
		gitOps := git.NewOperations(projectRoot)
		if !gitOps.IsGitRepository() {
			return fmt.Errorf("--root-dir must be a git repository: %s", projectRoot)
		}
	} else {
		// Try to find project root from current directory
		projectRoot, err = config.FindProjectRoot()
		if err != nil {
			return fmt.Errorf("not in an amux project and --root-dir not specified")
		}
	}

	// Create configuration manager

	configManager := config.NewManager(projectRoot)

	// Load configuration or use defaults

	cfg, err := configManager.Load()

	if err != nil {

		// If config doesn't exist, use defaults for MCP server
		cfg = config.DefaultConfig()

	}

	// Determine transport

	transport := serveTransport

	if transport == "" {

		transport = cfg.MCP.Transport.Type

	}

	// Build HTTP config if needed

	var httpConfig *config.HTTPConfig

	if transport == "http" || transport == "https" {

		httpConfig = &config.HTTPConfig{

			Port: servePort,
		}

		// Configure authentication

		if serveAuthType != "" {

			httpConfig.Auth.Type = serveAuthType

			switch serveAuthType {

			case "bearer":

				if serveAuthToken == "" {

					return fmt.Errorf("bearer token required for bearer authentication")

				}

				httpConfig.Auth.Bearer = serveAuthToken

			case "basic":

				if serveAuthUser == "" || serveAuthPass == "" {

					return fmt.Errorf("username and password required for basic authentication")

				}

				httpConfig.Auth.Basic.Username = serveAuthUser

				httpConfig.Auth.Basic.Password = serveAuthPass

			}

		} else if cfg.MCP.Transport.HTTP.Port != 0 {

			// Use config file settings

			httpConfig = &cfg.MCP.Transport.HTTP

		}

	}

	// Create MCP server (using new mcp-go implementation)

	server, err := mcp.NewServerV2(configManager, transport, httpConfig)

	if err != nil {

		return fmt.Errorf("failed to create MCP server: %w", err)

	}

	// Setup context with signal handling

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	// Handle shutdown signals

	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {

		<-sigChan

		if transport == "stdio" {

			fmt.Fprintf(os.Stderr, "Shutting down MCP server...\n")

		} else {

			ui.Info("Shutting down MCP server...")

		}

		cancel()

	}()

	// Start server

	if transport == "stdio" {

		// For stdio transport, all UI output must go to stderr
		// to avoid interfering with the MCP protocol on stdout
		fmt.Fprintf(os.Stderr, "Starting MCP server with stdio transport\n")

		fmt.Fprintf(os.Stderr, "Ready for AI agent connections via stdio\n")

		fmt.Fprintf(os.Stderr, "Debug logs will appear here. Press Ctrl+C to stop\n")

	} else {

		ui.Info("Starting MCP server with %s transport", transport)

	}

	if err := server.Start(ctx); err != nil {

		if err == context.Canceled {

			if transport == "stdio" {

				fmt.Fprintf(os.Stderr, "MCP server stopped\n")

			} else {

				ui.Success("MCP server stopped")

			}

			return nil

		}

		// For stdio, log errors to stderr
		if transport == "stdio" {

			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)

		}

		return fmt.Errorf("server error: %w", err)

	}

	return nil

}
