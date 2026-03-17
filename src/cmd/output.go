package cmd

import "github.com/pterm/pterm"

// NewSpinner creates and starts a spinner with the given text.
// It swallows the non-fatal terminal-detection error from pterm.
func NewSpinner(text string) *pterm.SpinnerPrinter {
	spinner, _ := pterm.DefaultSpinner.WithText(text).Start()
	return spinner
}
