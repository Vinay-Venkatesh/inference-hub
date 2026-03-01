package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// TableRow represents a row in a table
type TableRow struct {
	Columns []string
}

// Table represents a formatted table
type Table struct {
	Headers []string
	Rows    []TableRow
}

// PrintTable prints a formatted table
func (u *UI) PrintTable(table Table) {
	if len(table.Headers) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(table.Headers))
	for i, header := range table.Headers {
		widths[i] = len(header)
	}

	for _, row := range table.Rows {
		for i, col := range row.Columns {
			if i < len(widths) && len(col) > widths[i] {
				widths[i] = len(col)
			}
		}
	}

	// Print headers
	u.printTableRow(table.Headers, widths, true)

	// Print separator
	u.printTableSeparator(widths)

	// Print rows
	for _, row := range table.Rows {
		u.printTableRow(row.Columns, widths, false)
	}

	fmt.Println()
}

func (u *UI) printTableRow(columns []string, widths []int, isHeader bool) {
	parts := make([]string, len(columns))
	for i, col := range columns {
		if i < len(widths) {
			parts[i] = u.padString(col, widths[i])
		}
	}

	if isHeader {
		fmt.Println(color.CyanString("│ " + strings.Join(parts, " │ ") + " │"))
	} else {
		fmt.Println("│ " + strings.Join(parts, " │ ") + " │")
	}
}

func (u *UI) printTableSeparator(widths []int) {
	parts := make([]string, len(widths))
	for i, width := range widths {
		parts[i] = strings.Repeat("─", width)
	}
	fmt.Println(color.HiBlackString("├─" + strings.Join(parts, "─┼─") + "─┤"))
}

func (u *UI) padString(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// PrintStatusTable prints a status table with colored status indicators
func (u *UI) PrintStatusTable(items []StatusItem) {
	for _, item := range items {
		statusColor := color.GreenString("✓")
		if !item.Success {
			statusColor = color.RedString("✗")
		} else if item.Warning {
			statusColor = color.YellowString("⚠")
		}

		fmt.Printf("%s %s: %s\n", statusColor, color.CyanString(item.Name), item.Message)
		if item.Details != "" {
			fmt.Printf("  %s\n", color.HiBlackString(item.Details))
		}
	}
	fmt.Println()
}

// StatusItem represents an item in a status display
type StatusItem struct {
	Name    string
	Status  string
	Message string
	Details string
	Success bool
	Warning bool
}
