package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/aki/agentcave/internal/cli/ui"
	"github.com/aki/agentcave/internal/core/config"
	"github.com/aki/agentcave/internal/mcp"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long:  "Start the Model Context Protocol server for AI agent integration",
	RunE:  runServe,
}

var (
	serveTransport string
	servePort      int
	serveAuthType  string
	serveAuthToken string
	serveAuthUser  string
	serveAuthPass  string
)

func init() {
	serveCmd.Flags().StringVarP(&serveTransport, "transport", "t", "", "Transport type (stdio, http, https)")
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 3000, "Port for HTTP/HTTPS transport")
	serveCmd.Flags().StringVar(&serveAuthType, "auth", "", "Authentication type (none, bearer, basic)")
	serveCmd.Flags().StringVar(&serveAuthToken, "auth-token", "", "Bearer token for authentication")
	serveCmd.Flags().StringVar(&serveAuthUser, "auth-user", "", "Username for basic authentication")
	serveCmd.Flags().StringVar(&serveAuthPass, "auth-pass", "", "Password for basic authentication")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Create configuration manager
	configManager := config.NewManager(projectRoot)

	// Load configuration
	cfg, err := configManager.Load()
	if err != nil {
		return err
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
		ui.Info("Shutting down MCP server...")
		cancel()
	}()

	// Start server
	ui.Info("Starting MCP server with %s transport", transport)

	if transport == "stdio" {
		ui.Info("Ready for AI agent connections via stdio")
		ui.Info("Press Ctrl+C to stop")
	}

	if err := server.Start(ctx); err != nil {
		if err == context.Canceled {
			ui.Success("MCP server stopped")
			return nil
		}
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
