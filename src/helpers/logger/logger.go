// Package logger provides terminal log formatting for the CLI logs command.
package logger

import (
	"strings"

	"github.com/pterm/pterm"

	"github.com/vdyalex/lens-daemon/src/ipc"
)

// New returns a pterm logger configured for log event display.
// Sets the minimum level to DEBUG so all events are rendered regardless of severity.
func New() *pterm.Logger {
	return pterm.DefaultLogger.WithLevel(pterm.LogLevelDebug)
}

// Print renders a single log event to the terminal using the provided pterm logger.
// Routes to the appropriate pterm method based on event.Level; unknown levels default to Info.
func Print(logger *pterm.Logger, event ipc.LogEvent) {
	args := logger.Args(AttrsToArgs(event.Attrs)...)
	switch event.Level {
	case "DEBUG":
		logger.Debug(event.Message, args)
	case "INFO":
		logger.Info(event.Message, args)
	case "WARN":
		logger.Warn(event.Message, args)
	case "ERROR":
		logger.Error(event.Message, args)
	default:
		logger.Info(event.Message, args)
	}
}

// AttrsToArgs converts a map of attributes into a flattened slice of alternating key-value pairs
// suitable for use with pterm logger methods. Values are sanitized to escape control characters.
func AttrsToArgs(attrs map[string]any) []any {
	pairs := make([]any, 0, len(attrs)*2)
	for key, value := range attrs {
		sanitized := SanitizeAttrValue(value)
		pairs = append(pairs, key, sanitized)
	}
	return pairs
}

// SanitizeAttrValue escapes control characters in string attribute values to prevent multiline output.
// Escapes: newline (\n), tab (\t), carriage return (\r), and backslash (\\).
// Non-string values are returned unchanged.
func SanitizeAttrValue(value any) any {
	s, ok := value.(string)
	if !ok {
		return value
	}
	// Escape backslash first to avoid double-escaping
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}
