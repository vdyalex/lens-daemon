//go:build darwin

package capturer

// ParseWindowInfo is the exported reference to the unexported parseWindowInfo function.
// It parses a comma-separated window info string into a WindowInfo struct.
var ParseWindowInfo = parseWindowInfo

// ComputeCaptureRect is the exported reference to the unexported computeCaptureRect function.
// It computes the final capture rectangle given window bounds, optional custom bounds, and screen dimensions.
var ComputeCaptureRect = computeCaptureRect
