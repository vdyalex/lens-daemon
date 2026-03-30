package cmd

// FormatUptime is the exported reference to the unexported formatUptimeSeconds function.
// It formats uptime duration in seconds to a human-readable string.
var FormatUptime = formatUptimeSeconds

// WithFlags temporarily sets global flags during fn, restoring original after.
// It is used by external tests (package cmd_test) to inject test flag values.
// The original flags are restored in a defer block to ensure cleanup even if fn panics.
func WithFlags(newFlags GlobalFlags, fn func()) {
	orig := flags
	flags = newFlags
	defer func() { flags = orig }()
	fn()
}
