//go:build darwin

package capture

// HideProcess on macOS is a no-op at the Go level.
// The process has no GUI window or Dock icon since it's a pure CLI daemon.
// To hide from the Dock explicitly, the application would need an Info.plist
// with LSUIElement=true, but as a CLI binary this is not needed.
func HideProcess() {
	// No GUI elements are created, so nothing shows in Dock or Cmd-Tab.
}
