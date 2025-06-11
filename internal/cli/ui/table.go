package ui

import (
	"fmt"

	"github.com/rodaine/table"
)

// NewTable creates a new table with consistent styling
func NewTable(headers ...interface{}) table.Table {
	tbl := table.New(headers...)

	// Apply consistent formatting
	tbl.WithHeaderFormatter(func(format string, vals ...interface{}) string {
		return HeaderStyle.Render(fmt.Sprintf(format, vals...))
	})

	tbl.WithFirstColumnFormatter(func(format string, vals ...interface{}) string {
		return BoldStyle.Render(fmt.Sprintf(format, vals...))
	})

	// Add some padding
	tbl.WithPadding(2)

	return tbl
}

// PrintSectionHeader prints a consistent section header
func PrintSectionHeader(icon string, title string, count int) {
	fmt.Printf("\n%s %s (%d)\n\n", icon, title, count)

}
