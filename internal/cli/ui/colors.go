package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Color styles
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	InfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#0099FF"))
	WarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA00"))
	DimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	BoldStyle    = lipgloss.NewStyle().Bold(true)

	// Status indicators
	ActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	IdleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

	// Icons
	CaveIcon    = "🕳️"
	SuccessIcon = "✅"
	ErrorIcon   = "❌"
	InfoIcon    = "ℹ️"
	WarningIcon = "⚠️"
	ActiveIcon  = "🟢"
	IdleIcon    = "⚪"
)