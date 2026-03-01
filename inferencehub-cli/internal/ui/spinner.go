package ui

import (
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// activeSpinner holds the currently active spinner
var activeSpinner *spinner.Spinner

// StartSpinner starts a spinner with a message
func (u *UI) StartSpinner(message string) {
	if activeSpinner != nil {
		activeSpinner.Stop()
	}

	activeSpinner = spinner.New(
		spinner.CharSets[11], // Use character set 11 (dots)
		100*time.Millisecond,
		spinner.WithColor("cyan"),
		spinner.WithSuffix(color.CyanString(" "+message)),
	)
	activeSpinner.Start()
}

// UpdateSpinner updates the spinner message
func (u *UI) UpdateSpinner(message string) {
	if activeSpinner != nil {
		activeSpinner.Suffix = color.CyanString(" " + message)
	}
}

// StopSpinner stops the spinner with a success or failure indicator
func (u *UI) StopSpinner(success bool) {
	if activeSpinner != nil {
		activeSpinner.Stop()
		activeSpinner = nil
	}
}

// StopSpinnerWithMessage stops the spinner and prints a message
func (u *UI) StopSpinnerWithMessage(success bool, message string) {
	u.StopSpinner(success)
	if success {
		u.Success(message)
	} else {
		u.Error(message)
	}
}
