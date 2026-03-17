// Package daemon provides daemon lifecycle management including PID file operations and process daemonization.
package daemon

// Options configures the daemonize operation.
type Options struct {
	// PIDPath is the path where the child's PID file will be written.
	// The child writes this itself on startup.
	PIDPath string
	// LogPath is an optional path to redirect child stdout/stderr.
	// If empty, output is discarded (os.DevNull).
	LogPath string
	// ExtraEnv is appended to the child's environment.
	// Use to pass LENSD_DAEMON_MODE=1 without polluting the current env.
	ExtraEnv []string
}
