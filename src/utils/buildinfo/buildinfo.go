// Package buildinfo holds variables injected at compile time via -ldflags.
package buildinfo

// BinaryName is the name of the compiled binary.
// Override at build time: -ldflags "-X 'github.com/vdyalex/lens-daemon/src/utils/buildinfo.BinaryName=<name>'"
var BinaryName = "lensd"
