package ui

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

// Confirm asks the user for confirmation
func (u *UI) Confirm(message string) bool {
	return u.ConfirmWithDefault(message, false)
}

// ConfirmWithDefault asks the user for confirmation with a default value
func (u *UI) ConfirmWithDefault(message string, defaultValue bool) bool {
	prompt := &survey.Confirm{
		Message: message,
		Default: defaultValue,
	}

	var result bool
	err := survey.AskOne(prompt, &result)
	if err != nil {
		// If there's an error (e.g., non-interactive terminal), use default
		return defaultValue
	}

	return result
}

// AskString asks the user for a string input
func (u *UI) AskString(message string, defaultValue string) string {
	prompt := &survey.Input{
		Message: message,
		Default: defaultValue,
	}

	var result string
	err := survey.AskOne(prompt, &result)
	if err != nil {
		return defaultValue
	}

	return result
}

// AskPassword asks the user for a password input
func (u *UI) AskPassword(message string) string {
	prompt := &survey.Password{
		Message: message,
	}

	var result string
	err := survey.AskOne(prompt, &result)
	if err != nil {
		return ""
	}

	return result
}

// AskSelect asks the user to select from a list of options
func (u *UI) AskSelect(message string, options []string) string {
	prompt := &survey.Select{
		Message: message,
		Options: options,
	}

	var result string
	err := survey.AskOne(prompt, &result)
	if err != nil {
		return ""
	}

	return result
}

// AskMultiSelect asks the user to select multiple items from a list
func (u *UI) AskMultiSelect(message string, options []string) []string {
	prompt := &survey.MultiSelect{
		Message: message,
		Options: options,
	}

	var result []string
	err := survey.AskOne(prompt, &result)
	if err != nil {
		return []string{}
	}

	return result
}

// PrintConfirmation prints a confirmation message with details
func (u *UI) PrintConfirmation(title string, items map[string]string) bool {
	fmt.Println()
	fmt.Println(color.CyanString("═══════════════════════════════════════"))
	fmt.Println(color.CyanString("  %s", title))
	fmt.Println(color.CyanString("═══════════════════════════════════════"))
	fmt.Println()

	for key, value := range items {
		u.PrintKeyValue(key, value)
	}

	fmt.Println()
	return u.Confirm("Proceed with this configuration?")
}
