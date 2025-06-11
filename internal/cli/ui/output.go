package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/aki/amux/internal/core/workspace"
)

// Print functions for consistent output

func Error(format string, args ...interface{}) {

	fmt.Fprintf(os.Stderr, "%s %s\n", ErrorIcon, ErrorStyle.Render(fmt.Sprintf(format, args...)))

}

func Success(format string, args ...interface{}) {

	fmt.Printf("%s %s\n", SuccessIcon, SuccessStyle.Render(fmt.Sprintf(format, args...)))

}

func Info(format string, args ...interface{}) {

	fmt.Printf("%s %s\n", InfoIcon, InfoStyle.Render(fmt.Sprintf(format, args...)))

}

func Warning(format string, args ...interface{}) {

	fmt.Printf("%s %s\n", WarningIcon, WarningStyle.Render(fmt.Sprintf(format, args...)))

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

	fmt.Printf("%s %s %s %s\n",

		"📁",

		BoldStyle.Render(w.Name),

		DimStyle.Render(fmt.Sprintf("(%s)", id)),

		DimStyle.Render(fmt.Sprintf("updated %s ago", ageStr)),
	)

	if w.Description != "" {

		fmt.Printf("   %s\n", w.Description)

	}

	fmt.Printf("   %s %s\n", DimStyle.Render("Branch:"), w.Branch)

	fmt.Printf("   %s %s\n", DimStyle.Render("Path:"), w.Path)

	if w.AgentID != "" {

		fmt.Printf("   %s %s\n", DimStyle.Render("Agent:"), w.AgentID)

	}

	fmt.Printf("   %s %s\n", DimStyle.Render("Created:"), FormatTime(w.CreatedAt))

	fmt.Printf("   %s %s\n", DimStyle.Render("Updated:"), FormatTime(w.UpdatedAt))

}

// FormatDuration formats a duration into a human-readable string
func FormatDuration(d time.Duration) string {

	if d < time.Minute {

		return "< 1m"

	} else if d < time.Hour {

		return fmt.Sprintf("%dm", int(d.Minutes()))

	} else if d < 24*time.Hour {

		return fmt.Sprintf("%dh", int(d.Hours()))

	} else {

		return fmt.Sprintf("%dd", int(d.Hours()/24))

	}

}

// PrintWorkspaceList displays a list of workspaces using a table
func PrintWorkspaceList(workspaces []*workspace.Workspace) {
	if len(workspaces) == 0 {
		Info("No workspaces found")
		return
	}

	// Create table
	tbl := NewTable("ID", "NAME", "BRANCH", "AGE", "DESCRIPTION")

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

		tbl.AddRow(id, w.Name, w.Branch, age, description)
	}

	// Print with header
	PrintSectionHeader(CaveIcon, "Workspaces", len(workspaces))
	tbl.Print()
	fmt.Println()
}

// PrintWorkspaceDetails displays detailed information about a single workspace
func PrintWorkspaceDetails(w *workspace.Workspace) {
	fmt.Printf("%s Workspace Details\n\n", CaveIcon)
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
