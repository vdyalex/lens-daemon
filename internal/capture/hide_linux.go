//go:build linux

package capture

// HideProcess on Linux is a no-op at the Go level.
// The process has no GUI window, so it won't appear in Alt-Tab.
// To further hide from process lists, consider renaming the binary
// or running via a systemd service with a neutral name.
func HideProcess() {
	// No GUI window is created, so nothing appears in Alt-Tab or taskbar.
	// The process runs as a pure background daemon.
}
