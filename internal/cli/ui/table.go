package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/rodaine/table"
)

// NewTable creates a new table with consistent styling
func NewTable(headers ...interface{}) table.Table {
	tbl := table.New(headers...)

	// Don't use header formatter as it causes layout issues
	// Instead, we'll style the headers when we create them

	// Only format the first column (ID) with bold
	tbl.WithFirstColumnFormatter(func(format string, vals ...interface{}) string {
		return BoldStyle.Render(fmt.Sprintf(format, vals...))
	})

	// Add some padding
	tbl.WithPadding(2)

	// Use lipgloss Width function to properly calculate string width with ANSI codes
	tbl.WithWidthFunc(lipgloss.Width)

	return tbl
}

// PrintSectionHeader prints a consistent section header
func PrintSectionHeader(icon string, title string, count int) {
	OutputLine("\n%s %s (%d)", icon, title, count)
}
