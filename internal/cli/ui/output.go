package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aki/amux/internal/core/workspace"
)

// Error prints an error message with formatting
func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s %s\n", ErrorIcon, ErrorStyle.Render(fmt.Sprintf(format, args...)))
}

// Success prints a success message with formatting
func Success(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", SuccessIcon, SuccessStyle.Render(fmt.Sprintf(format, args...)))
}

// Info prints an informational message with formatting
func Info(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", InfoIcon, InfoStyle.Render(fmt.Sprintf(format, args...)))
}

// Warning prints a warning message with formatting
func Warning(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", WarningIcon, WarningStyle.Render(fmt.Sprintf(format, args...)))
}

// Output prints plain text without any formatting or icons
func Output(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// OutputLine prints plain text with a newline
func OutputLine(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

// PrintKeyValue prints a key-value pair with styling
func PrintKeyValue(key, value string) {
	OutputLine("%s %s", DimStyle.Render(key+":"), value)
}

// PrintIndented prints text with indentation
func PrintIndented(indent int, format string, args ...interface{}) {
	padding := strings.Repeat(" ", indent)
	OutputLine("%s%s", padding, fmt.Sprintf(format, args...))
}

// Separator prints a separator line
func Separator(char string, width int) {
	OutputLine("%s", strings.Repeat(char, width))
}

// Prompt displays a prompt and reads user input
func Prompt(message string) string {
	Output("%s", message)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// PrintTSV prints tab-separated values
func PrintTSV(rows [][]string) {
	for _, row := range rows {
		OutputLine("%s", strings.Join(row, "\t"))
	}
}

// Raw prints raw content without any processing
func Raw(content string) {
	Output("%s", content)
}

// PrintWorkspace displays a single workspace with formatting
func PrintWorkspace(w *workspace.Workspace) {
	// Calculate age from last modified time
	age := time.Since(w.UpdatedAt)
	ageStr := FormatDuration(age)

	id := w.ID
	if w.Index != "" {
		id = w.Index
	}

	OutputLine("%s %s %s %s",
		"📁",
		BoldStyle.Render(w.Name),
		DimStyle.Render(fmt.Sprintf("(%s)", id)),
		DimStyle.Render(fmt.Sprintf("updated %s ago", ageStr)),
	)

	if w.Description != "" {
		OutputLine("   %s", w.Description)
	}

	OutputLine("   %s %s", DimStyle.Render("Branch:"), w.Branch)
	OutputLine("   %s %s", DimStyle.Render("Path:"), w.Path)

	if w.StoragePath != "" {
		OutputLine("   %s %s", DimStyle.Render("Storage:"), w.StoragePath)
	}

	OutputLine("   %s %s", DimStyle.Render("Created:"), FormatTime(w.CreatedAt))
	OutputLine("   %s %s", DimStyle.Render("Updated:"), FormatTime(w.UpdatedAt))

	// Show consistency status
	var statusStr string
	switch w.Status {
	case workspace.StatusConsistent:
		statusStr = SuccessStyle.Render("✓ Consistent")
	case workspace.StatusFolderMissing:
		statusStr = WarningStyle.Render("⚠ Folder missing (run 'amux ws rm' to clean up)")
	case workspace.StatusWorktreeMissing:
		statusStr = WarningStyle.Render("⚠ Git worktree missing (run 'amux ws rm' to clean up)")
	case workspace.StatusOrphaned:
		statusStr = ErrorStyle.Render("✗ Orphaned (both folder and worktree missing)")
	default:
		statusStr = DimStyle.Render("Unknown")
	}
	OutputLine("   %s %s", DimStyle.Render("Status:"), statusStr)
}

// FormatDuration formats a duration into a human-readable string
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// PrintWorkspaceList displays a list of workspaces using a table
func PrintWorkspaceList(workspaces []*workspace.Workspace) {
	if len(workspaces) == 0 {
		Info("No workspaces found")
		return
	}

	// Create table
	tbl := NewTable("ID", "NAME", "BRANCH", "AGE", "STATUS", "DESCRIPTION")

	// Add rows
	for _, w := range workspaces {
		id := w.ID
		if w.Index != "" {
			id = w.Index
		}
		age := FormatDuration(time.Since(w.UpdatedAt))
		description := w.Description
		if description == "" {
			description = "-"
		}

		// Format status with appropriate icon
		var status string
		switch w.Status {
		case workspace.StatusConsistent:
			status = SuccessStyle.Render("✓ ok")
		case workspace.StatusFolderMissing:
			status = WarningStyle.Render("⚠ folder missing")
		case workspace.StatusWorktreeMissing:
			status = WarningStyle.Render("⚠ worktree missing")
		case workspace.StatusOrphaned:
			status = ErrorStyle.Render("✗ orphaned")
		default:
			status = DimStyle.Render("unknown")
		}

		tbl.AddRow(id, w.Name, w.Branch, age, status, description)
	}

	// Print with header
	PrintSectionHeader(WorkspaceIcon, "Workspaces", len(workspaces))
	tbl.Print()
}

// PrintWorkspaceDetails displays detailed information about a single workspace
func PrintWorkspaceDetails(w *workspace.Workspace) {
	OutputLine("%s Workspace Details\n", WorkspaceIcon)
	PrintWorkspace(w)
}

// FormatTime formats a time for display
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	now := time.Now()

	diff := now.Sub(t)

	switch {

	case diff < time.Minute:

		return "just now"

	case diff < time.Hour:

		minutes := int(diff.Minutes())

		if minutes == 1 {
			return "1 minute ago"
		}

		return fmt.Sprintf("%d minutes ago", minutes)

	case diff < 24*time.Hour:

		hours := int(diff.Hours())

		if hours == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", hours)

	case diff < 7*24*time.Hour:

		days := int(diff.Hours() / 24)

		if days == 1 {
			return "1 day ago"
		}

		return fmt.Sprintf("%d days ago", days)

	default:

		return t.Format("2006-01-02 15:04")

	}
}

// FormatSize formats a file size in bytes to a human-readable string
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes < KB:
		return fmt.Sprintf("%dB", bytes)
	case bytes < MB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/KB)
	case bytes < GB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/MB)
	default:
		return fmt.Sprintf("%.1fGB", float64(bytes)/GB)
	}
}

// Confirm asks the user for confirmation
func Confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
