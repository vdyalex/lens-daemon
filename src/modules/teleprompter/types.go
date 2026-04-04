//go:generate mockgen -destination=../../tests/mocks/mock_teleprompter_service.go -package=mocks -source=types.go -mock_names Service=MockTeleprompterService Service

package teleprompter

// Service abstracts the teleprompter overlay for testability.
type Service interface {
	// Display updates the overlay text content.
	// text: the short answer to show on the teleprompter.
	Display(text string)

	// Toggle switches the overlay visibility (hidden <-> visible).
	// Returns true if the overlay is now visible, false if hidden.
	Toggle() bool
}

// Teleprompter manages a stealth macOS overlay window for displaying short answers.
type Teleprompter struct {
	visible bool
}
