package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// UI handles all user interface operations
type UI struct {
	verbose bool
}

// New creates a new UI instance
func New(verbose bool) *UI {
	return &UI{
		verbose: verbose,
	}
}

// PrintHeader prints a formatted header
func (u *UI) PrintHeader(title string) {
	width := 60
	border := strings.Repeat("═", width)

	fmt.Println()
	fmt.Println(color.CyanString(border))
	fmt.Println(color.CyanString(fmt.Sprintf("  %s", title)))
	fmt.Println(color.CyanString(border))
	fmt.Println()
}

// PrintPhase prints a phase header
func (u *UI) PrintPhase(phase string) {
	fmt.Println()
	fmt.Println(color.HiMagentaString(">>> %s", phase))
	fmt.Println()
}

// Info prints an info message
func (u *UI) Info(format string, args ...interface{}) {
	fmt.Printf(color.CyanString("ℹ ")+format+"\n", args...)
}

// Success prints a success message
func (u *UI) Success(format string, args ...interface{}) {
	fmt.Printf(color.GreenString("✓ ")+format+"\n", args...)
}

// Warn prints a warning message
func (u *UI) Warn(format string, args ...interface{}) {
	fmt.Printf(color.YellowString("⚠ ")+format+"\n", args...)
}

// Error prints an error message
func (u *UI) Error(format string, args ...interface{}) {
	fmt.Printf(color.RedString("✗ ")+format+"\n", args...)
}

// Debug prints a debug message (only if verbose is enabled)
func (u *UI) Debug(format string, args ...interface{}) {
	if u.verbose {
		fmt.Printf(color.HiBlackString("DEBUG: ")+format+"\n", args...)
	}
}

// Printf prints a formatted message
func (u *UI) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// Println prints a line
func (u *UI) Println(args ...interface{}) {
	fmt.Println(args...)
}

// PrintKeyValue prints a key-value pair
func (u *UI) PrintKeyValue(key, value string) {
	fmt.Printf("  %s: %s\n", color.CyanString(key), color.YellowString(value))
}

// PrintStep prints a step indicator
func (u *UI) PrintStep(step int, total int, message string) {
	fmt.Printf(color.HiCyanString("[%d/%d] ")+"%s\n", step, total, message)
}

// PrintSeparator prints a separator line
func (u *UI) PrintSeparator() {
	fmt.Println(color.HiBlackString(strings.Repeat("─", 60)))
}
