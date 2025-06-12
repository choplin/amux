// Package ui provides UI styling and output functions for the CLI.
package ui

import "github.com/charmbracelet/lipgloss"

var (
	// ErrorStyle is the style for error messages
	ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))

	// SuccessStyle is the style for success messages
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))

	// InfoStyle is the style for informational messages
	InfoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#0099FF"))

	// WarningStyle is the style for warning messages
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00"))

	// DimStyle is the style for dimmed text
	DimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	// BoldStyle is the style for bold text
	BoldStyle = lipgloss.NewStyle().Bold(true)

	// HeaderStyle is the style for headers
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))

	// WorkspaceIcon is the icon for workspaces
	WorkspaceIcon = "📋"

	// SuccessIcon is the icon for success messages
	SuccessIcon = "✅"

	// ErrorIcon is the icon for error messages
	ErrorIcon = "❌"

	// InfoIcon is the icon for informational messages
	InfoIcon = "ⓘ"

	// WarningIcon is the icon for warning messages
	WarningIcon = "⚠️"
)
